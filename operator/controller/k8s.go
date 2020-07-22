package controller

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/inloco/kube-actions/operator/models"
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/pkg/apis/inloco/v1alpha1"
	inlocov1alpha1client "github.com/inloco/kube-actions/operator/pkg/generated/clientset/versioned/typed/inloco/v1alpha1"
	"github.com/inloco/kube-actions/operator/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
)

type facadeKubernetes struct {
	context context.Context

	k8sConfig       *restclient.Config
	k8sBatchClient  *batchv1client.BatchV1Client
	k8sCoreClient   *corev1client.CoreV1Client
	k8sInLocoClient *inlocov1alpha1client.InlocoV1alpha1Client

	actionsRunner         *inlocov1alpha1.ActionsRunner
	configMap             *corev1.ConfigMap
	secret                *corev1.Secret
	job                   *batchv1.Job
	persistentVolumeClaim *corev1.PersistentVolumeClaim
}

func (k8s *facadeKubernetes) Init() error {
	if err := k8s.InitK8sBatchClient(); err != nil {
		return err
	}

	if err := k8s.InitK8sCoreClient(); err != nil {
		return err
	}

	if err := k8s.InitK8sInLocoClient(); err != nil {
		return err
	}

	if err := k8s.InitConfigMap(); err != nil {
		return err
	}

	if err := k8s.InitSecret(); err != nil {
		return err
	}

	return nil
}

func (k8s *facadeKubernetes) InitK8sBatchClient() error {
	if k8s.k8sConfig == nil {
		return errors.New(".k8sConfig == nil")
	}

	client, err := batchv1client.NewForConfig(k8s.k8sConfig)
	if err != nil {
		return err
	}

	k8s.k8sBatchClient = client
	return nil
}

func (k8s *facadeKubernetes) InitK8sCoreClient() error {
	if k8s.k8sConfig == nil {
		return errors.New(".k8sConfig == nil")
	}

	client, err := corev1client.NewForConfig(k8s.k8sConfig)
	if err != nil {
		return err
	}

	k8s.k8sCoreClient = client
	return nil
}

func (k8s *facadeKubernetes) InitK8sInLocoClient() error {
	if k8s.k8sConfig == nil {
		return errors.New(".k8sConfig == nil")
	}

	client, err := inlocov1alpha1client.NewForConfig(k8s.k8sConfig)
	if err != nil {
		return err
	}

	k8s.k8sInLocoClient = client
	return nil
}

func (k8s *facadeKubernetes) InitConfigMap() error {
	if k8s.context == nil {
		return errors.New(".context == nil")
	}

	if k8s.actionsRunner == nil {
		return errors.New(".actionsRunner == nil")
	}

	if k8s.k8sCoreClient == nil {
		return errors.New(".k8sCoreClient == nil")
	}

	if configMapName := k8s.actionsRunner.Status.ConfigMapName; configMapName != "" {
		configMap, err := k8s.k8sCoreClient.ConfigMaps(k8s.actionsRunner.GetNamespace()).Get(k8s.context, configMapName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		k8s.configMap = configMap
		return nil
	}

	k8s.configMap = &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: k8s.actionsRunner.GetName() + "-",
			Namespace:    k8s.actionsRunner.GetNamespace(),
		},
		BinaryData: map[string][]byte{
			".runner":      []byte("{}"),
			".credentials": []byte("{}"),
		},
	}

	return nil
}

func (k8s *facadeKubernetes) InitSecret() error {
	if k8s.context == nil {
		return errors.New(".context == nil")
	}

	if k8s.actionsRunner == nil {
		return errors.New(".actionsRunner == nil")
	}

	if k8s.k8sCoreClient == nil {
		return errors.New(".k8sCoreClient == nil")
	}

	if secretName := k8s.actionsRunner.Status.SecretName; secretName != "" {
		secret, err := k8s.k8sCoreClient.Secrets(k8s.actionsRunner.GetNamespace()).Get(k8s.context, secretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		k8s.secret = secret
		return nil
	}

	k8s.secret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: k8s.actionsRunner.GetName() + "-",
			Namespace:    k8s.actionsRunner.GetNamespace(),
		},
		Data: map[string][]byte{
			".credentials_rsaparams": []byte("{}"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	return nil
}

func (k8s *facadeKubernetes) Apply(runner *models.Runner) error {
	if err := k8s.applyConfigMap(runner); err != nil {
		return err
	}

	if err := k8s.applySecret(runner); err != nil {
		return err
	}

	return nil
}

func (k8s *facadeKubernetes) applyConfigMap(runner *models.Runner) error {
	dotRunner, err := json.Marshal(runner.RunnerSettings)
	if err != nil {
		return err
	}
	k8s.configMap.BinaryData[".runner"] = dotRunner

	dotCredentials, err := json.Marshal(runner.Credentials)
	if err != nil {
		return err
	}
	k8s.configMap.BinaryData[".credentials"] = dotCredentials

	configMaps := k8s.k8sCoreClient.ConfigMaps(k8s.configMap.GetNamespace())
	if k8s.configMap.GetName() == "" {
		configMap, err := configMaps.Create(k8s.context, k8s.configMap, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		k8s.configMap = configMap
	} else {
		configMap, err := configMaps.Update(k8s.context, k8s.configMap, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		k8s.configMap = configMap
	}

	return nil
}

func (k8s *facadeKubernetes) applySecret(runner *models.Runner) error {
	dotCredentialsRSAParams, err := json.Marshal(runner.RSAParameters)
	if err != nil {
		return err
	}
	k8s.secret.Data[".credentials_rsaparams"] = dotCredentialsRSAParams

	secrets := k8s.k8sCoreClient.Secrets(k8s.configMap.GetNamespace())
	if k8s.secret.GetName() == "" {
		secret, err := secrets.Create(k8s.context, k8s.secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		k8s.secret = secret
	} else {
		secret, err := secrets.Update(k8s.context, k8s.secret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		k8s.secret = secret
	}

	return nil
}

func (k8s *facadeKubernetes) CreatePersistentVolumeClaim() error {
	template := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: k8s.actionsRunner.GetName() + "-",
			Namespace:    k8s.actionsRunner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion:         inlocov1alpha1.SchemeGroupVersion.String(),
					Kind:               "ActionsRunner",
					Name:               k8s.actionsRunner.Name,
					UID:                k8s.actionsRunner.UID,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceStorage: *k8s.actionsRunner.Spec.Resources.Limits.Storage(),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *k8s.actionsRunner.Spec.Resources.Requests.Storage(),
				},
			},
		},
	}

	persistentVolumeClaims := k8s.k8sCoreClient.PersistentVolumeClaims(k8s.actionsRunner.GetNamespace())
	persistentVolumeClaim, err := persistentVolumeClaims.Create(k8s.context, template, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	k8s.persistentVolumeClaim = persistentVolumeClaim
	return nil
}

func (k8s *facadeKubernetes) DeletePersistentVolumeClaim() error {
	if k8s.persistentVolumeClaim == nil {
		return errors.New(".persistentVolumeClaim == nil")
	}

	persistentVolumeClaims := k8s.k8sCoreClient.PersistentVolumeClaims(k8s.persistentVolumeClaim.GetNamespace())
	if err := persistentVolumeClaims.Delete(k8s.context, k8s.persistentVolumeClaim.GetName(), metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	k8s.persistentVolumeClaim = nil
	return nil
}

func (k8s *facadeKubernetes) addSecretCapability(job *batchv1.Job) {
	podSpec := &job.Spec.Template.Spec
	runnerContainer := &podSpec.Containers[0]

	envFrom := util.FilterEnvFrom(k8s.actionsRunner.Spec.EnvFrom, func(envFromSource corev1.EnvFromSource) bool {
		return envFromSource.SecretRef != nil
	})
	runnerContainer.EnvFrom = append(runnerContainer.EnvFrom, envFrom...)

	env := util.FilterEnv(k8s.actionsRunner.Spec.Env, func(envVar corev1.EnvVar) bool {
		return envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil
	})
	runnerContainer.Env = append(runnerContainer.Env, env...)

	podSpec.ServiceAccountName = k8s.actionsRunner.Spec.ServiceAccountName

	*podSpec.AutomountServiceAccountToken = true
}

func (k8s *facadeKubernetes) addDockerCapability(job *batchv1.Job) {
	podSpec := &job.Spec.Template.Spec
	runnerContainer := &podSpec.Containers[0]

	runnerContainer.Env = append(runnerContainer.Env, corev1.EnvVar{
		Name:  "DOCKER_HOST",
		Value: "tcp://localhost:2375",
	})

	podSpec.Containers = append(podSpec.Containers, corev1.Container{
		Name:  "dind",
		Image: "inloco/kube-actions:dind",
		Args: []string{
			"--add-runtime", "crun=/usr/local/bin/crun",
			"--default-runtime", "crun",
			"--experimental",
		},
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

	podSpec.InitContainers = append(podSpec.InitContainers, corev1.Container{
		Name:  "init",
		Image: "busybox",
		Command: []string{
			"sh",
			"-c", "mkdir -p /mnt/dind && chown -R 1000:1000 /mnt/dind",
		},
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      "persistent-volume-claim",
				MountPath: "/mnt",
			},
		},
	})
}

func (k8s *facadeKubernetes) CreateJob() error {
	template := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: k8s.actionsRunner.GetName() + "-",
			Namespace:    k8s.actionsRunner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion:         inlocov1alpha1.SchemeGroupVersion.String(),
					Kind:               "ActionsRunner",
					Name:               k8s.actionsRunner.Name,
					UID:                k8s.actionsRunner.UID,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
			},
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
										Name: k8s.configMap.GetName(),
									},
								},
							},
						},
						corev1.Volume{
							Name: "secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: k8s.secret.GetName(),
								},
							},
						},
						corev1.Volume{
							Name: "persistent-volume-claim",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: k8s.persistentVolumeClaim.GetName(),
								},
							},
						},
					},
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "runner",
							Image: "inloco/kube-actions:runner",
							EnvFrom: util.FilterEnvFrom(k8s.actionsRunner.Spec.EnvFrom, func(envFromSource corev1.EnvFromSource) bool {
								return envFromSource.SecretRef == nil
							}),
							Env: util.FilterEnv(k8s.actionsRunner.Spec.Env, func(envVar corev1.EnvVar) bool {
								return envVar.ValueFrom == nil || envVar.ValueFrom.SecretKeyRef == nil
							}),
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:              *k8s.actionsRunner.Spec.Resources.Limits.Cpu(),
									corev1.ResourceMemory:           *k8s.actionsRunner.Spec.Resources.Limits.Memory(),
									corev1.ResourceEphemeralStorage: *k8s.actionsRunner.Spec.Resources.Limits.StorageEphemeral(),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:              *k8s.actionsRunner.Spec.Resources.Requests.Cpu(),
									corev1.ResourceMemory:           *k8s.actionsRunner.Spec.Resources.Requests.Memory(),
									corev1.ResourceEphemeralStorage: *k8s.actionsRunner.Spec.Resources.Requests.StorageEphemeral(),
								},
							},
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
							},
							ImagePullPolicy: corev1.PullAlways,
						},
					},
					RestartPolicy:                corev1.RestartPolicyNever,
					AutomountServiceAccountToken: new(bool),
					Affinity:                     k8s.actionsRunner.Spec.Affinity,
					Tolerations:                  k8s.actionsRunner.Spec.Tolerations,
				},
			},
			TTLSecondsAfterFinished: pointer.Int32Ptr(0),
		},
	}

	capabilities := make(map[inlocov1alpha1.ActionsRunnerCapability]struct{})
	for _, capability := range k8s.actionsRunner.Spec.Capabilities {
		capabilities[capability] = struct{}{}
	}
	if _, ok := capabilities[inlocov1alpha1.ActionsRunnerCapabilitySecret]; ok {
		k8s.addSecretCapability(template)
	}
	if _, ok := capabilities[inlocov1alpha1.ActionsRunnerCapabilityDocker]; ok {
		k8s.addDockerCapability(template)
	}

	jobs := k8s.k8sBatchClient.Jobs(k8s.actionsRunner.GetNamespace())
	job, err := jobs.Create(k8s.context, template, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	k8s.job = job
	return nil
}

func (k8s *facadeKubernetes) DeleteJob() error {
	if k8s.job == nil {
		return errors.New(".job == nil")
	}

	jobs := k8s.k8sBatchClient.Jobs(k8s.job.GetNamespace())
	if err := jobs.Delete(k8s.context, k8s.job.GetName(), metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	k8s.job = nil
	return nil
}

func (k8s *facadeKubernetes) WaitJob() error {
	if k8s.job == nil {
		return errors.New(".job == nil")
	}

	jobs := k8s.k8sBatchClient.Jobs(k8s.job.GetNamespace())
	return wait.PollImmediate(time.Second, time.Hour, func() (done bool, err error) {
		job, err := jobs.Get(k8s.context, k8s.job.GetName(), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		active := job.Status.CompletionTime == nil
		return !active, nil
	})
}
