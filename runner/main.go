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
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/push"
)

const (
	awsAccountEnv       = "AWS_ACCOUNT"
	awsCallerArnEnv     = "AWS_CALLER_ARN"
	awsCallerIdEnv      = "AWS_CALLER_ID"
	awsDefaultRegionEnv = "AWS_DEFAULT_REGION"
	awsRegionEnv        = "AWS_REGION"

	dockerHostEnv        = "DOCKER_HOST"
	dockerAuthsEnv       = "DOCKER_AUTHS"
	dockerCredHelpersEnv = "DOCKER_CREDENTIAL_HELPERS"
	dockerPluginsEnv     = "DOCKER_PLUGINS"

	dockerConfigAuthsKey       = "auths"
	dockerConfigCredHelpersKey = "credHelpers"
	dockerConfigPluginsKey     = "plugins"

	prometheusPushGatewayAddr = "push-gateway.prometheus.svc.cluster.local:9091"
	prometheusPushJob         = "kubeactions_runner"

	runnerListenerProcessName = "Runner.Listener"
)

var (
	logger = log.New(os.Stdout, "kube-actions[runner]: ", log.LstdFlags)

	gitHubActionsRunnerPath = "/opt/actions-runner/run.sh"
	gitHubActionsRunnerArgs = []string{"--once"}

	arRepositoryOwner = os.Getenv("KUBEACTIONS_ACTIONSRUNNER_REPOSITORY_OWNER")
	arRepositoryName  = os.Getenv("KUBEACTIONS_ACTIONSRUNNER_REPOSITORY_NAME")
	arRepository      = arRepositoryOwner + "/" + arRepositoryName

	arjName = os.Getenv("KUBEACTIONS_ACTIONSRUNNERJOB_NAME")

	_, hasDockerCapability = os.LookupEnv(dockerHostEnv)

	runnerRunningGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "runner",
			Name:      "job_running",
		},
		[]string{"repository", "runner_job"},
	)

	runnerStartedTimestampGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "runner",
			Name:      "job_started",
		},
		[]string{"repository", "runner_job"},
	)

	runnerFinishedTimestampGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubeactions",
			Subsystem: "runner",
			Name:      "job_finished",
		},
		[]string{"repository", "runner_job"},
	)

	collectorsToPush = []prometheus.Collector{
		runnerRunningGauge,
		runnerStartedTimestampGauge,
		runnerFinishedTimestampGauge,
	}
)

func main() {
	logger.Println("Initializing runner")
	ctx := context.Background()

	updateCaCertificatesC := async(func() error {
		return updateCaCertificates()
	})

	setupGitCredentialsC := async(func() error {
		return setupGitCredentials()
	})

	ensureAwsAndDockerEnvC := async(func() error {
		if err := ensureAwsEnv(ctx); err != nil {
			return err
		}
		return setupDockerConfig()
	})

	waitForDockerC := async(func() error {
		return waitForDocker()
	})

	for _, c := range []chan error{updateCaCertificatesC, setupGitCredentialsC, ensureAwsAndDockerEnvC, waitForDockerC} {
		if err := <-c; err != nil {
			panic(err)
		}
	}

	defer func() {
		if err := requestDindTermination(); err != nil {
			logger.Println(err)
		}
	}()

	runnerRunningGauge.WithLabelValues(arRepository, arjName).Set(1)
	runnerStartedTimestampGauge.WithLabelValues(arRepository, arjName).SetToCurrentTime()
	if err := pushMetrics(); err != nil {
		panic(err)
	}

	defer func() {
		runnerRunningGauge.WithLabelValues(arRepository, arjName).Set(0)
		runnerFinishedTimestampGauge.WithLabelValues(arRepository, arjName).SetToCurrentTime()
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
	if err := <-run("sudo", "git", "config", "--system", "user.name", "github-actions[bot]"); err != nil {
		return errors.Wrap(err, "Error setting system username for git")
	}

	if err := <-run("sudo", "git", "config", "--system", "user.email", "41898282+github-actions[bot]@users.noreply.github.com"); err != nil {
		return errors.Wrap(err, "Error setting system email for git")
	}

	return nil
}

func ensureAwsEnv(ctx context.Context) error {
	logger.Println("Ensuring AWS Environment")
	envVars := make(map[string]string)

	logger.Println("Detecting AWS Region")
	awsRegion := detectAwsRegion(ctx)
	envVars[awsRegionEnv] = awsRegion

	logger.Println("Loading AWS Configuration with Region")
	config, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(awsRegion))
	if err != nil {
		return errors.Wrap(err, "Error loading AWS Configuration with Region")
	}

	logger.Println("Getting AWS Caller Identity")
	stsClient := sts.NewFromConfig(config)
	stsOutput, err := stsClient.GetCallerIdentity(ctx, nil)
	if err != nil {
		logger.Println(errors.Wrap(err, "Error getting AWS Caller Identity"))
	}
	if stsOutput != nil {
		if account := stsOutput.Account; account != nil {
			envVars[awsAccountEnv] = *account
		}
		if arn := stsOutput.Arn; arn != nil {
			envVars[awsCallerArnEnv] = *arn
		}
		if userId := stsOutput.UserId; userId != nil {
			envVars[awsCallerIdEnv] = *userId
		}
	}

	for k, v := range envVars {
		logger.Printf("Setting %s to %s\n", k, v)
		if err := os.Setenv(k, v); err != nil {
			return errors.Wrapf(err, "Error setting %s to %s", k, v)
		}
	}

	return nil
}

func detectAwsRegion(ctx context.Context) string {
	if awsRegion, ok := os.LookupEnv(awsRegionEnv); ok {
		logger.Println("Detected AWS Region from AWS_REGION")
		return awsRegion
	}
	logger.Println("AWS_REGION is not defined")

	if awsDefaultRegion, ok := os.LookupEnv(awsDefaultRegionEnv); ok {
		logger.Println("Detected AWS Region from AWS_DEFAULT_REGION")
		return awsDefaultRegion
	}
	logger.Println("AWS_DEFAULT_REGION is not defined")

	imdsClient := imds.New(imds.Options{
		ClientEnableState: imds.ClientEnabled,
	})
	imdsOutput, err := imdsClient.GetRegion(ctx, nil)
	if err != nil {
		logger.Println(errors.Wrap(err, "AWS IMDS is unavailable"))
	}
	if imdsOutput != nil {
		logger.Println("Detected AWS Region from AWS IMDS")
		return imdsOutput.Region
	}

	logger.Println("Detected AWS Region from Hardcoded Fallback")
	return "us-east-1"
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
	prometheusPusher := push.New(prometheusPushGatewayAddr, prometheusPushJob)
	for _, collector := range collectorsToPush {
		prometheusPusher = prometheusPusher.Collector(collector)
	}

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
		gracefullyShutdownRunnerListener()
		return errors.Wrap(err, "Error waiting for GitHub Actions Runner process")
	}

	return nil
}

// gracefullyShutdownRunnerListener tries to send a SIGINT to the Runner.Listener
// process so the workflow can be properly cancelled
func gracefullyShutdownRunnerListener() {
	if err := <-run("pkill", "-SIGINT", runnerListenerProcessName); err != nil {
		logger.Println(errors.Wrap(err, "Error gracefully terminating Runner.Listener process"))
	}
}
