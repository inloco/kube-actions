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

	"github.com/go-logr/logr"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
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
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&inlocov1alpha1.ActionsRunnerJob{}).
		Owns(&batchv1.Job{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
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
	var actionsRunner inlocov1alpha1.ActionsRunner
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunner); {
	case apierrors.IsNotFound(err):
		logger.Info("ActionsRunner not found, delete ActionsRunnerJob")
		return ctrl.Result{}, client.IgnoreNotFound(r.deleteActionsRunnerJob(ctx, logger, &actionsRunnerJob))
	case err != nil:
		logger.Error(err, "Error retrieving ActionsRunner")
		return ctrl.Result{}, err
	}

	// if ARJ is still pending, create job
	if actionsRunnerJob.State == inlocov1alpha1.ActionsRunnerJobStatePending {
		logger.Info("ActionsRunnerJob pending")

		desiredPersistentVolumeClaim, err := util.ToPersistentVolumeClaim(&actionsRunner, &actionsRunnerJob, r.Scheme)
		if err != nil {
			logger.Error(err, "Error generating PVC definition")
			return ctrl.Result{}, err
		}
		if err := r.Patch(ctx, desiredPersistentVolumeClaim, client.Apply, patchOpts...); err != nil {
			logger.Error(err, "Error creating PVC")
			return ctrl.Result{}, err
		}

		desiredJob, err := util.ToJob(&actionsRunner, &actionsRunnerJob, r.Scheme)
		if err != nil {
			logger.Error(err, "Error generating Job definition")
			return ctrl.Result{}, err
		}
		if err := r.Patch(ctx, desiredJob, client.Apply, patchOpts...); err != nil {
			logger.Error(err, "Error creating Job")
			return ctrl.Result{}, err
		}

		actionsRunnerJob.State = inlocov1alpha1.ActionsRunnerJobStateRunning
		if err := r.Update(ctx, &actionsRunnerJob, updateOpts...); err != nil {
			logger.Error(err, "Error updating ActionsRunnerJob.State to running")
		}

		return ctrl.Result{}, nil
	}

	// if ARJ is running, check if job is already finished
	if actionsRunnerJob.State == inlocov1alpha1.ActionsRunnerJobStateRunning {
		logger.Info("ActionsRunnerJob running")

		var job batchv1.Job
		err := r.Get(ctx, req.NamespacedName, &job)

		if err != nil && !apierrors.IsNotFound(err) {
			logger.Error(err, "Error retrieving Job")
			return ctrl.Result{}, err
		}

		// if job is still running, ignore
		if err == nil && job.Status.Succeeded+job.Status.Failed == 0 {
			logger.Info("Job still running, wait")
			return ctrl.Result{}, nil
		}

		logger.Info("Job finished running, deleting ActionsRunnerJob")
		if err := r.deleteActionsRunnerJob(ctx, logger, &actionsRunnerJob); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, fmt.Errorf("ActionsRunnerJob %s in invalid state", req.NamespacedName.String())
}

func (r *Reconciler) deleteActionsRunnerJob(ctx context.Context, logger logr.Logger, actionsRunnerJob *inlocov1alpha1.ActionsRunnerJob) error {
	controllerutil.RemoveFinalizer(actionsRunnerJob, inlocov1alpha1.ActionsRunnerJobFinalizer)
	if err := r.Update(ctx, actionsRunnerJob); err != nil {
		logger.Error(err, "Error removing ActionsRunnerJob's finalizer")
		return err
	}

	if err := r.Delete(ctx, actionsRunnerJob, deleteOpts...); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Error deleting ActionsRunnerJob")
		return err
	}

	return nil
}
