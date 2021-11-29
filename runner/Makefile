IMAGE_REPO ?= 779099367007.dkr.ecr.us-east-1.amazonaws.com
IMAGE_NAME ?= incognia/kube-actions
IMAGE_VERSION ?= $(shell git describe --dirty --broken --always)
IMAGE_VARIANT ?= -runner

RUNNER_VERSION ?= 2.285.0
UBUNTU_VERSION ?= focal

# Build, tag, and push the docker image
docker: docker-build docker-tag docker-push

# Build the docker image
docker-build:
	docker build --build-arg RUNNER_VERSION=$(RUNNER_VERSION) --build-arg UBUNTU_VERSION=$(UBUNTU_VERSION) --tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)-$(UBUNTU_VERSION) .

# Tag the docker image
docker-tag:
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)-$(UBUNTU_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)-$(UBUNTU_VERSION)
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)-$(UBUNTU_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)-$(UBUNTU_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(UBUNTU_VERSION)
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)-$(UBUNTU_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)

# Push the docker image
docker-push:
	docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)-$(UBUNTU_VERSION)
	docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)
	docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(UBUNTU_VERSION)
	docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)

docker-save:
	docker save $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)-$(RUNNER_VERSION)-$(UBUNTU_VERSION) > ./image.tar

docker-load:
	docker load < ./image.tar

continuous-build: docker-build docker-save

continuous-push: docker-load docker-tag docker-push
