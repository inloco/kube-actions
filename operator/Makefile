IMAGE_NAME ?= inloco/kube-actions
IMAGE_VERSION ?= $(shell git describe --always)
IMAGE_VARIANT ?= -operator

AGENT_VERSION ?= 2.283.1

# Default rule
all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile ./cover.out

# Build manager binary
manager: generate fmt vet
	go build -o ./bin/manager -v ./main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build ./config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build ./config/crd | kubectl delete -f -

deploy/production: manifests
	kustomize build --enable-alpha-plugins k8s/production | yq -y '(select(.kind=="Deployment" and .metadata.name=="controller-manager") | .spec.template.spec.containers[] | select(.name=="manager") | .image) |= "$(IMAGE_NAME):$(IMAGE_VERSION)"' | kubectl apply -f -

deploy/development: manifests
	kustomize build --enable-alpha-plugins k8s/development | yq -y '(select(.kind=="Deployment" and .metadata.name=="controller-manager") | .spec.template.spec.containers[] | select(.name=="manager") | .image) |= "$(IMAGE_NAME):$(IMAGE_VERSION)"' | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	controller-gen crd rbac:roleName=manager-role webhook paths='./...' output:crd:artifacts:config='./config/crd/bases'

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet -v ./...

# Generate code
generate:
	controller-gen object:headerFile='./hack/boilerplate.go.txt' paths='./...'

# Build, tag, and push the docker image
docker: docker-build docker-tag docker-push

# Build the docker image
docker-build:
	docker build --build-arg VER=$(IMAGE_VERSION) --build-arg API=$(AGENT_VERSION) --tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT) .

# Tag the docker image
docker-tag:
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT) $(IMAGE_NAME):$(IMAGE_VERSION)

# Push the docker image
docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT)
	docker push $(IMAGE_NAME):$(IMAGE_VERSION)

docker-save:
	docker save $(IMAGE_NAME):$(IMAGE_VERSION)$(IMAGE_VARIANT) > ./image.tar

docker-load:
	docker load < ./image.tar

continuous-build: docker-build docker-save

continuous-push: docker-load docker-tag docker-push
