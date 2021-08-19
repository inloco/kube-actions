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
	"reflect"
	"sync"

	"github.com/go-logr/logr"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/dot"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Collection struct {
	eventChannel chan event.GenericEvent
	wireRegistry sync.Map // map[client.ObjectKey]*Wire
}

func (c *Collection) Init() {
	c.eventChannel = make(chan event.GenericEvent)
}

func (c *Collection) Deinit() {
	close(c.eventChannel)

	c.wireRegistry.Range(func(key, iwire interface{}) bool {
		wire := iwire.(*Wire)
		if err := wire.Close(); err != nil {
			wire.log.Error(err, err.Error())
		}
		return true
	})
}

func (c *Collection) EventSource() source.Source {
	return &source.Channel{
		Source: c.eventChannel,
	}
}

func (c *Collection) WireFor(ctx context.Context, log logr.Logger, actionsRunner *inlocov1alpha1.ActionsRunner, dotFiles *dot.Files) (*Wire, error) {
	if actionsRunner == nil {
		return nil, errors.New("ActionsRunner == nil")
	}

	namespacedName := client.ObjectKey{
		Namespace: actionsRunner.GetNamespace(),
		Name:      actionsRunner.GetName(),
	}

	if iwire, ok := c.wireRegistry.Load(namespacedName); ok {
		wire := iwire.(*Wire)
		if !wire.gone {
			if reflect.DeepEqual(*wire.ActionsRunner, *actionsRunner) {
				return wire, nil
			}

			if err := wire.Close(); err != nil {
				wire.log.Error(err, err.Error())
			}
		}
	}

	wire := &Wire{
		log:           log,
		events:        c.eventChannel,
		ActionsRunner: actionsRunner,
		DotFiles:      dotFiles,
	}

	log.Info("Initializing Wire")

	if err := wire.Init(ctx); err != nil {
		return nil, err
	}

	log.Info("Wire Initialized")

	c.wireRegistry.Store(namespacedName, wire)
	return wire, nil
}

func (c *Collection) TryClose(actionsRunner *inlocov1alpha1.ActionsRunner) error {
	if actionsRunner == nil {
		return errors.New("ActionsRunner == nil")
	}

	namespacedName := client.ObjectKey{
		Namespace: actionsRunner.GetNamespace(),
		Name:      actionsRunner.GetName(),
	}

	iwire, ok := c.wireRegistry.Load(namespacedName)
	if !ok {
		return nil
	}

	wire := iwire.(*Wire)
	if wire.gone {
		return nil
	}

	return wire.Close()
}
