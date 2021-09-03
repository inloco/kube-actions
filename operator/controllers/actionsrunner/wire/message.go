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
