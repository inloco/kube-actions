RST := \033[m
BLD := \033[1m
RED := \033[31m
GRN := \033[32m
YLW := \033[33m
BLU := \033[34m
EOL := \n

IMAGE_REGISTRY ?= docker.io
IMAGE_REPOSITORY ?= inloco/kube-actions
IMAGE_VERSION ?= $(shell git describe --dirty --broken --always)
IMAGE_VARIANT ?= -runner

RUNNER_VERSION ?= 2.313.0
UBUNTU_VERSION ?= jammy
UBUNTU_VERSION_NUMBER ?= 22

docker: docker-build docker-tag docker-push
	@printf '${BLD}${RED}make: *** [$@]${RST}${EOL}'

docker-build:
	@printf '${BLD}${RED}make: *** [$@]${RST}${EOL}'
	@printf '${BLD}${YLW}$$${RST} '
	docker build --build-arg RUNNER_VERSION=${RUNNER_VERSION} --build-arg UBUNTU_VERSION=${UBUNTU_VERSION} --build-arg UBUNTU_VERSION_NUMBER=${UBUNTU_VERSION_NUMBER} --tag kube-actions:${IMAGE_VERSION}${IMAGE_VARIANT} .

docker-tag:
	@printf '${BLD}${RED}make: *** [$@]${RST}${EOL}'
	@printf '${BLD}${YLW}$$${RST} '
	docker tag kube-actions:${IMAGE_VERSION}${IMAGE_VARIANT} ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${IMAGE_VERSION}${IMAGE_VARIANT}
	@printf '${BLD}${YLW}$$${RST} '
	docker tag kube-actions:${IMAGE_VERSION}${IMAGE_VARIANT} ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:latest${IMAGE_VARIANT}

docker-push:
	@printf '${BLD}${RED}make: *** [$@]${RST}${EOL}'
	@printf '${BLD}${YLW}$$${RST} '
	docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${IMAGE_VERSION}${IMAGE_VARIANT}
	@printf '${BLD}${YLW}$$${RST} '
	docker push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:latest${IMAGE_VARIANT}
