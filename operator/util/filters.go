package util

import (
	corev1 "k8s.io/api/core/v1"
)

func FilterEnvFrom(oldEnvFrom []corev1.EnvFromSource, predicate func(corev1.EnvFromSource) bool) []corev1.EnvFromSource {
	var newEnvFrom []corev1.EnvFromSource
	for _, envFromSource := range oldEnvFrom {
		if !predicate(envFromSource) {
			continue
		}

		newEnvFrom = append(newEnvFrom, envFromSource)
	}

	return newEnvFrom
}

func FilterEnv(oldEnv []corev1.EnvVar, predicate func(corev1.EnvVar) bool) []corev1.EnvVar {
	var newEnv []corev1.EnvVar
	for _, envVar := range oldEnv {
		if !predicate(envVar) {
			continue
		}

		newEnv = append(newEnv, envVar)
	}

	return newEnv
}
