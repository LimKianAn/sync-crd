
# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/metal-stack/syncrd:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: syncrd

# Run tests
test: preparation
	go test ./... -coverprofile cover.out

REPO_URL ?= github.com/metal-stack/firewall-controller
REPO_VERSION ?= latest
SUB_PATH ?= api/v1
CRD_KIND ?= ClusterwideNetworkPolicy

preparation: edit download fmt vet

# Build manager binary
syncrd: preparation
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o bin/syncrd main.go

edit:
	sed 's#repo-url => .*#repo-url => ${REPO_URL} ${REPO_VERSION}#' -i go.mod && \
	sed 's#repo-url/.*#repo-url/${SUB_PATH}"#' -i main.go && \
	sed 's#repo-url/.*#repo-url/${SUB_PATH}"#' -i controllers/instance_controller.go && \
	sed 's#type CRD = api.*#type CRD = api.${CRD_KIND}#' -i controllers/instance_controller.go && \
	go mod tidy

download:
	go mod download

# stall CRDs into a cluster
install:
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall:
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy:
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build the docker image
docker-build: edit test syncrd
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}
