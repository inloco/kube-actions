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
	"reflect"
	"syscall"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/wire"
	"github.com/inloco/kube-actions/operator/metrics"
)

var (
	createOpts = []client.CreateOption{
		client.FieldOwner("kube-actions"),
	}

	patchOpts = []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner("kube-actions"),
	}

	updateOpts = []client.UpdateOption{
		client.FieldOwner("kube-actions"),
	}

	deleteOpts = []client.DeleteOption{
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	}
)

// Reconciler reconciles an actionsRunner object
type Reconciler struct {
	client.Client
	Log                     logr.Logger
	Scheme                  *runtime.Scheme
	MaxConcurrentReconciles int

	gone  bool
	wires wire.Collection
}

// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunnerjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunnerjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.wires.Init()

	go func() {
		stop := make(chan os.Signal)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		close(stop)

		r.gone = true
		r.wires.Deinit(context.Background())
	}()

	return ctrl.NewControllerManagedBy(mgr).
		For(&inlocov1alpha1.ActionsRunner{}).
		Owns(&inlocov1alpha1.ActionsRunnerJob{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Watches(r.wires.EventSource(), &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		WithEventFilter(controllers.EventPredicate(eventFilter)).
		Complete(r)
}

func eventFilter(e controllers.Event) bool {
	switch o := controllers.EventObject(e); o.(type) {
	case *inlocov1alpha1.ActionsRunnerJob, *corev1.ConfigMap, *corev1.Secret, *policyv1.PodDisruptionBudget:
		switch e.(type) {
		case event.UpdateEvent, event.DeleteEvent:
			return true
		}

	default:
		return true
	}

	return false
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "namespacedName", req.NamespacedName.String())

	if r.gone {
		logger.Info("Reconciler gone")
		return ctrl.Result{}, nil
	}

	// TODO: add finalizer to AR
	var actionsRunner inlocov1alpha1.ActionsRunner
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunner); {
	case apierrors.IsNotFound(err):
		logger.Info("ActionsRunner not found")
		return ctrl.Result{}, r.wires.TryDestroy(ctx, req.NamespacedName)
	case err != nil:
		logger.Error(err, "Failed to get ActionsRunner")
		return ctrl.Result{}, err
	}

	var configMap corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &configMap); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get Secret")
		return ctrl.Result{}, err
	}

	w, err := r.wires.WireFor(ctx, &actionsRunner, util.ToDotFiles(&configMap, &secret))
	if err != nil {
		logger.Error(err, "Failed to get Wire")
		return ctrl.Result{}, err
	}

	if reflect.ValueOf(configMap).IsZero() {
		logger.Info("ConfigMap needs to be created")

		desiredConfigMap, err := util.ToConfigMap(w.DotFiles, &actionsRunner, r.Scheme)
		if err != nil {
			logger.Info("Failed to build desired ConfigMap")
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, desiredConfigMap, createOpts...); err != nil {
			logger.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}
	}

	if reflect.ValueOf(configMap).IsZero() {
		logger.Info("Secret needs to be created")

		desiredSecret, err := util.ToSecret(w.DotFiles, &actionsRunner, r.Scheme)
		if err != nil {
			logger.Info("Failed to build desired Secret")
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, desiredSecret, createOpts...); err != nil {
			logger.Error(err, "Failed to create Secret")
			return ctrl.Result{}, err
		}
	}

	var podDisruptionBudget policyv1.PodDisruptionBudget
	if err := r.Get(ctx, req.NamespacedName, &podDisruptionBudget); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get PodDisruptionBudget")
		return ctrl.Result{}, err
	}

	// TODO: move PDB to ARJ
	desiredPodDisruptionBudget, err := util.ToPodDisruptionBudget(&actionsRunner, r.Scheme)
	if err != nil {
		logger.Info("Failed to build desired PodDisruptionBudget")
		return ctrl.Result{}, err
	}
	if !reflect.DeepEqual(podDisruptionBudget.Spec, desiredPodDisruptionBudget.Spec) {
		logger.Info("PodDisruptionBudget needs to be patched")

		if err := r.Patch(ctx, desiredPodDisruptionBudget, client.Apply, patchOpts...); err != nil {
			logger.Error(err, "Failed to patch PodDisruptionBudget")
			return ctrl.Result{}, err
		}
	}

	var actionsRunnerJob inlocov1alpha1.ActionsRunnerJob
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunnerJob); {
	case err == nil:
		switch actionsRunnerJob.Status.Phase {
		case corev1.PodSucceeded, corev1.PodFailed:
			logger.Info("ActionsRunnerJob needs to be deleted")

			if err := r.Delete(ctx, &actionsRunnerJob, deleteOpts...); err != nil {
				logger.Error(err, "Failed to delete ActionsRunnerJob")
				return ctrl.Result{}, err
			}

			metrics.SetGitHubActionsJobDone(actionsRunner.Spec.Repository.Name, actionsRunnerJob.Name)

		default:
			logger.Info("Waiting ActionsRunnerJob to complete")
		}

	case apierrors.IsNotFound(err):
		select {
		case <-w.JobRequests():
			logger.Info("ActionsRunnerJob needs to be created")

			desiredActionsRunnerJob, err := util.ToActionsRunnerJob(&actionsRunner, r.Scheme)
			if err != nil {
				logger.Info("Failed to build desired ActionsRunnerJob")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, desiredActionsRunnerJob, createOpts...); err != nil {
				logger.Error(err, "Failed to create ActionsRunnerJob")
				return ctrl.Result{}, err
			}

			metrics.SetGitHubActionsJobAlive(actionsRunner.Spec.Repository.Name, desiredActionsRunnerJob.Name)

		default:
			logger.Info("ActionsRunnerJob needs to start listening")

			if !w.Listening() {
				w.Listen()
			}
		}

	default:
		logger.Error(err, "Failed to get ActionsRunnerJob")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
