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

package wire

import (
	"encoding/json"
	"errors"

	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
)

type MessageType string

const (
	MessageTypePipelineAgentJobRequest MessageType = "PipelineAgentJobRequest"
	MessageTypeAgentRefresh            MessageType = "AgentRefresh"
)

type MessageId uint64

const (
	MessageIdZero MessageId = 0
)

type Message struct {
	Id   MessageId
	Type MessageType
	body string
}

func (m Message) String() string {
	return m.body
}

func toMessage(taMessage taskagent.TaskAgentMessage) (*Message, error) {
	if taMessage.MessageId == nil {
		return nil, errors.New("taMessage.MessageId == nil")
	}

	if taMessage.MessageType == nil {
		return nil, errors.New("taMessage.MessageType == nil")
	}

	message := Message{
		Id:   MessageId(*taMessage.MessageId),
		Type: MessageType(*taMessage.MessageType),
		body: *taMessage.Body,
	}
	return &message, nil
}

type PipelineAgentJobRequest struct {
	// MessageType          interface{} `json:"messageType"`
	// Plan                 interface{} `json:"plan"`
	// Timeline             interface{} `json:"timeline"`
	// JobId                interface{} `json:"jobId"`
	// JobDisplayName       interface{} `json:"jobDisplayName"`
	// JobName              interface{} `json:"jobName"`
	// JobContainer         interface{} `json:"jobContainer"`
	// JobServiceContainers interface{} `json:"jobServiceContainers"`
	// JobOutputs           interface{} `json:"jobOutputs"`
	// RequestId            interface{} `json:"requestId"`
	// LockedUntil          interface{} `json:"lockedUntil"`
	// Resources            interface{} `json:"resources"`

	ContextData map[string]PipelineContextData `json:"contextData"`

	// Workspace            interface{} `json:"workspace"`
	// ActionsEnvironment   interface{} `json:"actionsEnvironment"`
	// EnvironmentVariables interface{} `json:"environmentVariables"`
	// Defaults             interface{} `json:"defaults"`
	// FileTable            interface{} `json:"fileTable"`
	// Mask                 interface{} `json:"mask"`
	// Steps                interface{} `json:"steps"`
	// Variables            interface{} `json:"variables"`
	// JobSidecarContainers interface{} `json:"jobSidecarContainers"`
}

func toPipelineAgentJobRequest(message Message) (*PipelineAgentJobRequest, error) {
	body := message.String()

	var pajr PipelineAgentJobRequest
	if err := json.Unmarshal([]byte(body), &pajr); err != nil {
		return nil, err
	}

	return &pajr, nil
}
