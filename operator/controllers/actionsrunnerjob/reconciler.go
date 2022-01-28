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

package actionsrunnerjob

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
)

// Reconciler reconciles an ActionsRunnerJob object
type Reconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	MaxConcurrentReconciles int
}

// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunnerjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunnerjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&inlocov1alpha1.ActionsRunnerJob{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		WithEventFilter(controllers.EventPredicate(eventFilter)).
		Complete(r)
}

func eventFilter(e controllers.Event) bool {
	switch o := controllers.EventObject(e); o.(type) {
	case *inlocov1alpha1.ActionsRunnerJob:
		switch e.(type) {
		case event.CreateEvent, event.UpdateEvent:
			return true
		}

	case *corev1.Pod, *corev1.PersistentVolumeClaim:
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

	// TODO: add finalizer to ARJ
	var actionsRunnerJob inlocov1alpha1.ActionsRunnerJob
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunnerJob); {
	case apierrors.IsNotFound(err):
		logger.Info("ActionsRunnerJob not found")
		return ctrl.Result{}, nil
	case err != nil:
		logger.Error(err, "Failed to get ActionsRunnerJob")
		return ctrl.Result{}, err
	}
	actionsRunnerJob.SetManagedFields(nil)

	var actionsRunner inlocov1alpha1.ActionsRunner
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunner); {
	case apierrors.IsNotFound(err):
		logger.Info("ActionsRunner not found")
		return ctrl.Result{}, nil
	case err != nil:
		logger.Error(err, "Failed to get ActionsRunner")
		return ctrl.Result{}, err
	}

	var persistentVolumeClaim corev1.PersistentVolumeClaim
	switch err := r.Get(ctx, req.NamespacedName, &persistentVolumeClaim); {
	case apierrors.IsNotFound(err):
		desiredPersistentVolumeClaim, err := util.ToPersistentVolumeClaim(&actionsRunner, &actionsRunnerJob, r.Scheme)
		if err != nil {
			logger.Info("Failed to build desired PersistentVolumeClaim")
			return ctrl.Result{}, err
		}
		if desiredPersistentVolumeClaim != nil {
			logger.Info("PersistentVolumeClaim needs to be created")

			if err := r.Create(ctx, desiredPersistentVolumeClaim, controllers.CreateOpts...); controllers.IgnoreAlreadyExists(err) != nil {
				logger.Error(err, "Failed to create PersistentVolumeClaim")
				return ctrl.Result{}, err
			}
		}

	case err != nil:
		logger.Error(err, "Failed to get PersistentVolumeClaim")
		return ctrl.Result{}, err
	}

	pvcPhase := persistentVolumeClaim.Status.Phase
	if actionsRunnerJob.Status.PersistentVolumeClaimPhase != pvcPhase {
		logger.Info("PersistentVolumeClaimPhase changed", "phase", pvcPhase)
		actionsRunnerJob.Status.PersistentVolumeClaimPhase = pvcPhase

		logger.Info("ActionsRunnerJobStatus needs to be patched")
		if err := r.Status().Patch(ctx, &actionsRunnerJob, client.Apply, controllers.PatchOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to patch ActionsRunnerJobStatus")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	switch pvcPhase {
	case corev1.ClaimPending:
		return ctrl.Result{}, nil
	case corev1.ClaimLost:
		return ctrl.Result{}, fmt.Errorf("corev1.Claim%s", pvcPhase)
	}

	var pod corev1.Pod
	switch err := r.Get(ctx, req.NamespacedName, &pod); {
	case apierrors.IsNotFound(err):
		desiredPod, err := util.ToPod(&actionsRunner, &actionsRunnerJob, r.Scheme)
		if err != nil {
			logger.Info("Failed to build desired Pod")
			return ctrl.Result{}, err
		}
		if desiredPod != nil {
			logger.Info("Pod needs to be created")

			if err := r.Create(ctx, desiredPod, controllers.CreateOpts...); controllers.IgnoreAlreadyExists(err) != nil {
				logger.Error(err, "Failed to create Pod")
				return ctrl.Result{}, err
			}
		}

	case err != nil:
		logger.Error(err, "Failed to get Pod")
		return ctrl.Result{}, err
	}

	podPhase := pod.Status.Phase
	if actionsRunnerJob.Status.PodPhase != podPhase {
		logger.Info("PodPhase changed", "phase", podPhase)
		actionsRunnerJob.Status.PodPhase = podPhase

		logger.Info("ActionsRunnerJobStatus needs to be patched")
		if err := r.Status().Patch(ctx, &actionsRunnerJob, client.Apply, controllers.PatchOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to patch ActionsRunnerJobStatus")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	switch podPhase {
	case corev1.PodPending:
		return ctrl.Result{}, nil
	case corev1.PodUnknown:
		return ctrl.Result{}, fmt.Errorf("corev1.Pod%s", podPhase)
	}

	return ctrl.Result{}, nil
}
