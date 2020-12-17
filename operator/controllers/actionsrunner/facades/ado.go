/*
Copyright 2020 In Loco Tecnologia da Informação S.A.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package facades

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
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/dot"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/location"
	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
)

var (
	agentNameTemplate = "KA %v %v"

	agentVersion = "2.275.1"

	agentLabelsBase = []string{
		"self-hosted",
		runtime.GOOS,
		runtime.GOARCH,
	}

	poolId = 1
)

type AzureDevOps struct {
	RSAPrivateKey *rsa.PrivateKey

	Connection      *azuredevops.Connection
	TaskAgentClient taskagent.Client
	LocationClient  location.Client
	TaskAgent       *taskagent.TaskAgent

	BridgeConnection      *azuredevops.Connection
	TaskAgentBridgeClient taskagent.Client
	TaskAgentSession      *taskagent.TaskAgentSession
}

func (ado *AzureDevOps) Init(ctx context.Context, token string, url string, dotFiles *dot.Files, labels []string) error {
	if err := ado.initRSAPrivateKey(dotFiles); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsConnection(token, url); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsLocationClient(ctx); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsTaskAgentClient(ctx); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsTaskAgent(ctx, dotFiles, append(agentLabelsBase, labels...)); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsBridgeConnection(ctx, dotFiles); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsBridgeTaskAgentClient(ctx); err != nil {
		return err
	}

	return nil
}

func (ado *AzureDevOps) initRSAPrivateKey(runner *dot.Files) error {
	rsaPrivateKey, err := runner.RSAParameters.ToRSAPrivateKey()
	if err != nil {
		return err
	}

	ado.RSAPrivateKey = rsaPrivateKey
	return nil
}

func (ado *AzureDevOps) initAzureDevOpsConnection(token string, url string) error {
	if token == "" {
		return errors.New(`token == ""`)
	}

	if url == "" {
		return errors.New(`url == ""`)
	}

	ado.Connection = &azuredevops.Connection{
		AuthorizationString: fmt.Sprintf("Bearer %v", token),
		BaseUrl:             url,
	}
	return nil
}

func (ado *AzureDevOps) initAzureDevOpsLocationClient(ctx context.Context) error {
	if ado.Connection == nil {
		return errors.New(".Connection == nil")
	}

	ado.LocationClient = location.NewClient(ctx, ado.Connection)
	return nil
}

func (ado *AzureDevOps) initAzureDevOpsTaskAgentClient(ctx context.Context) error {
	if ado.Connection == nil {
		return errors.New(".Connection == nil")
	}

	client, err := taskagent.NewClient(ctx, ado.Connection)
	if err != nil {
		return err
	}

	ado.TaskAgentClient = client
	return nil
}

func (ado *AzureDevOps) GetAgent(ctx context.Context) (*taskagent.TaskAgent, error) {
	if ado.TaskAgentClient == nil {
		return nil, errors.New(".TaskAgentClient == nil")
	}

	agent, err := ado.TaskAgentClient.GetAgent(ctx, taskagent.GetAgentArgs{
		PoolId:  &poolId,
		AgentId: ado.TaskAgent.Id,
	})
	if err == nil {
		return agent, nil
	}

	agents, _ := ado.TaskAgentClient.GetAgents(ctx, taskagent.GetAgentsArgs{
		PoolId:    &poolId,
		AgentName: ado.TaskAgent.Name,
	})
	if agents != nil && len(*agents) == 1 {
		return &(*agents)[0], nil
	}

	return nil, err
}

func (ado *AzureDevOps) AddAgent(ctx context.Context) (*taskagent.TaskAgent, error) {
	if ado.TaskAgentClient == nil {
		return nil, errors.New(".TaskAgentClient == nil")
	}

	taskAgent, err := ado.TaskAgentClient.AddAgent(ctx, taskagent.AddAgentArgs{
		PoolId: &poolId,
		Agent:  ado.TaskAgent,
	})
	if err != nil {
		return nil, err
	}

	return taskAgent, nil
}

func (ado *AzureDevOps) ReplaceAgent(ctx context.Context) (*taskagent.TaskAgent, error) {
	if ado.TaskAgentClient == nil {
		return nil, errors.New(".TaskAgentClient == nil")
	}

	taskAgent, err := ado.TaskAgentClient.ReplaceAgent(ctx, taskagent.ReplaceAgentArgs{
		PoolId:  &poolId,
		AgentId: ado.TaskAgent.Id,
		Agent:   ado.TaskAgent,
	})
	if err != nil {
		return nil, err
	}

	return taskAgent, nil
}

func (ado *AzureDevOps) DeleteAgent(ctx context.Context) error {
	// TODO
	//if rc.runnerSettings == nil {
	//	return errors.New(".runnerSettings == nil")
	//}

	if ado.TaskAgentClient == nil {
		return errors.New(".TaskAgentClient == nil")
	}

	return ado.TaskAgentClient.DeleteAgent(ctx, taskagent.DeleteAgentArgs{
		PoolId:  &poolId,
		AgentId: ado.TaskAgent.Id,
	})
}

func (ado *AzureDevOps) initAzureDevOpsTaskAgent(ctx context.Context, runner *dot.Files, labels []string) error {
	// TODO
	//if rc.rsaParameters == nil {
	//	return errors.New(".rsaParameters == nil")
	//}

	ado.TaskAgent = &taskagent.TaskAgent{
		Id:      &runner.Runner.AgentId,
		Name:    &runner.Runner.AgentName,
		Version: &agentVersion,
		Labels:  &labels,
		Authorization: &taskagent.TaskAgentAuthorization{
			PublicKey: &taskagent.TaskAgentPublicKey{
				Exponent: &runner.RSAParameters.Exponent,
				Modulus:  &runner.RSAParameters.Modulus,
			},
		},
	}

	if agent, err := ado.GetAgent(ctx); err == nil {
		ado.TaskAgent.Id = agent.Id
	} else {
		ado.TaskAgent.Id = nil
	}

	var agent *taskagent.TaskAgent
	if ado.TaskAgent.Id == nil {
		taskAgent, err := ado.AddAgent(ctx)
		if err != nil {
			ado.TaskAgent = nil
			return err
		}

		agent = taskAgent
	} else {
		taskAgent, err := ado.ReplaceAgent(ctx)
		if err != nil {
			ado.TaskAgent = nil
			return err
		}

		agent = taskAgent
	}

	runner.Runner.AgentId = *agent.Id

	runner.Credentials.Data.ClientId = agent.Authorization.ClientId.String()
	runner.Credentials.Data.AuthorizationURL = *agent.Authorization.AuthorizationUrl
	runner.Credentials.Data.OAuthEndpointURL = *agent.Authorization.AuthorizationUrl

	ado.TaskAgent = agent
	return nil
}

func (ado *AzureDevOps) GetAzureDevOpsConnectionData(ctx context.Context) (*location.ConnectionData, error) {
	if ado.LocationClient == nil {
		return nil, errors.New(".LocationClient == nil")
	}

	connectOptions := webapi.ConnectOptions("1")
	return ado.LocationClient.GetConnectionData(ctx, location.GetConnectionDataArgs{
		ConnectOptions: &connectOptions,
	})
}

func (ado *AzureDevOps) initAzureDevOpsBridgeConnection(ctx context.Context, runner *dot.Files) error {
	// TODO
	//if rc.credentials == nil {
	//	return errors.New(".credentials == nil")
	//}

	assertion, err := util.ClientAssertion(runner.Credentials.Data.ClientId, runner.Credentials.Data.AuthorizationURL, ado.RSAPrivateKey)
	if err != nil {
		return err
	}

	token, err := util.AccessToken(ctx, runner.Credentials.Data.OAuthEndpointURL, assertion)
	if err != nil {
		return err
	}

	data, err := ado.GetAzureDevOpsConnectionData(ctx)
	if err != nil {
		return err
	}

	url := ado.Connection.BaseUrl
	for _, accessMapping := range *data.LocationServiceData.AccessMappings {
		if *accessMapping.Moniker == "HostGuidAccessMapping" {
			url = *accessMapping.AccessPoint
			break
		}
	}

	ado.BridgeConnection = &azuredevops.Connection{
		AuthorizationString: fmt.Sprintf("Bearer %v", token),
		BaseUrl:             url,
	}
	return nil
}

func (ado *AzureDevOps) initAzureDevOpsBridgeTaskAgentClient(ctx context.Context) error {
	if ado.BridgeConnection == nil {
		return errors.New(".BridgeConnection == nil")
	}

	client, err := taskagent.NewClient(ctx, ado.BridgeConnection)
	if err != nil {
		return err
	}

	ado.TaskAgentBridgeClient = client
	return nil
}

func (ado *AzureDevOps) CreateAgentSession(ctx context.Context) (*taskagent.TaskAgentSession, error) {
	if ado.TaskAgent == nil {
		return nil, errors.New(".TaskAgent == nil")
	}

	if ado.TaskAgentBridgeClient == nil {
		return nil, errors.New(".TaskAgentBridgeClient == nil")
	}

	ownerName := "Kube Actions"
	return ado.TaskAgentBridgeClient.CreateAgentSession(ctx, taskagent.CreateAgentSessionArgs{
		Session: &taskagent.TaskAgentSession{
			Agent: &taskagent.TaskAgentReference{
				Links:             ado.TaskAgent.Links,
				AccessPoint:       ado.TaskAgent.AccessPoint,
				Enabled:           ado.TaskAgent.Enabled,
				Id:                ado.TaskAgent.Id,
				Name:              ado.TaskAgent.Name,
				OsDescription:     ado.TaskAgent.OsDescription,
				ProvisioningState: ado.TaskAgent.ProvisioningState,
				Status:            ado.TaskAgent.Status,
				Version:           ado.TaskAgent.Version,
			},
			OwnerName: &ownerName,
			SessionId: &uuid.UUID{},
		},
		PoolId: &poolId,
	})
}

func (ado *AzureDevOps) InitAzureDevOpsTaskAgentSession(ctx context.Context) error {
	session, err := ado.CreateAgentSession(ctx)
	if err != nil {
		return err
	}

	if !*session.EncryptionKey.Encrypted {
		ado.TaskAgentSession = session
		return nil
	}

	encryptionKey, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, ado.RSAPrivateKey, *session.EncryptionKey.Value, nil)
	if err != nil {
		return err
	}

	session.EncryptionKey.Value = &encryptionKey

	ado.TaskAgentSession = session
	return nil
}

func (ado *AzureDevOps) DeleteAgentSession(ctx context.Context) error {
	if ado.TaskAgentBridgeClient == nil {
		return errors.New(".TaskAgentBridgeClient == nil")
	}

	if ado.TaskAgentSession == nil {
		return errors.New(".TaskAgentSession == nil")
	}

	return ado.TaskAgentBridgeClient.DeleteAgentSession(ctx, taskagent.DeleteAgentSessionArgs{
		PoolId:    &poolId,
		SessionId: ado.TaskAgentSession.SessionId,
	})
}

func (ado *AzureDevOps) DeinitAzureDevOpsTaskAgentSession(ctx context.Context) error {
	if err := ado.DeleteAgentSession(ctx); err != nil {
		return err
	}

	ado.TaskAgentSession = nil
	return nil
}

func (ado *AzureDevOps) GetMessage(ctx context.Context, lastMessageId *uint64) (*taskagent.TaskAgentMessage, error) {
	if ado.TaskAgentBridgeClient == nil {
		return nil, errors.New(".TaskAgentBridgeClient == nil")
	}

	if ado.TaskAgentSession == nil {
		return nil, errors.New(".TaskAgentSession == nil")
	}

	message, err := ado.TaskAgentBridgeClient.GetMessage(ctx, taskagent.GetMessageArgs{
		PoolId:        &poolId,
		SessionId:     ado.TaskAgentSession.SessionId,
		LastMessageId: lastMessageId,
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

	block, err := aes.NewCipher(*ado.TaskAgentSession.EncryptionKey.Value)
	if err != nil {
		return nil, err
	}

	cipher.NewCBCDecrypter(block, *message.Iv).CryptBlocks(bytes, bytes)
	message.Iv = nil

	body := string(bytes)
	message.Body = &body

	return message, nil
}

func (ado *AzureDevOps) DeleteMessage(ctx context.Context, taskAgentMessage taskagent.TaskAgentMessage) error {
	if ado.TaskAgentBridgeClient == nil {
		return errors.New(".TaskAgentBridgeClient == nil")
	}

	if ado.TaskAgentSession == nil {
		return errors.New(".TaskAgentSession == nil")
	}

	return ado.TaskAgentBridgeClient.DeleteMessage(ctx, taskagent.DeleteMessageArgs{
		PoolId:    &poolId,
		MessageId: taskAgentMessage.MessageId,
		SessionId: ado.TaskAgentSession.SessionId,
	})
}
