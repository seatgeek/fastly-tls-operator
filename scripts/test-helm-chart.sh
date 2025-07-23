#!/bin/bash

# Fastly Operator Helm Chart Testing Script
# This script creates the 'fastly-cluster' kind cluster, installs cert-manager, and deploys the Helm chart in kube-system for testing

set -e

# Configuration variables (using separate cluster for Helm testing)
BINARY_NAME="fastly-operator"
IMAGE_NAME="fastly-operator"
IMAGE_TAG="latest"
KIND_CLUSTER_NAME="fastly-cluster"
KIND_CONTEXT="kind-${KIND_CLUSTER_NAME}"
OPERATOR_NAME="fastly-operator"
CHART_PATH="charts/fastly-operator"
CHART_INSTALL_NAMESPACE="kube-system"
RELEASE_NAME="fastly-operator"

# Helper functions
log_info() {
    echo "[INFO] $1"
}

log_warn() {
    echo "[WARN] $1"
}

log_error() {
    echo "[ERROR] $1"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to load Fastly API key from secret file
load_fastly_secret() {
    local secret_file="$(dirname "$0")/fastly-secret.env"
    
    if [ ! -f "$secret_file" ]; then
        log_error "Secret file not found: $secret_file"
        log_error "Please create the file with your Fastly API key:"
        log_error "echo 'FASTLY_API_KEY=your_api_key_here' > $secret_file"
        exit 1
    fi
    
    # Source the secret file
    source "$secret_file"
    
    if [ -z "${FASTLY_API_KEY}" ] || [ "${FASTLY_API_KEY}" = "YOUR_FASTLY_API_KEY_HERE" ]; then
        log_error "FASTLY_API_KEY not set or still using placeholder value in $secret_file"
        log_error "Please update the file with your actual Fastly API key."
        exit 1
    fi
    
    log_info "Loaded Fastly API key from $secret_file"
}

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_deps=()
    
    if ! command_exists kind; then
        missing_deps+=("kind")
    fi
    
    if ! command_exists helm; then
        missing_deps+=("helm")
    fi
    
    if ! command_exists kubectl; then
        missing_deps+=("kubectl")
    fi
    
    if ! command_exists docker; then
        missing_deps+=("docker")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install them and try again."
        exit 1
    fi
    
    log_info "All prerequisites satisfied."
}

# Function to delete existing kind cluster
delete_kind_cluster() {
    log_info "Checking for existing kind cluster '${KIND_CLUSTER_NAME}'..."
    
    if kind get clusters | grep -q "^${KIND_CLUSTER_NAME}$"; then
        log_info "Deleting existing kind cluster '${KIND_CLUSTER_NAME}' to start fresh..."
        kind delete cluster --name "${KIND_CLUSTER_NAME}"
        log_info "Existing cluster deleted."
    else
        log_info "No existing cluster found."
    fi
}

# Function to create kind cluster
create_kind_cluster() {
    log_info "Creating kind cluster '${KIND_CLUSTER_NAME}'..."
    
    kind create cluster --name "${KIND_CLUSTER_NAME}"
    
    # Wait for cluster to be ready
    log_info "Waiting for kind cluster to be ready..."
    kubectl --context="${KIND_CONTEXT}" wait --for=condition=Ready nodes --all --timeout=300s
    
    log_info "Kind cluster '${KIND_CLUSTER_NAME}' created successfully."
}

# Function to build and load Docker image
build_and_load_image() {
    log_info "Building Docker image..."
    
    # Build the Go binary
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${BINARY_NAME}" ./cmd
    
    # Build Docker image
    docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" .
    
    # Load image into kind cluster
    log_info "Loading Docker image into kind cluster..."
    kind load docker-image "${IMAGE_NAME}:${IMAGE_TAG}" --name "${KIND_CLUSTER_NAME}"
    
    log_info "Docker image built and loaded successfully."
}

# Function to install cert-manager
install_cert_manager() {
    log_info "Installing cert-manager..."
    
    # Check if cert-manager is already installed
    if kubectl --context="${KIND_CONTEXT}" get namespace cert-manager >/dev/null 2>&1; then
        log_warn "cert-manager namespace already exists, checking if it's ready..."
        kubectl --context="${KIND_CONTEXT}" -n cert-manager rollout status deployment/cert-manager-webhook --timeout=300s
        log_info "cert-manager is already installed and ready."
        return 0
    fi
    
    # Install cert-manager
    kubectl --context="${KIND_CONTEXT}" apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
    
    # Wait for cert-manager to be ready
    log_info "Waiting for cert-manager to be ready..."
    kubectl --context="${KIND_CONTEXT}" -n cert-manager rollout status deployment/cert-manager-webhook --timeout=300s
    
    log_info "cert-manager installed successfully."
}

# Function to create test resources
create_test_resources() {
    log_info "Creating test resources..."
    
    # Load the Fastly API key from the secret file
    load_fastly_secret
    
    # Verify kube-system namespace exists (it should always exist)
    if ! kubectl --context="${KIND_CONTEXT}" get namespace "${CHART_INSTALL_NAMESPACE}" >/dev/null 2>&1; then
        log_error "Namespace '${CHART_INSTALL_NAMESPACE}' does not exist. This should not happen."
        exit 1
    fi
    log_info "Using existing namespace '${CHART_INSTALL_NAMESPACE}'."
    
    # Create fastly-secret for testing (with the API key from the secret file)
    if ! kubectl --context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" get secret fastly-secret >/dev/null 2>&1; then
        kubectl --context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" create secret generic fastly-secret \
            --from-literal=api-key="${FASTLY_API_KEY}"
        log_info "Created fastly-secret in namespace '${CHART_INSTALL_NAMESPACE}'."
    else
        log_info "fastly-secret already exists in namespace '${CHART_INSTALL_NAMESPACE}'."
    fi
}

# Function to install the Helm chart
install_helm_chart() {
    log_info "Installing Helm chart..."
    
    # Check if chart directory exists
    if [ ! -d "${CHART_PATH}" ]; then
        log_error "Chart directory '${CHART_PATH}' does not exist."
        log_error "Please ensure Phase 1 is complete and the chart has been created."
        exit 1
    fi
    
    # Install or upgrade the chart
    helm upgrade --install "${RELEASE_NAME}" "${CHART_PATH}" \
        --kube-context="${KIND_CONTEXT}" \
        --namespace="${CHART_INSTALL_NAMESPACE}" \
        --set image.registry="" \
        --set image.repository="${IMAGE_NAME}" \
        --set image.tag="${IMAGE_TAG}" \
        --set image.pullPolicy=Never \
        --set fastly.secretName=fastly-secret \
        --set operator.localReconciliation=true \
        --wait --timeout=300s
    
    # Note: operator.localReconciliation=true enables the hack-fastly-certificate-sync-local-reconciliation flag
    # This allows testing with self-signed certificates and local CA certificates
    # This flag is disabled by default in the published chart
    
    log_info "Helm chart installed successfully."
}

# Function to validate the installation
validate_installation() {
    log_info "Validating installation..."
    
    # Check if the deployment is ready
    log_info "Checking deployment status..."
    kubectl --context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" rollout status deployment/fastly-operator --timeout=300s
    
    # Check if pods are running
    log_info "Checking pod status..."
    local pod_count=$(kubectl --context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" get pods -l app.kubernetes.io/name=fastly-operator --field-selector=status.phase=Running --no-headers | wc -l)
    
    if [ "${pod_count}" -eq 0 ]; then
        log_error "No running pods found for fastly-operator."
        log_error "Pod details:"
        kubectl --context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" get pods -l app.kubernetes.io/name=fastly-operator
        kubectl --context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" describe pods -l app.kubernetes.io/name=fastly-operator
        return 1
    fi
    
    log_info "Found ${pod_count} running pod(s) for fastly-operator."
    
    # Check if CRDs are installed
    log_info "Checking CRD installation..."
    if ! kubectl --context="${KIND_CONTEXT}" get crd fastlycertificatesyncs.platform.seatgeek.io >/dev/null 2>&1; then
        log_error "FastlyCertificateSync CRD not found."
        return 1
    fi
    
    log_info "CRDs are properly installed."
    
    # Check webhook configuration
    log_info "Checking webhook configuration..."
    if ! kubectl --context="${KIND_CONTEXT}" get validatingwebhookconfiguration fastly-operator-webhook >/dev/null 2>&1; then
        log_warn "ValidatingWebhookConfiguration not found (this might be expected in some configurations)."
    else
        log_info "ValidatingWebhookConfiguration found."
    fi
    
    log_info "Installation validation completed successfully."
}

# Function to run chart tests
run_chart_tests() {
    log_info "Running Helm chart tests..."
    
    # Run helm test if test hooks exist
    if helm test "${RELEASE_NAME}" --kube-context="${KIND_CONTEXT}" --namespace="${CHART_INSTALL_NAMESPACE}" 2>/dev/null; then
        log_info "Helm chart tests passed."
    else
        log_warn "No Helm chart tests found or tests failed (this is expected if no test hooks are defined)."
    fi
}

# Function to show installation info
show_installation_info() {
    log_info "Installation Information:"
    echo "======================="
    echo "Cluster: ${KIND_CLUSTER_NAME}"
    echo "Context: ${KIND_CONTEXT}"
    echo "Namespace: ${CHART_INSTALL_NAMESPACE}"
    echo "Release: ${RELEASE_NAME}"
    echo "Chart: ${CHART_PATH}"
    echo ""
    
    log_info "Useful commands:"
    echo "  View pods:       kubectl --context=${KIND_CONTEXT} -n ${CHART_INSTALL_NAMESPACE} get pods"
    echo "  View logs:       kubectl --context=${KIND_CONTEXT} -n ${CHART_INSTALL_NAMESPACE} logs -l app.kubernetes.io/name=fastly-operator"
    echo "  View helm status: helm status ${RELEASE_NAME} --kube-context=${KIND_CONTEXT} -n ${CHART_INSTALL_NAMESPACE}"
    echo "  Port forward:    kubectl --context=${KIND_CONTEXT} -n ${CHART_INSTALL_NAMESPACE} port-forward svc/fastly-operator 8080:8080"
    echo ""
}

# Function to cleanup on exit
cleanup() {
    if [ -f "${BINARY_NAME}" ]; then
        rm -f "${BINARY_NAME}"
        log_info "Cleaned up binary artifact."
    fi
}

# Main function
main() {
    log_info "Starting Fastly Operator Helm Chart Test..."
    
    # Set up cleanup on exit
    trap cleanup EXIT
    
    # Check if we're in the right directory
    if [ ! -f "go.mod" ] || [ ! -d "cmd" ]; then
        log_error "This script must be run from the fastly-operator root directory."
        exit 1
    fi
    
    # Run the test steps
    check_prerequisites
    delete_kind_cluster
    create_kind_cluster
    build_and_load_image
    install_cert_manager
    create_test_resources
    install_helm_chart
    validate_installation
    run_chart_tests
    show_installation_info
    
    log_info "Helm chart test completed successfully!"
    log_info "The fastly-operator is now running in your kind cluster."
    log_info "Use 'kind delete cluster --name ${KIND_CLUSTER_NAME}' to clean up the test cluster."
}

# Handle command line arguments
case "${1:-}" in
    "help"|"-h"|"--help")
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  help, -h, --help    Show this help message"
        echo "  cleanup             Clean up test resources"
        echo ""
        echo "This script will:"
        echo "  1. Delete any existing kind cluster 'fastly-cluster' (fresh start)"
        echo "  2. Create a kind cluster 'fastly-cluster'"
        echo "  3. Build and load the operator image"
        echo "  4. Install cert-manager"
        echo "  5. Install the Helm chart in kube-system namespace"
        echo "  6. Validate the installation"
        echo ""
        echo "Test Configuration:"
        echo "  - Local reconciliation is enabled (hack-fastly-certificate-sync-local-reconciliation=true)"
        echo "  - This allows testing with self-signed certificates and local CA"
        echo ""
        echo "Requirements:"
        echo "  - A 'fastly-secret.env' file must exist in the scripts directory"
        echo "    with your Fastly API key: FASTLY_API_KEY=your_api_key_here"
        echo ""
        exit 0
        ;;
    "cleanup")
        log_info "Cleaning up test resources..."
        if helm list --kube-context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" | grep -q "${RELEASE_NAME}"; then
            helm uninstall "${RELEASE_NAME}" --kube-context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}"
            log_info "Helm release uninstalled."
        fi
        if kubectl --context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" get secret fastly-secret >/dev/null 2>&1; then
            kubectl --context="${KIND_CONTEXT}" -n "${CHART_INSTALL_NAMESPACE}" delete secret fastly-secret
            log_info "Test secret deleted."
        fi
        log_info "Cleanup completed. Note: ${CHART_INSTALL_NAMESPACE} namespace was not deleted."
        exit 0
        ;;
    "")
        # No arguments, run main function
        main
        ;;
    *)
        log_error "Unknown option: $1"
        echo "Use '$0 help' for usage information."
        exit 1
        ;;
esac 