package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ActionsRunner is a specification for an ActionsRunner resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ActionsRunner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ActionsRunnerSpec   `json:"spec"`
	Status ActionsRunnerStatus `json:"status,omitempty"`
}

// ActionsRunnerSpec is the spec for an ActionsRunner resource
type ActionsRunnerSpec struct {
	Repository   ActionsRunnerRepository   `json:"repository"`
	Capabilities []ActionsRunnerCapability `json:"capabilities,omitempty"`
	Labels       []string                  `json:"labels,omitempty"`

	EnvFrom   []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	Env       []corev1.EnvVar             `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	ServiceAccountName string              `json:"serviceAccountName,omitempty"`
	Affinity           *corev1.Affinity    `json:"affinity,omitempty"`
	Tolerations        []corev1.Toleration `json:"tolerations,omitempty"`
}

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

// ActionsRunnerStatus is the status for an ActionsRunner resource
type ActionsRunnerStatus struct {
	State              ActionsRunnerState `json:"phase,omitempty"`
	Message            string             `json:"message,omitempty"`
	LastProbeTime      metav1.Time        `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time        `json:"lastTransitionTime,omitempty"`
	ConfigMapName      string             `json:"configMapName,omitempty"`
	SecretName         string             `json:"secretName,omitempty"`
}

type ActionsRunnerState string

const (
	ActionsRunnerStatePending   ActionsRunnerState = "Pending"
	ActionsRunnerStateRunning   ActionsRunnerState = "Running"
	ActionsRunnerStateSucceeded ActionsRunnerState = "Succeeded"
	ActionsRunnerStateFailed    ActionsRunnerState = "Failed"
	ActionsRunnerStateUnknown   ActionsRunnerState = "Unknown"
)

// ActionsRunnerList is a list of ActionsRunner resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ActionsRunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ActionsRunner `json:"items"`
}
