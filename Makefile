
# Image URL to use all building/pushing image targets
APISERVER_IMG ?= apiserver:latest
CONTROLLER_MANAGER_IMG ?= controller:latest
APINETLET_IMG ?= apinetlet:latest
METALNETLET_IMG ?= metalnetlet:latest
KIND_CLUSTER_NAME ?= kind

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.31

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

SSH_KEY ?= ${HOME}/.ssh/id_rsa

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
manifests: controller-gen ## Generate rbac objects.
	# ironcore-net
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths="./internal/controllers/..." output:rbac:artifacts:config=config/controller/rbac

	# apinetlet
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths="./apinetlet/controllers/..." output:rbac:artifacts:config=config/apinetlet/rbac
	CONTROLLER_GEN=$(CONTROLLER_GEN) ./hack/cluster-controller-gen.sh cluster=apinet namespace=system rbac:roleName=apinet-role paths="./apinetlet/controllers/..." output:rbac:artifacts:config=config/apinetlet/apinet-rbac

	# metalnetlet
	CONTROLLER_GEN=$(CONTROLLER_GEN) ./hack/cluster-controller-gen.sh cluster=metalnet rbac:roleName=manager-role paths="./metalnetlet/controllers/..." output:rbac:artifacts:config=config/metalnetlet/rbac
	$(CONTROLLER_GEN) rbac:roleName=apinet-role paths="./metalnetlet/controllers/..." output:rbac:artifacts:config=config/metalnetlet/apinet-rbac

	# Promote *let roles.
	./hack/promote-let-role.sh config/apinetlet/apinet-rbac/role.yaml config/apiserver/rbac/apinetlet_role.yaml apinet.ironcore.dev:system:apinetlets
	./hack/promote-let-role.sh config/metalnetlet/apinet-rbac/role.yaml config/apiserver/rbac/metalnetlet_role.yaml apinet.ironcore.dev:system:metalnetlets

.PHONY: generate
generate: vgopath models-schema openapi-gen
	VGOPATH=$(VGOPATH) \
	MODELS_SCHEMA=$(MODELS_SCHEMA) \
	OPENAPI_GEN=$(OPENAPI_GEN) \
	./hack/update-codegen.sh

.PHONY: add-license
add-license: addlicense ## Add license headers to all go files.
	find . -name '*.go' -exec $(ADDLICENSE) -f hack/license-header.txt {} +

.PHONY: fmt
fmt: goimports ## Run goimports against code.
	$(GOIMPORTS) -w .

.PHONY: check-license
check-license: addlicense ## Check that every file has a license header present.
	find . -name '*.go' -exec $(ADDLICENSE) -check -c 'IronCore authors' {} +

.PHONY: lint
lint: golangci-lint ## Run golangci-lint on the code.
	$(GOLANGCI_LINT) run ./...

.PHONY: clean
clean: ## Clean any artifacts that can be regenerated.
	rm -rf client-go/applyconfigurations
	rm -rf client-go/informers
	rm -rf client-go/listers
	rm -rf client-go/ironcorenet
	rm -rf client-go/openapi

.PHONY: check
check: generate manifests add-license fmt lint test ## Lint and run tests.

.PHONY: test
test: envtest generate fmt check-license test-only ## Run tests.

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
.PHONY: test-only
test-only: envtest ## Only run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

.PHONY: extract-openapi
extract-openapi: envtest openapi-extractor
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(OPENAPI_EXTRACTOR) \
		--apiserver-package="github.com/ironcore-dev/ironcore-net/cmd/apiserver" \
		--apiserver-build-opts=mod \
		--apiservices="./config/apiserver/apiservice/bases" \
		--output="./gen"

.PHONY: docs
docs: gen-crd-api-reference-docs ## Run go generate to generate API reference documentation.
	$(GEN_CRD_API_REFERENCE_DOCS) -api-dir ./api/core/v1alpha1 -config ./hack/api-reference/config.json -template-dir ./hack/api-reference/template -out-file ./docs/api-reference/core.md

##@ Build

.PHONY: build-ironcore-net
build-ironcore-net: generate fmt addlicense lint ## Build ironcore-net binary.
	go build -o bin/manager ./cmd/controller-manager/main.go
	go build -o bin/apiserver ./cmd/apiserver/main.go

.PHONY: build-apinetlet
build-apinetlet: generate fmt addlicense lint ## Build apinetlet.
	go build -o bin/apinetlet ./cmd/apinetlet/main.go

.PHONY: build-metalnetlet
build-metalnetlet: generate fmt addlicense lint ## Build metalnetlet.
	go build -o bin/metalnetlet ./cmd/metalnetlet/main.go

.PHONY: build
build: build-ironcore-net build-apinetlet build-metalnetlet ## Build ironcore-net, apinetlet, metalnetlet.

.PHONY: run-ironcore-net
run-ironcore-net: manifests generate fmt lint ## Run a ironcore-net from your host.
	go run ./cmd/ironcore-net/main.go

.PHONY: run-apinetlet
run-apinetlet: manifests generate fmt lint ## Run apinetlet from your host.
	go run ./cmd/apinetlet/main.go

.PHONY: run-metalnetlet
run-metalnetlet: manifests generate fmt lint ## Run metalnetlet from your host.
	go run ./cmd/metalnetlet/main.go

.PHONY: docker-build-apiserver
docker-build-apiserver: ## Build apiserver image.
	docker build --ssh default=${SSH_KEY} --target apiserver -t ${APISERVER_IMG} .

.PHONY: docker-build-controller-manager
docker-build-controller-manager: ## Build controller-manager image.
	docker build --ssh default=${SSH_KEY} --target controller-manager -t ${CONTROLLER_MANAGER_IMG} .

.PHONY: docker-build-apinetlet
docker-build-apinetlet: ## Build apinetlet image with the manager.
	docker build --ssh default=${SSH_KEY} --target apinetlet-manager -t ${APINETLET_IMG} .

.PHONY: docker-build-metalnetlet
docker-build-metalnetlet: ## Build metalnetlet image with the manager.
	docker build --ssh default=${SSH_KEY} --target metalnetlet-manager -t ${METALNETLET_IMG} .

.PHONY: docker-build
docker-build: docker-build-apiserver docker-build-controller-manager docker-build-apinetlet docker-build-metalnetlet ## Build docker images.

.PHONY: docker-push-apiserver
docker-push-apiserver: ## Push apiserver image.
	docker push ${APISERVER_IMG}

.PHONY: docker-push-controller-manager
docker-push-controller-manager: ## Push controller-manager image.
	docker push ${CONTROLLER_MANAGER_IMG}

.PHONY: docker-push-apinetlet
docker-push-apinetlet: ## Push apinetlet image.
	docker push ${APINETLET_IMG}

.PHONY: docker-push-metalnetlet
docker-push-metalnetlet: ## Push metalnetlet image.
	docker push ${METALNETLET_IMG}

.PHONY: docker-push
docker-push: docker-push-apiserver docker-push-controller-manager docker-push-apinetlet docker-push-metalnetlet ## Push ironcore-net, apinetlet, metalnetlet image.

##@ Deployment

.PHONY: install
install: manifests kustomize ## Install ironcore-net API server & API services into the K8s cluster specified in ~/.kube/config. This requires APISERVER_IMG to be available for the cluster.
	cd config/apiserver/server && $(KUSTOMIZE) edit set image apiserver=${APISERVER_IMG}
	kubectl apply -k config/apiserver/default

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall ironcore-net API server & API services from the K8s cluster specified in ~/.kube/config.
	kubectl delete -k config/apiserver/default

.PHONY: deploy-ironcore-net
deploy-ironcore-net: manifests kustomize ## Deploy ironcore-net controller to the K8s cluster specified in ~/.kube/config.
	cd config/controller/manager && $(KUSTOMIZE) edit set image controller=${CONTROLLER_MANAGER_IMG}
	kubectl apply -k config/controller/default

.PHONY: deploy-apinetlet
deploy-apinetlet: manifests kustomize ## Deploy apinetlet controller to the K8s cluster specified in ~/.kube/config.
	cd config/apinetlet/manager && $(KUSTOMIZE) edit set image apinetlet=${APINETLET_IMG}
	kubectl apply -k config/apinetlet/default

.PHONY: deploy-metalnetlet
deploy-metalnetlet: manifests kustomize ## Deploy metalnetlet controller to the K8s cluster specified in ~/.kube/config.
	cd config/metalnetlet/manager && $(KUSTOMIZE) edit set image metalnetlet=${METALNETLET_IMG}
	kubectl apply -k config/metalnetlet/default

.PHONY: deploy
deploy: deploy-ironcore-net deploy-apinetlet deploy-metalnetlet ## Deploy ironcore-net, apinetlet, metalnetlet

.PHONY: undeploy-ironcore-net
undeploy-ironcore-net: ## Undeploy ironcore-net controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -k config/controller/default

.PHONY: undeploy-apinetlet
undeploy-apinetlet: ## Undeploy apinetlet controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -k config/apinetlet/default

.PHONY: undeploy-metalnetlet
undeploy-metalnetlet: ## Undeploy metalnetlet controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -k config/metalnetlet/default

.PHONY: undeploy
undeploy: undeploy-ironcore-net undeploy-apinetlet undeploy-metalnetlet ## Undeploy ironcore-net, apinetlet, metalnetlet controller from the K8s cluster specified in ~/.kube/config.

##@ Kind Deployment plumbing

.PHONY: kind-load-apiserver
kind-load-apiserver: docker-build-apiserver ## Load apiserver image to kind cluster.
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${APISERVER_IMG}

.PHONY: kind-load-controller-manager
kind-load-controller-manager: docker-build-controller-manager ## Load controller-manager image to kind cluster.
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${CONTROLLER_MANAGER_IMG}

.PHONY: kind-load-apinetlet
kind-load-apinetlet: docker-build-apinetlet ## Load apinetlet image to kind cluster.
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${APINETLET_IMG}

.PHONY: kind-load-metalnetlet
kind-load-metalnetlet: docker-build-metalnetlet ## Load metalnetlet image to kind cluster.
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${METALNETLET_IMG}

.PHONY: kind-load
kind-load: kind-load-apiserver kind-load-controller-manager ## Load apiserver, controller-manager image to kind cluster.

.PHONY: kind-restart-apiserver
kind-restart-apiserver: ## Restarts the apiserver.
	kubectl -n ironcore-net-system delete rs -l control-plane=apiserver

.PHONY: kind-restart-controller-manager
kind-restart-controller-manager: ## Restarts the controller-manager.
	kubectl -n ironcore-net-system delete rs -l control-plane=controller-manager

.PHONY: kind-restart-apinetlet
kind-restart-apinetlet: ## Restarts the apinetlet controller manager.
	kubectl -n apinetlet-system delete rs -l control-plane=controller-manager

.PHONY: kind-restart-metalnetlet
kind-restart-metalnetlet: ## Restarts the metalnetlet controller manager.
	kubectl -n metalnetlet-system delete rs -l control-plane=controller-manager

.PHONY: kind-restart
kind-restart: kind-restart-apiserver kind-restart-controller-manager ## Restarts the apiserver, controller manager.

.PHONY: kind-build-load-restart-apiserver
kind-build-load-restart-apiserver: docker-build-apiserver kind-load-apiserver kind-restart-apiserver ## Build, load and restart apiserver.

.PHONY: kind-build-load-restart-controller-manager
kind-build-load-restart-controller-manager: docker-build-controller-manager kind-load-controller-manager kind-restart-controller-manager ## Build, load and restart controller-manager.

.PHONY: kind-build-load-restart-apinetlet
kind-build-load-restart-apinetlet: docker-build-apinetlet kind-load-apinetlet kind-restart-apinetlet ## Build, load and restart apinetlet.

.PHONY: kind-build-load-restart-metalnetlet
kind-build-load-restart-metalnetlet: docker-build-metalnetlet kind-load-metalnetlet kind-restart-metalnetlet ## Build, load and restart metalnetlet.

.PHONY: kind-build-load-restart
kind-build-load-restart: kind-build-load-restart-apiserver kind-build-load-restart-controller-manager

.PHONY: kind-apply-apinetlet
kind-apply-apinetlet: manifests ## Apply apinetlet to the cluster specified in ~/.kube/config.
	kubectl apply -k config/apinetlet/kind

.PHONY: kind-apply-metalnetlet
kind-apply-metalnetlet: manifests ## Apply metalnetlet to the cluster specified in ~/.kube/config.
	kubectl apply -k config/metalnetlet/kind

.PHONY: kind-apply
kind-apply: manifests ## Apply config/kind to the cluster specified in ~/.kube/config.
	kubectl apply -k config/kind

.PHONY: kind-deploy
kind-deploy: kind-build-load-restart kind-apply ## Build, load and restart ironcore-net apiserver and controller manager and apply them.

.PHONY: kind-delete
kind-delete: ## Delete config/kind from the cluster specified in ~/.kube/config.
	kubectl delete -k config/kind

.PHONY: kind-delete-apinetlet
kind-delete-apinetlet: ## Delete apinetlet from the cluster specified in ~/.kube/config.
	kubectl delete -k config/apinetlet/kind

.PHONY: kind-delete-metalnetlet
kind-delete-metalnetlet: ## Delete metalnetlet from the cluster specified in ~/.kube/config.
	kubectl delete -k config/metalnetlet/kind

##@ Tools

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
OPENAPI_EXTRACTOR ?= $(LOCALBIN)/openapi-extractor
OPENAPI_GEN ?= $(LOCALBIN)/openapi-gen
VGOPATH ?= $(LOCALBIN)/vgopath
GEN_CRD_API_REFERENCE_DOCS ?= $(LOCALBIN)/gen-crd-api-reference-docs
ADDLICENSE ?= $(LOCALBIN)/addlicense
MODELS_SCHEMA ?= $(LOCALBIN)/models-schema
GOIMPORTS ?= $(LOCALBIN)/goimports
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.1.1
VGOPATH_VERSION ?= v0.1.5
CONTROLLER_TOOLS_VERSION ?= v0.16.0
GEN_CRD_API_REFERENCE_DOCS_VERSION ?= v0.3.0
ADDLICENSE_VERSION ?= v1.1.1
GOIMPORTS_VERSION ?= v0.25.0
GOLANGCI_LINT_VERSION ?= v1.62.2
OPENAPI_EXTRACTOR_VERSION ?= v0.1.9

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: openapi-gen
openapi-gen: $(OPENAPI_GEN) ## Download openapi-gen locally if necessary.
$(OPENAPI_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/openapi-gen || GOBIN=$(LOCALBIN) go install k8s.io/kube-openapi/cmd/openapi-gen


.PHONY: vgopath
vgopath: $(VGOPATH) ## Download vgopath locally if necessary.
.PHONY: $(VGOPATH)
$(VGOPATH): $(LOCALBIN)
	@if test -x $(LOCALBIN)/vgopath && ! $(LOCALBIN)/vgopath version | grep -q $(VGOPATH_VERSION); then \
		echo "$(LOCALBIN)/vgopath version is not expected $(VGOPATH_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/vgopath; \
	fi
	test -s $(LOCALBIN)/vgopath || GOBIN=$(LOCALBIN) go install github.com/ironcore-dev/vgopath@$(VGOPATH_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: openapi-extractor
openapi-extractor: $(OPENAPI_EXTRACTOR) ## Download openapi-extractor locally if necessary.
$(OPENAPI_EXTRACTOR): $(LOCALBIN)
	test -s $(LOCALBIN)/openapi-extractor || GOBIN=$(LOCALBIN) go install github.com/ironcore-dev/openapi-extractor/cmd/openapi-extractor@$(OPENAPI_EXTRACTOR_VERSION)

.PHONY: gen-crd-api-reference-docs
gen-crd-api-reference-docs: $(GEN_CRD_API_REFERENCE_DOCS) ## Download gen-crd-api-reference-docs locally if necessary.
$(GEN_CRD_API_REFERENCE_DOCS): $(LOCALBIN)
	test -s $(LOCALBIN)/gen-crd-api-reference-docs || GOBIN=$(LOCALBIN) go install github.com/ahmetb/gen-crd-api-reference-docs@$(GEN_CRD_API_REFERENCE_DOCS_VERSION)

.PHONY: addlicense
addlicense: $(ADDLICENSE) ## Download addlicense locally if necessary.
$(ADDLICENSE): $(LOCALBIN)
	test -s $(LOCALBIN)/addlicense || GOBIN=$(LOCALBIN) go install github.com/google/addlicense@$(ADDLICENSE_VERSION)

.PHONY: models-schema
models-schema: $(MODELS_SCHEMA) ## Install models-schema locally if necessary.
$(MODELS_SCHEMA): $(LOCALBIN)
	test -s $(LOCALBIN)/models-schema || GOBIN=$(LOCALBIN) go install github.com/ironcore-dev/ironcore-net/models-schema

.PHONY: goimports
goimports: $(GOIMPORTS) ## Download goimports locally if necessary.
$(GOIMPORTS): $(LOCALBIN)
	test -s $(LOCALBIN)/goimports || GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
