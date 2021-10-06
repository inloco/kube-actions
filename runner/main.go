package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	dockercliconfig "github.com/docker/cli/cli/config"
	dockercliconfigfile "github.com/docker/cli/cli/config/configfile"
	dockerconfig "github.com/docker/docker/cli/config"
	docker "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

const (
	awsRegionEnv    = "AWS_REGION"
	awsAccountEnv   = "AWS_ACCOUNT"
	awsCallerArnEnv = "AWS_CALLER_ARN"
	awsCallerIdEnv  = "AWS_CALLER_ID"

	dockerHostEnv = "DOCKER_HOST"
	dockerAuthsEnv = "DOCKER_AUTHS"
	dockerCredHelpersEnv = "DOCKER_CREDENTIAL_HELPERS"
	dockerPluginsEnv = "DOCKER_PLUGINS"

	dockerConfigAuthsKey = "auths"
	dockerConfigCredHelpersKey = "credHelpers"
	dockerConfigPluginsKey = "plugins"

	prometheusPushGatewayAddr = "push-gateway.prometheus.svc.cluster.local:9091"
)

var (
	logger = log.New(os.Stdout, "kube-actions[runner]: ", log.LstdFlags)

	gitHubActionsRunnerPath = "/opt/actions-runner/run.sh"
	gitHubActionsRunnerArgs = []string{"--once"}

	runnerRepository = os.Getenv("RUNNER_REPOSITORY")
	runnerJob = os.Getenv("RUNNER_JOB")

	_, hasDockerCapability = os.LookupEnv(dockerHostEnv)

	runnerRunningGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "runner",
			Name: "job_running",
		},
		[]string{"runner_repository", "runner_job"},
	)

	runnerStartedTimestampGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "runner",
			Name: "job_started",
		},
		[]string{"runner_repository", "runner_job"},
	)

	runnerFinishedTimestampGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "runner",
			Name: "job_finished",
		},
		[]string{"runner_repository", "runner_job"},
	)

	prometheusPusher = push.New(prometheusPushGatewayAddr, "kubeactions_runner").
		Collector(runnerRunningGauge).
		Collector(runnerStartedTimestampGauge).
		Collector(runnerFinishedTimestampGauge)
)

func main() {
	logger.Println("Initializing runner")
	ctx := context.Background()

	updateCaCertificatesC := async(func() error {
		return updateCaCertificates()
	})

	assureAwsEnvC := async(func() error {
		return assureAwsEnv(ctx)
	})

	waitForDockerC := async(func() error {
		return waitForDocker()
	})

	setupDockerConfigC := async(func() error {
		return setupDockerConfig()
	})

	for _, c := range []chan error{updateCaCertificatesC, assureAwsEnvC, waitForDockerC, setupDockerConfigC} {
		if err := <-c; err != nil {
			panic(err)
		}
	}

	defer func() {
		if err := requestDindTermination(); err != nil {
			logger.Println(err)
		}
	}()

	runnerRunningGauge.WithLabelValues(runnerRepository, runnerJob).Set(1)
	runnerStartedTimestampGauge.WithLabelValues(runnerRepository, runnerJob).SetToCurrentTime()
	if err := pushMetrics(); err != nil {
		panic(err)
	}

	defer func() {
		runnerRunningGauge.WithLabelValues(runnerRepository, runnerJob).Set(0)
		runnerFinishedTimestampGauge.WithLabelValues(runnerRepository, runnerJob).SetToCurrentTime()
		if err := pushMetrics(); err != nil {
			logger.Println(err)
		}
	}()

	logger.Println("Running GitHub Actions Runner")
	if err := runGitHubActionsRunner(); err != nil {
		panic(err)
	}
}

func updateCaCertificates() error {
	logger.Println("Updating CA certificates")

	if err := <-run("sudo", "update-ca-certificates"); err != nil {
		return errors.Wrap(err, "Error running update-ca-certificates")
	}

	return nil
}

func setupGitCredentials() error {
	if err := <-run("git", "config", "--global", "user.name", "github-actions[bot]"); err != nil {
		return errors.Wrap(err, "Error setting global username for git")
	}

	if err := <-run("git", "config", "--global", "user.email", "41898282+github-actions[bot]@users.noreply.github.com"); err != nil {
		return errors.Wrap(err, "Error setting global email for git")
	}

	return nil
}

func assureAwsEnv(ctx context.Context) error {
	logger.Println("Assuring AWS environment")

	awsRegion, ok := os.LookupEnv(awsRegionEnv)
	if !ok {
		logger.Println("AWS_REGION not present, calling metadata server")
		imdsClient := imds.New(imds.Options{})
		output, err := imdsClient.GetRegion(ctx, nil)
		if err != nil {
			logger.Printf("Error creating aws imds client: %v\n", err)
			return nil
		}

		logger.Printf("Detected aws region: %s\n", output.Region)
		awsRegion = output.Region
		if err := os.Setenv(awsRegionEnv, awsRegion); err != nil {
			return errors.Wrapf(err, "Error setting %s env var", awsRegionEnv)
		}
	}

	logger.Println("Loading aws configuration with credentials")
	config, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(awsRegion))
	if err != nil {
		return errors.Wrap(err, "Error loading aws config")
	}

	logger.Println("Retrieving aws caller identity")
	stsClient := sts.NewFromConfig(config)
	output, err := stsClient.GetCallerIdentity(ctx, nil)
	if err != nil {
		logger.Printf("Error requesting STS caller identity: %v\n", err)
		return nil
	}

	logger.Printf("Detected aws account: %s\n", *output.Account)
	if err := os.Setenv(awsAccountEnv, *output.Account); err != nil {
		return errors.Wrapf(err, "Error setting %s env var", awsAccountEnv)
	}

	logger.Printf("Detected aws arn: %s\n", *output.Arn)
	if err := os.Setenv(awsCallerArnEnv, *output.Arn); err != nil {
		return errors.Wrapf(err, "Error setting %s env var", awsCallerArnEnv)
	}

	logger.Printf("Detected aws caller id: %s\n", *output.UserId)
	if err := os.Setenv(awsCallerIdEnv, *output.UserId); err != nil {
		return errors.Wrapf(err, "Error setting %s env var", awsCallerIdEnv)
	}

	return nil
}

func waitForDocker() error {
	if !hasDockerCapability {
		return nil
	}

	logger.Println("Waiting Docker daemon")

	docker, err := docker.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "Error creating docker client")
	}

	for i := 0; i < 15; i++ {
		_, err = docker.ServerVersion(context.Background())
		if err == nil {
			logger.Println("Docker daemon responded successfully")
			break
		}

		logger.Println("Could not connect to docker daemon, trying again...")
		time.Sleep(time.Second)
	}

	return errors.Wrap(err, "Timeout waiting for docker daemon")
}

func setupDockerConfig() error {
	if !hasDockerCapability {
		return nil
	}

	logger.Println("Preparing Docker config")

	dockerConfigDir := dockerconfig.Dir()
	if err := os.MkdirAll(dockerConfigDir, 0700); err != nil {
		return errors.Wrap(err, "Error assuring existence of docker config dir")
	}

	dockerConfigPath := path.Join(dockerConfigDir, "config.json")
	dockerConfig := dockercliconfigfile.New(dockerConfigPath)
	if _, err := os.Stat(dockerConfigPath); err == nil {
		file, err := os.Open(dockerConfigPath)
		defer file.Close()
		if err != nil {
			return errors.Wrap(err, "Error opening docker config file")
		}

		config, err := dockercliconfig.LoadFromReader(file)
		if err != nil {
			return errors.Wrap(err, "Error reading docker config file")
		}
		dockerConfig = config
	}

	if auths, ok := os.LookupEnv(dockerAuthsEnv); ok {
		logger.Println("Loading DOCKER_AUTHS")
		if err := dockerConfig.LoadFromReader(strings.NewReader(wrapInJson(dockerConfigAuthsKey, auths))); err != nil {
			return errors.Wrap(err, "Error reading config from DOCKER_AUTHS")
		}
	}

	if credHelpers, ok := os.LookupEnv(dockerCredHelpersEnv); ok {
		logger.Println("Loading DOCKER_CREDENTIAL_HELPERS")
		if err := dockerConfig.LoadFromReader(strings.NewReader(wrapInJson(dockerConfigCredHelpersKey, credHelpers))); err != nil {
			return errors.Wrap(err, "Error reading config from DOCKER_CREDENTIAL_HELPERS")
		}
	}

	if plugins, ok := os.LookupEnv(dockerPluginsEnv); ok {
		logger.Println("Loading DOCKER_PLUGINS")
		if err := dockerConfig.LoadFromReader(strings.NewReader(wrapInJson(dockerConfigPluginsKey, plugins))); err != nil {
			return errors.Wrap(err, "Error reading config from DOCKER_PLUGINS")
		}
	}

	if err := dockerConfig.Save(); err != nil {
		return errors.Wrap(err, "Error writing new docker config to file")
	}

	return nil
}

func wrapInJson(key, value string) string {
	return fmt.Sprintf(`{ "%s": %s }`, key, os.ExpandEnv(value))
}

func pushMetrics() error {
	if err := prometheusPusher.Push(); err != nil {
		panic(errors.Wrap(err, "Error pushing metrics to Prometheus' Push Gateway"))
	}

	return nil
}

func requestDindTermination() error {
	if !hasDockerCapability {
		return nil
	}

	logger.Println("Requesting dind termination")
	conn, err := net.Dial("tcp", ":2378")

	if conn != nil {
		defer conn.Close()
	}

	if err != nil {
		return errors.Wrap(err, "Error dialing dind termination endpoint")
	}

	return nil
}

func runGitHubActionsRunner() error {
	if err := <-run(gitHubActionsRunnerPath, gitHubActionsRunnerArgs...); err != nil {
		return errors.Wrap(err, "Error waiting for GitHub Actions Runner process")
	}

	return nil
}
