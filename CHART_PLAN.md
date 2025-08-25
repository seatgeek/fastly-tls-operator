# Fastly TLS Operator - GitHub Publishing & Container Build Plan

## Overview

This document outlines the plan for hosting the Fastly TLS Operator Helm chart on GitHub Pages and building/publishing Docker images to GitHub Container Registry (GHCR). The chart is already functional and ready for distribution.

## Goals

- 🎯 **Host Helm Chart on GitHub Pages** - Make the chart easily installable via `helm repo add`
- 🐳 **Publish Docker Images to GHCR** - Automated multi-architecture container builds
- 🚀 **Automated CI/CD Pipeline** - Tag-based releases with GitHub Actions
- 📦 **Semantic Versioning** - Proper release management with changelogs
- 🔒 **Security & Best Practices** - Signed containers and vulnerability scanning

## Current State

✅ **Helm Chart**: Ready in `charts/fastly-tls-operator/` (v0.1.0)  
✅ **Docker Build**: Working Dockerfile with multi-stage build  
✅ **Local Testing**: Makefile targets for validation  
❌ **GitHub Actions**: Not configured  
❌ **Chart Repository**: Not hosted  
❌ **Container Registry**: Not publishing  

## Phase 1: GitHub Container Registry Setup

### 1.1 Configure Docker Image Publishing
- [ ] Set up GitHub Container Registry (ghcr.io) authentication
- [ ] Configure multi-architecture builds (linux/amd64, linux/arm64)  
- [ ] Implement semantic versioning for images
- [ ] Add image signing with cosign
- [ ] Set up vulnerability scanning

**Deliverables:**
- Docker images published to `ghcr.io/seatgeek/fastly-tls-operator`
- Multi-architecture support
- Signed and scanned container images

### 1.2 Update Makefile for Registry Publishing
- [ ] Add `docker-push` target for local publishing
- [ ] Add `docker-tag` target for proper versioning
- [ ] Add `docker-multiarch` target for cross-platform builds
- [ ] Update existing targets to use configurable registry

**Deliverables:**
- Enhanced Makefile with registry publishing support
- Local development workflow for testing builds

## Phase 2: GitHub Pages Helm Repository

### 2.1 Set up GitHub Pages Repository
- [ ] Create `gh-pages` branch for chart hosting
- [ ] Configure GitHub Pages settings for the repository
- [ ] Set up chart packaging and indexing
- [ ] Create repository index.yaml
- [ ] Test chart installation from GitHub Pages

**Deliverables:**
- Live Helm repository at `https://seatgeek.github.io/fastly-tls-operator/`
- Chart installable via `helm repo add fastly-tls-operator https://seatgeek.github.io/fastly-tls-operator/`

### 2.2 Chart Packaging and Indexing
- [ ] Create `scripts/package-chart.sh` for chart packaging
- [ ] Implement chart versioning strategy
- [ ] Set up index.yaml generation
- [ ] Add chart validation before packaging
- [ ] Create cleanup for old chart versions

**Deliverables:**
- Automated chart packaging workflow
- Proper chart versioning and indexing

## Phase 3: GitHub Actions CI/CD Pipeline

### 3.1 Create Release Workflow
```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  docker:
    # Build and push Docker images
  helm:
    # Package and publish Helm chart
  release:
    # Create GitHub release with assets
```

**Tasks:**
- [ ] Create `.github/workflows/release.yml`
- [ ] Set up Docker build and push job
- [ ] Set up Helm chart packaging job
- [ ] Set up GitHub release creation
- [ ] Configure job dependencies and artifacts

**Deliverables:**
- Automated release pipeline triggered by git tags
- Docker images and Helm charts published on release

### 3.2 Create Development Workflow
```yaml
# .github/workflows/ci.yml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  test:
    # Run tests and linting
  docker:
    # Build and test Docker image
  helm:
    # Validate Helm chart
```

**Tasks:**
- [ ] Create `.github/workflows/ci.yml`
- [ ] Set up Go testing and linting
- [ ] Set up Docker build validation
- [ ] Set up Helm chart linting and testing
- [ ] Add security scanning (gosec, trivy)

**Deliverables:**
- Comprehensive CI pipeline for all pull requests
- Quality gates for code, containers, and charts

### 3.3 Configure Repository Secrets
- [ ] `GITHUB_TOKEN` (automatically provided)
- [ ] Optional: `COSIGN_KEY` for image signing
- [ ] Optional: Additional registry credentials if needed

**Deliverables:**
- Secure CI/CD pipeline with proper authentication

## Phase 4: Release Management

### 4.1 Semantic Versioning Strategy
- [ ] Define version bump rules (major.minor.patch)
- [ ] Align chart version with app version
- [ ] Set up automated changelog generation
- [ ] Create release notes templates

**Version Strategy:**
- **Docker Image**: `ghcr.io/seatgeek/fastly-tls-operator:v1.0.0`
- **Helm Chart**: `version: 1.0.0, appVersion: "1.0.0"`
- **Git Tag**: `v1.0.0`

### 4.2 Release Process
1. **Development**: Work on feature branches
2. **Testing**: CI validates all changes
3. **Tagging**: Create git tag `v1.0.0`
4. **Automation**: GitHub Actions builds and publishes
5. **Distribution**: Images on GHCR, chart on GitHub Pages

**Deliverables:**
- Clear release process documentation
- Automated changelog generation

## Phase 5: Documentation and Usage

### 5.1 Update README
- [ ] Add installation instructions using Helm repository
- [ ] Include container registry information
- [ ] Add example configurations
- [ ] Document release process for contributors

### 5.2 Create Usage Examples
- [ ] Basic installation from GitHub Pages
- [ ] Production deployment with custom values
- [ ] Upgrading between versions
- [ ] Troubleshooting common issues

**Deliverables:**
- Comprehensive documentation for end users
- Clear contribution guidelines

## Implementation Timeline

**Phase 1: Container Registry** (1-2 days)
- Set up GHCR publishing
- Configure multi-architecture builds
- Update Makefile targets

**Phase 2: Helm Repository** (1 day)
- Set up GitHub Pages
- Create chart packaging scripts
- Test chart installation

**Phase 3: CI/CD Pipeline** (2-3 days)
- Create GitHub Actions workflows
- Set up release automation
- Configure security scanning

**Phase 4: Release Management** (1 day)
- Define versioning strategy
- Create release documentation
- Test release process

**Phase 5: Documentation** (1 day)
- Update README and docs
- Create usage examples
- Finalize contributor guides

**Total Estimated Time: 6-8 days**

## Success Criteria

### Technical Success
- [ ] Docker images published to GHCR on every release
- [ ] Helm chart hosted on GitHub Pages
- [ ] Automated CI/CD pipeline working
- [ ] Multi-architecture container support
- [ ] Security scanning and signing implemented

### User Success
- [ ] Simple installation: `helm repo add fastly-tls-operator https://seatgeek.github.io/fastly-tls-operator/`
- [ ] Easy image access: `docker pull ghcr.io/seatgeek/fastly-tls-operator:latest`
- [ ] Clear documentation and examples
- [ ] Reliable release process

### Operational Success
- [ ] Automated release workflow
- [ ] Proper versioning and changelog
- [ ] Security compliance
- [ ] Maintainable CI/CD pipeline

## File Structure (Post-Implementation)

```
fastly-tls-operator/
├── .github/
│   └── workflows/
│       ├── ci.yml           # Development CI pipeline
│       └── release.yml      # Release automation
├── charts/
│   └── fastly-tls-operator/     # Existing Helm chart
├── scripts/
│   ├── package-chart.sh     # Chart packaging
│   └── build-multiarch.sh   # Multi-arch Docker builds
├── docs/
│   ├── installation.md      # Installation guide
│   └── releasing.md         # Release process
└── ...                      # Existing project files
```

## Next Steps

1. **Start with Phase 1**: Set up GitHub Container Registry publishing
2. **Phase 2**: Configure GitHub Pages for Helm repository
3. **Phase 3**: Create GitHub Actions workflows
4. **Phase 4**: Define and test release process
5. **Phase 5**: Update documentation and examples

## Commands for Users (Post-Implementation)

```bash
# Add Helm repository
helm repo add fastly-tls-operator https://seatgeek.github.io/fastly-tls-operator/
helm repo update

# Install the operator
helm install fastly-tls-operator fastly-tls-operator/fastly-tls-operator

# Pull Docker image
docker pull ghcr.io/seatgeek/fastly-tls-operator:latest

# Check available versions
helm search repo fastly-tls-operator --versions
```

This plan focuses exclusively on publishing and distribution, leveraging GitHub's built-in capabilities for hosting both container images and Helm charts. 