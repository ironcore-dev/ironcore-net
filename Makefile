
# Image URL to use all building/pushing image targets
ONMETAL_API_NET_IMG ?= onmetal-api-net:latest
APINETLET_IMG ?= apinetlet:latest
KIND_CLUSTER_NAME ?= kind

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.26.1

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
BUILDARGS ?=

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
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

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role crd paths="./api/...;./onmetal-api-net/..." output:crd:artifacts:config=config/onmetal-api-net/crd/bases output:rbac:artifacts:config=config/onmetal-api-net/rbac

	# apinetlet
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role paths="./apinetlet/..." output:rbac:artifacts:config=config/apinetlet/rbac

	# poollet system roles
	cp config/apinetlet/apinet-rbac/role.yaml config/onmetal-api-net/rbac/apinetlet_role.yaml
	./hack/replace.sh config/onmetal-api-net/rbac/apinetlet_role.yaml 's/apinet-role/apinet.api.onmetal.de:system:apinetlets/g'
	./hack/replace.sh config/onmetal-api-net/rbac/apinetlet_role.yaml 's/Role/ClusterRole/g'
	./hack/replace.sh config/onmetal-api-net/rbac/apinetlet_role.yaml '/namespace: system/d'

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: add-license
add-license: addlicense ## Add license headers to all go files.
	find . -name '*.go' -exec $(ADDLICENSE) -c 'OnMetal authors' {} +

.PHONY: fmt
fmt: goimports ## Run goimports against code.
	$(GOIMPORTS) -w .

.PHONY: check-license
check-license: addlicense ## Check that every file has a license header present.
	find . -name '*.go' -exec $(ADDLICENSE) -check -c 'OnMetal authors' {} +

.PHONY: lint
lint: ## Run golangci-lint against code.
	golangci-lint run ./...

.PHONY: check
check: manifests generate add-license fmt lint test ## Lint and run tests.

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
.PHONY: test
test: envtest generate fmt check-license ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build-onmetal-api-net
build-onmetal-api-net: generate fmt addlicense lint ## Build onmetal-api-net binary.
	go build -o bin/manager ./onmetal-api-net/main.go

.PHONY: build-apinetlet
build-apinetlet: generate fmt addlicense lint ## Build apinetlet.
	go build -o bin/apinetlet ./apinetlet/main.go

.PHONY: build
build: build-onmetal-api-net build-apinetlet ## Build onmetal-api-net and apinetlet.

.PHONY: run-onmetal-api-net
run-onmetal-api-net: manifests generate fmt lint ## Run a onmetal-api-net from your host.
	go run ./onmetal-api-net/main.go

.PHONY: run-apinetlet
run-apinetlet: manifests generate fmt lint ## Run apinetlet from your host.
	go run ./apinetlet/main.go

.PHONY: docker-build-onmetal-api-net
docker-build-onmetal-api-net: ## Build onmetal-api-net image with the manager.
	docker build $(BUILDARGS) --target onmetal-api-net-manager -t ${ONMETAL_API_NET_IMG} .

.PHONY: docker-build-apinetlet
docker-build-apinetlet: ## Build apinetlet image with the manager.
	docker build $(BUILDARGS) --target apinetlet-manager -t ${APINETLET_IMG} .

.PHONY: docker-build
docker-build: docker-build-onmetal-api-net docker-build-apinetlet

.PHONY: docker-push-onmetal-api-net
docker-push-onmetal-api-net: ## Push onmetal-api-net image.
	docker push ${ONMETAL_API_NET_IMG}

.PHONY: docker-push-apinetlet
docker-push-apinetlet: ## Push apinetlet image.
	docker push ${APINETLET_IMG}

.PHONY: docker-push
docker-push: docker-push-onmetal-api-net docker-build-apinetlet ## Push onmetal-api-net and apinetlet image.

##@ Deployment

.PHONY: install-onmetal-api-net
install-onmetal-api-net: manifests ## Install onmetal-api-net CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -k config/onmetal-api-net/crd

.PHONY: uninstall-onmetal-api-net
uninstall-onmetal-api-net: manifests ## Uninstall onmetal-api-net CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete-k config/onmetal-api-net/crd

.PHONY: install-apinetlet
install-apinetlet: manifests ## Install apinetlet CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -k config/apinetlet/crd

.PHONY: uninstall-apinetlet
uninstall-apinetlet: manifests ## Uninstall apinetlet CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete-k config/apinetlet/crd

.PHONY: install
install: install-onmetal-api-net install-apinetlet ## Uninstall onmetal-api-net and apinetlet.

.PHONY: uninstall
uninstall: uninstall-onmetal-api-net uninstall-apinetlet ## Uninstall onmetal-api-net and apinetlet.

.PHONY: deploy-onmetal-api-net
deploy-onmetal-api-net: manifests kustomize ## Deploy onmetal-api-net controller to the K8s cluster specified in ~/.kube/config.
	cd config/onmetal-api-net/manager && $(KUSTOMIZE) edit set image controller=${ONMETAL_API_NET_IMG}
	kubectl apply -k config/onmetal-api-net/default

.PHONY: deploy-apinetlet
deploy-apinetlet: manifests kustomize ## Deploy apinetlet controller to the K8s cluster specified in ~/.kube/config.
	cd config/apinetlet/manager && $(KUSTOMIZE) edit set image controller=${APINETLET_IMG}
	kubectl apply -k config/apinetlet/default

.PHONY: deploy
deploy: deploy-onmetal-api-net deploy-apinetlet

.PHONY: undeploy-onmetal-api-net
undeploy-onmetal-api-net: ## Undeploy onmetal-api-net controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -k config/onmetal-api-net

.PHONY: undeploy-apinetlet
undeploy-apinetlet: ## Undeploy apinetlet controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -k config/apinetlet

.PHONY: undeploy
undeploy: undeploy-onmetal-api-net undeploy-apinetlet ## Undeploy onmetal-api-net and apinetlet controller from the K8s cluster specified in ~/.kube/config.

##@ Kind Deployment plumbing

.PHONY: kind-load-onmetal-api-net
kind-load-onmetal-api-net: docker-build-onmetal-api-net ## Load onmetal-api-net image to kind cluster.
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${ONMETAL_API_NET_IMG}

.PHONY: kind-load-apinetlet
kind-load-apinetlet: docker-build-apinetlet ## Load apinetlet image to kind cluster.
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${APINETLET_IMG}

.PHONY: kind-load
kind-load: kind-load-onmetal-api-net kind-load-apinetlet ## Load onmetal-api-net and apinetlet image to kind cluster.

.PHONY: kind-restart-onmetal-api-net
kind-restart-onmetal-api-net: ## Restarts the onmetal-api-net controller manager.
	kubectl -n onmetal-api-net-system delete rs -l control-plane=controller-manager

.PHONY: kind-restart-apinetlet
kind-restart-apinetlet: ## Restarts the apinetlet controller manager.
	kubectl -n apinetlet-system delete rs -l control-plane=controller-manager

.PHONY: kind-restart
kind-restart: kind-restart-onmetal-api-net kind-restart-apinetlet ## Restarts the onmetal-api-net and apinetlet controller manager.

.PHONY: kind-build-load-restart-onmetal-api-net
kind-build-load-restart-onmetal-api-net: docker-build-onmetal-api-net kind-load-onmetal-api-net kind-restart-onmetal-api-net ## Build, load and restart onmetal-api-net.

.PHONY: kind-build-load-restart-apinetlet
kind-build-load-restart-apinetlet: docker-build-apinetlet kind-load-apinetlet kind-restart-apinetlet ## Build, load and restart apinetlet.

.PHONY: kind-build-load-restart
kind-build-load-restart: kind-build-load-restart-onmetal-api-net kind-build-load-restart-apinetlet

.PHONY: kind-apply-onmetal-api-net
kind-apply-onmetal-api-net: manifests ## Apply onmetal-api-net to the cluster specified in ~/.kube/config.
	kubectl apply -k config/onmetal-api-net/kind

.PHONY: kind-apply-apinetlet
kind-apply-apinetlet: manifests ## Apply apinetlet to the cluster specified in ~/.kube/config.
	kubectl apply -k config/apinetlet/kind

.PHONY: kind-apply
kind-apply: kind-apply-onmetal-api-net kind-apply-apinetlet  ## Apply onmetal-api-net apinetlet to the cluster specified in ~/.kube/config.

.PHONY: kind-deploy
kind-deploy: kind-build-load-restart kind-apply ## Build, load and restart onmetal-api-net and apinetlet and apply them.

.PHONY: kind-delete-onmetal-api-net
kind-delete-onmetal-api-net: ## Delete onmetal-api-net from the cluster specified in ~/.kube/config.
	kubectl delete -k config/onmetal-api-net/kind

.PHONY: kind-delete-apinetlet
kind-delete-apinetlet: ## Delete apinetlet from the cluster specified in ~/.kube/config.
	kubectl delete -k config/apinetlet/kind

.PHONY: kind-delete
kind-delete: kind-delete-onmetal-api-net kind-delete-apinetlet ## Delete onmetal-api-net and apinetlet from the cluster specified in ~/.kube/config.

##@ Tools

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
ADDLICENSE ?= $(LOCALBIN)/addlicense
GOIMPORTS ?= $(LOCALBIN)/goimports

## Tool Versions
KUSTOMIZE_VERSION ?= v5.0.0
CONTROLLER_TOOLS_VERSION ?= v0.11.3
ADDLICENSE_VERSION ?= v1.1.0
GOIMPORTS_VERSION ?= v0.5.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: addlicense
addlicense: $(ADDLICENSE) ## Download addlicense locally if necessary.
$(ADDLICENSE): $(LOCALBIN)
	test -s $(LOCALBIN)/addlicense || GOBIN=$(LOCALBIN) go install github.com/google/addlicense@$(ADDLICENSE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: goimports
goimports: $(GOIMPORTS) ## Download goimports locally if necessary.
$(GOIMPORTS): $(LOCALBIN)
	test -s $(LOCALBIN)/goimports || GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
