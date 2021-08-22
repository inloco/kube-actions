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
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/wire"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	patchOpts = []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner("kube-actions"),
	}

	deleteOpts = []client.DeleteOption{
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	}
)

// Reconciler reconciles an actionsRunner object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	MaxConcurrentReconciles int

	watchers WatcherCollection
	wires    wire.Collection

	gone bool
}

// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunners/status,verbs=get;update;patch

// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunnerjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunnerjobs/status,verbs=get;update;patch

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.watchers.Init(r.Client)
	r.wires.Init()

	go func() {
		stop := make(chan os.Signal)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		close(stop)

		r.gone = true
		r.wires.Deinit()
		r.watchers.Deinit()
	}()

	return ctrl.NewControllerManagedBy(mgr).
		For(&inlocov1alpha1.ActionsRunner{}).
		Owns(&inlocov1alpha1.ActionsRunnerJob{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.Job{}).
		Watches(r.watchers.EventSource(), &handler.EnqueueRequestForObject{}).
		Watches(r.wires.EventSource(), &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if r.gone {
		logger.Info("Reconciler Gone")
		return ctrl.Result{}, nil
	}

	var actionsRunner inlocov1alpha1.ActionsRunner
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunner); {
	case apierrors.IsNotFound(err):
		return ctrl.Result{}, r.wires.TryClose(&actionsRunner)
	case err != nil:
		return ctrl.Result{}, err
	}

	var actionsRunnerJob inlocov1alpha1.ActionsRunnerJob
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunnerJob); {
	case err == nil:
		r.watchers.Watch(ctx, logger, &actionsRunnerJob, nil)
		return ctrl.Result{}, nil
	case !apierrors.IsNotFound(err):
		return ctrl.Result{}, err
	}

	var configMap corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &configMap); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	dotFiles := util.ToDotFiles(&configMap, &secret)

	wire, err := r.wires.WireFor(ctx, logger, &actionsRunner, dotFiles)
	if err != nil {
		return ctrl.Result{}, err
	}

	desiredConfigMap, err := util.ToConfigMap(wire.DotFiles, &actionsRunner, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Patch(ctx, desiredConfigMap, client.Apply, patchOpts...); err != nil {
		return ctrl.Result{}, err
	}

	desiredSecret, err := util.ToSecret(wire.DotFiles, &actionsRunner, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Patch(ctx, desiredSecret, client.Apply, patchOpts...); err != nil {
		return ctrl.Result{}, err
	}

	consumer := &Consumer{
		Client: r.Client,
		Log:    r.Log,
		Scheme: r.Scheme,

		wire: wire,
		watch: func(job *inlocov1alpha1.ActionsRunnerJob, ack <-chan struct{}) {
			r.watchers.Watch(ctx, logger, job, ack)
		},
	}
	if err := consumer.Consume(ctx); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
