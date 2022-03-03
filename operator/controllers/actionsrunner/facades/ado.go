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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/google/go-github/v32/github"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/location"
	"github.com/microsoft/azure-devops-go-api/azuredevops/serviceendpoint"
	"github.com/microsoft/azure-devops-go-api/azuredevops/task"
	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
	"golang.org/x/text/encoding/unicode"

	"github.com/inloco/kube-actions/operator/constants"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/dot"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
)

const (
	poolId = 1
)

var (
	agentVersion = constants.API()

	agentLabelsBase = []string{
		"self-hosted",
		runtime.GOOS,
		runtime.GOARCH,
	}
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

	Plan          *task.TaskOrchestrationPlanReference
	Timeline      *build.TimelineReference
	JobConnection *azuredevops.Connection
	TaskClient    task.Client
}

func (ado *AzureDevOps) InitForCRUD(ctx context.Context, dotFiles *dot.Files, labels []string, token string, url string) error {
	if err := ado.initRSAPrivateKey(dotFiles); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsConnection(token, url); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsTaskAgentClient(ctx); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsTaskAgent(ctx, dotFiles, append(agentLabelsBase, labels...)); err != nil {
		return err
	}

	return nil
}

func (ado *AzureDevOps) InitForRun(ctx context.Context, dotFiles *dot.Files, labels []string) error {
	if err := ado.initRSAPrivateKey(dotFiles); err != nil {
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
func (ado *AzureDevOps) RefreshForRun(ctx context.Context, dotFiles *dot.Files) error {
	// TODO: improve refresh mechanism so it refreshes automatically and only when necessary
	// TODO: refresh other tokens if needed

	if err := ado.initAzureDevOpsBridgeConnection(ctx, dotFiles); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsBridgeTaskAgentClient(ctx); err != nil {
		return err
	}

	return nil
}

func (ado *AzureDevOps) initRSAPrivateKey(dotFiles *dot.Files) error {
	rsaPrivateKey, err := dotFiles.RSAParameters.ToRSAPrivateKey()
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
		PoolId:  github.Int(poolId),
		AgentId: ado.TaskAgent.Id,
	})
	if err == nil {
		return agent, nil
	}

	agents, _ := ado.TaskAgentClient.GetAgents(ctx, taskagent.GetAgentsArgs{
		PoolId:    github.Int(poolId),
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
		PoolId: github.Int(poolId),
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
		PoolId:  github.Int(poolId),
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
		PoolId:  github.Int(poolId),
		AgentId: ado.TaskAgent.Id,
	})
}

func (ado *AzureDevOps) initAzureDevOpsTaskAgent(ctx context.Context, dotFiles *dot.Files, labels []string) error {
	// TODO
	//if rc.rsaParameters == nil {
	//	return errors.New(".rsaParameters == nil")
	//}

	publicKey := taskagent.TaskAgentPublicKey{
		Exponent: &dotFiles.RSAParameters.Exponent,
		Modulus:  &dotFiles.RSAParameters.Modulus,
	}

	authorization := taskagent.TaskAgentAuthorization{
		PublicKey: &publicKey,
	}

	ado.TaskAgent = &taskagent.TaskAgent{
		DisableUpdate: github.Bool(true),
		Id:            github.Int(dotFiles.Runner.AgentId),
		Name:          github.String(dotFiles.Runner.AgentName),
		Version:       github.String(agentVersion),
		Authorization: &authorization,
		Labels:        &labels,
	}

	if ado.TaskAgentClient == nil {
		return nil
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

	dotFiles.Runner.AgentId = *agent.Id

	dotFiles.Credentials.Data.ClientId = agent.Authorization.ClientId.String()
	dotFiles.Credentials.Data.AuthorizationURL = *agent.Authorization.AuthorizationUrl
	dotFiles.Credentials.Data.OAuthEndpointURL = *agent.Authorization.AuthorizationUrl

	dotFiles.Credentials.Data.RequireFipsCryptography = "False"
	if v := util.GetPropertyValue(agent.Properties, "RequireFipsCryptography"); v == true {
		dotFiles.Credentials.Data.RequireFipsCryptography = "True"
	}

	ado.TaskAgent = agent
	return nil
}

func (ado *AzureDevOps) initAzureDevOpsBridgeConnection(ctx context.Context, dotFiles *dot.Files) error {
	// TODO
	//if rc.credentials == nil {
	//	return errors.New(".credentials == nil")
	//}

	assertion, err := util.ClientAssertion(dotFiles.Credentials.Data.ClientId, dotFiles.Credentials.Data.AuthorizationURL, ado.RSAPrivateKey)
	if err != nil {
		return err
	}

	token, err := util.AccessToken(ctx, dotFiles.Credentials.Data.OAuthEndpointURL, assertion)
	if err != nil {
		return err
	}

	ado.BridgeConnection = &azuredevops.Connection{
		AuthorizationString: fmt.Sprintf("Bearer %v", token),
		BaseUrl:             dotFiles.ServerUrl,
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
		PoolId: github.Int(poolId),
	})
}

func (ado *AzureDevOps) InitAzureDevOpsTaskAgentSession(ctx context.Context) error {
	session, err := ado.CreateAgentSession(ctx)
	if err != nil {
		unmarshalTypeError, ok := err.(*json.UnmarshalTypeError)
		if !ok {
			return err
		}

		// sometimes ADO returns TaskAgentStatus as an int instead of a string
		if unmarshalTypeError.Struct != "TaskAgentReference" || unmarshalTypeError.Field != "agent.status" || unmarshalTypeError.Value != "number" {
			return err
		}
	}

	if !*session.EncryptionKey.Encrypted {
		ado.TaskAgentSession = session
		return nil
	}

	hash := sha1.New()
	if *session.UseFipsEncryption {
		hash = sha256.New()
	}

	encryptionKey, err := rsa.DecryptOAEP(hash, rand.Reader, ado.RSAPrivateKey, *session.EncryptionKey.Value, nil)
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
		PoolId:    github.Int(poolId),
		SessionId: ado.TaskAgentSession.SessionId,
	})
}

func (ado *AzureDevOps) DeinitAzureDevOpsTaskAgentSession(ctx context.Context) error {
	if ado.TaskAgentSession == nil {
		return nil
	}

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
		PoolId:        github.Int(poolId),
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

	encBytes, err := base64.StdEncoding.DecodeString(*message.Body)
	if err != nil {
		return nil, err
	}

	decBytes, err := util.DecryptPKCS7(encBytes, *message.Iv, *ado.TaskAgentSession.EncryptionKey.Value)
	if err != nil {
		return nil, err
	}

	body, err := unicode.UTF8BOM.NewDecoder().String(string(decBytes))
	if err != nil {
		return nil, err
	}

	message.Body = github.String(body)
	message.Iv = nil

	return message, nil
}

func (ado *AzureDevOps) DeleteMessage(ctx context.Context, messageId uint64) error {
	if ado.TaskAgentBridgeClient == nil {
		return errors.New(".TaskAgentBridgeClient == nil")
	}

	if ado.TaskAgentSession == nil {
		return errors.New(".TaskAgentSession == nil")
	}

	return ado.TaskAgentBridgeClient.DeleteMessage(ctx, taskagent.DeleteMessageArgs{
		PoolId:    github.Int(poolId),
		MessageId: &messageId,
		SessionId: ado.TaskAgentSession.SessionId,
	})
}

func (ado *AzureDevOps) UpdateAgentRequest(ctx context.Context, request *taskagent.TaskAgentJobRequest, orchestrationId *string) (*taskagent.TaskAgentJobRequest, error) {
	// TODO
	//if rc.runnerSettings == nil {
	//	return errors.New(".runnerSettings == nil")
	//}

	if ado.TaskAgentBridgeClient == nil {
		return nil, errors.New(".TaskAgentBridgeClient == nil")
	}

	if request == nil {
		return nil, errors.New("request == nil")
	}

	return ado.TaskAgentBridgeClient.UpdateAgentRequest(ctx, taskagent.UpdateAgentRequestArgs{
		Request:         request,
		PoolId:          github.Int(poolId),
		RequestId:       request.RequestId,
		LockToken:       new(uuid.UUID),
		OrchestrationId: orchestrationId,
	})
}

func (ado *AzureDevOps) initAzureDevOpsPlan(plan *task.TaskOrchestrationPlanReference) error {
	if plan == nil {
		return errors.New("plan == nil")
	}

	if plan.ScopeIdentifier == nil {
		return errors.New("plan.ScopeIdentifier == nil")
	}

	if plan.PlanType == nil {
		return errors.New("plan.PlanType == nil")
	}

	if plan.PlanId == nil {
		return errors.New("plan.PlanId == nil")
	}

	ado.Plan = plan
	return nil
}

func (ado *AzureDevOps) initAzureDevOpsTimeline(timeline *build.TimelineReference) error {
	if timeline == nil {
		return errors.New("timeline == nil")
	}

	if timeline.Id == nil {
		return errors.New("timeline.Id == nil")
	}

	ado.Timeline = timeline
	return nil
}

func (ado *AzureDevOps) initAzureDevOpsJobConnection(serviceEndpoints []serviceendpoint.ServiceEndpoint) error {
	serviceEndpoint, err := util.GetSystemVssConnectionEndpoint(serviceEndpoints)
	if err != nil {
		return err
	}

	accessToken, err := util.GetServiceEndpointAccessToken(serviceEndpoint)
	if err != nil {
		return err
	}

	url, err := util.GetServiceEndpointURL(serviceEndpoint)
	if err != nil {
		return err
	}

	ado.JobConnection = &azuredevops.Connection{
		AuthorizationString: fmt.Sprintf("Bearer %s", accessToken),
		BaseUrl:             url,
	}
	return nil
}

func (ado *AzureDevOps) InitAzureDevOpsTaskClient(plan *task.TaskOrchestrationPlanReference, timeline *build.TimelineReference, endpoints []serviceendpoint.ServiceEndpoint) error {
	if err := ado.initAzureDevOpsPlan(plan); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsTimeline(timeline); err != nil {
		return err
	}

	if err := ado.initAzureDevOpsJobConnection(endpoints); err != nil {
		return err
	}

	ado.TaskClient = task.NewClient(context.TODO(), ado.JobConnection)
	return nil
}

func (ado *AzureDevOps) UpdateRecord(ctx context.Context, timelineRecords []task.TimelineRecord) ([]task.TimelineRecord, error) {
	if ado.Plan == nil {
		return nil, errors.New(".Plan == nil")
	}

	if ado.Timeline == nil {
		return nil, errors.New(".Timeline == nil")
	}

	if ado.TaskClient == nil {
		return nil, errors.New(".TaskClient == nil")
	}

	count := len(timelineRecords)

	value := make([]interface{}, count)
	for i := range timelineRecords {
		value[i] = timelineRecords[i]
	}

	records := azuredevops.VssJsonCollectionWrapper{
		Count: &count,
		Value: &value,
	}

	res, err := ado.TaskClient.UpdateRecords(ctx, task.UpdateRecordsArgs{
		Records:         &records,
		ScopeIdentifier: ado.Plan.ScopeIdentifier,
		HubName:         ado.Plan.PlanType,
		PlanId:          ado.Plan.PlanId,
		TimelineId:      ado.Timeline.Id,
	})

	var ret []task.TimelineRecord
	if res != nil {
		ret = *res
	}

	return ret, err
}

func (ado *AzureDevOps) RaisePlanEvent(ctx context.Context, eventData *task.JobEvent) error {
	if ado.Plan == nil {
		return errors.New(".Plan == nil")
	}

	if ado.TaskClient == nil {
		return errors.New(".TaskClient == nil")
	}

	if eventData == nil {
		return errors.New("eventData == nil")
	}

	return ado.TaskClient.RaisePlanEvent(ctx, task.RaisePlanEventArgs{
		EventData:       eventData,
		ScopeIdentifier: ado.Plan.ScopeIdentifier,
		HubName:         ado.Plan.PlanType,
		PlanId:          ado.Plan.PlanId,
	})
}
