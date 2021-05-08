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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ActionsRunnerRepository struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	APIEndpoint string `json:"apiEndpoint,omitempty"`
}

type ActionsRunnerCapability string

const (
	ActionsRunnerCapabilitySecret ActionsRunnerCapability = "secret"
	ActionsRunnerCapabilityDocker ActionsRunnerCapability = "docker"
)

// ActionsRunnerSpec defines the desired state of ActionsRunner
type ActionsRunnerSpec struct {
	Repository   ActionsRunnerRepository   `json:"repository"`
	Capabilities []ActionsRunnerCapability `json:"capabilities,omitempty"`
	Labels       []string                  `json:"labels,omitempty"`

	Volumes      []corev1.Volume             `json:"volumes,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
	VolumeMounts []corev1.VolumeMount        `json:"volumeMounts,omitempty" patchStrategy:"merge" patchMergeKey:"mountPath"`
	EnvFrom      []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	Env          []corev1.EnvVar             `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	Resources    corev1.ResourceRequirements `json:"resources,omitempty"`

	ServiceAccountName string              `json:"serviceAccountName,omitempty"`
	Affinity           *corev1.Affinity    `json:"affinity,omitempty"`
	Tolerations        []corev1.Toleration `json:"tolerations,omitempty"`
}

type ActionsRunnerState string

const (
	ActionsRunnerStatePending   ActionsRunnerState = "Pending"
	ActionsRunnerStateRunning   ActionsRunnerState = "Running"
	ActionsRunnerStateSucceeded ActionsRunnerState = "Succeeded"
	ActionsRunnerStateFailed    ActionsRunnerState = "Failed"
	ActionsRunnerStateUnknown   ActionsRunnerState = "Unknown"
)

// ActionsRunnerStatus defines the observed state of ActionsRunner
type ActionsRunnerStatus struct {
	State              ActionsRunnerState `json:"phase,omitempty"`
	Message            string             `json:"message,omitempty"`
	LastProbeTime      metav1.Time        `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time        `json:"lastTransitionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ActionsRunner is the Schema for the actionsrunners API
type ActionsRunner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionsRunnerSpec   `json:"spec,omitempty"`
	Status ActionsRunnerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ActionsRunnerList contains a list of ActionsRunner
type ActionsRunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionsRunner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ActionsRunner{}, &ActionsRunnerList{})
}
