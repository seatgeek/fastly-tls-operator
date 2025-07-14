# Fastly Operator Helm Chart Development Plan

## Overview

This document outlines the comprehensive plan for creating, testing, and publishing a Helm chart for the Fastly Operator. The chart will enable easy installation and management of the operator across different Kubernetes environments.

## Current Status (Updated)

📈 **Overall Progress: ~50% Complete**

- ✅ **Phase 1: Chart Structure Creation** - **COMPLETE** (100%)
- ✅ **Phase 2: Local Development and Testing** - **MOSTLY COMPLETE** (90%)
- 🔄 **Phase 3: Chart Configuration and Best Practices** - **IN PROGRESS** (30%)
- 📋 **Phase 4: Publishing Strategy** - **PENDING** (0%)
- 📋 **Phase 5: Testing and Validation** - **PENDING** (0%)
- 📋 **Phase 6: Documentation and Best Practices** - **PENDING** (0%)

**Ready for Production Testing:** The chart can be installed and tested locally. Chart metadata is now properly configured with version 0.1.0. Focus now shifts to comprehensive values.yaml configuration and security best practices.

## Project Goals

- ✅ Create a production-ready Helm chart *(basic structure complete)*
- ✅ Enable local testing via kind *(fully implemented)*
- [ ] Provide comprehensive configuration options *(needs enhancement)*
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
- [x] Add `helm-lint` target (local-only: runs `helm lint` on chart)
- [x] Add `helm-template` target (local-only: runs `helm template` to render templates)
- [x] Add `helm-test` target (local-only: basic chart validation without cluster)
- [x] Add `helm-validate-all` target (local-only: runs helm-lint, helm-template, and helm-test)
- [x] Add `helm-integration-test` target (cluster-based: runs the test script)
- [ ] Update `kind-deploy` to support Helm option

**Deliverables:**
- ✅ Enhanced Makefile with Helm support
- ✅ Consistent development workflow
- ✅ Local-only validation targets for fast feedback
- ✅ Cluster-based integration testing

**Status:** ✅ **MOSTLY COMPLETE** - All Helm targets implemented, only `kind-deploy` Helm option remaining

**Notes:**
- `helm-lint`, `helm-template`, and `helm-test` are local-only and do not require kind cluster
- `helm-validate-all` combines all local validation targets for comprehensive local testing
- `helm-integration-test` uses the existing test script for full cluster testing

### 2.3 Create Validation Tests
- [x] Implement chart linting (covered by `helm-lint` target in 2.2)
- [x] Add template rendering tests (covered by `helm-template` target in 2.2)
- [ ] Include manifest validation (kubeval or similar tools)
- [x] Test installation in kind cluster (covered by `helm-integration-test` target in 2.2)

**Deliverables:**
- ✅ Comprehensive test suite
- ✅ Automated validation pipeline

**Status:** ✅ **MOSTLY COMPLETE** - Basic validation complete, additional tooling (kubeval) optional

**Notes:**
- Most validation is now handled by Makefile targets in Phase 2.2
- This phase focuses on additional tooling (kubeval, etc.) not covered by basic Helm commands

## Phase 3: Chart Configuration and Best Practices

### 3.1 Configure Chart.yaml
- [x] Set proper chart metadata
- [x] Define version and appVersion (0.1.0)
- [x] Add keywords and descriptions
- [x] Include maintainer information
- [x] Add home/source URLs and annotations
- [ ] ~~Define dependencies (cert-manager)~~ *Skipped - users will install cert-manager separately*

**Deliverables:**
- ✅ Complete Chart.yaml with all metadata
- ✅ Professional chart description and keywords
- ✅ Maintainer and source information
- ✅ Proper semantic versioning (0.1.0)

**Status:** ✅ **COMPLETE**

### 3.2 Create Comprehensive values.yaml

**Current Status:** Basic structure complete, needs comprehensive enhancements

#### 3.2.1 Image Configuration Options (Minor enhancements)
- [x] Basic image repository, tag, and pullPolicy *(already implemented)*
- [x] Add registry configuration support
- [x] Add digest support for immutable deployments
- [x] Basic imagePullSecrets configuration *(already implemented)*
- [~] Enhanced imagePullSecrets configuration *(skipped by choice)*

#### 3.2.2 Operator Configuration Settings (Add advanced options)
- [x] Basic operator settings (leaderElection, webhookPort, localReconciliation) *(already implemented)*
- [x] Basic probe configuration *(already implemented)*
- [x] **HIGH PRIORITY**: Add unified environment variable configuration (operator.env array)
- [x] **HIGH PRIORITY**: Add LOG_LEVEL default (INFO)
- [~] **HIGH PRIORITY**: Add metrics configuration *(skipped by choice)
- [~] **MEDIUM PRIORITY**: Add health check configuration *(skipped by choice)*
- [~] **MEDIUM PRIORITY**: Add concurrent reconciler controls *(skipped by choice)*
- [~] **LOW PRIORITY**: Add sync period configuration *(skipped by choice)*

#### 3.2.3 Resource Limits and Requests (Add granular control)
- [x] Basic resource limits and requests *(already implemented)*
- [ ] **MEDIUM PRIORITY**: Add separate webhook resource configuration
- [ ] **LOW PRIORITY**: Add resource monitoring and alerting configuration

#### 3.2.4 RBAC Configuration (Enhance with granular control)
- [x] Basic RBAC creation toggle *(already implemented)*
- [ ] **HIGH PRIORITY**: Add granular RBAC rule control
- [ ] **MEDIUM PRIORITY**: Add custom additional rules support
- [ ] **MEDIUM PRIORITY**: Add specific permission controls (secrets, certificates)

#### 3.2.5 Webhook Settings (Add advanced configuration)
- [x] Basic webhook settings (enabled, failurePolicy, timeoutSeconds) *(already implemented)*
- [x] Basic cert-manager integration *(already implemented)*
- [ ] **HIGH PRIORITY**: Add namespace and object selector support
- [ ] **MEDIUM PRIORITY**: Add admission review versions configuration
- [ ] **MEDIUM PRIORITY**: Add custom webhook rules
- [ ] **LOW PRIORITY**: Add custom issuer reference for cert-manager

#### 3.2.6 Security Contexts (Enhance security hardening)
- [x] Basic pod security context with non-root user *(already implemented)*
- [x] Basic seccomp profile *(already implemented)*
- [ ] **HIGH PRIORITY**: Add runAsGroup and fsGroup configuration
- [ ] **HIGH PRIORITY**: Add comprehensive container security context
- [ ] **MEDIUM PRIORITY**: Add SELinux support
- [ ] **MEDIUM PRIORITY**: Add capabilities management (drop ALL, selective add)
- [ ] **HIGH PRIORITY**: Add readOnlyRootFilesystem and privilege escalation controls

#### 3.2.7 Dependency Management (NEW - Critical missing piece)
- [ ] **CRITICAL**: Add cert-manager dependency configuration
- [ ] **HIGH PRIORITY**: Add cert-manager version requirements
- [ ] **HIGH PRIORITY**: Add dependency verification logic
- [ ] **MEDIUM PRIORITY**: Add webhook certificate auto-creation
- [ ] **MEDIUM PRIORITY**: Add namespace configuration for dependencies

#### 3.2.8 Additional Production-Ready Features (NEW)
- [x] **HIGH PRIORITY**: Add simplified annotations support (pod-level)
- [x] **HIGH PRIORITY**: Add simplified labels support (pod-level)
- [ ] **MEDIUM PRIORITY**: Add deployment strategy configuration
- [ ] **MEDIUM PRIORITY**: Add pod disruption budget support
- [ ] **LOW PRIORITY**: Add horizontal pod autoscaler configuration
- [ ] **LOW PRIORITY**: Add network policy support
- [ ] **LOW PRIORITY**: Add priority class and runtime class support
- [ ] **LOW PRIORITY**: Add topology spread constraints
- [ ] **LOW PRIORITY**: Add init containers and sidecar support
- [ ] **LOW PRIORITY**: Add additional volumes and environment variables

**Deliverables:**
- ✅ Production-ready values.yaml foundation *(complete)*
- 🔄 Comprehensive configuration options *(in progress)*
- 📋 Advanced security and operational features *(planned)*

**Implementation Priority:**
1. **CRITICAL**: Dependency management (cert-manager verification)
2. **HIGH**: Enhanced security contexts and RBAC
3. **HIGH**: Advanced operator configuration (logging, metrics)
4. **MEDIUM**: Advanced webhook and resource configuration
5. **LOW**: Additional production features (PDB, HPA, network policies)

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

- **Phase 1**: ✅ **COMPLETE** *(was 2-3 days)*
- **Phase 2**: ✅ **MOSTLY COMPLETE** *(was 1-2 days)*
- **Phase 3**: ✅ **30% COMPLETE** - 1-2 days remaining *(current focus)*
- **Phase 4**: 1-2 days
- **Phase 5**: 2-3 days
- **Phase 6**: 1-2 days

**Original Total Estimated Time**: 9-15 days  
**Remaining Estimated Time**: 5-9 days  
**Progress**: ~50% complete

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

## Next Steps (Updated)

1. **✅ Phase 1 & 2 Complete** - Basic chart structure and testing infrastructure ready
2. **✅ Phase 3.1 Complete** - Chart metadata properly configured with version 0.1.0
3. **🔄 Current Focus: Phase 3.2** - Enhance values.yaml with comprehensive configuration options
4. **📋 Upcoming: Production Readiness** - Security best practices and comprehensive configuration
5. **📋 Future: Publishing Strategy** - Choose and implement chart distribution method
6. **📋 Final: Documentation** - Complete user guides and best practices

### Immediate Priorities:
- **Phase 3.2**: Enhance values.yaml with comprehensive configuration options (resource limits, security contexts, etc.)
- **Phase 3.3**: Implement security best practices and validation
- **Phase 4.1**: Choose publishing method (GitHub Pages, Artifact Hub, or Private Registry)

## Notes

- Keep existing `config/` manifests as reference
- Ensure backward compatibility with current deployment method
- Consider chart versioning strategy early
- Plan for both development and production use cases
- Include monitoring and observability considerations 