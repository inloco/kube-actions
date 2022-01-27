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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ActionsRunnerJobSpec defines the desired state of ActionsRunnerJob
type ActionsRunnerJobSpec struct{}

// ActionsRunnerJobStatus defines the observed state of ActionsRunnerJob
type ActionsRunnerJobStatus struct {
	Phase string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=actions,shortName=arj
// +kubebuilder:subresource:status

// ActionsRunnerJob is the Schema for the actionsrunnerjobs API
type ActionsRunnerJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionsRunnerJobSpec   `json:"spec,omitempty"`
	Status ActionsRunnerJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ActionsRunnerJobList contains a list of ActionsRunnerJob
type ActionsRunnerJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionsRunnerJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ActionsRunnerJob{}, &ActionsRunnerJobList{})
}
