package controller

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"

	"github.com/google/uuid"
	"github.com/inloco/kube-actions/operator/models"
	"github.com/inloco/kube-actions/operator/util"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/location"
	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
)

var (
	agentNameTemplate = "KA %v %v"

	agentVersion = "2.267.1"

	agentLabelsBase = []string{
		"self-hosted",
		runtime.GOOS,
		runtime.GOARCH,
	}

	poolId = 1
)

type facadeAzureDevOps struct {
	context       context.Context
	rsaPrivateKey *rsa.PrivateKey

	adoConnection      *azuredevops.Connection
	adoTaskAgentClient taskagent.Client
	adoLocationClient  location.Client
	adoTaskAgent       *taskagent.TaskAgent

	adoBridgeConnection      *azuredevops.Connection
	adoTaskAgentBridgeClient taskagent.Client
	adoTaskAgentSession      *taskagent.TaskAgentSession
}

func (ado *facadeAzureDevOps) Init(token string, url string, runner *models.Runner, labels []string) error {
	if err := ado.initRSAPrivateKey(runner); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsConnection(token, url); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsLocationClient(); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsTaskAgentClient(); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsTaskAgent(runner, labels); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsBridgeConnection(runner); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsBridgeTaskAgentClient(); err != nil {
		return err
	}

	return nil
}

func (ado *facadeAzureDevOps) initRSAPrivateKey(runner *models.Runner) error {
	rsaPrivateKey, err := runner.RSAParameters.ToRSAPrivateKey()
	if err != nil {
		return err
	}

	ado.rsaPrivateKey = rsaPrivateKey
	return nil
}

func (ado *facadeAzureDevOps) initAzureDevOpsConnection(token string, url string) error {
	if token == "" {
		return errors.New(`token == ""`)
	}

	if url == "" {
		return errors.New(`url == ""`)
	}

	ado.adoConnection = &azuredevops.Connection{
		AuthorizationString: fmt.Sprintf("Bearer %v", token),
		BaseUrl:             url,
	}
	return nil
}

func (ado *facadeAzureDevOps) initAzureDevOpsLocationClient() error {
	if ado.context == nil {
		return errors.New(".context == nil")
	}

	if ado.adoConnection == nil {
		return errors.New(".adoConnection == nil")
	}

	ado.adoLocationClient = location.NewClient(ado.context, ado.adoConnection)
	return nil
}

func (ado *facadeAzureDevOps) initAzureDevOpsTaskAgentClient() error {
	if ado.context == nil {
		return errors.New(".context == nil")
	}

	if ado.adoConnection == nil {
		return errors.New(".adoConnection == nil")
	}

	client, err := taskagent.NewClient(ado.context, ado.adoConnection)
	if err != nil {
		return err
	}

	ado.adoTaskAgentClient = client
	return nil
}

func (ado *facadeAzureDevOps) GetAgent() (*taskagent.TaskAgent, error) {
	if ado.context == nil {
		return nil, errors.New(".context == nil")
	}

	if ado.adoTaskAgentClient == nil {
		return nil, errors.New(".adoTaskAgentClient == nil")
	}

	agent, err := ado.adoTaskAgentClient.GetAgent(ado.context, taskagent.GetAgentArgs{
		PoolId:  &poolId,
		AgentId: ado.adoTaskAgent.Id,
	})
	if err == nil {
		return agent, nil
	}

	agents, _ := ado.adoTaskAgentClient.GetAgents(ado.context, taskagent.GetAgentsArgs{
		PoolId:    &poolId,
		AgentName: ado.adoTaskAgent.Name,
	})
	if agents != nil && len(*agents) == 1 {
		return &(*agents)[0], nil
	}

	return nil, err
}

func (ado *facadeAzureDevOps) AddAgent() (*taskagent.TaskAgent, error) {
	if ado.context == nil {
		return nil, errors.New(".context == nil")
	}

	if ado.adoTaskAgentClient == nil {
		return nil, errors.New(".adoTaskAgentClient == nil")
	}

	taskAgent, err := ado.adoTaskAgentClient.AddAgent(ado.context, taskagent.AddAgentArgs{
		PoolId: &poolId,
		Agent:  ado.adoTaskAgent,
	})
	if err != nil {
		return nil, err
	}

	return taskAgent, nil
}

func (ado *facadeAzureDevOps) ReplaceAgent() (*taskagent.TaskAgent, error) {
	if ado.context == nil {
		return nil, errors.New(".context == nil")
	}

	if ado.adoTaskAgentClient == nil {
		return nil, errors.New(".adoTaskAgentClient == nil")
	}

	taskAgent, err := ado.adoTaskAgentClient.ReplaceAgent(ado.context, taskagent.ReplaceAgentArgs{
		PoolId:  &poolId,
		AgentId: ado.adoTaskAgent.Id,
		Agent:   ado.adoTaskAgent,
	})
	if err != nil {
		return nil, err
	}

	return taskAgent, nil
}

func (ado *facadeAzureDevOps) DeleteAgent() error {
	if ado.context == nil {
		return errors.New(".context == nil")
	}

	// TODO
	//if rc.runnerSettings == nil {
	//	return errors.New(".runnerSettings == nil")
	//}

	if ado.adoTaskAgentClient == nil {
		return errors.New(".adoTaskAgentClient == nil")
	}

	return ado.adoTaskAgentClient.DeleteAgent(ado.context, taskagent.DeleteAgentArgs{
		PoolId:  &poolId,
		AgentId: ado.adoTaskAgent.Id,
	})
}

func (ado *facadeAzureDevOps) initAzureDevOpsTaskAgent(runner *models.Runner, labels []string) error {
	// TODO
	//if rc.rsaParameters == nil {
	//	return errors.New(".rsaParameters == nil")
	//}

	ado.adoTaskAgent = &taskagent.TaskAgent{
		Id:      &runner.RunnerSettings.AgentId,
		Name:    &runner.RunnerSettings.AgentName,
		Version: &agentVersion,
		Labels:  &labels,
		Authorization: &taskagent.TaskAgentAuthorization{
			PublicKey: &taskagent.TaskAgentPublicKey{
				Exponent: &runner.RSAParameters.Exponent,
				Modulus:  &runner.RSAParameters.Modulus,
			},
		},
	}

	if agent, err := ado.GetAgent(); err == nil {
		ado.adoTaskAgent.Id = agent.Id
	} else {
		ado.adoTaskAgent.Id = nil
	}

	var agent *taskagent.TaskAgent
	if ado.adoTaskAgent.Id == nil {
		taskAgent, err := ado.AddAgent()
		if err != nil {
			ado.adoTaskAgent = nil
			return err
		}

		agent = taskAgent
	} else {
		taskAgent, err := ado.ReplaceAgent()
		if err != nil {
			ado.adoTaskAgent = nil
			return err
		}

		agent = taskAgent
	}

	runner.RunnerSettings.AgentId = *agent.Id

	runner.Credentials.Data.ClientId = agent.Authorization.ClientId.String()
	runner.Credentials.Data.AuthorizationURL = *agent.Authorization.AuthorizationUrl
	runner.Credentials.Data.OAuthEndpointURL = *agent.Authorization.AuthorizationUrl

	ado.adoTaskAgent = agent
	return nil
}

func (ado *facadeAzureDevOps) GetAzureDevOpsConnectionData() (*location.ConnectionData, error) {
	if ado.context == nil {
		return nil, errors.New(".context == nil")
	}

	if ado.adoLocationClient == nil {
		return nil, errors.New(".adoLocationClient == nil")
	}

	connectOptions := webapi.ConnectOptions("1")
	return ado.adoLocationClient.GetConnectionData(ado.context, location.GetConnectionDataArgs{
		ConnectOptions: &connectOptions,
	})
}

func (ado *facadeAzureDevOps) initAzureDevOpsBridgeConnection(runner *models.Runner) error {
	if ado.context == nil {
		return errors.New(".context == nil")
	}

	//if rc.credentials == nil {
	//	return errors.New(".credentials == nil")
	//}

	assertion, err := util.ClientAssertion(runner.Credentials.Data.ClientId, runner.Credentials.Data.AuthorizationURL, ado.rsaPrivateKey)
	if err != nil {
		return err
	}

	token, err := util.AccessToken(ado.context, runner.Credentials.Data.OAuthEndpointURL, assertion)
	if err != nil {
		return err
	}

	data, err := ado.GetAzureDevOpsConnectionData()
	if err != nil {
		return err
	}

	url := ado.adoConnection.BaseUrl
	for _, accessMapping := range *data.LocationServiceData.AccessMappings {
		if *accessMapping.Moniker == "HostGuidAccessMapping" {
			url = *accessMapping.AccessPoint
			break
		}
	}

	ado.adoBridgeConnection = &azuredevops.Connection{
		AuthorizationString: fmt.Sprintf("Bearer %v", token),
		BaseUrl:             url,
	}
	return nil
}

func (ado *facadeAzureDevOps) initAzureDevOpsBridgeTaskAgentClient() error {
	if ado.context == nil {
		return errors.New(".context == nil")
	}

	if ado.adoBridgeConnection == nil {
		return errors.New(".adoBridgeConnection == nil")
	}

	client, err := taskagent.NewClient(ado.context, ado.adoBridgeConnection)
	if err != nil {
		return err
	}

	ado.adoTaskAgentBridgeClient = client
	return nil
}

func (ado *facadeAzureDevOps) CreateAgentSession() (*taskagent.TaskAgentSession, error) {
	if ado.context == nil {
		return nil, errors.New(".context == nil")
	}

	if ado.adoTaskAgent == nil {
		return nil, errors.New(".adoTaskAgent == nil")
	}

	if ado.adoTaskAgentBridgeClient == nil {
		return nil, errors.New(".adoTaskAgentBridgeClient == nil")
	}

	ownerName := "Kube Actions"
	return ado.adoTaskAgentBridgeClient.CreateAgentSession(ado.context, taskagent.CreateAgentSessionArgs{
		Session: &taskagent.TaskAgentSession{
			Agent: &taskagent.TaskAgentReference{
				Links:             ado.adoTaskAgent.Links,
				AccessPoint:       ado.adoTaskAgent.AccessPoint,
				Enabled:           ado.adoTaskAgent.Enabled,
				Id:                ado.adoTaskAgent.Id,
				Name:              ado.adoTaskAgent.Name,
				OsDescription:     ado.adoTaskAgent.OsDescription,
				ProvisioningState: ado.adoTaskAgent.ProvisioningState,
				Status:            ado.adoTaskAgent.Status,
				Version:           ado.adoTaskAgent.Version,
			},
			OwnerName: &ownerName,
			SessionId: &uuid.UUID{},
		},
		PoolId: &poolId,
	})
}

func (ado *facadeAzureDevOps) InitAzureDevOpsTaskAgentSession() error {
	session, err := ado.CreateAgentSession()
	if err != nil {
		return err
	}

	if !*session.EncryptionKey.Encrypted {
		ado.adoTaskAgentSession = session
		return nil
	}

	encryptionKey, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, ado.rsaPrivateKey, *session.EncryptionKey.Value, nil)
	if err != nil {
		return err
	}

	session.EncryptionKey.Value = &encryptionKey

	ado.adoTaskAgentSession = session
	return nil
}

func (ado *facadeAzureDevOps) DeleteAgentSession() error {
	if ado.context == nil {
		return errors.New(".context == nil")
	}

	if ado.adoTaskAgentBridgeClient == nil {
		return errors.New(".adoTaskAgentBridgeClient == nil")
	}

	if ado.adoTaskAgentSession == nil {
		return errors.New(".adoTaskAgentSession == nil")
	}

	return ado.adoTaskAgentBridgeClient.DeleteAgentSession(ado.context, taskagent.DeleteAgentSessionArgs{
		PoolId:    &poolId,
		SessionId: ado.adoTaskAgentSession.SessionId,
	})
}

func (ado *facadeAzureDevOps) DeinitAzureDevOpsTaskAgentSession() error {
	if err := ado.DeleteAgentSession(); err != nil {
		return err
	}

	ado.adoTaskAgentSession = nil
	return nil
}

func (ado *facadeAzureDevOps) GetMessage() (*taskagent.TaskAgentMessage, error) {
	if ado.context == nil {
		return nil, errors.New(".context == nil")
	}

	if ado.adoTaskAgentBridgeClient == nil {
		return nil, errors.New(".adoTaskAgentBridgeClient == nil")
	}

	if ado.adoTaskAgentSession == nil {
		return nil, errors.New(".adoTaskAgentSession == nil")
	}

	message, err := ado.adoTaskAgentBridgeClient.GetMessage(ado.context, taskagent.GetMessageArgs{
		PoolId:    &poolId,
		SessionId: ado.adoTaskAgentSession.SessionId,
	})
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, nil
	}

	if message.Iv == nil {
		return message, nil
	}

	bytes, err := base64.StdEncoding.DecodeString(*message.Body)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(*ado.adoTaskAgentSession.EncryptionKey.Value)
	if err != nil {
		return nil, err
	}

	cipher.NewCBCDecrypter(block, *message.Iv).CryptBlocks(bytes, bytes)
	message.Iv = nil

	body := string(bytes)
	message.Body = &body

	return message, nil
}

func (ado *facadeAzureDevOps) DeleteMessage(taskAgentMessage taskagent.TaskAgentMessage) error {
	if ado.context == nil {
		return errors.New(".context == nil")
	}

	if ado.adoTaskAgentBridgeClient == nil {
		return errors.New(".adoTaskAgentBridgeClient == nil")
	}

	if ado.adoTaskAgentSession == nil {
		return errors.New(".adoTaskAgentSession == nil")
	}

	return ado.adoTaskAgentBridgeClient.DeleteMessage(ado.context, taskagent.DeleteMessageArgs{
		PoolId:    &poolId,
		MessageId: taskAgentMessage.MessageId,
		SessionId: ado.adoTaskAgentSession.SessionId,
	})
}

type MessageType string

const (
	MessageTypePipelineAgentJobRequest MessageType = "PipelineAgentJobRequest"
	MessageTypeJobCancellation         MessageType = "JobCancellation"
	MessageTypeAgentRefresh            MessageType = "AgentRefresh"
)
