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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var actionsrunnerlog = logf.Log.WithName("actionsrunner-resource")

func (r *ActionsRunner) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:verbs=create;update,sideEffects=none,admissionReviewVersions=v1,path=/validate-inloco-com-br-v1alpha1-actionsrunner,mutating=false,failurePolicy=fail,groups=inloco.com.br,resources=actionsrunners,versions=v1alpha1,name=vactionsrunner.kb.io

var _ webhook.Validator = &ActionsRunner{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ActionsRunner) ValidateCreate() error {
	if r.State == "" {
		return errors.New("missing Status.State for ActionsRunner")
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ActionsRunner) ValidateUpdate(old runtime.Object) error {
	actionsrunnerlog.Info("validate update", "name", r.Name)

	oldRunner, ok := old.(*ActionsRunner)
	if !ok {
		return errors.New("could not read old object to validate update operation")
	}

	if !reflect.DeepEqual(r.Spec.Labels, oldRunner.Spec.Labels) {
		return errors.New("ActionsRunner's labels field is immutable")
	}

	if !reflect.DeepEqual(r.Spec.Repository, oldRunner.Spec.Repository) {
		return errors.New("ActionsRunner's repository field is immutable")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ActionsRunner) ValidateDelete() error {
	return nil
}
