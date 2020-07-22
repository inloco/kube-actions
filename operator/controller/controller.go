package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/inloco/kube-actions/operator/models"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/pkg/apis/inloco/v1alpha1"
	"github.com/inloco/kube-actions/operator/util"
	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
	restclient "k8s.io/client-go/rest"
	"k8s.io/utils/strings"
)

type RunnerController struct {
	lock chan struct{}
	loop bool

	context context.Context
	runner  models.Runner

	k8sFacade facadeKubernetes
	ghFacade  facadeGitHub
	adoFacade facadeAzureDevOps
}

func NewRunnerController(k8sConfig *restclient.Config, actionsRunner *inlocov1alpha1.ActionsRunner) (*RunnerController, error) {
	if actionsRunner == nil {
		return nil, errors.New("actionsRunner == nil")
	}

	ctx := context.Background()
	runnerController := &RunnerController{
		lock:    make(chan struct{}, 1),
		context: ctx,
		k8sFacade: facadeKubernetes{
			context:       ctx,
			k8sConfig:     k8sConfig,
			actionsRunner: actionsRunner,
		},
		ghFacade: facadeGitHub{
			context: ctx,
		},
		adoFacade: facadeAzureDevOps{
			context: ctx,
		},
	}
	runnerController.lock <- struct{}{}

	return runnerController, nil
}

func (rc *RunnerController) Loop() error {
	select {
	case <-rc.lock:
		defer func() {
			rc.lock <- struct{}{}
		}()

	default:
		return errors.New("instance is locked")
	}

	log.Printf("%s/%s: init", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.init(); err != nil {
		return err
	}

	log.Printf("%s/%s: Apply", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.k8sFacade.Apply(&rc.runner); err != nil {
		return err
	}

	log.Printf("%s/%s: CreateAgentSession", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.adoFacade.InitAzureDevOpsTaskAgentSession(); err != nil {
		return err
	}

	defer func() {
		log.Printf("%s/%s: DeinitAzureDevOpsTaskAgentSession()", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
		if err := rc.adoFacade.DeinitAzureDevOpsTaskAgentSession(); err != nil {
			log.Panic(err)
		}
	}()

	rc.loop = true
	for rc.loop {
		log.Printf("%s/%s: GetMessage", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
		message, err := rc.adoFacade.GetMessage()
		if err != nil {
			return err
		}

		log.Printf("%s/%s: consumeMessage", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
		if err := rc.consumeMessage(message); err != nil {
			continue
		}
	}

	return nil
}

func (rc *RunnerController) init() error {
	if err := rc.k8sFacade.Init(); err != nil {
		return err
	}

	if err := rc.initRunnerSettings(); err != nil {
		return err
	}

	if err := rc.initCredentials(); err != nil {
		return err
	}

	if err := rc.initRSAParameters(); err != nil {
		return err
	}

	if err := rc.ghFacade.Init(githubPAT, rc.k8sFacade.actionsRunner.Spec.Repository.Owner, rc.k8sFacade.actionsRunner.Spec.Repository.Name); err != nil {
		return err
	}
	rc.runner.RunnerSettings.GitHubUrl = rc.ghFacade.ghRepository.GetGitCommitsURL()

	credential, err := rc.ghFacade.GetGitHubTenantCredential(RunnerEventRegister)
	if err != nil {
		return err
	}
	rc.runner.RunnerSettings.ServerUrl = credential.GetURL()

	if err := rc.adoFacade.Init(credential.GetToken(), credential.GetURL(), &rc.runner, append(agentLabelsBase, rc.k8sFacade.actionsRunner.Spec.Labels...)); err != nil {
		return err
	}

	return nil
}

func (rc *RunnerController) initRunnerSettings() error {
	if rc.k8sFacade.actionsRunner == nil {
		return errors.New(".actionsRunner == nil")
	}

	if rc.k8sFacade.configMap == nil {
		return errors.New(".configMap == nil")
	}

	dotRunner, ok := rc.k8sFacade.configMap.BinaryData[".runner"]
	if !ok {
		return errors.New(".runner not in BinaryData")
	}

	rc.runner.RunnerSettings.AgentName = strings.ShortenString(fmt.Sprintf(agentNameTemplate, rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName()), 64)
	rc.runner.RunnerSettings.PoolId = 1
	rc.runner.RunnerSettings.PoolName = "Default"
	rc.runner.RunnerSettings.WorkFolder = "_work"

	return json.Unmarshal(dotRunner, &rc.runner.RunnerSettings)
}

func (rc *RunnerController) initCredentials() error {
	if rc.k8sFacade.configMap == nil {
		return errors.New(".configMap == nil")
	}

	dotCredentials, ok := rc.k8sFacade.configMap.BinaryData[".credentials"]
	if !ok {
		return errors.New(".credentials not in BinaryData")
	}

	rc.runner.Credentials.Scheme = "OAuth"

	return json.Unmarshal(dotCredentials, &rc.runner.Credentials)
}

func (rc *RunnerController) initRSAParameters() error {
	if rc.k8sFacade.secret == nil {
		return errors.New(".secret == nil")
	}

	dotCredentialsRSAParams, ok := rc.k8sFacade.secret.Data[".credentials_rsaparams"]
	if !ok {
		return errors.New(".credentials_rsaparams not in Data")
	}

	if string(dotCredentialsRSAParams) != "{}" {
		return json.Unmarshal(dotCredentialsRSAParams, &rc.runner.RSAParameters)
	}

	parameters, err := models.NewRSAParameters()
	if err != nil {
		return err
	}

	rc.runner.RSAParameters = *parameters
	return nil
}

func (rc *RunnerController) onPipelineAgentJobRequest(taskAgentMessage taskagent.TaskAgentMessage) error {
	if MessageType(*taskAgentMessage.MessageType) != MessageTypePipelineAgentJobRequest {
		return fmt.Errorf("taskAgentMessage.MessageType != MessageTypePipelineAgentJobRequest")
	}

	log.Printf("%s/%s: DeinitAzureDevOpsTaskAgentSession()", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.adoFacade.DeinitAzureDevOpsTaskAgentSession(); err != nil {
		return err
	}

	defer func() {
		log.Printf("%s/%s: InitAzureDevOpsTaskAgentSession", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
		if err := rc.adoFacade.InitAzureDevOpsTaskAgentSession(); err != nil {
			log.Panic(err)
		}
	}()

	log.Printf("%s/%s: CreatePersistentVolumeClaim", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.k8sFacade.CreatePersistentVolumeClaim(); err != nil {
		return err
	}

	log.Printf("%s/%s: CreateJob", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.k8sFacade.CreateJob(); err != nil {
		return err
	}

	log.Printf("%s/%s: WaitJob", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.k8sFacade.WaitJob(); err != nil {
		return err
	}

	log.Printf("%s/%s: DeleteJob", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.k8sFacade.DeleteJob(); err != nil {
		return err
	}

	log.Printf("%s/%s: DeletePersistentVolumeClaim", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	if err := rc.k8sFacade.DeletePersistentVolumeClaim(); err != nil {
		return err
	}

	return nil
}

func (rc *RunnerController) onJobCancellation(taskAgentMessage taskagent.TaskAgentMessage) error {
	if MessageType(*taskAgentMessage.MessageType) != MessageTypeJobCancellation {
		return errors.New("taskAgentMessage.MessageType != MessageTypeJobCancellation")
	}

	return rc.adoFacade.DeleteMessage(taskAgentMessage)
}

// TODO
func (rc *RunnerController) onAgentRefresh(taskAgentMessage taskagent.TaskAgentMessage) error {
	if MessageType(*taskAgentMessage.MessageType) != MessageTypeAgentRefresh {
		return errors.New("taskAgentMessage.MessageType != MessageTypeAgentRefresh")
	}

	rc.loop = false
	return nil
}

func (rc *RunnerController) consumeMessage(taskAgentMessage *taskagent.TaskAgentMessage) error {
	if taskAgentMessage == nil {
		return errors.New("taskAgentMessage == nil")
	}

	log.Printf("%s/%s: #%d .%s", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName(), *taskAgentMessage.MessageId, *taskAgentMessage.MessageType)
	util.Tracef("%s/%s: %s", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName(), *taskAgentMessage.Body)

	switch MessageType(*taskAgentMessage.MessageType) {
	case MessageTypePipelineAgentJobRequest:
		if err := rc.onPipelineAgentJobRequest(*taskAgentMessage); err != nil {
			return err
		}

	case MessageTypeJobCancellation:
		if err := rc.onPipelineAgentJobRequest(*taskAgentMessage); err != nil {
			return err
		}

	case MessageTypeAgentRefresh:
		if err := rc.onPipelineAgentJobRequest(*taskAgentMessage); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown MessageType(%s)", *taskAgentMessage.MessageType)
	}

	return nil
}

// TODO
func (rc *RunnerController) Notify() error {
	log.Printf("%s/%s: Notify", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())
	return nil
}

// TODO
func (rc *RunnerController) Close() error {
	log.Printf("%s/%s: Close", rc.k8sFacade.actionsRunner.GetNamespace(), rc.k8sFacade.actionsRunner.GetName())

	if !rc.loop {
		return errors.New("!.loop")
	}

	rc.loop = false
	return nil
}
