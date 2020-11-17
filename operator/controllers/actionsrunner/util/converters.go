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

package util

import (
	"encoding/json"
	"errors"
	"fmt"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/constants"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner/dot"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	runnerImageName    = EnvVar("KUBEACTIONS_RUNNER_IMAGE_NAME", "inloco/kube-actions")
	runnerImageVersion = EnvVar("KUBEACTIONS_RUNNER_IMAGE_VERSION", constants.Ver())
	runnerImageVariant = EnvVar("KUBEACTIONS_RUNNER_IMAGE_VARIANT", "-runner")

	dindImageName    = EnvVar("KUBEACTIONS_DIND_IMAGE_NAME", "inloco/kube-actions")
	dindImageVersion = EnvVar("KUBEACTIONS_DIND_IMAGE_VERSION", constants.Ver())
	dindImageVariant = EnvVar("KUBEACTIONS_DIND_IMAGE_VARIANT", "-dind")
)

func ToDotFiles(configMap *corev1.ConfigMap, secret *corev1.Secret) *dot.Files {
	dotFiles := dot.Files{}

	if configMap == nil {
		return nil
	}

	if configMap.BinaryData == nil {
		return nil
	}

	runner, ok := configMap.BinaryData[".runner"]
	if !ok {
		return nil
	}

	if err := json.Unmarshal(runner, &dotFiles.Runner); err != nil {
		return nil
	}

	credentials, ok := configMap.BinaryData[".credentials"]
	if !ok {
		return nil
	}

	if err := json.Unmarshal(credentials, &dotFiles.Credentials); err != nil {
		return nil
	}

	if secret == nil {
		return nil
	}

	if secret.Data == nil {
		return nil
	}

	rsaparams, ok := secret.Data[".credentials_rsaparams"]
	if !ok {
		return nil
	}

	if err := json.Unmarshal(rsaparams, &dotFiles.RSAParameters); err != nil {
		return nil
	}

	return &dotFiles
}

func ToConfigMap(dotFiles *dot.Files, actionsRunner *inlocov1alpha1.ActionsRunner, scheme *runtime.Scheme) (*corev1.ConfigMap, error) {
	if dotFiles == nil {
		return nil, errors.New("dotFiles == nil")
	}

	if actionsRunner == nil {
		return nil, errors.New("actionsRunner == nil")
	}

	if scheme == nil {
		return nil, errors.New("scheme == nil")
	}

	runner, err := json.Marshal(dotFiles.Runner)
	if err != nil {
		return nil, err
	}

	credentials, err := json.Marshal(dotFiles.Credentials)
	if err != nil {
		return nil, err
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionsRunner.GetName(),
			Namespace: actionsRunner.GetNamespace(),
		},
		BinaryData: map[string][]byte{
			".runner":      runner,
			".credentials": credentials,
		},
	}

	if err := ctrl.SetControllerReference(actionsRunner, &configMap, scheme); err != nil {
		return nil, err
	}

	return &configMap, nil
}

func ToSecret(dotFiles *dot.Files, actionsRunner *inlocov1alpha1.ActionsRunner, scheme *runtime.Scheme) (*corev1.Secret, error) {
	if dotFiles == nil {
		return nil, errors.New("dotFiles == nil")
	}

	if actionsRunner == nil {
		return nil, errors.New("actionsRunner == nil")
	}

	if scheme == nil {
		return nil, errors.New("scheme == nil")
	}

	rsaparams, err := json.Marshal(dotFiles.RSAParameters)
	if err != nil {
		return nil, err
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionsRunner.GetName(),
			Namespace: actionsRunner.GetNamespace(),
		},
		Data: map[string][]byte{
			".credentials_rsaparams": rsaparams,
		},
	}

	if err := ctrl.SetControllerReference(actionsRunner, &secret, scheme); err != nil {
		return nil, err
	}

	return &secret, nil
}

func ToActionsRunnerJob(actionsRunner *inlocov1alpha1.ActionsRunner, scheme *runtime.Scheme) (*inlocov1alpha1.ActionsRunnerJob, error) {
	if actionsRunner == nil {
		return nil, errors.New("actionsRunner == nil")
	}

	if scheme == nil {
		return nil, errors.New("scheme == nil")
	}

	actionsRunnerJob := inlocov1alpha1.ActionsRunnerJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: inlocov1alpha1.GroupVersion.String(),
			Kind:       "ActionsRunnerJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionsRunner.GetName(),
			Namespace: actionsRunner.GetNamespace(),
		},
	}

	if err := ctrl.SetControllerReference(actionsRunner, &actionsRunnerJob, scheme); err != nil {
		return nil, err
	}

	return &actionsRunnerJob, nil
}

func ToPersistentVolumeClaim(actionsRunner *inlocov1alpha1.ActionsRunner, actionsRunnerJob *inlocov1alpha1.ActionsRunnerJob, scheme *runtime.Scheme) (*corev1.PersistentVolumeClaim, error) {
	if actionsRunner == nil {
		return nil, errors.New("actionsRunner == nil")
	}

	if actionsRunnerJob == nil {
		return nil, errors.New("actionsRunnerJob == nil")
	}

	if scheme == nil {
		return nil, errors.New("scheme == nil")
	}

	persistentVolumeClaim := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionsRunner.GetName(),
			Namespace: actionsRunner.GetNamespace(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: FilterResources(actionsRunner.Spec.Resources, corev1.ResourceStorage),
		},
	}

	if err := ctrl.SetControllerReference(actionsRunnerJob, &persistentVolumeClaim, scheme); err != nil {
		return nil, err
	}

	return &persistentVolumeClaim, nil
}

func ToJob(actionsRunner *inlocov1alpha1.ActionsRunner, actionsRunnerJob *inlocov1alpha1.ActionsRunnerJob, scheme *runtime.Scheme) (*batchv1.Job, error) {
	if actionsRunner == nil {
		return nil, errors.New("actionsRunner == nil")
	}

	if actionsRunnerJob == nil {
		return nil, errors.New("actionsRunnerJob == nil")
	}

	if scheme == nil {
		return nil, errors.New("scheme == nil")
	}

	job := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionsRunner.GetName(),
			Namespace: actionsRunner.GetNamespace(),
		},
		Spec: batchv1.JobSpec{
			Parallelism:           pointer.Int32Ptr(1),
			Completions:           pointer.Int32Ptr(1),
			ActiveDeadlineSeconds: pointer.Int64Ptr(3000),
			BackoffLimit:          pointer.Int32Ptr(0),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						corev1.Volume{
							Name: "config-map",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: actionsRunner.GetName(),
									},
								},
							},
						},
						corev1.Volume{
							Name: "secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: actionsRunner.GetName(),
								},
							},
						},
						corev1.Volume{
							Name: "persistent-volume-claim",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: actionsRunner.GetName(),
								},
							},
						},
					},
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "runner",
							Image: fmt.Sprintf("%s:%s%s", runnerImageName, runnerImageVersion, runnerImageVariant),
							EnvFrom: FilterEnvFrom(actionsRunner.Spec.EnvFrom, func(envFromSource corev1.EnvFromSource) bool {
								return envFromSource.SecretRef == nil
							}),
							Env: FilterEnv(actionsRunner.Spec.Env, func(envVar corev1.EnvVar) bool {
								return envVar.ValueFrom == nil || envVar.ValueFrom.SecretKeyRef == nil
							}),
							Resources: FilterResources(actionsRunner.Spec.Resources, corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceEphemeralStorage),
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									Name:      "config-map",
									MountPath: "/opt/actions-runner/.runner",
									SubPath:   ".runner",
								},
								corev1.VolumeMount{
									Name:      "config-map",
									MountPath: "/opt/actions-runner/.credentials",
									SubPath:   ".credentials",
								},
								corev1.VolumeMount{
									Name:      "secret",
									MountPath: "/opt/actions-runner/.credentials_rsaparams",
									SubPath:   ".credentials_rsaparams",
								},
								corev1.VolumeMount{
									Name:      "persistent-volume-claim",
									MountPath: "/opt/actions-runner/_work",
									SubPath:   "runner",
								},
								corev1.VolumeMount{
									Name:      "persistent-volume-claim",
									MountPath: "/root",
									SubPath:   "root",
								},
								corev1.VolumeMount{
									Name:      "persistent-volume-claim",
									MountPath: "/home/user",
									SubPath:   "user",
								},
							},
							ImagePullPolicy: corev1.PullAlways,
						},
					},
					RestartPolicy:                corev1.RestartPolicyNever,
					AutomountServiceAccountToken: pointer.BoolPtr(false),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:    pointer.Int64Ptr(1000),
						RunAsGroup:   pointer.Int64Ptr(1000),
						RunAsNonRoot: pointer.BoolPtr(true),
						FSGroup:      pointer.Int64Ptr(1000),
					},
					Affinity:    withRuntimeAffinity(actionsRunner.Spec.Affinity),
					Tolerations: actionsRunner.Spec.Tolerations,
				},
			},
			TTLSecondsAfterFinished: pointer.Int32Ptr(0),
		},
	}

	capabilities := make(map[inlocov1alpha1.ActionsRunnerCapability]struct{})
	for _, capability := range actionsRunner.Spec.Capabilities {
		capabilities[capability] = struct{}{}
	}
	if _, ok := capabilities[inlocov1alpha1.ActionsRunnerCapabilitySecret]; ok {
		addSecretCapability(&job, actionsRunner)
	}
	if _, ok := capabilities[inlocov1alpha1.ActionsRunnerCapabilityDocker]; ok {
		addDockerCapability(&job)
	}

	if err := ctrl.SetControllerReference(actionsRunnerJob, &job, scheme); err != nil {
		return nil, err
	}

	return &job, nil
}

func withRuntimeAffinity(affinity *corev1.Affinity) *corev1.Affinity {
	if affinity == nil {
		affinity = &corev1.Affinity{}
	}

	if affinity.NodeAffinity == nil {
		affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	nodeAffinity := affinity.NodeAffinity

	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}
	nodeSelector := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution

	if len(nodeSelector.NodeSelectorTerms) == 0 {
		nodeSelector.NodeSelectorTerms = append(nodeSelector.NodeSelectorTerms, corev1.NodeSelectorTerm{})
	}
	nodeSelectorTerms := nodeSelector.NodeSelectorTerms

	for i, nodeSelectorTerm := range nodeSelectorTerms {
		nodeSelectorTerms[i].MatchExpressions = append(
			nodeSelectorTerm.MatchExpressions,
			corev1.NodeSelectorRequirement{
				Key:      "kubernetes.io/os",
				Operator: corev1.NodeSelectorOpIn,
				Values: []string{
					constants.OS(),
				},
			},
			corev1.NodeSelectorRequirement{
				Key:      "kubernetes.io/arch",
				Operator: corev1.NodeSelectorOpIn,
				Values: []string{
					constants.Arch(),
				},
			},
		)
	}

	return affinity
}

func addSecretCapability(job *batchv1.Job, actionsRunner *inlocov1alpha1.ActionsRunner) {
	podSpec := &job.Spec.Template.Spec
	runnerContainer := &podSpec.Containers[0]

	envFrom := FilterEnvFrom(actionsRunner.Spec.EnvFrom, func(envFromSource corev1.EnvFromSource) bool {
		return envFromSource.SecretRef != nil
	})
	runnerContainer.EnvFrom = append(runnerContainer.EnvFrom, envFrom...)

	env := FilterEnv(actionsRunner.Spec.Env, func(envVar corev1.EnvVar) bool {
		return envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil
	})
	runnerContainer.Env = append(runnerContainer.Env, env...)

	podSpec.ServiceAccountName = actionsRunner.Spec.ServiceAccountName
	podSpec.AutomountServiceAccountToken = pointer.BoolPtr(true)
}

func addDockerCapability(job *batchv1.Job) {
	podSpec := &job.Spec.Template.Spec
	runnerContainer := &podSpec.Containers[0]

	runnerContainer.Env = append(runnerContainer.Env, corev1.EnvVar{
		Name:  "DOCKER_HOST",
		Value: "tcp://localhost:2375",
	})

	podSpec.Containers = append(podSpec.Containers, corev1.Container{
		Name:  "dind",
		Image: fmt.Sprintf("%s:%s%s", dindImageName, dindImageVersion, dindImageVariant),
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "DOCKER_TLS_CERTDIR",
				Value: "",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      "persistent-volume-claim",
				MountPath: "/home/rootless",
				SubPath:   "dind",
			},
			corev1.VolumeMount{
				Name:      "persistent-volume-claim",
				MountPath: "/opt/actions-runner/_work",
				SubPath:   "runner",
			},
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"nc",
						"localhost", "2375",
					},
				},
			},
			InitialDelaySeconds: 3,
		},
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			Privileged: pointer.BoolPtr(true),
		},
	})
}
