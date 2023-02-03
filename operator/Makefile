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
IMAGE_VARIANT ?= -operator

AGENT_VERSION ?= 2.301.1

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
	kustomize build --enable-alpha-plugins k8s/production | yq -y '(select(.kind=="Deployment" and .metadata.name=="controller-manager") | .spec.template.spec.containers[] | select(.name=="manager") | .image) |= "$(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)"' | kubectl apply -f -

deploy/development: manifests
	kustomize build --enable-alpha-plugins k8s/development | yq -y '(select(.kind=="Deployment" and .metadata.name=="controller-manager") | .spec.template.spec.containers[] | select(.name=="manager") | .image) |= "$(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)"' | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	CGO_ENABLED=0 controller-gen crd rbac:roleName=manager-role webhook paths='./...' output:crd:artifacts:config='./config/crd/bases'

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet -v ./...

# Generate code
generate:
	CGO_ENABLED=0 controller-gen object:headerFile='./hack/boilerplate.go.txt' paths='./...'

docker: docker-build docker-tag docker-push
	@printf '${BLD}${RED}make: *** [$@]${RST}${EOL}'

docker-build:
	@printf '${BLD}${RED}make: *** [$@]${RST}${EOL}'
	@printf '${BLD}${YLW}$$${RST} '
	docker build --build-arg VER=${IMAGE_VERSION} --build-arg API=${AGENT_VERSION} --tag kube-actions:${IMAGE_VERSION}${IMAGE_VARIANT} .

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
