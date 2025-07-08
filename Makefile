# Operator Variables
BINARY_NAME=fastly-operator
IMAGE_NAME=fastly-operator
IMAGE_TAG=latest
KIND_CLUSTER_NAME=fastly-cluster
KIND_CONTEXT=kind-$(KIND_CLUSTER_NAME)
OPERATOR_NAME=fastly-operator

# Go build flags
GOOS=linux
GOARCH=amd64
CGO_ENABLED=0

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl --context=$(KIND_CONTEXT)
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

## Tool Versions
KUSTOMIZE_VERSION ?= v5.0.1
CONTROLLER_TOOLS_VERSION ?= v0.15.0

.PHONY: help build docker-build kind-create kind-load kind-deploy kind-restart kind-delete clean controller-gen generate manifests kustomize install apply-examples kind-dependencies

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build the Go binary"
	@echo "  docker-build  - Build Docker image (depends on build)"
	@echo "  kind-create   - Create kind cluster"
	@echo "  kind-load     - Load Docker image into kind cluster (depends on docker-build, kind-create)"
	@echo "  kind-deploy   - Deploy to kind cluster and restart (depends on kind-load)"
	@echo "  kind-restart  - Restart deployment to pick up new image"
	@echo "  kind-delete   - Delete kind cluster"
	@echo "  clean         - Clean build artifacts"
	@echo "  generate      - Generate code (DeepCopy, etc.)"
	@echo "  manifests     - Generate CRDs and RBAC"
	@echo "  install       - Install CRDs into cluster"
	@echo "  apply-examples - Apply example resources"

# Build the Go binary
build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY_NAME) ./cmd

# Build Docker image (depends on build)
docker-build: build
	@echo "Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)..."
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Create kind cluster
kind-create:
	@if kind get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		echo "Kind cluster '$(KIND_CLUSTER_NAME)' already exists."; \
	else \
		echo "Creating kind cluster '$(KIND_CLUSTER_NAME)'..."; \
		kind create cluster --name $(KIND_CLUSTER_NAME); \
	fi

# Load Docker image into kind cluster (depends on docker-build and kind-create)
kind-load: docker-build kind-create
	@echo "Loading Docker image into kind cluster..."
	kind load docker-image $(IMAGE_NAME):$(IMAGE_TAG) --name $(KIND_CLUSTER_NAME)

.PHONY: kind-install-dependencies
kind-install-dependencies:
	@echo "Installing cert-manager in kind cluster..."
	$(KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
	@echo "Waiting for cert-manager deployment to be ready..."
	$(KUBECTL) -n cert-manager rollout status deployment/cert-manager-webhook


# Deploy to kind cluster and restart (depends on kind-load)
kind-deploy: kind-load kind-install-dependencies
	@echo "Deploying to kind cluster..."
	$(KUSTOMIZE) build config/ | $(KUBECTL) apply -f -
	@$(MAKE) _kind-restart-deployment
	@$(MAKE) _clean-artifacts

# Restart deployment to pick up new image
kind-restart:
	@$(MAKE) _kind-restart-deployment

# Internal target to restart deployment (not meant to be called directly)
_kind-restart-deployment:
	@echo "Restarting deployment to pick up new image..."
	kubectl --context $(KIND_CONTEXT) -n kube-system rollout restart deployment/fastly-operator
	@echo "Waiting for rollout to complete..."
	kubectl --context $(KIND_CONTEXT) -n kube-system rollout status deployment/fastly-operator

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)

# Internal target to clean artifacts (kept for backward compatibility)
_clean-artifacts: clean

# Delete kind cluster
kind-delete:
	@echo "Deleting kind cluster '$(KIND_CLUSTER_NAME)'..."
	kind delete cluster --name $(KIND_CLUSTER_NAME)

# Download controller-gen locally if necessary
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

# Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=$(OPERATOR_NAME) crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases output:webhook:dir=config/operator/webhook

# Download kustomize locally if necessary
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

# Install CRDs into the K8s cluster specified in ~/.kube/config
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

# Apply example resources
apply-examples: install
	@if ! $(KUBECTL) get namespace test >/dev/null 2>&1; then \
		echo "Creating namespace test..."; \
		$(KUBECTL) create namespace test; \
	else \
		echo "Namespace test already exists"; \
	fi
	$(KUBECTL) -n test apply -f hack/fastlycertificatesync/example.yaml
