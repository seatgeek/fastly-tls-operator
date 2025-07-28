# Operator Variables
BINARY_NAME=fastly-operator
IMAGE_NAME=fastly-operator
IMAGE_TAG=latest
KIND_CLUSTER_NAME=fastly-cluster
KIND_CONTEXT=kind-$(KIND_CLUSTER_NAME)
OPERATOR_NAME=fastly-operator

# Namespace Variables
OPERATOR_NAMESPACE=kube-system  # Where the operator and CA secret are deployed
TEST_NAMESPACE=fastly-test      # Where test resources (examples) are deployed

# Helm variables
CHART_PATH=charts/fastly-operator

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
OPENSSL ?= openssl
KIND ?= kind
HELM ?= helm

## Tool Versions
KUSTOMIZE_VERSION ?= v5.0.1
CONTROLLER_TOOLS_VERSION ?= v0.15.0

.PHONY: help build docker-build kind-create kind-load kind-deploy kind-restart kind-delete clean controller-gen generate manifests kustomize install apply-issuer-secret apply-examples kind-dependencies helm-lint helm-template helm-test helm-validate-all helm-integration-test lint fmt vet test check

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Development:"
	@echo "  build         - Build the Go binary"
	@echo "  docker-build  - Build Docker image (depends on build)"
	@echo "  clean         - Clean build artifacts"
	@echo "  generate      - Generate code (DeepCopy, etc.)"
	@echo "  manifests     - Generate CRDs and RBAC, sync to Helm chart"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint          - Run golangci-lint (same as CI)"
	@echo "  fmt           - Run gofumpt formatting"
	@echo "  vet           - Run go vet"
	@echo "  test          - Run tests with coverage"
	@echo "  check         - Run all code quality checks (fmt, vet, lint, test)"
	@echo ""
	@echo "Kind Cluster Management:"
	@echo "  kind-create   - Create kind cluster"
	@echo "  kind-load     - Load Docker image into kind cluster (depends on docker-build, kind-create)"
	@echo "  kind-deploy   - Deploy to kind cluster using Kustomize (depends on kind-load)"
	@echo "  kind-restart  - Restart deployment to pick up new image"
	@echo "  kind-delete   - Delete kind cluster"
	@echo "  install       - Install CRDs into cluster"
	@echo "  apply-issuer-secret - Generate and apply CA issuer secret to $(OPERATOR_NAMESPACE)"
	@echo "  apply-examples - Apply ClusterIssuer and example resources to $(TEST_NAMESPACE)"
	@echo ""
	@echo "Helm Chart (Local Validation):"
	@echo "  helm-lint     - Lint Helm chart (local-only)"
	@echo "  helm-template - Render Helm chart templates (local-only)"
	@echo "  helm-test     - Basic Helm chart validation (local-only)"
	@echo "  helm-validate-all - Run all local Helm validation targets"
	@echo ""
	@echo "Helm Chart (Cluster Testing):"
	@echo "  helm-integration-test - Run full Helm chart integration test with kind cluster"

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
	@if $(KIND) get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		echo "Kind cluster '$(KIND_CLUSTER_NAME)' already exists."; \
	else \
		echo "Creating kind cluster '$(KIND_CLUSTER_NAME)'..."; \
		$(KIND) create cluster --name $(KIND_CLUSTER_NAME); \
	fi

# Load Docker image into kind cluster (depends on docker-build and kind-create)
kind-load: docker-build kind-create
	@echo "Loading Docker image into kind cluster..."
	$(KIND) load docker-image $(IMAGE_NAME):$(IMAGE_TAG) --name $(KIND_CLUSTER_NAME)

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
	kubectl --context $(KIND_CONTEXT) -n $(OPERATOR_NAMESPACE) rollout restart deployment/fastly-operator
	@echo "Waiting for rollout to complete..."
	kubectl --context $(KIND_CONTEXT) -n $(OPERATOR_NAMESPACE) rollout status deployment/fastly-operator

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)

# Internal target to clean artifacts (kept for backward compatibility)
_clean-artifacts: clean

# ============================
# Code Quality Targets
# ============================

# Run golangci-lint (same as CI)
lint:
	@echo "Running golangci-lint..."
	golangci-lint run

# Run gofumpt formatting
fmt:
	@echo "Running gofumpt formatting..."
	$(shell go env GOPATH)/bin/gofumpt -w .

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run tests with coverage (internal packages only)
test:
	@echo "Running tests with coverage for internal packages..."
	go test -v -coverprofile=coverage.out ./internal/...
	@echo "Coverage by package (excluding 100% covered functions):"
	@go tool cover -func=coverage.out | awk '$$NF != "100.0%" || /^total:/'
	@echo ""
	@echo "Total coverage: $$(go tool cover -func=coverage.out | grep total | awk '{print $$3}')"



# Run all code quality checks (mimics CI)
check: fmt vet lint test
	@echo "All code quality checks completed successfully!"

# Delete kind cluster
kind-delete:
	@echo "Deleting kind cluster '$(KIND_CLUSTER_NAME)'..."
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)

# ============================
# Helm Chart Targets
# ============================

# Lint Helm chart (local-only)
helm-lint:
	@echo "Linting Helm chart..."
	@if [ ! -d "$(CHART_PATH)" ]; then \
		echo "Error: Chart directory '$(CHART_PATH)' does not exist."; \
		echo "Please ensure Phase 1 is complete and the chart has been created."; \
		exit 1; \
	fi
	$(HELM) lint $(CHART_PATH)

# Render Helm chart templates (local-only)
helm-template:
	@echo "Rendering Helm chart templates..."
	@if [ ! -d "$(CHART_PATH)" ]; then \
		echo "Error: Chart directory '$(CHART_PATH)' does not exist."; \
		echo "Please ensure Phase 1 is complete and the chart has been created."; \
		exit 1; \
	fi
	$(HELM) template fastly-operator $(CHART_PATH) \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG) \
		--set fastly.secretName=fastly-secret

# Basic Helm chart validation (local-only)
helm-test:
	@echo "Running basic Helm chart validation..."
	@if [ ! -d "$(CHART_PATH)" ]; then \
		echo "Error: Chart directory '$(CHART_PATH)' does not exist."; \
		echo "Please ensure Phase 1 is complete and the chart has been created."; \
		exit 1; \
	fi
	@echo "Checking Chart.yaml syntax..."
	@$(HELM) show chart $(CHART_PATH) > /dev/null
	@echo "Checking values.yaml syntax..."
	@$(HELM) show values $(CHART_PATH) > /dev/null
	@echo "Checking template rendering (dry-run)..."
	@$(HELM) template fastly-operator $(CHART_PATH) \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG) \
		--set fastly.secretName=fastly-secret \
		--dry-run > /dev/null
	@echo "Basic validation completed successfully."

# Run all local Helm validation targets
helm-validate-all: helm-lint helm-template helm-test
	@echo "All local Helm validation targets completed successfully."

# Run full Helm chart integration test with kind cluster
helm-integration-test:
	@echo "Running Helm chart integration test..."
	@if [ ! -f "scripts/test-helm-chart.sh" ]; then \
		echo "Error: Test script 'scripts/test-helm-chart.sh' not found."; \
		echo "Please ensure Phase 2.1 is complete."; \
		exit 1; \
	fi
	@chmod +x scripts/test-helm-chart.sh
	@./scripts/test-helm-chart.sh
	@echo "Applying test resources..."
	@$(MAKE) apply-issuer-secret
	@$(MAKE) apply-examples
	@echo "Helm chart integration test with test resources completed successfully!"

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
	@echo "Syncing CRDs to Helm chart..."
	@mkdir -p charts/fastly-operator/crds
	@cp config/crd/bases/*.yaml charts/fastly-operator/crds/
	@mv charts/fastly-operator/crds/platform.seatgeek.io_fastlycertificatesyncs.yaml charts/fastly-operator/crds/fastlycertificatesyncs.platform.seatgeek.io.yaml

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

test-namespace:
	@$(KUBECTL) create namespace $(TEST_NAMESPACE) --dry-run=client -o yaml | $(KUBECTL) apply -f -

apply-issuer-secret: test-namespace
	@echo "Generating issuer PK..."
	$(OPENSSL) genpkey -algorithm RSA -out root-ca.key -pkeyopt rsa_keygen_bits:4096
	@echo "Generating local CA issuer secret..."
	$(OPENSSL) req -x509 -new -key root-ca.key -out root-ca.crt -days 3650 \
	-subj "/CN=My Root CA" \
	-extensions v3_ca \
	-addext "keyUsage = critical, digitalSignature, keyCertSign, cRLSign" \
	-addext "extendedKeyUsage = serverAuth, clientAuth"
	@echo "Creating local CA issuer secret..."
	$(KUBECTL) create secret -n $(TEST_NAMESPACE) tls local-ca-issuer --key=root-ca.key --cert=root-ca.crt
	@echo "Removing local CA issuer secret files..."
	rm root-ca.key root-ca.crt

# Apply example resources
apply-examples: install test-namespace
	@echo "Creating test namespace '$(TEST_NAMESPACE)'..."
	@echo "Applying ClusterIssuer..."
	$(KUBECTL) -n $(TEST_NAMESPACE)apply -f hack/fastlycertificatesync/issuer.yaml
	@echo "Applying example resources to '$(TEST_NAMESPACE)' namespace..."
	$(KUBECTL) -n $(TEST_NAMESPACE) apply -f hack/fastlycertificatesync/example.yaml
