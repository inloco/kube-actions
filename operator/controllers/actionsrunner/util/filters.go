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

func filteredEnvFromSources(in []corev1.EnvFromSource, predicate func(corev1.EnvFromSource) bool) []corev1.EnvFromSource {
	var out []corev1.EnvFromSource
	for _, envFromSource := range in {
		if !predicate(envFromSource) {
			continue
		}

		out = append(out, envFromSource)
	}

	return out
}

func filteredEnvVars(in []corev1.EnvVar, predicate func(corev1.EnvVar) bool) []corev1.EnvVar {
	var out []corev1.EnvVar
	for _, envVar := range in {
		if !predicate(envVar) {
			continue
		}

		out = append(out, envVar)
	}

	return out
}

func filteredResourceRequirements(in corev1.ResourceRequirements, resourceNames ...corev1.ResourceName) corev1.ResourceRequirements {
	out := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}

	for _, resourceName := range resourceNames {
		if val, ok := in.Limits[resourceName]; ok {
			out.Limits[resourceName] = val
		}
		if val, ok := in.Requests[resourceName]; ok {
			out.Requests[resourceName] = val
		}
	}

	return out
}
