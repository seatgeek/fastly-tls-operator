# Variables
BINARY_NAME=fastly-operator
IMAGE_NAME=fastly-operator
IMAGE_TAG=latest
KIND_CLUSTER_NAME=fastly-cluster
KIND_CONTEXT=kind-$(KIND_CLUSTER_NAME)

# Go build flags
GOOS=linux
GOARCH=amd64
CGO_ENABLED=0

.PHONY: help build docker-build kind-create kind-load kind-deploy kind-restart kind-delete clean

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

# Deploy to kind cluster and restart (depends on kind-load)
kind-deploy: kind-load
	@echo "Deploying to kind cluster..."
	kubectl --context $(KIND_CONTEXT) apply -f config/deployment.yaml
	@$(MAKE) _kind-restart-deployment
	@$(MAKE) _clean-artifacts

# Restart deployment to pick up new image
kind-restart:
	@$(MAKE) _kind-restart-deployment

# Internal target to restart deployment (not meant to be called directly)
_kind-restart-deployment:
	@echo "Restarting deployment to pick up new image..."
	kubectl --context $(KIND_CONTEXT) rollout restart deployment/fastly-operator
	@echo "Waiting for rollout to complete..."
	kubectl --context $(KIND_CONTEXT) rollout status deployment/fastly-operator

# Clean build artifacts
_clean-artifacts:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME) 

# Delete kind cluster
kind-delete:
	@echo "Deleting kind cluster '$(KIND_CLUSTER_NAME)'..."
	kind delete cluster --name $(KIND_CLUSTER_NAME)
