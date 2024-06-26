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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	controllers "github.com/inloco/kube-actions/operator/internal/controller"
	"github.com/inloco/kube-actions/operator/internal/controller/actionsrunner/util"
	"github.com/inloco/kube-actions/operator/internal/controller/actionsrunner/wire"
	"github.com/inloco/kube-actions/operator/metrics"
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
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		close(stop)

		r.gone = true
		r.wires.Deinit(context.Background())
	}()

	return ctrl.NewControllerManagedBy(mgr).
		For(&inlocov1alpha1.ActionsRunner{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&inlocov1alpha1.ActionsRunnerJob{}).
		WatchesRawSource(r.wires.EventSource(), &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		WithEventFilter(controllers.EventPredicate(eventFilter)).
		Complete(r)
}

func eventFilter(e controllers.Event) bool {
	switch o := controllers.EventObject(e); o.(type) {
	case *corev1.ConfigMap, *corev1.Secret, *policyv1.PodDisruptionBudget, *inlocov1alpha1.ActionsRunnerJob:
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
	actionsRunner.SetManagedFields(nil)

	if controllers.IsBeingDeleted(&actionsRunner) {
		logger.Info("ActionsRunner is being deleted")
		return ctrl.Result{}, nil
	}

	var configMap corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &configMap); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	if controllers.IsBeingDeleted(&configMap) {
		logger.Info("ConfigMap is being deleted")
		return ctrl.Result{}, nil
	}

	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get Secret")
		return ctrl.Result{}, err
	}

	if controllers.IsBeingDeleted(&secret) {
		logger.Info("Secret is being deleted")
		return ctrl.Result{}, nil
	}

	w, err := r.wires.WireFor(ctx, &actionsRunner, nil)
	if err != nil {
		logger.Error(err, "Failed to get Wire")

		if !wire.IsUnrecoverable(err) {
			return ctrl.Result{}, err
		}

		logger.Info("ConfigMap needs to be deleted")
		if err := r.Delete(ctx, &configMap, controllers.DeleteOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to delete ConfigMap")
		}

		logger.Info("Secret needs to be deleted")
		if err := r.Delete(ctx, &secret, controllers.DeleteOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to delete Secret")
		}

		return ctrl.Result{}, err
	}

	if controllers.IsZero(configMap) {
		logger.Info("ConfigMap needs to be created")

		desiredConfigMap, err := util.ToConfigMap(w.DotFiles, &actionsRunner, r.Scheme)
		if err != nil {
			logger.Info("Failed to build desired ConfigMap")
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, desiredConfigMap, controllers.CreateOpts...); controllers.IgnoreAlreadyExists(err) != nil {
			logger.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}
	}

	if controllers.IsZero(secret) {
		logger.Info("Secret needs to be created")

		desiredSecret, err := util.ToSecret(w.DotFiles, &actionsRunner, r.Scheme)
		if err != nil {
			logger.Info("Failed to build desired Secret")
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, desiredSecret, controllers.CreateOpts...); controllers.IgnoreAlreadyExists(err) != nil {
			logger.Error(err, "Failed to create Secret")
			return ctrl.Result{}, err
		}
	}

	desiredConfigMap, err := util.ToConfigMap(w.DotFiles, &actionsRunner, r.Scheme)
	if err != nil {
		logger.Info("Failed to build desired ConfigMap")
		return ctrl.Result{}, err
	}
	if err := r.Update(ctx, desiredConfigMap, controllers.UpdateOpts...); err != nil {
		logger.Error(err, "Failed to update ConfigMap")
		return ctrl.Result{}, err
	}

	desiredSecret, err := util.ToSecret(w.DotFiles, &actionsRunner, r.Scheme)
	if err != nil {
		logger.Info("Failed to build desired Secret")
		return ctrl.Result{}, err
	}
	if err := r.Update(ctx, desiredSecret, controllers.UpdateOpts...); err != nil {
		logger.Error(err, "Failed to update Secret")
		return ctrl.Result{}, err
	}

	var podDisruptionBudget policyv1.PodDisruptionBudget
	if err := r.Get(ctx, req.NamespacedName, &podDisruptionBudget); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get PodDisruptionBudget")
		return ctrl.Result{}, err
	}

	if controllers.IsBeingDeleted(&podDisruptionBudget) {
		logger.Info("PodDisruptionBudget is being deleted")
		return ctrl.Result{}, nil
	}

	desiredPodDisruptionBudget, err := util.ToPodDisruptionBudget(&actionsRunner, r.Scheme)
	if err != nil {
		logger.Info("Failed to build desired PodDisruptionBudget")
		return ctrl.Result{}, err
	}

	switch controllers.CalculateReconciliationAction(&podDisruptionBudget, desiredPodDisruptionBudget) {
	case controllers.ReconciliationActionCreate:
		logger.Info("PodDisruptionBudget needs to be created")

		if err := r.Create(ctx, desiredPodDisruptionBudget, controllers.CreateOpts...); controllers.IgnoreAlreadyExists(err) != nil {
			logger.Error(err, "Failed to create PodDisruptionBudget")
			return ctrl.Result{}, err
		}

	case controllers.ReconciliationActionUpdate:
		if reflect.DeepEqual(podDisruptionBudget.Spec, desiredPodDisruptionBudget.Spec) {
			break
		}

		logger.Info("PodDisruptionBudget needs to be updated")

		if err := r.Update(ctx, desiredPodDisruptionBudget, controllers.UpdateOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to update PodDisruptionBudget")
			return ctrl.Result{}, err
		}

	case controllers.ReconciliationActionDelete:
		logger.Info("PodDisruptionBudget needs to be deleted")

		if err := r.Delete(ctx, desiredPodDisruptionBudget, controllers.DeleteOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to delete PodDisruptionBudget")
			return ctrl.Result{}, err
		}
	}

	var actionsRunnerJob inlocov1alpha1.ActionsRunnerJob
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunnerJob); {
	case apierrors.IsNotFound(err):
		select {
		case <-w.JobRequests():
			logger.Info("ActionsRunnerJob needs to be created")

			desiredActionsRunnerJob, err := util.ToActionsRunnerJob(&actionsRunner, r.Scheme)
			if err != nil {
				logger.Error(err, "Failed to build desired ActionsRunnerJob")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, desiredActionsRunnerJob, controllers.CreateOpts...); controllers.IgnoreAlreadyExists(err) != nil {
				logger.Error(err, "Failed to create ActionsRunnerJob")
				return ctrl.Result{}, err
			}

			metrics.SetGitHubActionsJobAlive(actionsRunner.Spec.Repository.Name, desiredActionsRunnerJob.Name)
			return ctrl.Result{}, nil

		default:
			logger.Info("Wire needs to start listening")
			if !w.Listening() {
				w.Listen()
			}

			return ctrl.Result{}, nil
		}

	case err != nil:
		logger.Error(err, "Failed to get ActionsRunnerJob")
		return ctrl.Result{}, err
	}

	if controllers.IsBeingDeleted(&actionsRunnerJob) {
		logger.Info("ActionsRunnerJob is being deleted")
		return ctrl.Result{}, nil
	}

	persistentVolumeClaimPhase := actionsRunnerJob.Status.PersistentVolumeClaimPhase
	podPhase := actionsRunnerJob.Status.PodPhase
	logger = logger.WithValues("persistentVolumeClaimPhase", persistentVolumeClaimPhase, "podPhase", podPhase)

	var completed bool
	switch persistentVolumeClaimPhase {
	case corev1.ClaimLost:
		completed = true
	}
	switch podPhase {
	case corev1.PodSucceeded, corev1.PodFailed, corev1.PodUnknown:
		completed = true
	}

	if !completed {
		logger.Info("Waiting ActionsRunnerJob to complete")
		return ctrl.Result{}, nil
	}

	logger.Info("ActionsRunnerJob needs to be deleted")
	if err := r.Delete(ctx, &actionsRunnerJob, controllers.DeleteOpts...); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to delete ActionsRunnerJob")
		return ctrl.Result{}, err
	}

	metrics.SetGitHubActionsJobDone(actionsRunner.Spec.Repository.Name, actionsRunnerJob.Name)
	return ctrl.Result{}, nil
}
