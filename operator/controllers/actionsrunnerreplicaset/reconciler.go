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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
)

var (
	createOpts = []client.CreateOption{
		client.FieldOwner("kube-actions"),
	}

	updateOpts = []client.UpdateOption{
		client.FieldOwner("kube-actions"),
	}

	deleteOpts = []client.DeleteOption{
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	}
)

func listOpts(actionsRunnerReplicaSet inlocov1alpha1.ActionsRunnerReplicaSet) []client.ListOption {
	return []client.ListOption{
		client.InNamespace(actionsRunnerReplicaSet.GetNamespace()),
		client.MatchingLabels{
			"kube-actions.inloco.com.br/actions-runner-replica-set": actionsRunnerReplicaSet.GetName(),
		},
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
			Labels: map[string]string{
				"kube-actions.inloco.com.br/actions-runner-replica-set": actionsRunnerReplicaSet.GetName(),
			},
		},
		Spec: actionsRunnerReplicaSet.Spec.Template,
	}

	if err := ctrl.SetControllerReference(actionsRunnerReplicaSet, &actionsRunner, scheme); err != nil {
		return nil, err
	}

	return &actionsRunner, nil
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
	logger := log.FromContext(ctx)

	var actionsRunnerReplicaSet inlocov1alpha1.ActionsRunnerReplicaSet
	if err := r.Get(ctx, req.NamespacedName, &actionsRunnerReplicaSet); err != nil {
		return ctrl.Result{}, err
	}
	actionsRunnerReplicaSet.SetManagedFields(nil)

	var actionsRunnerList inlocov1alpha1.ActionsRunnerList
	if err := r.List(ctx, &actionsRunnerList, listOpts(actionsRunnerReplicaSet)...); err != nil {
		return ctrl.Result{}, err
	}

	expected := int(actionsRunnerReplicaSet.Spec.Replicas)
	items := actionsRunnerList.Items

	if len(items) < expected {
		actionsRunner, err := desiredActionsRunner(&actionsRunnerReplicaSet, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Less replicas than expected, creating ActionsRunner " + actionsRunner.GetGenerateName())

		return ctrl.Result{}, r.Create(ctx, actionsRunner, createOpts...)
	}

	if len(items) > expected {
		// TODO: prioritize deletion of idle ARs
		actionsRunner := &items[0]
		logger.Info("More replicas than expected, deleting ActionsRunner " + actionsRunner.GetName())

		return ctrl.Result{}, r.Delete(ctx, actionsRunner, deleteOpts...)
	}

	for _, item := range items {
		if !reflect.DeepEqual(item.Spec, actionsRunnerReplicaSet.Spec.Template) {
			actionsRunner := item

			logger.Info("Undesired replica, patching ActionsRunner " + actionsRunner.GetName())
			actionsRunner.Spec = actionsRunnerReplicaSet.Spec.Template
			return ctrl.Result{}, r.Update(ctx, &actionsRunner, updateOpts...)
		}
	}

	return ctrl.Result{}, nil
}
