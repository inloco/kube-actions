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

package actionsrunnerreplicaset

import (
	"context"
	"errors"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers"
)

func matchingLabels(actionsRunnerReplicaSet inlocov1alpha1.ActionsRunnerReplicaSet) client.MatchingLabels {
	return client.MatchingLabels{
		"kube-actions.inloco.com.br/actions-runner-replica-set": actionsRunnerReplicaSet.GetName(),
	}
}

func listOpts(actionsRunnerReplicaSet inlocov1alpha1.ActionsRunnerReplicaSet) []client.ListOption {
	return []client.ListOption{
		client.InNamespace(actionsRunnerReplicaSet.GetNamespace()),
		matchingLabels(actionsRunnerReplicaSet),
	}
}

func desiredActionsRunner(actionsRunnerReplicaSet *inlocov1alpha1.ActionsRunnerReplicaSet, scheme *runtime.Scheme) (*inlocov1alpha1.ActionsRunner, error) {
	if actionsRunnerReplicaSet == nil {
		return nil, errors.New("actionsRunnerReplicaSet == nil")
	}

	if scheme == nil {
		return nil, errors.New("scheme == nil")
	}

	actionsRunner := inlocov1alpha1.ActionsRunner{
		TypeMeta: metav1.TypeMeta{
			APIVersion: inlocov1alpha1.GroupVersion.String(),
			Kind:       "ActionsRunner",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: actionsRunnerReplicaSet.GetName() + "-",
			Namespace:    actionsRunnerReplicaSet.GetNamespace(),
			Labels:       matchingLabels(*actionsRunnerReplicaSet),
		},
		Spec: actionsRunnerReplicaSet.Spec.Template,
	}

	if err := ctrl.SetControllerReference(actionsRunnerReplicaSet, &actionsRunner, scheme); err != nil {
		return nil, err
	}

	return &actionsRunner, nil
}

func desiredSelector(actionsRunnerReplicaSet *inlocov1alpha1.ActionsRunnerReplicaSet) (string, error) {
	if actionsRunnerReplicaSet == nil {
		return "", errors.New("actionsRunnerReplicaSet == nil")
	}

	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: matchingLabels(*actionsRunnerReplicaSet),
	})
	if err != nil {
		return "", err
	}

	return selector.String(), nil
}

// Reconciler reconciles an ActionsRunnerReplicaSet object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	MaxConcurrentReconciles int
}

// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunnerreplicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunnerreplicasets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunner,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inloco.com.br,resources=actionsrunner/status,verbs=get;update;patch

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&inlocov1alpha1.ActionsRunnerReplicaSet{}).
		Owns(&inlocov1alpha1.ActionsRunner{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "namespacedName", req.NamespacedName.String())

	var actionsRunnerReplicaSet inlocov1alpha1.ActionsRunnerReplicaSet
	switch err := r.Get(ctx, req.NamespacedName, &actionsRunnerReplicaSet); {
	case apierrors.IsNotFound(err):
		logger.Info("ActionsRunnerReplicaSet not found")
		return ctrl.Result{}, nil
	case err != nil:
		logger.Error(err, "Failed to get ActionsRunnerReplicaSet")
		return ctrl.Result{}, err
	}
	actionsRunnerReplicaSet.SetManagedFields(nil)

	if controllers.IsBeingDeleted(&actionsRunnerReplicaSet) {
		logger.Info("ActionsRunnerReplicaSet is being deleted")
		return ctrl.Result{}, nil
	}

	selector, err := desiredSelector(&actionsRunnerReplicaSet)
	if err != nil {
		return ctrl.Result{}, err
	}

	if actionsRunnerReplicaSet.Status.Selector != selector {
		logger.Info("ActionsRunnerReplicaSetStatus needs to be updated")
		actionsRunnerReplicaSet.Status.Selector = selector

		if err := r.Status().Update(ctx, &actionsRunnerReplicaSet, controllers.UpdateOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to update ActionsRunnerReplicaSetStatus")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	var actionsRunnerList inlocov1alpha1.ActionsRunnerList
	if err := r.List(ctx, &actionsRunnerList, listOpts(actionsRunnerReplicaSet)...); err != nil {
		return ctrl.Result{}, err
	}
	items := actionsRunnerList.Items

	actionsRunners := make([]inlocov1alpha1.ActionsRunner, 0, len(items))
	for _, actionsRunner := range items {
		if !controllers.IsBeingDeleted(&actionsRunner) {
			actionsRunners = append(actionsRunners, actionsRunner)
		}
	}

	actual := len(actionsRunners)
	desired := int(actionsRunnerReplicaSet.Spec.Replicas)

	if actual < desired {
		actionsRunner, err := desiredActionsRunner(&actionsRunnerReplicaSet, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		logger := logger.WithValues("actionsRunner", actionsRunner.GetName())
		logger.Info("Less replicas than desired, creating ActionsRunner")

		if err := r.Create(ctx, actionsRunner, controllers.CreateOpts...); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("ActionsRunnerReplicaSetStatus needs to be updated")
		actionsRunnerReplicaSet.Status.Replicas = uint(actual + 1)

		if err := r.Status().Update(ctx, &actionsRunnerReplicaSet, controllers.UpdateOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to update ActionsRunnerReplicaSetStatus")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if actual > desired {
		// TODO: prioritize deletion of idle ARs
		actionsRunner := &items[0]

		logger := logger.WithValues("actionsRunner", actionsRunner.GetName())
		logger.Info("More replicas than desired, deleting ActionsRunner")

		if err := r.Delete(ctx, actionsRunner, controllers.DeleteOpts...); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("ActionsRunnerReplicaSetStatus needs to be updated")
		actionsRunnerReplicaSet.Status.Replicas = uint(actual - 1)

		if err := r.Status().Update(ctx, &actionsRunnerReplicaSet, controllers.UpdateOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to update ActionsRunnerReplicaSetStatus")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	for _, actionsRunner := range actionsRunners {
		if reflect.DeepEqual(actionsRunner.Spec, actionsRunnerReplicaSet.Spec.Template) {
			continue
		}

		logger := logger.WithValues("actionsRunner", actionsRunner.GetName())
		actionsRunner.SetManagedFields(nil)

		logger.Info("Undesired spec, updating ActionsRunner")
		actionsRunner.Spec = actionsRunnerReplicaSet.Spec.Template

		if err := r.Update(ctx, &actionsRunner, controllers.UpdateOpts...); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to update ActionsRunner")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
