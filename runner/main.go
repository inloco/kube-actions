package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
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
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	awsRegionEnv    = "AWS_REGION"
	awsAccountEnv   = "AWS_ACCOUNT"
	awsCallerArnEnv = "AWS_CALLER_ARN"
	awsCallerIdEnv  = "AWS_CALLER_ID"

	dockerHostEnv = "DOCKER_HOST"
	dockerConfigEnv = "DOCKER_CONFIG"
	dockerAuthsEnv = "DOCKER_AUTHS"
	dockerCredHelpersEnv = "DOCKER_CREDENTIAL_HELPERS"
	dockerPluginsEnv = "DOCKER_PLUGINS"

	dockerConfigAuthsKey = "auths"
	dockerConfigCredHelpersKey = "credHelpers"
	dockerConfigPluginsKey = "plugins"

	prometheusAddr = ":9102"
	prometheusMetricsPath = "/metrics"
)

var (
	logger = log.New(os.Stdout, "kube-actions[runner]: ", log.LstdFlags)

	gitHubActionsRunnerPath = "/opt/actions-runner/run.sh"
	gitHubActionsRunnerArgs = []string{"--once"}

	runnerRepository = os.Getenv("RUNNER_REPOSITORY")
	runnerJob = os.Getenv("RUNNER_JOB")

	runnerRunningGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "actions",
			Name: "job_running",
		},
		[]string{"repository", "job"},
	)
)

func main() {
	logger.Println("Initializing runner")
	ctx := context.Background()

	logger.Println("Updating CA certificates")
	if err := updateCaCertificates(); err != nil {
		panic(err)
	}

	logger.Println("Assuring AWS environment")
	if err := assureAwsEnv(ctx); err != nil {
		panic(err)
	}

	logger.Println("Waiting Docker daemon")
	if err := waitForDocker(); err != nil {
		panic(err)
	}

	logger.Println("Preparing Docker config")
	if err := setupDockerConfig(); err != nil {
		panic(err)
	}

	logger.Println("Starting metrics server")
	if err := startMetricsServer(); err != nil {
		panic(err)
	}

	defer func() {
		if err := requestDindTermination(); err != nil {
			logger.Println(err)
		}
	}()

	runnerRunningGauge.WithLabelValues(runnerRepository, runnerJob).Set(1)

	logger.Println("Running GitHub Actions Runner")
	if err := runGitHubActionsRunner(); err != nil {
		panic(err)
	}
}

func updateCaCertificates() error {
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

func startMetricsServer() error {
	http.Handle(prometheusMetricsPath, promhttp.Handler())
	errC := make(chan error)
	go func() {
		errC <- http.ListenAndServe(prometheusAddr, nil)
	}()

	select {
	case err := <-errC:
		return err
	case <-time.After(time.Second):
		return nil
	}
}

func requestDindTermination() error {
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
