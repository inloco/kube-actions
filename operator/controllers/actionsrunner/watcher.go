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
	"sync"
	"time"

	"github.com/go-logr/logr"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type WatcherCollection struct {
	client.Client
	events chan event.GenericEvent
	guards sync.Map
}

func (wc *WatcherCollection) Init(client client.Client) {
	wc.Client = client
	wc.events = make(chan event.GenericEvent)
}

func (wc *WatcherCollection) EventSource() source.Source {
	return &source.Channel{
		Source: wc.events,
	}
}

func (wc *WatcherCollection) Watch(ctx context.Context, log logr.Logger, actionsRunnerJob *inlocov1alpha1.ActionsRunnerJob, ack <-chan struct{}) {
	objectKey := client.ObjectKey{
		Namespace: actionsRunnerJob.GetNamespace(),
		Name:      actionsRunnerJob.GetName(),
	}

	if _, ok := wc.guards.Load(objectKey); ok {
		return
	}

	go wc.watch(ctx, log, actionsRunnerJob, ack)
}

func (wc *WatcherCollection) watch(ctx context.Context, log logr.Logger, actionsRunnerJob *inlocov1alpha1.ActionsRunnerJob, ack <-chan struct{}) {
	objectKey := client.ObjectKey{
		Namespace: actionsRunnerJob.GetNamespace(),
		Name:      actionsRunnerJob.GetName(),
	}

	defer func() {
		if ack != nil {
			<-ack
		}

		wc.guards.Delete(objectKey)
	}()

	condition := func() (done bool, err error) {
		var job batchv1.Job
		if err := wc.Get(ctx, objectKey, &job); client.IgnoreNotFound(err) != nil {
			return false, err
		}

		isActive := job.Status.Succeeded+job.Status.Failed == 0
		return !isActive, nil
	}

	log.Info("Waiting Job")

	if err := wait.PollImmediate(time.Second, time.Hour, condition); err != nil {
		log.Error(err, "wait.PollImmediate")
	}

	log.Info("Job Completed")

	var job batchv1.Job
	if err := wc.Get(ctx, objectKey, &job); client.IgnoreNotFound(err) != nil {
		log.Error(err, "wc.Get")
	}

	if job.Status.Failed > 0 {
		log.Info("Recovering from Failed Job")
		time.Sleep(time.Minute)
	}

	if err := wc.Delete(ctx, actionsRunnerJob, deleteOpts...); err != nil {
		log.Error(err, "wc.Delete")
	}
}
