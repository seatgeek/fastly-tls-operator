# Fastly Operator

[![Build Status](https://github.com/seatgeek/fastly-operator/workflows/CI/badge.svg)](https://github.com/seatgeek/fastly-operator/actions)
[![Codecov](https://img.shields.io/codecov/c/github/seatgeek/fastly-operator?style=flat-square)](https://codecov.io/gh/seatgeek/fastly-operator)

A Kubernetes operator that powers Fastly TLS automation.

It is currently focused on synchronizing TLS certificates between [cert-manager](https://cert-manager.io/) and Fastly.

You can think of this operator as the "glue" between your certificate definitions and how they are uploaded to Fastly.

Simply point to the `cert-manager` certificate, and the operator will take care of the rest!

![Architecture Diagram](./docs/architecture.png)

```yaml
apiVersion: platform.seatgeek.io/v1alpha1
kind: FastlyCertificateSync
metadata:
  name: example-1
spec:
  certificateName: example-1
  suspend: false
  tlsConfigurationIds:
  - abcd1234 # customer specific reference to tls configuration
```

## Context

### TLS Subscriptions (Alternative)

Fastly-managed TLS Certificates can be defined using [TLS Subscriptions](https://www.fastly.com/documentation/reference/api/tls/subs/), typically via their [terraform provider](https://registry.terraform.io/providers/fastly/fastly/latest/docs/resources/tls_subscription).

The certificate, private key(s), and all details are managed on Fastly's end.

This approach doesn't allow you to own your private keys or leverage tls certificates you might already be defining on your k8s cluster(s).

### Custom Certificates

Fastly also allows you to define custom, self-defined certificates via their [Custom TLS Certificates API](https://www.fastly.com/documentation/reference/api/tls/custom-certs/).

Although the API is complete, what is lacking is automation tying things together.

This operator bridges that gap by building the necessary automation for you!

### Cert-Manager

[cert-manager](https://cert-manager.io/) is the canonical k8s native solution for managing certificates. Fastly-Operator builds on top of this familiar solution without reinventing how certificates in k8s are managed.

## How it Works

1. You create a `Certificate` resource using cert-manager (with Let's Encrypt, internal CA, etc.)
2. You create a `FastlyCertificateSync` resource pointing to that certificate and specifying Fastly TLS configuration IDs
3. The operator watches both resources and automatically:
   - Uploads the certificate and private key to Fastly
   - Associates them with your specified TLS configurations
   - Monitors for certificate renewals and re-syncs as needed
   - Reports status and any issues back to Kubernetes

## Why Use This Operator?

- ✅ **Own Your Private Keys**: Maintain control over your Private Keys instead of delegating to Fastly
- ✅ **Automate Certificate Syncronization**: Certificates automatically flow from cert-manager to Fastly
- ✅ **Automatic Renewals**: Certificate renewals are automatically synced to Fastly
- ✅ **Kubernetes Native**: Manage everything through Kubernetes resources and tooling
- ✅ **Status Visibility**: Clear status reporting and conditions in Kubernetes

## Quickstart Guide

This guide should get you up and running with an example certificate.

### Prerequisites

1. **Kubernetes Cluster running cert-manager**: [installation guide](https://cert-manager.io/docs/installation/)
2. **Fastly Account**: You must be a Fastly customer with an account in order to use this operator.
3. **Fastly API Token**: A token must be created with the proper permissions:
  * Type: `Automation Token`
  * Role: `Engineer - Read, write, purge & activate on service configuration`
  * TLS management: `Grant access to modify TLS configuration across all services, including TLS certificates and domains.
  * Scope: `All services on SeatGeek`
4. A namespace to deploy the operator to, we'll use `fastly-system` in the examples below.


### Step 1: Set Up Credentials

Create a Kubernetes secret with your Fastly API token:

```bash
kubectl create secret generic fastly-operator-secrets \
  --from-literal=api-key="YOUR_FASTLY_API_TOKEN" \
  --namespace=fastly-system
```

In production, you may want to store this secret in a secure secret storage system.

### Step 2: Install the Operator Using Helm

```bash
# Add the Helm repository
helm repo add seatgeek https://seatgeek.github.io/charts
helm repo update

# Install the operator
helm install fastly-operator seatgeek/fastly-operator \
  --namespace fastly-system
```

### Step 3: Create a Certificate

Please refer to [cert-manager documentation](https://cert-manager.io/docs/) if you aren't familiar with this tool.

First, ensure you have a cert-manager Issuer configured, then create a Certificate:

**Note**: Make sure to annotate your target certificates with `platform.seatgeek.io/enable-fastly-sync: true`! This helps our controller avoid reconciling every single Certificate on the cluster when changes take place. We only want to fully watch and inspect the subset of all Certificates that are being synced to Fastly.

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  annotations:
    platform.seatgeek.io/enable-fastly-sync: true
  name: my-app-cert
  namespace: default
spec:
  secretName: my-app-tls
  issuerRef:
    name: my-letsencrypt-issuer  # Your configured issuer
    kind: ClusterIssuer
  commonName: my-app.example.com
  # In order for this example to work, you will need to choose owned domains that the Issuer can solve ACME challenges for
  dnsNames:
  - my-app.example.com
  - api.my-app.example.com
```

### Step 4: Create a FastlyCertificateSync Resource

Create a sync resource to connect your certificate to Fastly:

```yaml
apiVersion: platform.seatgeek.io/v1alpha1
kind: FastlyCertificateSync
metadata:
  name: my-app-cert-sync
  namespace: default
spec:
  certificateName: my-app-cert
  tlsConfigurationIds:
  - "your-fastly-tls-config-id-1"  # Replace with your actual TLS configuration ID
  - "your-fastly-tls-config-id-2"  # Optional: activate against multiple TLS configurations
```

> 💡 **Finding TLS Configuration IDs**: You can find these in the Fastly dashboard under Security > TLS Management -> Configurations. These differ from customer to customer so we cannot provide default values here or in the operator.

### Step 5: Verify Synchronization

Check the status of your certificate sync:

```bash
# Check the sync resource status
kubectl get fastlycertificatesync my-app-cert-sync -o yaml

# Check operator logs
kubectl logs -n kube-system deployment/fastly-operator
```

The `FastlyCertificateSync` resource will show status conditions indicating whether the certificate has been successfully uploaded and configured in Fastly.

## Development Setup

All development is powered with a `kind` cluster where we can install our helm chart and define example resources:

* `config/` contains the resources that will be installed in order to run the operator.
* `hack/` contains some example resources that will demonstrate the behavior of the operator.

Our `makefile` setup has some high level commands that will bootstrap everything for you

### Local Development Workflow

```bash
# Clone the repository
git clone https://github.com/seatgeek/fastly-operator.git
cd fastly-operator

# Create a local kind cluster and deploy the operator
make kind-deploy

# Apply example resources for testing
make apply-examples
```

## Configuration

### FastlyCertificateSync Resource Spec

| Field | Type | Description |
|-------|------|-------------|
| `certificateName` | string | Name of the cert-manager Certificate resource to sync |
| `tlsConfigurationIds` | []string | List of Fastly TLS configuration IDs to sync the certificate to |
| `suspend` | bool | Temporarily suspend reconciliation of this resource |

### Status Conditions

The operator reports several status conditions:

- **Ready**: Overall readiness of the certificate sync
- **PrivateKeyReady**: Whether the private key has been uploaded to Fastly
- **CertificateReady**: Whether the certificate has been uploaded and is current
- **CleanupRequired**: Whether old/unused certificates need cleanup

## Security Considerations

- **API Token Storage**: Use Kubernetes secrets or external secret management systems
- **RBAC**: The operator requires permissions to manage secrets and cert-manager resources
- **Network Policy**: Consider restricting the operator's network access to only Fastly APIs
- **Private Key Handling**: Private keys are handled in-memory only and uploaded securely to Fastly

## Contributing

Please feel free to open


## Support

For issues and questions:

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/seatgeek/fastly-operator/issues)
