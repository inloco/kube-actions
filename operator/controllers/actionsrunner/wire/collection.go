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

	"github.com/go-logr/logr"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/dot"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Collection struct {
	eventChannel chan event.GenericEvent
	wireRegistry map[client.ObjectKey]*Wire
}

func (c *Collection) Init() {
	c.eventChannel = make(chan event.GenericEvent)
	c.wireRegistry = make(map[client.ObjectKey]*Wire)
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

	if dotFiles == nil {
		return nil, errors.New("DotFiles == nil")
	}

	namespacedName := client.ObjectKey{
		Namespace: actionsRunner.GetNamespace(),
		Name:      actionsRunner.GetName(),
	}

	if wire, ok := c.wireRegistry[namespacedName]; ok {
		return wire, nil
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

	c.wireRegistry[namespacedName] = wire
	return wire, nil
}
