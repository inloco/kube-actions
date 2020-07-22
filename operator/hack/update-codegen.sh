#!/bin/sh
set -ex

TEMP=$(mktemp -d)
trap "rm -fR ${TEMP}" EXIT INT QUIT TERM

git clone --depth 1 https://github.com/kubernetes/code-generator ${TEMP}/k8s.io/code-generator

ROOT=$(dirname "${BASH_SOURCE}")/..

${TEMP}/k8s.io/code-generator/generate-groups.sh \
	all \
	github.com/inloco/kube-actions/operator/pkg/generated \
	github.com/inloco/kube-actions/operator/pkg/apis \
	inloco:v1alpha1 \
	--go-header-file "${ROOT}/hack/custom-boilerplate.go.txt" \
	--output-base ${TEMP}

cp -fR ${TEMP}/github.com/inloco/kube-actions/operator/ "${ROOT}"
