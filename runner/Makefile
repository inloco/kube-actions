IMAGE_REPOSITORY ?= inloco/kube-actions
IMAGE_VERSION ?= 0.0.1

RUNNER_VERSION ?= 2.272.0
UBUNTU_VERSION ?= focal

# Build, tag, and push the docker image
docker: docker-build docker-tag docker-push

# Build the docker image
docker-build:
	docker build --build-arg RUNNER_VERSION=$(RUNNER_VERSION) --build-arg UBUNTU_VERSION=$(UBUNTU_VERSION) --tag $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION) .

# Tag the docker image
docker-tag:
	docker tag $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION) $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(RUNNER_VERSION)
	docker tag $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION) $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(UBUNTU_VERSION)
	docker tag $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION) $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner

# Push the docker image
docker-push:
	docker push $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(RUNNER_VERSION)-$(UBUNTU_VERSION)
	docker push $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(RUNNER_VERSION)
	docker push $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner-$(UBUNTU_VERSION)
	docker push $(IMAGE_REPOSITORY):$(IMAGE_VERSION)-runner
