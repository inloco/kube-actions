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

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/constants"
	controllers "github.com/inloco/kube-actions/operator/internal/controller"
	"github.com/inloco/kube-actions/operator/internal/controller/actionsrunner/dot"
)

var (
	runnerImageName    = getEnv("KUBEACTIONS_RUNNER_IMAGE_NAME", "inloco/kube-actions")
	runnerImageVersion = getEnv("KUBEACTIONS_RUNNER_IMAGE_VERSION", constants.Ver())
	runnerImageVariant = getEnv("KUBEACTIONS_RUNNER_IMAGE_VARIANT", "-runner")

	dindImageName    = getEnv("KUBEACTIONS_DIND_IMAGE_NAME", "inloco/kube-actions")
	dindImageVersion = getEnv("KUBEACTIONS_DIND_IMAGE_VERSION", constants.Ver())
	dindImageVariant = getEnv("KUBEACTIONS_DIND_IMAGE_VARIANT", "-dind")

	runnerContainerName = "runner"
	runnerResourcesKey  = "runner"
	dindContainerName   = "dind"
	dindResourcesKey    = "docker"
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

func ToPodDisruptionBudget(actionsRunner *inlocov1alpha1.ActionsRunner, scheme *runtime.Scheme) (*policyv1.PodDisruptionBudget, error) {
	if actionsRunner == nil {
		return nil, errors.New("actionsRunner == nil")
	}

	if scheme == nil {
		return nil, errors.New("scheme == nil")
	}

	podDisruptionBudget := policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyv1.SchemeGroupVersion.String(),
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionsRunner.GetName(),
			Namespace: actionsRunner.GetNamespace(),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{
				IntVal: 0,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kube-actions.inloco.com.br/actions-runner": actionsRunner.GetName(),
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(actionsRunner, &podDisruptionBudget, scheme); err != nil {
		return nil, err
	}

	return &podDisruptionBudget, nil
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
			Labels: map[string]string{
				"kube-actions.inloco.com.br/actions-runner": actionsRunner.GetName(),
			},
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

	if !controllers.HasActionsRunnerRequestedStorage(actionsRunner) {
		return nil, nil
	}

	resources, ok := actionsRunner.Spec.Resources[runnerResourcesKey]
	if ok {
		resources = filteredResourceRequirements(
			resources,
			corev1.ResourceStorage,
		)
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
			Resources: resources,
		},
	}

	if err := ctrl.SetControllerReference(actionsRunnerJob, &persistentVolumeClaim, scheme); err != nil {
		return nil, err
	}

	return &persistentVolumeClaim, nil
}

func ToPod(actionsRunner *inlocov1alpha1.ActionsRunner, actionsRunnerJob *inlocov1alpha1.ActionsRunnerJob, scheme *runtime.Scheme) (*corev1.Pod, error) {
	if actionsRunner == nil {
		return nil, errors.New("actionsRunner == nil")
	}

	if actionsRunnerJob == nil {
		return nil, errors.New("actionsRunnerJob == nil")
	}

	if scheme == nil {
		return nil, errors.New("scheme == nil")
	}

	resources, ok := actionsRunner.Spec.Resources[runnerResourcesKey]
	if ok {
		resources = filteredResourceRequirements(
			resources,
			corev1.ResourceCPU,
			corev1.ResourceMemory,
			corev1.ResourceEphemeralStorage,
		)
	}

	imageVersion := runnerImageVersion
	if actionsRunner.Spec.Version != "" {
		imageVersion = actionsRunner.Spec.Version
	}

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        actionsRunnerJob.GetName(),
			Namespace:   actionsRunner.GetNamespace(),
			Annotations: actionsRunner.Spec.Annotations,
			Labels: map[string]string{
				"kube-actions.inloco.com.br/actions-runner": actionsRunner.GetName(),
			},
		},
		Spec: corev1.PodSpec{
			ActiveDeadlineSeconds: pointer.Int64(3000),
			Volumes:               withVolumes(actionsRunner),
			Containers: []corev1.Container{
				{
					Name:  runnerContainerName,
					Image: fmt.Sprintf("%s:%s%s", runnerImageName, imageVersion, runnerImageVariant),
					EnvFrom: filteredEnvFromSources(actionsRunner.Spec.EnvFrom, func(envFromSource corev1.EnvFromSource) bool {
						return envFromSource.SecretRef == nil
					}),
					Env: append(
						filteredEnvVars(actionsRunner.Spec.Env, func(envVar corev1.EnvVar) bool {
							return envVar.ValueFrom == nil || envVar.ValueFrom.SecretKeyRef == nil
						}),
						corev1.EnvVar{
							Name:  "KUBEACTIONS_ACTIONSRUNNER_NAME",
							Value: actionsRunner.GetName(),
						},
						corev1.EnvVar{
							Name:  "KUBEACTIONS_ACTIONSRUNNER_REPOSITORY_OWNER",
							Value: actionsRunner.Spec.Repository.Owner,
						},
						corev1.EnvVar{
							Name:  "KUBEACTIONS_ACTIONSRUNNER_REPOSITORY_NAME",
							Value: actionsRunner.Spec.Repository.Name,
						},
						corev1.EnvVar{
							Name:  "KUBEACTIONS_ACTIONSRUNNERJOB_NAME",
							Value: actionsRunnerJob.GetName(),
						},
					),
					Resources:    resources,
					VolumeMounts: withVolumeMounts(actionsRunner),
				},
			},
			RestartPolicy:                corev1.RestartPolicyNever,
			AutomountServiceAccountToken: pointer.Bool(false),
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:    pointer.Int64(1000),
				RunAsGroup:   pointer.Int64(1000),
				RunAsNonRoot: pointer.Bool(true),
				FSGroup:      pointer.Int64(1000),
				Sysctls: []corev1.Sysctl{
					{Name: "net.ipv4.ping_group_range", Value: "0 2147483647"},
				},
			},
			Affinity: withRuntimeAffinity(actionsRunner.Spec.Affinity),
			Tolerations: []corev1.Toleration{
				{
					Key:      "node-role.incognia.com/ci",
					Operator: corev1.TolerationOpEqual,
					Value:    "true",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			NodeSelector: map[string]string{
				"node-role.incognia.com/ci": "true",
			},
		},
	}

	capabilities := make(map[inlocov1alpha1.ActionsRunnerCapability]struct{})
	for _, capability := range actionsRunner.Spec.Capabilities {
		capabilities[capability] = struct{}{}
	}
	if _, ok := capabilities[inlocov1alpha1.ActionsRunnerCapabilitySecret]; ok {
		addSecretCapability(&pod, actionsRunner)
	}
	if _, ok := capabilities[inlocov1alpha1.ActionsRunnerCapabilityDocker]; ok {
		if err := addDockerCapability(&pod, actionsRunner); err != nil {
			return nil, err
		}
	}

	if err := ctrl.SetControllerReference(actionsRunnerJob, &pod, scheme); err != nil {
		return nil, err
	}

	return &pod, nil
}

func withVolumes(actionsRunner *inlocov1alpha1.ActionsRunner) []corev1.Volume {
	volumeByName := make(map[string]corev1.Volume, len(actionsRunner.Spec.Volumes))

	for _, volume := range actionsRunner.Spec.Volumes {
		volumeByName[volume.Name] = volume
	}

	volumeByName["config-map"] = corev1.Volume{
		Name: "config-map",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: actionsRunner.GetName(),
				},
			},
		},
	}

	volumeByName["secret"] = corev1.Volume{
		Name: "secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: actionsRunner.GetName(),
			},
		},
	}

	if controllers.HasActionsRunnerRequestedStorage(actionsRunner) {
		volumeByName["persistent-volume-claim"] = corev1.Volume{
			Name: "persistent-volume-claim",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: actionsRunner.GetName(),
				},
			},
		}
	}

	volumes := make([]corev1.Volume, 0, len(volumeByName))
	for _, volume := range volumeByName {
		volumes = append(volumes, volume)
	}

	return volumes
}

func withVolumeMounts(actionsRunner *inlocov1alpha1.ActionsRunner) []corev1.VolumeMount {
	volumeMountByPath := make(map[string]corev1.VolumeMount, len(actionsRunner.Spec.Volumes))

	for _, volumeMount := range actionsRunner.Spec.VolumeMounts {
		volumeMountByPath[volumeMount.MountPath] = volumeMount
	}

	volumeMountByPath["/opt/actions-runner/.runner"] = corev1.VolumeMount{
		MountPath: "/opt/actions-runner/.runner",
		Name:      "config-map",
		SubPath:   ".runner",
	}
	volumeMountByPath["/opt/actions-runner/.credentials"] = corev1.VolumeMount{
		MountPath: "/opt/actions-runner/.credentials",
		Name:      "config-map",
		SubPath:   ".credentials",
	}
	volumeMountByPath["/opt/actions-runner/.credentials_rsaparams"] = corev1.VolumeMount{
		MountPath: "/opt/actions-runner/.credentials_rsaparams",
		Name:      "secret",
		SubPath:   ".credentials_rsaparams",
	}

	if controllers.HasActionsRunnerRequestedStorage(actionsRunner) {
		volumeMountByPath["/home/linuxbrew"] = corev1.VolumeMount{
			MountPath: "/home/linuxbrew",
			Name:      "persistent-volume-claim",
			SubPath:   ":home:linuxbrew",
		}
		volumeMountByPath["/home/runner"] = corev1.VolumeMount{
			MountPath: "/home/runner",
			Name:      "persistent-volume-claim",
			SubPath:   ":home:user",
		}
		volumeMountByPath["/home/user"] = corev1.VolumeMount{
			MountPath: "/home/user",
			Name:      "persistent-volume-claim",
			SubPath:   ":home:user",
		}
		volumeMountByPath["/opt/actions-runner/_work"] = corev1.VolumeMount{
			MountPath: "/opt/actions-runner/_work",
			Name:      "persistent-volume-claim",
			SubPath:   ":opt:actions-runner:_work",
		}
		volumeMountByPath["/opt/hostedtoolcache"] = corev1.VolumeMount{
			MountPath: "/opt/hostedtoolcache",
			Name:      "persistent-volume-claim",
			SubPath:   ":opt:hostedtoolcache",
		}
		volumeMountByPath["/root"] = corev1.VolumeMount{
			MountPath: "/root",
			Name:      "persistent-volume-claim",
			SubPath:   ":root",
		}
	}

	volumeMounts := make([]corev1.VolumeMount, 0, len(volumeMountByPath))
	for _, volumeMount := range volumeMountByPath {
		volumeMounts = append(volumeMounts, volumeMount)
	}

	return volumeMounts
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

func addSecretCapability(pod *corev1.Pod, actionsRunner *inlocov1alpha1.ActionsRunner) {
	runnerContainer := &pod.Spec.Containers[0]

	envFrom := filteredEnvFromSources(actionsRunner.Spec.EnvFrom, func(envFromSource corev1.EnvFromSource) bool {
		return envFromSource.SecretRef != nil
	})
	runnerContainer.EnvFrom = append(runnerContainer.EnvFrom, envFrom...)

	env := filteredEnvVars(actionsRunner.Spec.Env, func(envVar corev1.EnvVar) bool {
		return envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil
	})
	runnerContainer.Env = append(runnerContainer.Env, env...)

	pod.Spec.ServiceAccountName = actionsRunner.Spec.ServiceAccountName
	pod.Spec.AutomountServiceAccountToken = pointer.Bool(true)
}

func addDockerCapability(pod *corev1.Pod, actionsRunner *inlocov1alpha1.ActionsRunner) error {
	runnerContainer := &pod.Spec.Containers[0]
	runnerContainer.Env = append(runnerContainer.Env, corev1.EnvVar{
		Name:  "DOCKER_HOST",
		Value: "tcp://localhost:2375",
	})

	resources, ok := actionsRunner.Spec.Resources[dindResourcesKey]
	if ok {
		resources = filteredResourceRequirements(
			resources,
			corev1.ResourceCPU,
			corev1.ResourceMemory,
			corev1.ResourceEphemeralStorage,
		)
	}

	var volumeMounts []corev1.VolumeMount
	if controllers.HasActionsRunnerRequestedStorage(actionsRunner) {
		volumeMounts = append(
			volumeMounts,
			corev1.VolumeMount{
				MountPath: "/home/rootless",
				Name:      "persistent-volume-claim",
				SubPath:   ":home:rootless",
			},
			corev1.VolumeMount{
				MountPath: "/home/rootless/.local/share/docker",
				Name:      "persistent-volume-claim",
				SubPath:   ":home:rootless:.local:share:docker",
			},
			corev1.VolumeMount{
				MountPath: "/opt/actions-runner/_work",
				Name:      "persistent-volume-claim",
				SubPath:   ":opt:actions-runner:_work",
			},
		)
	}

	imageVersion := dindImageVersion
	if actionsRunner.Spec.Version != "" {
		imageVersion = actionsRunner.Spec.Version
	}

	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:  dindContainerName,
		Image: fmt.Sprintf("%s:%s%s", dindImageName, imageVersion, dindImageVariant),
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "DOCKER_TLS_CERTDIR",
				Value: "",
			},
		},
		VolumeMounts: volumeMounts,
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"nc",
						"-z",
						"127.0.0.1",
						"2375",
					},
				},
			},
			InitialDelaySeconds: 3,
		},
		Resources: resources,
		SecurityContext: &corev1.SecurityContext{
			Privileged: pointer.Bool(true),
		},
	})

	return nil
}
