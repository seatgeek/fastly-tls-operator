# Fastly Operator Helm Chart Development Plan

## Overview

This document outlines the comprehensive plan for creating, testing, and publishing a Helm chart for the Fastly Operator. The chart will enable easy installation and management of the operator across different Kubernetes environments.

## Project Goals

- [ ] Create a production-ready Helm chart
- [ ] Enable local testing via kind
- [ ] Provide comprehensive configuration options
- [ ] Establish publishing workflow
- [ ] Document chart usage and configuration

## Phase 1: Chart Structure Creation

### 1.1 Initialize Helm Chart
- [x] Create `charts/fastly-operator` directory
- [x] Initialize chart with `helm create`
- [x] Clean up default template files
- [x] Set up proper directory structure

**Deliverables:**
- Basic chart structure with proper organization
- Clean `Chart.yaml` and `values.yaml` templates

### 1.2 Create Chart Directory Structure
```
charts/fastly-operator/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── crds/
│   │   └── fastlycertificatesync.yaml
│   ├── rbac/
│   │   ├── serviceaccount.yaml
│   │   ├── clusterrole.yaml
│   │   └── clusterrolebinding.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── webhook/
│   │   ├── certificate.yaml
│   │   ├── issuer.yaml
│   │   └── webhook-config.yaml
│   └── secret.yaml
└── README.md
```

### 1.3 Convert Existing Manifests
- [x] Convert `config/operator/operator.yaml` to `templates/deployment.yaml`
- [x] Convert `config/rbac/*.yaml` to `templates/rbac/`
- [x] Convert CRDs to `templates/crds/`
- [x] Convert webhook configs to `templates/webhook/`
- [x] Add proper templating and value substitution

**Deliverables:**
- All existing manifests converted to Helm templates
- Proper Go template syntax throughout

## Phase 2: Local Development and Testing

### 2.1 Create Testing Infrastructure
- [x] Create `scripts/test-helm-chart.sh` script
- [x] Add kind cluster creation logic
- [x] Include cert-manager installation
- [x] Add chart installation and validation

**Deliverables:**
- ✅ Automated testing script for local development
- ✅ Integration with existing build process

**Status:** ✅ **COMPLETE** - Script created and verified working

### 2.2 Update Makefile
- [ ] Add `helm-lint` target (local-only: runs `helm lint` on chart)
- [ ] Add `helm-template` target (local-only: runs `helm template` to render templates)
- [ ] Add `helm-test` target (local-only: basic chart validation without cluster)
- [ ] Add `helm-validate-all` target (local-only: runs helm-lint, helm-template, and helm-test)
- [ ] Add `helm-integration-test` target (cluster-based: runs the test script)
- [ ] Update `kind-deploy` to support Helm option

**Deliverables:**
- Enhanced Makefile with Helm support
- Consistent development workflow
- Local-only validation targets for fast feedback
- Cluster-based integration testing

**Notes:**
- `helm-lint`, `helm-template`, and `helm-test` are local-only and do not require kind cluster
- `helm-validate-all` combines all local validation targets for comprehensive local testing
- `helm-integration-test` uses the existing test script for full cluster testing

### 2.3 Create Validation Tests
- [ ] Implement chart linting (covered by `helm-lint` target in 2.2)
- [ ] Add template rendering tests (covered by `helm-template` target in 2.2)
- [ ] Include manifest validation (kubeval or similar tools)
- [ ] Test installation in kind cluster (covered by `helm-integration-test` target in 2.2)

**Deliverables:**
- Comprehensive test suite
- Automated validation pipeline

**Notes:**
- Most validation is now handled by Makefile targets in Phase 2.2
- This phase focuses on additional tooling (kubeval, etc.) not covered by basic Helm commands

## Phase 3: Chart Configuration and Best Practices

### 3.1 Configure Chart.yaml
- [ ] Set proper chart metadata
- [ ] Define version and appVersion
- [ ] Add keywords and descriptions
- [ ] Include maintainer information
- [ ] Define dependencies (cert-manager)

**Deliverables:**
- Complete Chart.yaml with all metadata
- Clear dependency declarations

### 3.2 Create Comprehensive values.yaml
- [ ] Image configuration options
- [ ] Operator configuration settings
- [ ] Resource limits and requests
- [ ] RBAC configuration
- [ ] Webhook settings
- [ ] Security contexts
- [ ] Dependency management

**Deliverables:**
- Production-ready values.yaml
- Comprehensive configuration options

### 3.3 Implement Security Best Practices
- [ ] Non-root security contexts
- [ ] Proper RBAC with least privilege
- [ ] Pod Security Standards support
- [ ] Network policies (optional)
- [ ] Service mesh integration (optional)

**Deliverables:**
- Security-hardened templates
- Compliance with Kubernetes security standards

## Phase 4: Publishing Strategy

### 4.1 Choose Publishing Method
Select one or more options:
- [ ] **Option A: GitHub Pages** (Recommended for open source)
  - [ ] Create `gh-pages` branch
  - [ ] Set up chart packaging
  - [ ] Configure repository index
  - [ ] Test repository access
  
- [ ] **Option B: Artifact Hub** (For broader distribution)
  - [ ] Create Artifact Hub metadata
  - [ ] Submit repository for inclusion
  - [ ] Verify listing and searchability
  
- [ ] **Option C: Private Registry** (For enterprise)
  - [ ] Configure OCI-compatible registry
  - [ ] Set up push automation
  - [ ] Test chart installation from registry

**Deliverables:**
- Working chart repository
- Automated publishing pipeline

### 4.2 Set Up CI/CD Pipeline
- [ ] Create `.github/workflows/helm-publish.yml`
- [ ] Configure automated chart packaging
- [ ] Set up release trigger (tags)
- [ ] Add chart testing in CI
- [ ] Configure security scanning

**Deliverables:**
- Automated CI/CD pipeline
- Release automation

### 4.3 Version Management
- [ ] Implement semantic versioning
- [ ] Create changelog automation
- [ ] Set up release notes generation
- [ ] Configure tag-based releases

**Deliverables:**
- Version management strategy
- Automated release process

## Phase 5: Testing and Validation

### 5.1 Comprehensive Testing Strategy
- [ ] Create `scripts/comprehensive-test.sh`
- [ ] Implement chart linting
- [ ] Add template rendering validation
- [ ] Include Kubernetes manifest validation
- [ ] Test installation scenarios
- [ ] Add upgrade/rollback testing

**Deliverables:**
- Complete testing suite
- Automated validation pipeline

### 5.2 Integration Testing
- [ ] Test with different Kubernetes versions
- [ ] Validate with multiple cert-manager versions
- [ ] Test resource scaling scenarios
- [ ] Validate webhook functionality
- [ ] Test certificate synchronization

**Deliverables:**
- Multi-environment test results
- Compatibility matrix

### 5.3 Performance Testing
- [ ] Resource usage validation
- [ ] Startup time measurement
- [ ] Memory and CPU profiling
- [ ] Scaling behavior testing

**Deliverables:**
- Performance benchmarks
- Resource optimization recommendations

## Phase 6: Documentation and Best Practices

### 6.1 Create Chart Documentation
- [ ] Write comprehensive README.md for chart
- [ ] Document all configuration options
- [ ] Include common use case examples
- [ ] Add troubleshooting guide
- [ ] Create upgrade/migration guides

**Deliverables:**
- Complete chart documentation
- User-friendly installation guide

### 6.2 Create User Examples
- [ ] Basic installation example
- [ ] Production deployment example
- [ ] Custom configuration examples
- [ ] Integration with monitoring
- [ ] Multi-environment setup

**Deliverables:**
- Example configurations
- Best practice guidelines

### 6.3 Community Preparation
- [ ] Create contribution guidelines
- [ ] Set up issue templates
- [ ] Define support channels
- [ ] Create security policy
- [ ] Add code of conduct

**Deliverables:**
- Community-ready project
- Clear contribution process

## Dependencies and Prerequisites

### External Dependencies
- [ ] Helm 3.x installed
- [ ] kind cluster for testing
- [ ] cert-manager (runtime dependency)
- [ ] kubectl configured
- [ ] Docker for image building

### Internal Dependencies
- [ ] Existing operator manifests in `config/`
- [ ] Working Dockerfile and build process
- [ ] Proper RBAC definitions
- [ ] CRD definitions
- [ ] Webhook configurations

## Success Criteria

### Technical Criteria
- [ ] Chart passes all lint checks
- [ ] All templates render correctly
- [ ] Installation works in kind cluster
- [ ] Operator functions correctly when deployed via Helm
- [ ] Chart follows Helm best practices

### User Experience Criteria
- [ ] Simple installation process
- [ ] Clear configuration options
- [ ] Comprehensive documentation
- [ ] Easy troubleshooting
- [ ] Smooth upgrade path

### Operational Criteria
- [ ] Automated testing pipeline
- [ ] Reliable publishing process
- [ ] Version management
- [ ] Security compliance
- [ ] Performance validation

## Timeline Estimates

- **Phase 1**: 2-3 days
- **Phase 2**: 1-2 days
- **Phase 3**: 2-3 days
- **Phase 4**: 1-2 days
- **Phase 5**: 2-3 days
- **Phase 6**: 1-2 days

**Total Estimated Time**: 9-15 days

## Risk Assessment

### High Risk
- [ ] Webhook certificate management in Helm
- [ ] CRD installation ordering
- [ ] Existing deployment compatibility

### Medium Risk
- [ ] Chart complexity management
- [ ] Testing environment setup
- [ ] Publishing workflow automation

### Low Risk
- [ ] Basic template conversion
- [ ] Documentation creation
- [ ] CI/CD pipeline setup

## Next Steps

1. **Start with Phase 1** - Create basic chart structure
2. **Validate locally** - Use kind for immediate testing
3. **Iterate quickly** - Focus on working deployment first
4. **Add complexity gradually** - Build up configuration options
5. **Document everything** - Keep documentation current

## Notes

- Keep existing `config/` manifests as reference
- Ensure backward compatibility with current deployment method
- Consider chart versioning strategy early
- Plan for both development and production use cases
- Include monitoring and observability considerations 