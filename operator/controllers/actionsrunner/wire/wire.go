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

	"github.com/microsoft/azure-devops-go-api/azuredevops/task"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/dot"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/facades"
	"github.com/inloco/kube-actions/operator/metrics"
	"k8s.io/utils/strings"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Wire struct {
	operatorNotifier chan<- event.GenericEvent

	actionsRunner *inlocov1alpha1.ActionsRunner
	DotFiles      *dot.Files

	ghFacade  facades.GitHub
	adoFacade facades.AzureDevOps

	jobRequests chan struct{}
	loopClose   chan struct{}

	invalid   bool
	listening bool

	validator *PolicyValidator
}

func (w *Wire) initGH(ctx context.Context) error {
	if err := w.ghFacade.Init(ctx, w.actionsRunner.Spec.Repository.Owner, w.actionsRunner.Spec.Repository.Name); err != nil {
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

	return w.adoFacade.InitForCRUD(ctx, w.DotFiles, w.actionsRunner.Spec.Labels, credential.GetToken(), credential.GetURL())
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

	if err := w.adoFacade.InitForRun(ctx, w.DotFiles, w.actionsRunner.Spec.Labels); err != nil {
		return err
	}

	if w.jobRequests == nil {
		w.jobRequests = make(chan struct{}, 1)
	}

	if w.validator == nil {
		w.validator = NewPolicyValidator()
	}

	return nil
}

func (w *Wire) GetRunnerName() string {
	return fmt.Sprintf("%s/%s", w.actionsRunner.GetNamespace(), w.actionsRunner.GetName())
}

func (w *Wire) Destroy() error {
	ctx := context.Background()

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

	if err := w.Destroy(); err != nil {
		logger.Error(err, "Error destroying wire")
	}

	if err := w.Close(); err != nil {
		logger.Error(err, "Error closing wire")
	}

	w.invalid = true

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
			AgentName:  strings.ShortenString(fmt.Sprintf("KA %s %s", w.actionsRunner.GetNamespace(), w.actionsRunner.GetName()), 64),
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

func (w *Wire) JobRequests() <-chan struct{} {
	return w.jobRequests
}

func (w *Wire) Valid() bool {
	return !w.invalid
}

func (w *Wire) Listening() bool {
	return w.listening
}

func (w *Wire) Listen() {
	ctx := context.Background()
	logger := log.FromContext(ctx, "runner", w.GetRunnerName())

	w.loopClose = make(chan struct{})
	logger.Info("Wire opened")

	go func() {
		genericEvent := event.GenericEvent{
			Object: w.actionsRunner,
		}

		defer func() {
			w.listening = false

			if err := w.Close(); err != nil {
				logger.Error(err, "Error closing agent session")
			}

			if r := recover(); r != nil {
				logger.Error(fmt.Errorf("%v", r), "Recovering from error in wire listener")

				// trigger reconciliation on error to setup listener again
				logger.Info("Trigger reconciliation to setup listener again")
				if err := w.trySendEvent(genericEvent); err != nil {
					logger.Error(err, "Error notifying event on recover")
				}
			}
		}()

		if err := w.adoFacade.InitAzureDevOpsTaskAgentSession(ctx); err != nil {
			w.invalid = true
			logger.Info("Wire gone")
			panic(err)
		}

		w.listening = true

		logger.Info("Getting message")

		var lastMessageId *uint64

		for !w.isClosed() {
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

			logger.Info("Message received", "id", message.Id, "type", message.Type)
			metrics.IncGitHubActionsEventCounter(w.actionsRunner.GetNamespace(), w.GetRunnerName(), string(message.Type))

			if message.Type == MessageTypePipelineAgentJobRequest {
				pajr, err := toPipelineAgentJobRequest(message)
				if err != nil {
					panic(err)
				}

				violatedRule, err := w.validator.Validate(ctx, &w.actionsRunner.Spec.Policy, pajr)
				if err != nil {
					panic(err)
				}

				if violatedRule == nil {
					logger.Info("PipelineAgentJobRequest validated, notifying reconciler and disabling listener")
					w.jobRequests <- struct{}{}
					w.operatorNotifier <- genericEvent
					break
				}

				logger.Info("PipelineAgentJobRequest aborted, job request violated rule: %s", violatedRule)
				if err := w.onPolicyViolation(ctx, pajr, violatedRule); err != nil {
					panic(err)
				}
			}

			// send ack unless it's a job request (ack will be sent by actual runner)
			logger.Info("Deleting message", "id", message.Id, "type", message.Type)
			if err := w.adoFacade.DeleteMessage(ctx, uint64(message.Id)); err != nil {
				panic(err)
			}
			logger.Info("Message deleted", "id", message.Id, "type", message.Type)

			if message.Type == MessageTypeAgentRefresh {
				logger.Info("AgentRefresh, deleting agent", "id", message.Id, "type", message.Type)
				if err := w.adoFacade.DeleteAgent(ctx); err != nil {
					panic(err)
				}
				logger.Info("Agent deleted", "id", message.Id, "type", message.Type)
			}
		}

		logger.Info("Stop listening")
	}()
}

func (w *Wire) onPolicyViolation(ctx context.Context, pajr *PipelineAgentJobRequest, violatedRule *inlocov1alpha1.ActionsRunnerPolicyRule) error {
	if violatedRule == nil {
		return errors.New("violatedRule == nil")
	}

	if pajr == nil {
		return errors.New("pajr == nil")
	}

	if pajr.JobId == nil {
		return errors.New("pajr.JobId == nil")
	}

	if pajr.RequestId == nil {
		return errors.New("pajr.RequestId == nil")
	}

	if pajr.Resources == nil {
		return errors.New("pajr.Resources == nil")
	}

	if pajr.Resources.Endpoints == nil {
		return errors.New("pajr.Resources.Endpoints == nil")
	}

	if err := w.adoFacade.InitAzureDevOpsTaskClient(pajr.Plan, *pajr.Resources.Endpoints); err != nil {
		return err
	}

	return w.adoFacade.RaisePlanEvent(ctx, &task.JobEvent{
		Name:      JobCompleted.StringReference(),
		JobId:     pajr.JobId,
		RequestId: pajr.RequestId,
		Result:    &task.TaskResultValues.Failed,
	})
}

func (w *Wire) Close() error {
	ctx := context.Background()
	logger := log.FromContext(ctx, "runner", w.GetRunnerName())

	if !w.isClosed() {
		close(w.loopClose)
	}

	logger.Info("Closing wire")
	if err := w.adoFacade.DeinitAzureDevOpsTaskAgentSession(ctx); err != nil {
		return err
	}

	logger.Info("Wire closed")
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

	w.operatorNotifier <- genericEvent

	return nil
}
