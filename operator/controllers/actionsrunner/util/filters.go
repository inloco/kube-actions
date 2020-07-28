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
