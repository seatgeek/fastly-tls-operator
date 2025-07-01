# Fastly Operator

A Go project that builds Kubernetes controllers for Fastly services.

## Project Structure

```
fastly-operator/
├── cmd/
│   └── main.go          # Application entry point
├── config/
│   └── deployment.yaml  # Kubernetes deployment manifest
├── Makefile             # Build and deployment automation
├── README.md
└── Dockerfile
```

## Quick Start

```bash
# Deploy to local kind cluster (builds, creates cluster, and deploys)
make kind-deploy

# Clean up when done
make kind-delete
make clean
```

## Available Commands

```bash
make help          # Show all available commands
make build         - Build the Go binary locally
make docker-build  - Build Docker image
make kind-create   - Create kind cluster
make kind-load     - Load Docker image into kind cluster
make kind-deploy   - Build and deploy to kind cluster
make kind-delete   - Delete kind cluster
make clean         - Clean build artifacts
```

## Kubernetes Resources

The project uses a simple deployment manifest in the `config/` directory:

- **deployment.yaml**: Defines the Fastly operator deployment

This approach makes it easy to add more Kubernetes resources (ConfigMaps, Secrets, RBAC, etc.) as the project grows.
