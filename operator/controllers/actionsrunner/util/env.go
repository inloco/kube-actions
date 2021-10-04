package util

import (
	"os"

	corev1 "k8s.io/api/core/v1"
)

func EnvVar(variable string, defaultValue string) string {
	if value, ok := os.LookupEnv(variable); ok {
		return value
	}

	return defaultValue
}

func ConcatEnvVars(envVars []corev1.EnvVar, moreEnvVars []corev1.EnvVar) []corev1.EnvVar {
	allEnvVars := []corev1.EnvVar{}
	for _, envVar := range envVars {
		allEnvVars = append(allEnvVars, envVar)
	}
	for _, envVar := range moreEnvVars {
		allEnvVars = append(allEnvVars, envVar)
	}
	return allEnvVars
}
