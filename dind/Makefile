IMAGE_REPOSITORY ?= inloco/kube-actions
IMAGE_VERSION ?= 0.0.1

DOCKER_VERSION ?= 19.03

# Build, tag, and push the docker image
docker: docker-build docker-tag docker-push

# Build the docker image
docker-build:
	docker build --build-arg DOCKER_VERSION=$(DOCKER_VERSION) --tag $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-dind-$(DOCKER_VERSION) .

# Tag the docker image
docker-tag:
	docker tag $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-dind-$(DOCKER_VERSION) $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-dind

# Push the docker image
docker-push:
	docker push $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-dind-$(DOCKER_VERSION)
	docker push $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-dind
