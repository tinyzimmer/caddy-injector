
# Image URL to use all building/pushing image targets
TAG ?= latest
IMG ?= ghcr.io/tinyzimmer/caddy-injector:$(TAG)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
# Applies to various helm operations
HELM_ARGS ?=

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

GOLANGCI_LINT    ?= $(CURDIR)/bin/golangci-lint
GOLANGCI_VERSION ?= v1.42.1
$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(PWD)/bin" $(GOLANGCI_VERSION)

lint: $(GOLANGCI_LINT) ## Run linting.
	"$(GOLANGCI_LINT)" run -v

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

PLATFORM ?= amd64
docker-build: ## Build docker image with the manager. Set PLATFORM to your preferred arch. Default is amd64.
	$(call docker_build_platform,$(PLATFORM),$(IMG))

DIST_PLATFORMS ?= amd64 arm64
docker-build-multiarch: ## Build docker image for all platforms. Override values with space-separated DIST_PLATFORMS.
	$(foreach platform,$(DIST_PLATFORMS),$(call docker_build_platform,$(platform),$(IMG)-$(platform)) ;)

docker-push-multiarch: docker-build-multiarch ## Publish a multiarch build to a repository.
	docker manifest create $(IMG) $(foreach platform,$(DIST_PLATFORMS),$(IMG)-$(platform))
	docker manifest push $(IMG)

define docker_build_platform
	set -x ; docker build \
		--platform linux/$(1) \
		--tag $(2) .
endef

BUNDLE_NAMESPACE ?= caddy-system
BUNDLE_OUTPUT    ?= config/bundle.yaml
bundle: ## Generate a full deployment manifest from the helm chart.
	echo -en "# Namespace\n---\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: $(BUNDLE_NAMESPACE)\n" \
		> $(BUNDLE_OUTPUT)
	helm template \
		caddy-injector config/charts/caddy-injector \
		--create-namespace \
		--namespace $(BUNDLE_NAMESPACE) \
		--include-crds \
		$(HELM_ARGS) | \
	sed -e '/^metadata:/{:a; N; /  name:.*/!ba; a\  namespace: $(BUNDLE_NAMESPACE)' -e '}' \
		>> $(BUNDLE_OUTPUT)

##@ Local In-Cluster Development

CLUSTER_NAME ?= caddy-injector
KUBECTL      := kubectl --context k3d-$(CLUSTER_NAME)
HELM         := helm --kube-context k3d-$(CLUSTER_NAME)

full-cluster: cluster import install-deps install ## A combination of cluster, import, install-deps, and install

cluster: ## Create a local k3d cluster for testing
	k3d cluster create $(CLUSTER_NAME) \
		--k3s-arg --disable=traefik@server:* \
		--port 443:443@loadbalancer

import: docker-build ## Build and import the controller image into the k3d cluster
	k3d image import --cluster $(CLUSTER_NAME) $(IMG)

install-deps: ## Install cert-manager into the cluster
	$(KUBECTL) create ns cert-manager
	$(HELM) upgrade --install cert-manager jetstack/cert-manager \
		--namespace cert-manager \
		--set installCRDs=true

install: ## Install the controller and webhook config into the cluster
	$(HELM) upgrade --install caddy-injector config/charts/caddy-injector $(HELM_ARGS)

restart: ## Restart the controller (in case of importing a new image)
	$(KUBECTL) delete pod -l app.kubernetes.io/name=caddy-injector

remove-cluster: ## Remove the local k3d cluster
	k3d cluster delete $(CLUSTER_NAME)

##@ Documentation Generation

helm-docs: ## Generate helm chart documentation.
	docker run --rm \
		--volume "$(CURDIR)/config/charts:/helm-docs" \
		--user $(shell id -u) \
		jnorwood/helm-docs:latest