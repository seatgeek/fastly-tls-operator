# Fastly Operator

[![Build Status](https://github.com/seatgeek/fastly-operator/workflows/CI/badge.svg)](https://github.com/seatgeek/fastly-operator/actions)
[![Codecov](https://img.shields.io/codecov/c/github/seatgeek/fastly-operator?style=flat-square)](https://codecov.io/gh/seatgeek/fastly-operator)

A Go project that builds Kubernetes controllers for Fastly services.

## Currently In Development

⚠️ This project is currently in development and not intended for public use ⚠️

## Setup

### Prerequisites

1. **Fastly API Token**: You'll need a Fastly API token to interact with Fastly services.

### Configuration

Before deploying the operator, you need to create a secret file with your Fastly API credentials:

1. **Create the Fastly secret file**:
   ```bash
   # Create the secret file
   cat > config/operator/webhook/fastly-secret.yaml << 'EOF'
   apiVersion: v1
   kind: Secret
   metadata:
     name: fastly-secret
     namespace: kube-system
   type: Opaque
   stringData:
     api-key: <YOUR_FASTLY_API_TOKEN_HERE>
   EOF
   ```

2. **Replace the placeholder** with your actual Fastly API token:
   - Edit `config/operator/webhook/fastly-secret.yaml`
   - Replace `<YOUR_FASTLY_API_TOKEN_HERE>` with your Fastly API token

   > ⚠️  **Security Note**: This file contains sensitive credentials and is ignored by git. Never commit actual API tokens to version control.

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
# 1. First, set up your Fastly credentials (see Setup section above)

# 2. Deploy to local kind cluster (builds, creates cluster, and deploys)
make kind-deploy

# 3. Clean up when done
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
make test          - Run tests with coverage
make clean         - Clean build artifacts
```

## Test Coverage

The project includes test coverage reporting:

- Run `make test` to see coverage output in the terminal
- Coverage data is automatically generated in `coverage.out`
- Coverage reports are uploaded to [Codecov](https://codecov.io/gh/seatgeek/fastly-operator) automatically by CI
- Current coverage percentage is shown in the Codecov badge above

## Kubernetes Resources

The project uses a simple deployment manifest in the `config/` directory:

- **deployment.yaml**: Defines the Fastly operator deployment

This approach makes it easy to add more Kubernetes resources (ConfigMaps, Secrets, RBAC, etc.) as the project grows.
