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

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var actionsrunnerreplicasetlog = logf.Log.WithName("actionsrunnerreplicaset-resource")

func (r *ActionsRunnerReplicaSet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-inloco-com-br-v1alpha1-actionsrunnerreplicaset,mutating=false,failurePolicy=fail,sideEffects=None,groups=inloco.com.br,resources=actionsrunnerreplicasets,verbs=create;update,versions=v1alpha1,name=vactionsrunnerreplicaset.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ActionsRunnerReplicaSet{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (arrs *ActionsRunnerReplicaSet) ValidateCreate() error {
	actionsrunnerreplicasetlog.Info("validate create", "name", arrs.Name)

	ar := &ActionsRunner{
		Spec: arrs.Spec.Template,
	}
	return ar.ValidateCreate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (arrs *ActionsRunnerReplicaSet) ValidateUpdate(old runtime.Object) error {
	actionsrunnerreplicasetlog.Info("validate update", "name", arrs.Name)

	oldARRS, ok := old.(*ActionsRunnerReplicaSet)
	if !ok {
		return errors.New("old.(*ActionsRunnerReplicaSet) == nil")
	}

	ar := &ActionsRunner{
		Spec: arrs.Spec.Template,
	}
	oldAR := &ActionsRunner{
		Spec: oldARRS.Spec.Template,
	}
	return ar.ValidateUpdate(oldAR)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (arrs *ActionsRunnerReplicaSet) ValidateDelete() error {
	actionsrunnerreplicasetlog.Info("validate delete", "name", arrs.Name)

	ar := &ActionsRunner{
		Spec: arrs.Spec.Template,
	}
	return ar.ValidateDelete()
}
