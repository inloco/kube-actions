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
	"context"
	"errors"
	"fmt"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/dot"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/facades"
	"k8s.io/utils/strings"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Wire struct {
	events chan<- event.GenericEvent

	ActionsRunner *inlocov1alpha1.ActionsRunner
	DotFiles      *dot.Files

	ghFacade  facades.GitHub
	adoFacade facades.AzureDevOps

	loopClose    chan struct{}
	loopAck      chan struct{}
	loopMessages chan Message

	gone bool
}

func (w *Wire) initGH(ctx context.Context) error {
	if err := w.ghFacade.Init(ctx, w.ActionsRunner.Spec.Repository.Owner, w.ActionsRunner.Spec.Repository.Name); err != nil {
		return err
	}
	w.DotFiles.Runner.GitHubUrl = w.ghFacade.Repository.GetGitCommitsURL()

	return nil
}

func (w *Wire) initADO(ctx context.Context, runnerEvent facades.RunnerEvent) error {
	credential, err := facades.GetGitHubTenantCredential(ctx, w.ghFacade.Repository, runnerEvent)
	if err != nil {
		return err
	}
	w.DotFiles.Runner.ServerUrl = credential.GetURL()

	return w.adoFacade.InitForCRUD(ctx, w.DotFiles, w.ActionsRunner.Spec.Labels, credential.GetToken(), credential.GetURL())
}

func (w *Wire) init(ctx context.Context) error {
	if w.DotFiles == nil {
		if err := w.initDotFiles(); err != nil {
			return err
		}

		if err := w.initGH(ctx); err != nil {
			return err
		}

		if err := w.initADO(ctx, facades.RunnerEventRegister); err != nil {
			return err
		}
	}

	if err := w.adoFacade.InitForRun(ctx, w.DotFiles, w.ActionsRunner.Spec.Labels); err != nil {
		return err
	}

	if err := w.adoFacade.InitAzureDevOpsTaskAgentSession(ctx); err != nil {
		return err
	}

	if err := w.adoFacade.DeinitAzureDevOpsTaskAgentSession(ctx); err != nil {
		return err
	}

	if w.loopAck == nil {
		w.loopAck = make(chan struct{})
	}

	if w.loopMessages == nil {
		w.loopMessages = make(chan Message)
	}

	return nil
}

func (w *Wire) destroy(ctx context.Context) error {
	if err := w.initDotFiles(); err != nil {
		return err
	}

	if err := w.initGH(ctx); err != nil {
		return err
	}

	if err := w.initADO(ctx, facades.RunnerEventRemove); err != nil {
		return err
	}

	if err := w.adoFacade.DeleteAgent(ctx); err != nil {
		return err
	}

	return nil
}

func (w *Wire) Init(ctx context.Context) error {
	logger := log.FromContext(ctx)

	err := w.init(ctx)
	if err == nil {
		return nil
	}

	if err := w.destroy(ctx); err != nil {
		logger.Error(err, err.Error())
	}

	if err := w.Close(ctx); err != nil {
		logger.Error(err, err.Error())
	}

	w.gone = true
	return err
}

func (w *Wire) initDotFiles() error {
	if w.DotFiles != nil {
		return nil
	}

	dotRSAParameters, err := dot.NewRSAParameters()
	if err != nil {
		return err
	}

	w.DotFiles = &dot.Files{
		Runner: dot.Runner{
			AgentName:  strings.ShortenString(fmt.Sprintf("KA %s %s", w.ActionsRunner.GetNamespace(), w.ActionsRunner.GetName()), 64),
			PoolId:     1,
			PoolName:   "Default",
			WorkFolder: "_work",
		},
		Credentials: dot.Credentials{
			Scheme: "OAuth",
		},
		RSAParameters: *dotRSAParameters,
	}
	return nil
}

func (w *Wire) Channels(ctx context.Context) (<-chan struct{}, <-chan Message) {
	logger := log.FromContext(ctx)

	if !w.isClosed() {
		return w.loopAck, w.loopMessages
	}

	w.loopClose = make(chan struct{})
	logger.Info("Wire Opened")

	go func() {
		genericEvent := event.GenericEvent{
			Object: w.ActionsRunner,
		}

		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("%v", r)
				logger.Error(err, err.Error())

				if err := w.trySendEvent(genericEvent); err != nil {
					logger.Error(err, err.Error())
				}

				if err := w.Close(ctx); err != nil {
					logger.Error(err, err.Error())
				}
			}
		}()

		if err := w.adoFacade.InitAzureDevOpsTaskAgentSession(ctx); err != nil {
			w.gone = true
			logger.Info("Wire Gone")

			panic(err)
		}
		defer w.adoFacade.DeinitAzureDevOpsTaskAgentSession(ctx)

		var lastMessageId *uint64
		for !w.isClosed() {
			logger.Info("Getting Message")

			taMessage, err := w.adoFacade.GetMessage(ctx, lastMessageId)
			if err != nil {
				panic(err)
			}
			if taMessage == nil {
				continue
			}

			lastMessageId = taMessage.MessageId

			message, err := toMessage(*taMessage)
			if err != nil {
				panic(err)
			}

			logger.Info("Message Received", "id", message.Id, "type", message.Type)

			if err := w.adoFacade.DeinitAzureDevOpsTaskAgentSession(ctx); err != nil {
				panic(err)
			}

			w.events <- genericEvent
			w.loopMessages <- *message
			w.loopAck <- struct{}{}

			if w.isClosed() {
				break
			}

			if err := w.adoFacade.InitAzureDevOpsTaskAgentSession(ctx); err != nil {
				panic(err)
			}

			if message.Type != MessageTypePipelineAgentJobRequest {
				logger.Info("Deleting Message", "id", message.Id, "type", message.Type)

				if err := w.adoFacade.DeleteMessage(ctx, *taMessage); err != nil {
					panic(err)
				}

				logger.Info("Message Deleted", "id", message.Id, "type", message.Type)
			}

			if message.Type == MessageTypeAgentRefresh {
				logger.Info("Deleting Agent", "id", message.Id, "type", message.Type)

				if err := w.adoFacade.DeleteAgent(ctx); err != nil {
					panic(err)
				}

				logger.Info("Agent Deleted", "id", message.Id, "type", message.Type)

				break
			}
		}
	}()

	return w.loopAck, w.loopMessages
}

func (w *Wire) Close(ctx context.Context) error {
	logger := log.FromContext(ctx)

	if w.isClosed() {
		return errors.New(".isClosed")
	}

	close(w.loopClose)
	w.adoFacade.DeinitAzureDevOpsTaskAgentSession(context.Background())

	logger.Info("Wire Closed")
	return nil
}

func (w *Wire) isClosed() bool {
	select {
	case _, ok := <-w.loopClose:
		return !ok
	default:
		return w.loopClose == nil
	}
}

func (w *Wire) trySendEvent(genericEvent event.GenericEvent) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	w.events <- genericEvent

	return nil
}
