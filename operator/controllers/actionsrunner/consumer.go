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

package actionsrunner

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/wire"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Consumer struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	wire  *wire.Wire
	watch func(*inlocov1alpha1.ActionsRunnerJob, <-chan struct{})
}

func (c *Consumer) Consume(ctx context.Context) error {
	ack, msg := c.wire.Channels(ctx)
	select {
	case msg := <-msg:
		switch typ := msg.Type; typ {
		case wire.MessageTypePipelineAgentJobRequest:
			return c.onPipelineAgentJobRequest(ctx, ack, msg)
		case wire.MessageTypeJobCancellation:
			return c.onJobCancellation(ctx, ack, msg)
		case wire.MessageTypeAgentRefresh:
			return c.onAgentRefresh(ctx, ack, msg)
		default:
			return fmt.Errorf("unknown MessageType(%s)", typ)
		}
	default:
		// don't wait for channels and return if there's nothing to be consumed
		return nil
	}
}

func (c *Consumer) onPipelineAgentJobRequest(ctx context.Context, ack <-chan struct{}, message wire.Message) error {
	c.Log.Info("On PipelineAgentJobRequest")

	desiredActionsRunnerJob, err := util.ToActionsRunnerJob(c.wire.ActionsRunner, c.Scheme)
	if err != nil {
		return err
	}
	if err := c.Patch(ctx, desiredActionsRunnerJob, client.Apply, patchOpts...); err != nil {
		return err
	}

	desiredPersistentVolumeClaim, err := util.ToPersistentVolumeClaim(c.wire.ActionsRunner, desiredActionsRunnerJob, c.Scheme)
	if err != nil {
		return err
	}
	if err := c.Patch(ctx, desiredPersistentVolumeClaim, client.Apply, patchOpts...); err != nil {
		return err
	}

	desiredJob, err := util.ToJob(c.wire.ActionsRunner, desiredActionsRunnerJob, c.Scheme)
	if err != nil {
		return err
	}
	if err := c.Patch(ctx, desiredJob, client.Apply, patchOpts...); err != nil {
		return err
	}

	c.watch(desiredActionsRunnerJob, ack)
	return nil
}

func (c *Consumer) onJobCancellation(ctx context.Context, ack <-chan struct{}, message wire.Message) error {
	c.Log.Info("On JobCancellation")

	if message.Type != wire.MessageTypeJobCancellation {
		return errors.New("message.Type != MessageTypeJobCancellation")
	}

	c.Log.Info(message.String())

	actionsRunnerJob, err := util.ToActionsRunnerJob(c.wire.ActionsRunner, c.Scheme)
	if err != nil {
		return err
	}

	if err := c.Delete(ctx, actionsRunnerJob, deleteOpts...); client.IgnoreNotFound(err) != nil {
		c.Log.Error(err, "c.Delete")
	}

	<-ack
	return nil
}

// TODO: trigger upgrade procedure
func (c *Consumer) onAgentRefresh(ctx context.Context, ack <-chan struct{}, message wire.Message) error {
	c.Log.Info("On AgentRefresh")

	if message.Type != wire.MessageTypeAgentRefresh {
		return errors.New("message.Type != MessageTypeAgentRefresh")
	}

	c.Log.Info(message.String())

	<-ack
	return nil
}
