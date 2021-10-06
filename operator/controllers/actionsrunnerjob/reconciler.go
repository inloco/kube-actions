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

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
	controllersutil "github.com/inloco/kube-actions/operator/controllers/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
		Owns(&corev1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		WithEventFilter(controllersutil.PredicateOfFunction(eventFilter)).
		Complete(r)
}

func eventFilter(object client.Object, event controllersutil.PredicateEvent) bool {
	// ignore events for Pod, except Delete and completion Update
	if pod, isPod := object.(*corev1.Pod); isPod {
		isDelete := event == controllersutil.PredicateEventDelete
		isUpdate := event == controllersutil.PredicateEventUpdate
		if isDelete || (isUpdate && (pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded)) {
			return true
		}
		return false
	}

	// ignore events for ActionsRunnerJob, except Create
	_, isActionsRunnerJob := object.(*inlocov1alpha1.ActionsRunnerJob)
	if isActionsRunnerJob && event != controllersutil.PredicateEventCreate {
		return false
	}

	return true
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "namespacedName", req.NamespacedName.String())

	// if ARJ was deleted, ignore it
	var actionsRunnerJob inlocov1alpha1.ActionsRunnerJob
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunnerJob); {
	case apierrors.IsNotFound(err):
		logger.Info("ActionsRunnerJob not found, ignore")
		return ctrl.Result{}, nil
	case err != nil:
		logger.Error(err, "Error retrieving ActionsRunnerJob")
		return ctrl.Result{}, err
	}

	// if AR was deleted, ignore it
	actionsRunnerNamespacedName := req.NamespacedName
	actionsRunnerNamespacedName.Name = actionsRunnerJob.Labels[util.LabelRunner]
	var actionsRunner inlocov1alpha1.ActionsRunner
	switch err := r.Get(ctx, actionsRunnerNamespacedName, &actionsRunner); {
	case apierrors.IsNotFound(err):
		logger.Info("ActionsRunner not found, ignore")
		return ctrl.Result{}, nil
	case err != nil:
		logger.Error(err, "Error retrieving ActionsRunner")
		return ctrl.Result{}, err
	}

	// if ARJ is still pending, create pod
	if actionsRunnerJob.State == inlocov1alpha1.ActionsRunnerJobStatePending {
		logger.Info("ActionsRunnerJob pending, creating Pod")

		desiredPersistentVolumeClaim, err := util.ToPersistentVolumeClaim(&actionsRunner, &actionsRunnerJob, r.Scheme)
		if err != nil {
			logger.Error(err, "Error generating PVC definition")
			return ctrl.Result{}, err
		}
		if err := r.Patch(ctx, desiredPersistentVolumeClaim, client.Apply, patchOpts...); err != nil {
			logger.Error(err, "Error creating PVC")
			return ctrl.Result{}, err
		}

		desiredPod, err := util.ToPod(&actionsRunner, &actionsRunnerJob, r.Scheme)
		if err != nil {
			logger.Error(err, "Error generating Pod definition")
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, desiredPod, createOpts...); err != nil {
			logger.Error(err, "Error creating Pod")
			return ctrl.Result{}, err
		}

		logger.Info("Set ActionsRunnerJob.State to running")
		actionsRunnerJob.State = inlocov1alpha1.ActionsRunnerJobStateRunning
		if err := r.Update(ctx, &actionsRunnerJob, updateOpts...); err != nil {
			logger.Error(err, "Error updating ActionsRunnerJob.State to running")
		}

		return ctrl.Result{}, nil
	}

	// if ARJ is running, check if pod is already finished
	if actionsRunnerJob.State == inlocov1alpha1.ActionsRunnerJobStateRunning {
		logger.Info("ActionsRunnerJob running")

		var pod corev1.Pod
		err := r.Get(ctx, req.NamespacedName, &pod)

		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}

		if err == nil {
			if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
				logger.Info("Pod still running, wait")
				return ctrl.Result{}, nil
			}
		}

		// if pod is no longer present, set ARJ to completed
		logger.Info("Pod no longer active, set ActionsRunnerJob.State to completed")
		actionsRunnerJob.State = inlocov1alpha1.ActionsRunnerJobStateCompleted
		if err := r.Update(ctx, &actionsRunnerJob, updateOpts...); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// if ARJ is completed, ignore event, ARJ will be deleted
	if actionsRunnerJob.State == inlocov1alpha1.ActionsRunnerJobStateCompleted {
		logger.Info("ActionsRunnerJob completed, waiting to be deleted")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, fmt.Errorf("ActionsRunnerJob %s in invalid state", req.NamespacedName.String())
}
