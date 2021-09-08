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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/util"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/wire"
	controllersutil "github.com/inloco/kube-actions/operator/controllers/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets;poddisruptionbudgets/status,verbs=get;list;watch;create;update;patch;delete

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
		Watches(r.wires.EventSource(), &handler.EnqueueRequestForObject{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		WithEventFilter(controllersutil.PredicateOfFunction(eventFilter)).
		Complete(r)
}

func eventFilter(object client.Object, event controllersutil.PredicateEvent) bool {
	// ignore events for ActionsRunnerJob, except Delete
	actionsRunnerJob, isActionRunnerJob := object.(*inlocov1alpha1.ActionsRunnerJob)
	if isActionRunnerJob {
		if event == controllersutil.PredicateEventDelete {
			return true
		}

		if actionsRunnerJob.State == inlocov1alpha1.ActionsRunnerJobStateCompleted {
			return true
		}

		return false
	}

	// ignore Update for ActionsRunner
	_, isActionRunner := object.(*inlocov1alpha1.ActionsRunner)
	if isActionRunner && event == controllersutil.PredicateEventUpdate {
		return false
	}

	return true
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "namespacedName", req.NamespacedName.String())

	if r.gone {
		logger.Info("Reconciler gone")
		return ctrl.Result{}, nil
	}

	// if AR was deleted, close leftover connection to GitHub
	var actionsRunner inlocov1alpha1.ActionsRunner
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunner); {
	case apierrors.IsNotFound(err):
		logger.Info("ActionsRunner not found, TryDestroy wire")
		return ctrl.Result{}, r.wires.TryDestroy(ctx, client.ObjectKey(req.NamespacedName))
	case err != nil:
		return ctrl.Result{}, err
	}

	wire, err := r.wires.GetWire(ctx, &actionsRunner)
	if err != nil {
		logger.Error(err, "Error retrieving ActionsRunner")
		return ctrl.Result{}, client.IgnoreNotFound(r.Delete(ctx, &actionsRunner, deleteOpts...))
	}

	// reload setup in case of new connection (e.g. operator restarts)
	if wire == nil {
		var configMap corev1.ConfigMap
		if err := r.Get(ctx, req.NamespacedName, &configMap); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Error retrieving ActionsRunner's ConfigMap")
			return ctrl.Result{}, err
		}
		var secret corev1.Secret
		if err := r.Get(ctx, req.NamespacedName, &secret); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Error retrieving ActionsRunner's Secret")
			return ctrl.Result{}, err
		}

		dotFiles := util.ToDotFiles(&configMap, &secret)
		if wire, err = r.wires.MakeWire(ctx, &actionsRunner, dotFiles); err != nil {
			logger.Error(err, "Error initializing ActionsRunner's wire")
			return ctrl.Result{}, client.IgnoreNotFound(r.Delete(ctx, &actionsRunner, deleteOpts...))
		}
	}

	// if AR is pending, setup connection and create related resources
	if actionsRunner.State == inlocov1alpha1.ActionsRunnerStatePending {
		logger.Info("ActionsRunner pending")

		desiredConfigMap, err := util.ToConfigMap(wire.DotFiles, &actionsRunner, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Patch(ctx, desiredConfigMap, client.Apply, patchOpts...); err != nil {
			logger.Error(err, "Error creating ConfigMap for ActionsRunner")
			return ctrl.Result{}, err
		}

		desiredSecret, err := util.ToSecret(wire.DotFiles, &actionsRunner, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Patch(ctx, desiredSecret, client.Apply, patchOpts...); err != nil {
			logger.Error(err, "Error creating Secret for ActionsRunner")
			return ctrl.Result{}, err
		}

		desiredPodDisruptionBudget, err := util.ToPodDisruptionBudget(wire.DotFiles, &actionsRunner, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Patch(ctx, desiredPodDisruptionBudget, client.Apply, patchOpts...); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("Set ActionsRunner.State to idle ")

		actionsRunner.State = inlocov1alpha1.ActionsRunnerStateIdle
		if err := r.Update(ctx, &actionsRunner, updateOpts...); err != nil {
			logger.Error(err, "Error updating ActionsRunner.State to Idle")
			return ctrl.Result{}, err
		}

		logger.Info("Listening for events")
		wire.Listen()

		return ctrl.Result{}, nil
	}

	// if AR is idle, look for job requests to be processed
	if actionsRunner.State == inlocov1alpha1.ActionsRunnerStateIdle {
		logger.Info("ActionsRunner idle")

		select {
		case <-wire.JobRequests():
			break
		default:
			// wire should always be listening when idle, except when operator restarts or listener panics
			if !wire.Listening() {
				logger.Info("Wire not listening on idle, Listen() again")
				wire.Listen()
			}

			logger.Info("No jobs to process")
			return ctrl.Result{}, nil
		}

		logger.Info("New job request, creating ActionsRunnerJob")

		desiredActionsRunnerJob, err := util.ToActionsRunnerJob(&actionsRunner, r.Scheme)
		if err != nil {
			logger.Error(err, "Error generating ActionsRunnerJob definition")
			return ctrl.Result{}, err
		}
		if err := r.Patch(ctx, desiredActionsRunnerJob, client.Apply, patchOpts...); err != nil {
			logger.Error(err, "Error creating ActionsRunnerJob")
			return ctrl.Result{}, err
		}

		logger.Info("Set ActionsRunner.State to active ")

		actionsRunner.State = inlocov1alpha1.ActionsRunnerStateActive
		if err := r.Update(ctx, &actionsRunner, updateOpts...); err != nil {
			logger.Error(err, "Error updating ActionsRunner.State to Active")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// if AR is active and ARJ is no more, than let's listen for events again
	if actionsRunner.State == inlocov1alpha1.ActionsRunnerStateActive {
		logger.Info("ActionsRunner active")

		var actionsRunnerJob inlocov1alpha1.ActionsRunnerJob
		err := r.Get(ctx, req.NamespacedName, &actionsRunnerJob)
		if err != nil && !apierrors.IsNotFound(err) {
			logger.Error(err, "Error retrieving ActionsRunnerJob")
			return ctrl.Result{}, err
		}

		if err == nil {
			if actionsRunnerJob.State != inlocov1alpha1.ActionsRunnerJobStateCompleted {
				logger.Info(fmt.Sprintf("ActionsRunnerJob still running (%s), wait", actionsRunnerJob.State))
				return ctrl.Result{}, nil
			}

			if controllerutil.ContainsFinalizer(&actionsRunnerJob, inlocov1alpha1.ActionsRunnerJobFinalizer) {
				logger.Info("Removing finalizer")

				controllerutil.RemoveFinalizer(&actionsRunnerJob, inlocov1alpha1.ActionsRunnerJobFinalizer)
				if err := r.Update(ctx, &actionsRunnerJob); err != nil {
					logger.Error(err, "Error removing ActionsRunnerJob's finalizer")
					return ctrl.Result{}, nil
				}

				return ctrl.Result{}, nil
			}

			logger.Info("ActionsRunnerJob completed, delete it")
			if err := r.Delete(ctx, &actionsRunnerJob, deleteOpts...); client.IgnoreNotFound(err) != nil {
				logger.Error(err, "Error deleting ActionsRunnerJob")
				return ctrl.Result{}, nil
			}

			return ctrl.Result{}, nil
		}

		logger.Info("ActionsRunnerJob not present, set ActionsRunner.State to idle ")

		actionsRunner.State = inlocov1alpha1.ActionsRunnerStateIdle
		if err := r.Update(ctx, &actionsRunner, updateOpts...); err != nil {
			logger.Error(err, "Error updating ActionsRunner.State to idle")
			return ctrl.Result{}, err
		}

		wire.Listen()

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, fmt.Errorf("ActionsRunner %s in invalid state", req.NamespacedName.String())
}
