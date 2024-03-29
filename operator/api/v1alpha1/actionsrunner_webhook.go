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
	"fmt"
	"reflect"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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
func (ar *ActionsRunner) ValidateCreate() (warnings admission.Warnings, err error) {
	actionsrunnerlog.Info("validate create", "name", ar.Name)

	for _, policyRule := range ar.Spec.Policy.Must {
		if err := validatePolicyRule(policyRule); err != nil {
			return nil, err
		}
	}

	for _, policyRule := range ar.Spec.Policy.MustNot {
		if err := validatePolicyRule(policyRule); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (ar *ActionsRunner) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	actionsrunnerlog.Info("validate update", "name", ar.Name)

	oldAR, ok := old.(*ActionsRunner)
	if !ok {
		return nil, errors.New("old.(*ActionsRunner) == nil")
	}

	if !reflect.DeepEqual(ar.Spec.Repository, oldAR.Spec.Repository) {
		return nil, errors.New(".Spec.Repository is immutable")
	}

	if !reflect.DeepEqual(ar.Spec.Labels, oldAR.Spec.Labels) {
		return nil, errors.New(".Spec.Labels is immutable")
	}

	for _, policyRule := range ar.Spec.Policy.Must {
		if err := validatePolicyRule(policyRule); err != nil {
			return nil, err
		}
	}

	for _, policyRule := range ar.Spec.Policy.MustNot {
		if err := validatePolicyRule(policyRule); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (ar *ActionsRunner) ValidateDelete() (admission.Warnings, error) {
	actionsrunnerlog.Info("validate delete", "name", ar.Name)

	return nil, nil
}

func validatePolicyRule(policyRule ActionsRunnerPolicyRule) error {
	q, err := gojq.Parse(string(policyRule))
	if err != nil {
		return fmt.Errorf("unable to parse policy rule `%s`: %w", policyRule, err)
	}

	c, err := gojq.Compile(q)
	if err != nil {
		return fmt.Errorf("unable to compile policy rule `%s`: %w", policyRule, err)
	}

	var input map[string]interface{}
	it := c.Run(input)

	v, ok := it.Next()
	if !ok {
		return nil
	}

	if err, ok := v.(error); ok {
		return fmt.Errorf("unable to run policy rule `%s`: %w", policyRule, err)
	}

	return nil
}
