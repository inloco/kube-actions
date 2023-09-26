package wire

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/serviceendpoint"
	"github.com/microsoft/azure-devops-go-api/azuredevops/task"
)

type ContainerResource struct {
	Environment *map[string]string `json:"environment,omitempty"`
	Image       *string            `json:"image,omitempty"`
	Options     *string            `json:"options,omitempty"`
	Volumes     *[]string          `json:"volumes,omitempty"`
	Ports       *[]string          `json:"ports,omitempty"`
}

type RepositoryResource struct {
	Id      *string `json:"id,omitempty"`
	Type    *string `json:"type,omitempty"`
	Url     *string `json:"url,omitempty"`
	Version *string `json:"version,omitempty"`
}

type JobResources struct {
	Containers   *[]ContainerResource               `json:"containers,omitempty"`
	Endpoints    *[]serviceendpoint.ServiceEndpoint `json:"endpoints,omitempty"`
	Repositories *[]RepositoryResource              `json:"repositories,omitempty"`
}

type WorkspaceOptions struct {
	Clean *string `json:"clean,omitempty"`
}

type JobStep struct {
	Condition        *string             `json:"condition,omitempty"`
	ContinueOnError  *task.TemplateToken `json:"continueOnError,omitempty"`
	TimeoutInMinutes *task.TemplateToken `json:"timeoutInMinutes,omitempty"`
}

type PipelineAgentJobRequest struct {
	MessageType          *string                              `json:"messageType,omitempty"`
	Plan                 *task.TaskOrchestrationPlanReference `json:"plan,omitempty"`
	Timeline             *build.TimelineReference             `json:"timeline,omitempty"`
	JobId                *uuid.UUID                           `json:"jobId,omitempty"`
	JobDisplayName       *string                              `json:"jobDisplayName,omitempty"`
	JobName              *string                              `json:"jobName,omitempty"`
	JobContainer         *task.TemplateToken                  `json:"jobContainer,omitempty"`
	JobServiceContainers *task.TemplateToken                  `json:"jobServiceContainers,omitempty"`
	JobOutputs           *task.TemplateToken                  `json:"jobOutputs,omitempty"`
	RequestId            *uint64                              `json:"requestId,omitempty"`
	LockedUntil          *azuredevops.Time                    `json:"lockedUntil,omitempty"`
	Resources            *JobResources                        `json:"resources,omitempty"`
	ContextData          *map[string]PipelineContextData      `json:"contextData,omitempty"`
	Workspace            *WorkspaceOptions                    `json:"workspace,omitempty"`
	ActionsEnvironment   *task.ActionsEnvironmentReference    `json:"actionsEnvironment,omitempty"`
	EnvironmentVariables *[]task.TemplateToken                `json:"environmentVariables,omitempty"`
	Defaults             *[]task.TemplateToken                `json:"defaults,omitempty"`
	FileTable            *[]string                            `json:"fileTable,omitempty"`
	Mask                 *[]task.MaskHint                     `json:"mask,omitempty"`
	Steps                *[]JobStep                           `json:"steps,omitempty"`
	Variables            *map[string]task.VariableValue       `json:"variables,omitempty"`
	JobSidecarContainers *map[string]string                   `json:"jobSidecarContainers,omitempty"`
}

func toPipelineAgentJobRequest(message *Message) (*PipelineAgentJobRequest, error) {
	if message == nil {
		return nil, errors.New("message == nil")
	}

	body := message.String()

	var pajr PipelineAgentJobRequest
	if err := json.Unmarshal([]byte(body), &pajr); err != nil {
		return nil, err
	}

	return &pajr, nil
}
