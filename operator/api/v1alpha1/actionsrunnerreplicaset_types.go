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

// ActionsRunnerReplicaSetSpec defines the desired state of ActionsRunnerReplicaSet
type ActionsRunnerReplicaSetSpec struct {
	Replicas uint              `json:"replicas,omitempty"`
	Template ActionsRunnerSpec `json:"template,omitempty"`
}

// ActionsRunnerReplicaSetStatus defines the observed state of ActionsRunnerReplicaSet
type ActionsRunnerReplicaSetStatus struct {
	Replicas uint   `json:"replicas,omitempty"`
	Selector string `json:"selector,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=actions,shortName=arrs
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector

// ActionsRunnerReplicaSet is the Schema for the actionsrunnerjobs API
type ActionsRunnerReplicaSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionsRunnerReplicaSetSpec   `json:"spec,omitempty"`
	Status ActionsRunnerReplicaSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ActionsRunnerReplicaSetList contains a list of ActionsRunnerReplicaSet
type ActionsRunnerReplicaSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionsRunnerReplicaSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ActionsRunnerReplicaSet{}, &ActionsRunnerReplicaSetList{})
}
