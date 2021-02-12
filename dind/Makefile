IMAGE_NAME ?= inloco/kube-actions
IMAGE_VERSION ?= $(shell git describe --dirty --broken --always)
IMAGE_VARIANT ?= -dind

DOCKER_VERSION ?= 20.10

# Build, tag, and push the docker image
docker: docker-build docker-tag docker-push

# Build the docker image
docker-build:
	docker build --build-arg DOCKER_VERSION=$(DOCKER_VERSION) --tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(DOCKER_VERSION) .

# Tag the docker image
docker-tag:
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(DOCKER_VERSION) $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)

# Push the docker image
docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(DOCKER_VERSION)
	docker push $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)

docker-save:
	docker save $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(DOCKER_VERSION) > ./image.tar

docker-load:
	docker load < ./image.tar

continuous-build: docker-build docker-save

continuous-push: docker-load docker-tag docker-push
