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
	"sync"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/dot"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Collection struct {
	eventChannel chan event.GenericEvent
	wireRegistry sync.Map // map[client.ObjectKey]*Wire
}

func (c *Collection) Init() {
	c.eventChannel = make(chan event.GenericEvent)
}

func (c *Collection) Deinit(ctx context.Context) {
	logger := log.FromContext(ctx)

	close(c.eventChannel)

	c.wireRegistry.Range(func(key, i interface{}) bool {
		if err := i.(*Wire).Close(); err != nil {
			logger.Error(err, "Error closing wire on deinit")
		}

		return true
	})
}

func (c *Collection) EventSource() source.Source {
	return &source.Channel{
		Source: c.eventChannel,
	}
}

func (c *Collection) WireFor(ctx context.Context, actionsRunner *inlocov1alpha1.ActionsRunner, dotFiles *dot.Files) (*Wire, bool, error) {
	if actionsRunner == nil {
		return nil, false, errors.New("ActionsRunner == nil")
	}

	namespacedName := client.ObjectKey{
		Namespace: actionsRunner.GetNamespace(),
		Name:      actionsRunner.GetName(),
	}

	logger := log.FromContext(ctx, "runner", namespacedName.String())

	if i, ok := c.wireRegistry.Load(namespacedName); ok {
		if wire, ok := i.(*Wire); ok {
			return wire, false, nil
		}
	}

	wire := &Wire{
		operatorNotifier: c.eventChannel,
		actionsRunner:    actionsRunner,
		DotFiles:         dotFiles,
	}

	logger.Info("Initializing Wire")

	if err := wire.Init(ctx); err != nil {
		return nil, false, err
	}

	logger.Info("Wire Initialized")

	c.wireRegistry.Store(namespacedName, wire)
	return wire, true, nil
}

func (c *Collection) TryDestroy(ctx context.Context, namespacedName client.ObjectKey) error {
	i, ok := c.wireRegistry.LoadAndDelete(namespacedName)
	if !ok {
		return nil
	}

	wire, ok := i.(*Wire)
	if !ok {
		return nil
	}

	if err := wire.Close(); err != nil {
		return err
	}

	if err := wire.Destroy(); err != nil {
		return err
	}

	return nil
}
