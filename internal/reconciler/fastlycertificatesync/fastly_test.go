package fastlycertificatesync

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/fastly/go-fastly/v11/fastly"
	"github.com/go-logr/logr"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockFastlyClient implements FastlyClientInterface for testing
type MockFastlyClient struct {
	ListPrivateKeysFunc            func(ctx context.Context, input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error)
	CreatePrivateKeyFunc           func(ctx context.Context, input *fastly.CreatePrivateKeyInput) (*fastly.PrivateKey, error)
	DeletePrivateKeyFunc           func(ctx context.Context, input *fastly.DeletePrivateKeyInput) error
	ListCustomTLSCertificatesFunc  func(ctx context.Context, input *fastly.ListCustomTLSCertificatesInput) ([]*fastly.CustomTLSCertificate, error)
	CreateCustomTLSCertificateFunc func(ctx context.Context, input *fastly.CreateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error)
	UpdateCustomTLSCertificateFunc func(ctx context.Context, input *fastly.UpdateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error)
	ListTLSActivationsFunc         func(ctx context.Context, input *fastly.ListTLSActivationsInput) ([]*fastly.TLSActivation, error)
	CreateTLSActivationFunc        func(ctx context.Context, input *fastly.CreateTLSActivationInput) (*fastly.TLSActivation, error)
	DeleteTLSActivationFunc        func(ctx context.Context, input *fastly.DeleteTLSActivationInput) error

	// Track method calls
	DeletePrivateKeyCalls    []string
	DeleteTLSActivationCalls []string
	CreateTLSActivationCalls []*fastly.CreateTLSActivationInput
}

// MockKubernetesClient implements a simple mock for the Kubernetes client Get method
type MockKubernetesClient struct {
	GetFunc func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error
}

func (m *MockKubernetesClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key, obj, opts...)
	}
	return nil
}

// MockContextClient wraps the Kubernetes client to match Context.Client structure
type MockContextClient struct {
	Client *MockKubernetesClient
}

func (m *MockFastlyClient) ListPrivateKeys(ctx context.Context, input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error) {
	if m.ListPrivateKeysFunc != nil {
		return m.ListPrivateKeysFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockFastlyClient) CreatePrivateKey(ctx context.Context, input *fastly.CreatePrivateKeyInput) (*fastly.PrivateKey, error) {
	if m.CreatePrivateKeyFunc != nil {
		return m.CreatePrivateKeyFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockFastlyClient) DeletePrivateKey(ctx context.Context, input *fastly.DeletePrivateKeyInput) error {
	// Track the call
	m.DeletePrivateKeyCalls = append(m.DeletePrivateKeyCalls, input.ID)

	if m.DeletePrivateKeyFunc != nil {
		return m.DeletePrivateKeyFunc(ctx, input)
	}
	return nil
}

func (m *MockFastlyClient) ListCustomTLSCertificates(ctx context.Context, input *fastly.ListCustomTLSCertificatesInput) ([]*fastly.CustomTLSCertificate, error) {
	if m.ListCustomTLSCertificatesFunc != nil {
		return m.ListCustomTLSCertificatesFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockFastlyClient) CreateCustomTLSCertificate(ctx context.Context, input *fastly.CreateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error) {
	if m.CreateCustomTLSCertificateFunc != nil {
		return m.CreateCustomTLSCertificateFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockFastlyClient) UpdateCustomTLSCertificate(ctx context.Context, input *fastly.UpdateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error) {
	if m.UpdateCustomTLSCertificateFunc != nil {
		return m.UpdateCustomTLSCertificateFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockFastlyClient) ListTLSActivations(ctx context.Context, input *fastly.ListTLSActivationsInput) ([]*fastly.TLSActivation, error) {
	if m.ListTLSActivationsFunc != nil {
		return m.ListTLSActivationsFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockFastlyClient) CreateTLSActivation(ctx context.Context, input *fastly.CreateTLSActivationInput) (*fastly.TLSActivation, error) {
	// Track the call
	m.CreateTLSActivationCalls = append(m.CreateTLSActivationCalls, input)

	if m.CreateTLSActivationFunc != nil {
		return m.CreateTLSActivationFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockFastlyClient) DeleteTLSActivation(ctx context.Context, input *fastly.DeleteTLSActivationInput) error {
	// Track the call
	m.DeleteTLSActivationCalls = append(m.DeleteTLSActivationCalls, input.ID)

	if m.DeleteTLSActivationFunc != nil {
		return m.DeleteTLSActivationFunc(ctx, input)
	}
	return nil
}

func TestJoinErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		expected string
	}{
		{
			name:     "empty slice",
			errors:   []error{},
			expected: "",
		},
		{
			name:     "single error",
			errors:   []error{errors.New("first error")},
			expected: "first error",
		},
		{
			name:     "multiple errors",
			errors:   []error{errors.New("first error"), errors.New("second error")},
			expected: "first error\nsecond error",
		},
		{
			name:     "nil errors in slice",
			errors:   []error{errors.New("first error"), nil, errors.New("third error")},
			expected: "first error\nthird error",
		},
		{
			name:     "all nil errors",
			errors:   []error{nil, nil, nil},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinErrors(tt.errors)

			if tt.expected == "" {
				if result != nil {
					t.Errorf("joinErrors() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("joinErrors() = nil, want %q", tt.expected)
				} else if result.Error() != tt.expected {
					t.Errorf("joinErrors() = %q, want %q", result.Error(), tt.expected)
				}
			}
		})
	}
}

func TestLogic_getFastlyUnusedPrivateKeyIDs(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  []*fastly.PrivateKey
		mockError     error
		expectedIDs   []string
		expectedError string
	}{
		{
			name: "successful call with multiple keys",
			mockResponse: []*fastly.PrivateKey{
				{ID: "key1"},
				{ID: "key2"},
				{ID: "key3"},
			},
			expectedIDs:   []string{"key1", "key2", "key3"},
			expectedError: "",
		},
		{
			name:          "successful call with no keys",
			mockResponse:  []*fastly.PrivateKey{},
			expectedIDs:   []string{},
			expectedError: "",
		},
		{
			name:          "successful call with single key",
			mockResponse:  []*fastly.PrivateKey{{ID: "single-key"}},
			expectedIDs:   []string{"single-key"},
			expectedError: "",
		},
		{
			name:          "api call fails",
			mockResponse:  nil,
			mockError:     errors.New("api error"),
			expectedIDs:   nil,
			expectedError: "failed to list Fastly private keys: api error",
		},
		{
			name:          "api call returns nil response",
			mockResponse:  nil,
			mockError:     nil,
			expectedIDs:   []string{},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockFastlyClient{
				ListPrivateKeysFunc: func(ctx context.Context, input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error) {
					// Verify the correct filter is set
					if input.FilterInUse != "false" {
						t.Errorf("Expected FilterInUse = 'false', got %q", input.FilterInUse)
					}
					return tt.mockResponse, tt.mockError
				},
			}

			// Create Logic instance with mock client
			logic := &Logic{
				FastlyClient: mockClient,
			}

			// Call the actual function from fastly.go
			result, err := logic.getFastlyUnusedPrivateKeyIDs(nil)

			// Check error
			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("getFastlyUnusedPrivateKeyIDs() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("getFastlyUnusedPrivateKeyIDs() error = nil, want %q", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("getFastlyUnusedPrivateKeyIDs() error = %q, want %q", err.Error(), tt.expectedError)
				}
			}

			// Check result
			if err == nil {
				if len(result) != len(tt.expectedIDs) {
					t.Errorf("getFastlyUnusedPrivateKeyIDs() returned %d IDs, want %d", len(result), len(tt.expectedIDs))
				}
				for i, id := range result {
					if i >= len(tt.expectedIDs) || id != tt.expectedIDs[i] {
						t.Errorf("getFastlyUnusedPrivateKeyIDs() result[%d] = %q, want %q", i, id, tt.expectedIDs[i])
					}
				}
			}
		})
	}
}

func TestLogic_clearFastlyUnusedPrivateKeys(t *testing.T) {
	tests := []struct {
		name                string
		unusedPrivateKeyIDs []string
		deleteErrors        map[string]error // Map of keyID -> error to return
		expectedDeletedKeys []string
	}{
		{
			name:                "successful deletion of multiple keys",
			unusedPrivateKeyIDs: []string{"key1", "key2", "key3"},
			deleteErrors:        map[string]error{},
			expectedDeletedKeys: []string{"key1", "key2", "key3"},
		},
		{
			name:                "no keys to delete",
			unusedPrivateKeyIDs: []string{},
			deleteErrors:        map[string]error{},
			expectedDeletedKeys: []string{},
		},
		{
			name:                "successful deletion of single key",
			unusedPrivateKeyIDs: []string{"single-key"},
			deleteErrors:        map[string]error{},
			expectedDeletedKeys: []string{"single-key"},
		},
		{
			name:                "some deletions fail - should continue",
			unusedPrivateKeyIDs: []string{"key1", "key2", "key3"},
			deleteErrors: map[string]error{
				"key1": errors.New("delete failed"),
				"key3": errors.New("another delete failed"),
			},
			expectedDeletedKeys: []string{"key1", "key2", "key3"},
		},
		{
			name:                "all deletions fail - should continue",
			unusedPrivateKeyIDs: []string{"key1", "key2"},
			deleteErrors: map[string]error{
				"key1": errors.New("delete failed"),
				"key2": errors.New("another delete failed"),
			},
			expectedDeletedKeys: []string{"key1", "key2"},
		},
		{
			name:                "mixed success and failure",
			unusedPrivateKeyIDs: []string{"success-key", "fail-key", "another-success"},
			deleteErrors: map[string]error{
				"fail-key": errors.New("this one fails"),
			},
			expectedDeletedKeys: []string{"success-key", "fail-key", "another-success"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockFastlyClient{
				DeletePrivateKeyCalls: []string{}, // Reset calls
				DeletePrivateKeyFunc: func(ctx context.Context, input *fastly.DeletePrivateKeyInput) error {
					// Return error if specified for this key
					if err, exists := tt.deleteErrors[input.ID]; exists {
						return err
					}
					return nil
				},
			}

			// Create Logic instance with mock client and observed state
			logic := &Logic{
				FastlyClient: mockClient,
				ObservedState: ObservedState{
					UnusedPrivateKeyIDs: tt.unusedPrivateKeyIDs,
				},
			}

			// Create a mock context with logger
			ctx := &Context{
				Log: logr.Discard(),
			}

			// Call the actual function from fastly.go
			logic.clearFastlyUnusedPrivateKeys(ctx)

			// Verify the correct delete calls were made
			if len(mockClient.DeletePrivateKeyCalls) != len(tt.expectedDeletedKeys) {
				t.Errorf("clearFastlyUnusedPrivateKeys() made %d delete calls, want %d",
					len(mockClient.DeletePrivateKeyCalls), len(tt.expectedDeletedKeys))
			}

			// Verify each expected call was made
			for i, expectedID := range tt.expectedDeletedKeys {
				if i >= len(mockClient.DeletePrivateKeyCalls) {
					t.Errorf("clearFastlyUnusedPrivateKeys() missing delete call %d for key %s", i, expectedID)
				} else if mockClient.DeletePrivateKeyCalls[i] != expectedID {
					t.Errorf("clearFastlyUnusedPrivateKeys() delete call %d = %s, want %s",
						i, mockClient.DeletePrivateKeyCalls[i], expectedID)
				}
			}
		})
	}
}

func TestGetPublicKeySHA1FromPEM(t *testing.T) {
	// TEST DATA EXPLANATION:
	// The following RSA private keys are real test keys generated specifically for testing purposes.
	// These are NOT production keys and were created solely for this test using `openssl genrsa 1024`.
	// They are safe to include in the codebase as they're only used for testing the SHA1 calculation logic.

	tests := []struct {
		name          string
		privateKeyPEM string
		expectedSHA1  string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_1024_bit_rsa_key_1",
			privateKeyPEM: `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDSIX1v14YXhBhoXs4xMDFaqcw0BzFGN9BUetq4xCX0RQjOgwut
EVAQg+zqSwRzW0eQsNuWQBX0qFlNQSxtE5/Bt0mr9Vh5VTePHAj+kLqAWYwzpRK/
IN8oOndsvTNJQHhHWPcnopJTIB+ktuBJpqjDVn6tHmXIj2hYA9/AQJ4BywIDAQAB
AoGAEuXcKCDT+G1y3IAaPyY8ahD3Qn6bGduPKunZneBWIX/L6Pa0KB50eufCeNfC
ULWW3BZryTl+QACb92yzGCQ5q8KZvQ5OW2SWPc7gEh2EBUFPj/SX5u4oGFRFnVFS
dv7A97OFWjRN1FVCMHGwhLD73Rq4YHZgsyGz1ZcaUtWZfeECQQDu0Zp/z4uxg4Xk
QxEUYeQmRCLSPG7b3A8Ihi1EnkXrHbVnSV+2yflz7lNLAUE5/VpHdjqhzuiYUG8G
K3N86DvpAkEA4T+INKuDyxICkUChD1ImAIPc3qhLUMgYDMPrsIjWdON0TQSpL0cQ
IpIwVHZA6QpacIV8W1r1DoF8R0kFRoTjkwJAbwtlJHLTyJmYQzfwFCMkW6qo6kqR
XYeoMdV57QMPDbEFrV4PtEWbyQ0TC7gspRMpzDqsLpqvykr0JNFFZNnzKQJASqI1
bFZERf4CscQ7WYs7okIO5gvXYL3cEia8qnK8tGBFQdvAfzTJqNrNfr7sBQt0KgJg
0RhTSGopFqmgQNx5VwJAPp9VqDDjM053vTekmu4x9eG+ItUg9fHfEJR4IcIU13DD
nqCTMVzmHe6A84rU57AR8Cd3ns2wJCdVBVXqipCW+g==
-----END RSA PRIVATE KEY-----`,
			expectedSHA1: "1ccf8849ae82aaab5749d5c791a221354f182a73",
		},
		{
			name: "valid_1024_bit_rsa_key_2",
			privateKeyPEM: `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDcohqitNHcFz6UieW++OiZ0e5m3NBbG5T1JMDehlbywuEprj/g
hcp15DVN0QRrlpYfLo8gEGPocIEBPlVhqTApOH7KJeLKypu7nf5Oa+msOym+kNY5
ttC54k4TDSQeO6iFWfPvRExPsodiH/MYdvskqUNYo1tC+OfPvnzOTSDeDQIDAQAB
AoGADIpWMztN1lGn5+9ylIk3R07sWwJgAV2u+MQPBlbiaEf1XlYeIVfZaxv+f57K
voa/n6QY1Hy6AQMsAfMWDUf9ia83KdOksEjRlk9/zcsfGCWhlAtkBWTF03GF/+qu
WbIhL35qOJoPxebEhIdPr9DMobg6QycSIW6KX8+rbBcRMe0CQQD3tkIEbC69tcTC
1ZryHBuM6Cif5TkisI9+CKLFnSKRikhns9Sj90Qw4ec4awxqf8tEfCdrbrpa5GNx
CTywYd0TAkEA5APoOKgqRqLPrU/JD35OlhV8lXbTBzmCnEBkNK2mNOG3pcd9o6yI
wTAlfb/GMOAQauVWGc2SrHV7a0MQyc9cXwJAcEL8Nk7k+/sVugreVt0gK0LHrndO
5obH8SFuy0pEcVsPJ1hbhRe5osGubWYuUVGrSFVP9CNRd4HMA11hULp5WwJAF8po
knDJaHFYZebrPZiaLoKzawzo29oeTJtTWUO9EctzU/LKoyc/ZZjWcJZv4W2fiOfA
4hRW93OSmxB2Ufg21QJAMsgwXxLJXjy0ThU7YejExp+YUntrBVrAFed3NO+gBU51
N84chfBB9g2GDYw/6drAjG7oEHDD1KOttRB5gwRzhQ==
-----END RSA PRIVATE KEY-----`,
			expectedSHA1: "a41ed6258c0928ac2e61a70dc42d20a9d4f47254",
		},
		{
			name:          "invalid_pem_data",
			privateKeyPEM: "invalid pem data",
			expectError:   true,
			errorContains: "failed to parse PEM block",
		},
		{
			name: "valid_pem_but_invalid_rsa_key",
			privateKeyPEM: `-----BEGIN RSA PRIVATE KEY-----
invalidbase64data==
-----END RSA PRIVATE KEY-----`,
			expectError:   true,
			errorContains: "failed to parse PEM block", // PEM decode fails first due to invalid base64
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getPublicKeySHA1FromPEM([]byte(tt.privateKeyPEM))

			if tt.expectError {
				if err == nil {
					t.Error("getPublicKeySHA1FromPEM() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("getPublicKeySHA1FromPEM() error = %v, want error containing %q", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("getPublicKeySHA1FromPEM() unexpected error = %v", err)
			}

			// Verify format (40-character hex string)
			if len(result) != 40 {
				t.Errorf("getPublicKeySHA1FromPEM() result length = %d, want 40", len(result))
			}

			// Verify the result is a valid hex string
			if _, parseErr := hex.DecodeString(result); parseErr != nil {
				t.Errorf("getPublicKeySHA1FromPEM() result %q is not valid hex", result)
			}

			// Log the result for manual verification
			t.Logf("✓ SHA1 for %s: %s", tt.name, result)

			// Assert the result matches the expected SHA1 value
			if tt.expectedSHA1 != "" {
				if result != tt.expectedSHA1 {
					t.Errorf("getPublicKeySHA1FromPEM() = %s, want %s", result, tt.expectedSHA1)
				}
			}
		})
	}
}

func TestLogic_deleteExtraFastlyTLSActivations(t *testing.T) {
	tests := []struct {
		name                  string
		extraTLSActivationIDs []string
		deleteErrors          map[string]error // Map of activationID -> error to return
	}{
		{
			name:                  "successful deletion of multiple activations",
			extraTLSActivationIDs: []string{"activation1", "activation2", "activation3"},
			deleteErrors:          map[string]error{},
		},
		{
			name:                  "no activations to delete",
			extraTLSActivationIDs: []string{},
			deleteErrors:          map[string]error{},
		},
		{
			name:                  "successful deletion of single activation",
			extraTLSActivationIDs: []string{"single-activation"},
			deleteErrors:          map[string]error{},
		},
		{
			name:                  "some deletions fail - should return error",
			extraTLSActivationIDs: []string{"activation1", "activation2", "activation3"},
			deleteErrors: map[string]error{
				"activation1": errors.New("delete failed"),
				"activation3": errors.New("another delete failed"),
			},
		},
		{
			name:                  "all deletions fail - should return error",
			extraTLSActivationIDs: []string{"activation1", "activation2"},
			deleteErrors: map[string]error{
				"activation1": errors.New("delete failed"),
				"activation2": errors.New("another delete failed"),
			},
		},
		{
			name:                  "mixed success and failure",
			extraTLSActivationIDs: []string{"success-activation", "fail-activation", "another-success"},
			deleteErrors: map[string]error{
				"fail-activation": errors.New("this one fails"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockFastlyClient{
				DeleteTLSActivationCalls: []string{}, // Reset calls
				DeleteTLSActivationFunc: func(ctx context.Context, input *fastly.DeleteTLSActivationInput) error {
					// Return error if specified for this activation
					if err, exists := tt.deleteErrors[input.ID]; exists {
						return err
					}
					return nil
				},
			}

			// Create Logic instance with mock client and observed state
			logic := &Logic{
				FastlyClient: mockClient,
				ObservedState: ObservedState{
					ExtraTLSActivationIDs: tt.extraTLSActivationIDs,
				},
			}

			// Create a mock context (function ignores it anyway)
			ctx := &Context{
				Log: logr.Discard(),
			}

			// Call the actual function from fastly.go
			err := logic.deleteExtraFastlyTLSActivations(ctx)

			// Check error - expect error if any delete operations should fail
			expectedError := len(tt.deleteErrors) > 0
			if expectedError {
				if err == nil {
					t.Errorf("deleteExtraFastlyTLSActivations() expected error but got nil")
				} else if !strings.Contains(err.Error(), "failed to delete TLS activations") {
					t.Errorf("deleteExtraFastlyTLSActivations() error = %v, want error containing %q", err, "failed to delete TLS activations")
				}
			} else {
				if err != nil {
					t.Errorf("deleteExtraFastlyTLSActivations() unexpected error = %v", err)
				}
			}

			// Verify the correct delete calls were made - should attempt all deletions
			if len(mockClient.DeleteTLSActivationCalls) != len(tt.extraTLSActivationIDs) {
				t.Errorf("deleteExtraFastlyTLSActivations() made %d delete calls, want %d",
					len(mockClient.DeleteTLSActivationCalls), len(tt.extraTLSActivationIDs))
			}

			// Verify each expected call was made in order
			for i, expectedID := range tt.extraTLSActivationIDs {
				if i >= len(mockClient.DeleteTLSActivationCalls) {
					t.Errorf("deleteExtraFastlyTLSActivations() missing delete call %d for activation %s", i, expectedID)
				} else if mockClient.DeleteTLSActivationCalls[i] != expectedID {
					t.Errorf("deleteExtraFastlyTLSActivations() delete call %d = %s, want %s",
						i, mockClient.DeleteTLSActivationCalls[i], expectedID)
				}
			}
		})
	}
}

func TestLogic_createMissingFastlyTLSActivations(t *testing.T) {
	tests := []struct {
		name                     string
		missingTLSActivationData []TLSActivationData
		createErrors             map[string]error // Map of configID -> error to return
	}{
		{
			name: "successful creation of multiple activations",
			missingTLSActivationData: []TLSActivationData{
				{Certificate: &fastly.CustomTLSCertificate{ID: "cert1"}, Configuration: &fastly.TLSConfiguration{ID: "config1"}, Domain: &fastly.TLSDomain{ID: "domain1"}},
				{Certificate: &fastly.CustomTLSCertificate{ID: "cert1"}, Configuration: &fastly.TLSConfiguration{ID: "config2"}, Domain: &fastly.TLSDomain{ID: "domain1"}},
			},
			createErrors: map[string]error{},
		},
		{
			name:                     "no activations to create",
			missingTLSActivationData: []TLSActivationData{},
			createErrors:             map[string]error{},
		},
		{
			name: "successful creation of single activation",
			missingTLSActivationData: []TLSActivationData{
				{Certificate: &fastly.CustomTLSCertificate{ID: "cert1"}, Configuration: &fastly.TLSConfiguration{ID: "config1"}, Domain: &fastly.TLSDomain{ID: "domain1"}},
			},
			createErrors: map[string]error{},
		},
		{
			name: "some creations fail",
			missingTLSActivationData: []TLSActivationData{
				{Certificate: &fastly.CustomTLSCertificate{ID: "cert1"}, Configuration: &fastly.TLSConfiguration{ID: "config1"}, Domain: &fastly.TLSDomain{ID: "domain1"}},
				{Certificate: &fastly.CustomTLSCertificate{ID: "cert1"}, Configuration: &fastly.TLSConfiguration{ID: "config2"}, Domain: &fastly.TLSDomain{ID: "domain1"}},
			},
			createErrors: map[string]error{
				"config1": errors.New("create failed"),
			},
		},
		{
			name: "all creations fail",
			missingTLSActivationData: []TLSActivationData{
				{Certificate: &fastly.CustomTLSCertificate{ID: "cert1"}, Configuration: &fastly.TLSConfiguration{ID: "config1"}, Domain: &fastly.TLSDomain{ID: "domain1"}},
			},
			createErrors: map[string]error{
				"config1": errors.New("create failed"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockFastlyClient{
				CreateTLSActivationCalls: []*fastly.CreateTLSActivationInput{}, // Reset calls
				CreateTLSActivationFunc: func(ctx context.Context, input *fastly.CreateTLSActivationInput) (*fastly.TLSActivation, error) {
					// Return error if specified for this configuration
					if err, exists := tt.createErrors[input.Configuration.ID]; exists {
						return nil, err
					}
					return &fastly.TLSActivation{ID: "new-activation"}, nil
				},
			}

			// Create Logic instance with mock client and observed state
			logic := &Logic{
				FastlyClient: mockClient,
				ObservedState: ObservedState{
					MissingTLSActivationData: tt.missingTLSActivationData,
				},
			}

			// Create a mock context (function ignores it anyway)
			ctx := &Context{
				Log: logr.Discard(),
			}

			// Call the actual function from fastly.go
			err := logic.createMissingFastlyTLSActivations(ctx)

			// Check error - expect error if any create operations should fail
			expectedError := len(tt.createErrors) > 0
			if expectedError {
				if err == nil {
					t.Errorf("createMissingFastlyTLSActivations() expected error but got nil")
				} else if !strings.Contains(err.Error(), "failed to create TLS activations") {
					t.Errorf("createMissingFastlyTLSActivations() error = %v, want error containing %q", err, "failed to create TLS activations")
				}
			} else {
				if err != nil {
					t.Errorf("createMissingFastlyTLSActivations() unexpected error = %v", err)
				}
			}

			// Verify the correct create calls were made - should attempt all creations
			if len(mockClient.CreateTLSActivationCalls) != len(tt.missingTLSActivationData) {
				t.Errorf("createMissingFastlyTLSActivations() made %d create calls, want %d",
					len(mockClient.CreateTLSActivationCalls), len(tt.missingTLSActivationData))
			}

			// Verify each expected call was made in order with correct data
			for i, expectedData := range tt.missingTLSActivationData {
				if i >= len(mockClient.CreateTLSActivationCalls) {
					t.Errorf("createMissingFastlyTLSActivations() missing create call %d", i)
					continue
				}

				actualCall := mockClient.CreateTLSActivationCalls[i]
				if actualCall.Certificate.ID != expectedData.Certificate.ID {
					t.Errorf("createMissingFastlyTLSActivations() call %d certificate ID = %s, want %s",
						i, actualCall.Certificate.ID, expectedData.Certificate.ID)
				}
				if actualCall.Configuration.ID != expectedData.Configuration.ID {
					t.Errorf("createMissingFastlyTLSActivations() call %d configuration ID = %s, want %s",
						i, actualCall.Configuration.ID, expectedData.Configuration.ID)
				}
				if actualCall.Domain.ID != expectedData.Domain.ID {
					t.Errorf("createMissingFastlyTLSActivations() call %d domain ID = %s, want %s",
						i, actualCall.Domain.ID, expectedData.Domain.ID)
				}
			}
		})
	}
}

func TestLogic_getFastlyPrivateKeyExists(t *testing.T) {
	testPrivateKeyPEM := `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDSIX1v14YXhBhoXs4xMDFaqcw0BzFGN9BUetq4xCX0RQjOgwut
EVAQg+zqSwRzW0eQsNuWQBX0qFlNQSxtE5/Bt0mr9Vh5VTePHAj+kLqAWYwzpRK/
IN8oOndsvTNJQHhHWPcnopJTIB+ktuBJpqjDVn6tHmXIj2hYA9/AQJ4BywIDAQAB
AoGAEuXcKCDT+G1y3IAaPyY8ahD3Qn6bGduPKunZneBWIX/L6Pa0KB50eufCeNfC
ULWW3BZryTl+QACb92yzGCQ5q8KZvQ5OW2SWPc7gEh2EBUFPj/SX5u4oGFRFnVFS
dv7A97OFWjRN1FVCMHGwhLD73Rq4YHZgsyGz1ZcaUtWZfeECQQDu0Zp/z4uxg4Xk
QxEUYeQmRCLSPG7b3A8Ihi1EnkXrHbVnSV+2yflz7lNLAUE5/VpHdjqhzuiYUG8G
K3N86DvpAkEA4T+INKuDyxICkUChD1ImAIPc3qhLUMgYDMPrsIjWdON0TQSpL0cQ
IpIwVHZA6QpacIV8W1r1DoF8R0kFRoTjkwJAbwtlJHLTyJmYQzfwFCMkW6qo6kqR
XYeoMdV57QMPDbEFrV4PtEWbyQ0TC7gspRMpzDqsLpqvykr0JNFFZNnzKQJASqI1
bFZERf4CscQ7WYs7okIO5gvXYL3cEia8qnK8tGBFQdvAfzTJqNrNfr7sBQt0KgJg
0RhTSGopFqmgQNx5VwJAPp9VqDDjM053vTekmu4x9eG+ItUg9fHfEJR4IcIU13DD
nqCTMVzmHe6A84rU57AR8Cd3ns2wJCdVBVXqipCW+g==
-----END RSA PRIVATE KEY-----`

	expectedSHA1 := "1ccf8849ae82aaab5749d5c791a221354f182a73"

	tests := []struct {
		name                 string
		setupObjects         []client.Object // K8s objects to create in fake client
		mockKeys             []*fastly.PrivateKey
		mockKeyPages         [][]*fastly.PrivateKey // Support for pagination testing
		fastlyAPIError       error
		expectedExists       bool
		expectedError        string
		expectFastlyAPICall  bool
		expectedPageRequests int // Number of page requests expected
	}{
		{
			name: "key exists in fastly - single page",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte("test-cert-data"),
					},
				},
			},
			mockKeys: []*fastly.PrivateKey{
				{ID: "key1", PublicKeySHA1: "different_sha1"},
				{ID: "key2", PublicKeySHA1: expectedSHA1}, // This matches
				{ID: "key3", PublicKeySHA1: "another_sha1"},
			},
			expectedExists:       true,
			expectFastlyAPICall:  true,
			expectedPageRequests: 1,
		},
		{
			name: "key does not exist in fastly",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte("test-cert-data"),
					},
				},
			},
			mockKeys: []*fastly.PrivateKey{
				{ID: "key1", PublicKeySHA1: "different_sha1"},
				{ID: "key2", PublicKeySHA1: "another_sha1"},
			},
			expectedExists:       false,
			expectFastlyAPICall:  true,
			expectedPageRequests: 1,
		},
		{
			name: "key exists in fastly - multiple pages",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte("test-cert-data"),
					},
				},
			},
			// Use mockKeyPages to simulate pagination
			mockKeyPages: [][]*fastly.PrivateKey{
				// Page 1 - full page (20 keys)
				func() []*fastly.PrivateKey {
					keys := make([]*fastly.PrivateKey, defaultFastlyPageSize)
					for i := 0; i < defaultFastlyPageSize; i++ {
						keys[i] = &fastly.PrivateKey{ID: fmt.Sprintf("key%d", i), PublicKeySHA1: fmt.Sprintf("sha1_%d", i)}
					}
					return keys
				}(),
				// Page 2 - partial page with matching key
				{
					{ID: "key21", PublicKeySHA1: expectedSHA1}, // This matches
					{ID: "key22", PublicKeySHA1: "another_sha1"},
				},
			},
			expectedExists:       true,
			expectFastlyAPICall:  true,
			expectedPageRequests: 2,
		},
		{
			name: "no keys in fastly",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte("test-cert-data"),
					},
				},
			},
			mockKeys:             []*fastly.PrivateKey{},
			expectedExists:       false,
			expectFastlyAPICall:  true,
			expectedPageRequests: 1,
		},
		{
			name:                "certificate not found",
			setupObjects:        []client.Object{}, // No objects - certificate missing
			expectedError:       "failed to get TLS secret from context",
			expectFastlyAPICall: false,
		},
		{
			name: "secret not found",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret", // This secret doesn't exist
					},
				},
				// No secret object
			},
			expectedError:       "failed to get TLS secret from context",
			expectFastlyAPICall: false,
		},
		{
			name: "secret missing tls.key",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.crt": []byte("test-cert-data"),
						// Note: tls.key is missing
					},
				},
			},
			expectedError:       "secret test-namespace/test-secret does not contain tls.key",
			expectFastlyAPICall: false,
		},
		{
			name: "invalid private key PEM",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte("invalid-pem-data"),
						"tls.crt": []byte("test-cert-data"),
					},
				},
			},
			expectedError:       "failed to get public key SHA1",
			expectFastlyAPICall: false,
		},
		{
			name: "fastly api error",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte("test-cert-data"),
					},
				},
			},
			fastlyAPIError:       errors.New("fastly connection failed"),
			expectedError:        "failed to list Fastly private keys",
			expectFastlyAPICall:  true,
			expectedPageRequests: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track API calls
			var actualPageRequests int

			// Create mock Fastly client
			mockFastlyClient := &MockFastlyClient{
				ListPrivateKeysFunc: func(ctx context.Context, input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error) {
					actualPageRequests++

					if tt.fastlyAPIError != nil {
						return nil, tt.fastlyAPIError
					}

					// Handle pagination testing with mockKeyPages
					if len(tt.mockKeyPages) > 0 {
						pageIndex := input.PageNumber - 1 // Convert to 0-based index
						if pageIndex < len(tt.mockKeyPages) {
							return tt.mockKeyPages[pageIndex], nil
						}
						return []*fastly.PrivateKey{}, nil // Empty page for out-of-range requests
					}

					// Single page response for simple cases
					if input.PageNumber == 1 {
						return tt.mockKeys, nil
					}
					return []*fastly.PrivateKey{}, nil // Empty subsequent pages
				},
			}

			// Create fake k8s client with test objects
			scheme := runtime.NewScheme()
			_ = cmv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.setupObjects...).
				Build()

			// Create Logic instance
			logic := &Logic{
				FastlyClient: mockFastlyClient,
			}

			// Create test context with fake K8s client
			ctx := createTestContext()
			ctx.Client = &k8sutil.ContextClient{
				SchemedClient: k8sutil.SchemedClient{
					Client: fakeClient,
				},
				Context:   context.Background(),
				Namespace: "test-namespace",
			}

			// Call the actual function
			result, err := logic.getFastlyPrivateKeyExists(ctx)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("getFastlyPrivateKeyExists() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("getFastlyPrivateKeyExists() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return // Don't check result if we expected an error
			}

			if err != nil {
				t.Errorf("getFastlyPrivateKeyExists() unexpected error = %v", err)
				return
			}

			// Check result
			if result != tt.expectedExists {
				t.Errorf("getFastlyPrivateKeyExists() = %v, want %v", result, tt.expectedExists)
			}

			// Check API call expectations
			if tt.expectFastlyAPICall {
				if actualPageRequests == 0 {
					t.Error("getFastlyPrivateKeyExists() expected Fastly API to be called, but it wasn't")
				} else if tt.expectedPageRequests > 0 && actualPageRequests != tt.expectedPageRequests {
					t.Errorf("getFastlyPrivateKeyExists() made %d page requests, want %d", actualPageRequests, tt.expectedPageRequests)
				}
			} else {
				if actualPageRequests > 0 {
					t.Errorf("getFastlyPrivateKeyExists() expected Fastly API NOT to be called, but it was called %d times", actualPageRequests)
				}
			}
		})
	}
}

func TestLogic_createFastlyPrivateKey(t *testing.T) {
	tests := []struct {
		name                       string
		setupObjects               []client.Object // K8s objects to create in fake client
		fastlyAPIShouldNotBeCalled bool            // If true, fail test if API is called
		fastlyAPIError             string          // If set, return this error from API
		expectedError              string
		expectFastlyClientCall     bool
		expectedFastlyInput        *fastly.CreatePrivateKeyInput
	}{
		{
			name: "successful private key creation",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte("-----BEGIN RSA PRIVATE KEY-----\ntest-key-data\n-----END RSA PRIVATE KEY-----"),
						"tls.crt": []byte("test-cert-data"),
					},
				},
			},
			expectFastlyClientCall: true,
			expectedFastlyInput: &fastly.CreatePrivateKeyInput{
				Key:  "-----BEGIN RSA PRIVATE KEY-----\ntest-key-data\n-----END RSA PRIVATE KEY-----",
				Name: "test-secret",
			},
		},
		{
			name:                       "certificate not found",
			setupObjects:               []client.Object{}, // No objects - certificate missing
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get TLS secret from context",
			expectFastlyClientCall:     false,
		},
		{
			name: "secret not found",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret", // This secret doesn't exist
					},
				},
				// No secret object
			},
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get TLS secret from context",
			expectFastlyClientCall:     false,
		},
		{
			name: "secret missing tls.key",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.crt": []byte("test-cert-data"),
						// Note: tls.key is missing
					},
				},
			},
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "secret test-namespace/test-secret does not contain tls.key",
			expectFastlyClientCall:     false,
		},
		{
			name: "fastly api error",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte("-----BEGIN RSA PRIVATE KEY-----\ntest-key-data\n-----END RSA PRIVATE KEY-----"),
						"tls.crt": []byte("test-cert-data"),
					},
				},
			},
			fastlyAPIError:         "fastly api connection failed",
			expectedError:          "failed to create Fastly private key: fastly api connection failed",
			expectFastlyClientCall: true,
		},
	}

	// Helper function to create mock Fastly client based on raw parameters
	setupFastlyClient := func(t *testing.T, shouldNotBeCalled bool, apiError string) *MockFastlyClient {
		return &MockFastlyClient{
			CreatePrivateKeyFunc: func(ctx context.Context, input *fastly.CreatePrivateKeyInput) (*fastly.PrivateKey, error) {
				if shouldNotBeCalled {
					t.Error("CreatePrivateKey should not be called in this test case")
					return nil, nil
				}

				if apiError != "" {
					return nil, errors.New(apiError)
				}

				// Success case
				return &fastly.PrivateKey{ID: "new-key-123"}, nil
			},
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Fastly client mock with call tracking
			var actualFastlyInput *fastly.CreatePrivateKeyInput
			mockFastlyClient := setupFastlyClient(t, tt.fastlyAPIShouldNotBeCalled, tt.fastlyAPIError)

			// Wrap the original function to capture input
			originalFunc := mockFastlyClient.CreatePrivateKeyFunc
			mockFastlyClient.CreatePrivateKeyFunc = func(ctx context.Context, input *fastly.CreatePrivateKeyInput) (*fastly.PrivateKey, error) {
				actualFastlyInput = input
				return originalFunc(ctx, input)
			}

			// Create fake k8s client with test objects
			scheme := runtime.NewScheme()
			_ = cmv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.setupObjects...).
				Build()

			// Create Logic instance
			logic := &Logic{
				FastlyClient: mockFastlyClient,
			}

			// Create test context with fake K8s client
			ctx := createTestContext()
			ctx.Client = &k8sutil.ContextClient{
				SchemedClient: k8sutil.SchemedClient{
					Client: fakeClient,
				},
				Context:   context.Background(),
				Namespace: "test-namespace",
			}

			// Call the function
			err := logic.createFastlyPrivateKey(ctx)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("createFastlyPrivateKey() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("createFastlyPrivateKey() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("createFastlyPrivateKey() unexpected error = %v", err)
				}
			}

			// Check if Fastly client was called as expected
			if tt.expectFastlyClientCall {
				if actualFastlyInput == nil {
					t.Error("createFastlyPrivateKey() expected Fastly CreatePrivateKey to be called, but it wasn't")
				} else if tt.expectedFastlyInput != nil {
					// Verify the input to CreatePrivateKey
					if actualFastlyInput.Key != tt.expectedFastlyInput.Key {
						t.Errorf("createFastlyPrivateKey() Fastly input Key = %q, want %q", actualFastlyInput.Key, tt.expectedFastlyInput.Key)
					}
					if actualFastlyInput.Name != tt.expectedFastlyInput.Name {
						t.Errorf("createFastlyPrivateKey() Fastly input Name = %q, want %q", actualFastlyInput.Name, tt.expectedFastlyInput.Name)
					}
				}
			} else {
				if actualFastlyInput != nil {
					t.Error("createFastlyPrivateKey() expected Fastly CreatePrivateKey NOT to be called, but it was")
				}
			}
		})
	}
}

func TestLogic_getFastlyCertificateStatus(t *testing.T) {
	// Valid test certificate with serial number 46069880556468363886837689903758279604279389965
	testCertPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUCBHYStfMkkndTFGA8Ii8cwjbLw0wDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJdGVzdC1jZXJ0MB4XDTI1MDcyNTE2MDczMVoXDTI2MDcy
NTE2MDczMVowFDESMBAGA1UEAwwJdGVzdC1jZXJ0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEA0LU+3xmSv1m/8HQCDbX2vh8IFt81ihXqRPsCVaUFKooe
wQ51qfcf8QTrrFO9zTa4/jE5XCnc6mcaFyus5mGB5KaicTJFqWEnSMZ6EaTeriyw
igKmX21Fl7QmxHmEUPZ2SBgg3+3KZ6x6ZuBbZBEBqBmWwCgKLKYvL9VX2qPyxFbO
FQ/jTXEtycTL0CjvKrcVzSBudHfkK+p3YonfuhjyECon6RQYj8/JcKumywvOi4xS
Ly2632ylNXk3lnUR81VZMzpqACav/hZ8xfHCJSLGyvq+ie+g7UTmeRDay+WpXSaL
+pONsYmTlDkusINgN9gQHjvIpHtVYgMSO4nILVGgywIDAQABo1MwUTAdBgNVHQ4E
FgQUMsHrc5vZ2ZM+2hrlYKejCv+rzegwHwYDVR0jBBgwFoAUMsHrc5vZ2ZM+2hrl
YKejCv+rzegwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAfzvb
D7WypfPJdolfjWqJ1uTMt+3ZwsekHdYnrb6V45wHF0CZz3yjVRQ5X58GwlziuRPJ
7fJbs1IrhbyrfvFYAtti7AbEkMmir4IvXD7ptmsq4BM5R6T5b4sNVCs1wB+6Zt64
6naCo/pA/uFaYvXnKc1ehcXjqBcAbcrUw30QZQyxw+P3Bj1PrAJdJl+ziChS1Rwv
29Ufb2qfN/isbgNb6JHoErzKosMtPqXZK10QQXYEnoL9xmXPVSxufGGtdE18Q0c+
0vadtjmK1NtoNKvuWDjpKcZRsRUm/dzgDL/0Jq6EiKrz/fAAvwnFgyNzOfjJvkDh
uDRcMus2gDK/pMedtQ==
-----END CERTIFICATE-----`)

	tests := []struct {
		name                   string
		setupObjects           []client.Object
		mockFastlyCertificates []*fastly.CustomTLSCertificate
		fastlyAPIError         error
		expectedStatus         CertificateStatus
		expectedError          string
	}{
		{
			name: "certificate missing in fastly",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
			},
			mockFastlyCertificates: []*fastly.CustomTLSCertificate{}, // No certificates
			expectedStatus:         CertificateStatusMissing,
		},
		{
			name: "certificate exists and is stale",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte("-----BEGIN RSA PRIVATE KEY-----\ntest-key-data\n-----END RSA PRIVATE KEY-----"),
						"tls.crt": testCertPEM,
					},
				},
			},
			mockFastlyCertificates: []*fastly.CustomTLSCertificate{
				{
					ID:           "cert-123",
					Name:         "test-certificate",
					SerialNumber: "different-serial-number", // Different from certificate serial
				},
			},
			expectedStatus: CertificateStatusStale,
		},
		{
			name: "certificate exists and is not stale",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte("-----BEGIN RSA PRIVATE KEY-----\ntest-key-data\n-----END RSA PRIVATE KEY-----"),
						"tls.crt": testCertPEM,
					},
				},
			},
			mockFastlyCertificates: []*fastly.CustomTLSCertificate{
				{
					ID:           "cert-123",
					Name:         "test-certificate",
					SerialNumber: "46069880556468363886837689903758279604279389965", // Matching serial from testCertPEM
				},
			},
			expectedStatus: CertificateStatusSynced,
		},
		{
			name: "certificate exists but name doesn't match",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
			},
			mockFastlyCertificates: []*fastly.CustomTLSCertificate{
				{
					ID:           "cert-123",
					Name:         "different-certificate", // Different name
					SerialNumber: "some-serial",
				},
			},
			expectedStatus: CertificateStatusMissing, // Should be treated as missing
		},
		{
			name: "multiple certificates, match found",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte("-----BEGIN RSA PRIVATE KEY-----\ntest-key-data\n-----END RSA PRIVATE KEY-----"),
						"tls.crt": testCertPEM,
					},
				},
			},
			mockFastlyCertificates: []*fastly.CustomTLSCertificate{
				{
					ID:           "cert-111",
					Name:         "other-certificate",
					SerialNumber: "other-serial",
				},
				{
					ID:           "cert-123",
					Name:         "test-certificate",                                // This matches
					SerialNumber: "46069880556468363886837689903758279604279389965", // Matching serial from testCertPEM
				},
				{
					ID:           "cert-456",
					Name:         "another-certificate",
					SerialNumber: "another-serial",
				},
			},
			expectedStatus: CertificateStatusSynced,
		},
		{
			name:          "error getting certificate from k8s",
			setupObjects:  []client.Object{}, // No certificate object
			expectedError: "failed to get Fastly certificate matching subject",
		},
		{
			name: "error from fastly api",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
			},
			fastlyAPIError: errors.New("fastly api error"),
			expectedError:  "failed to get Fastly certificate matching subject",
		},
		{
			name: "error from isFastlyCertificateStale due to invalid cert PEM",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte("-----BEGIN RSA PRIVATE KEY-----\ntest-key-data\n-----END RSA PRIVATE KEY-----"),
						"tls.crt": []byte("invalid-cert-data"), // Invalid PEM data
					},
				},
			},
			mockFastlyCertificates: []*fastly.CustomTLSCertificate{
				{
					ID:           "cert-123",
					Name:         "test-certificate",
					SerialNumber: "some-serial",
				},
			},
			expectedError: "failed to check if certificate is stale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Fastly client
			mockFastlyClient := &MockFastlyClient{
				ListCustomTLSCertificatesFunc: func(ctx context.Context, input *fastly.ListCustomTLSCertificatesInput) ([]*fastly.CustomTLSCertificate, error) {
					if tt.fastlyAPIError != nil {
						return nil, tt.fastlyAPIError
					}

					// Simple single page response for testing
					if input.PageNumber == 1 {
						return tt.mockFastlyCertificates, nil
					}
					return []*fastly.CustomTLSCertificate{}, nil // Empty subsequent pages
				},
			}

			// Create fake k8s client with test objects
			scheme := runtime.NewScheme()
			_ = cmv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.setupObjects...).
				Build()

			// Create Logic instance
			logic := &Logic{
				FastlyClient: mockFastlyClient,
			}

			// Create test context with fake K8s client
			ctx := createTestContext()
			ctx.Client = &k8sutil.ContextClient{
				SchemedClient: k8sutil.SchemedClient{
					Client: fakeClient,
				},
				Context:   context.Background(),
				Namespace: "test-namespace",
			}

			// Call the function under test
			result, err := logic.getFastlyCertificateStatus(ctx)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("getFastlyCertificateStatus() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("getFastlyCertificateStatus() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return // Don't check result if we expected an error
			}

			if err != nil {
				t.Errorf("getFastlyCertificateStatus() unexpected error = %v", err)
				return
			}

			// Check result
			if result != tt.expectedStatus {
				t.Errorf("getFastlyCertificateStatus() = %v, want %v", result, tt.expectedStatus)
			}
		})
	}
}

func TestLogic_isFastlyCertificateStale(t *testing.T) {
	// Test certificates generated with OpenSSL
	testCert1PEM := `-----BEGIN CERTIFICATE-----
MIIDGTCCAgGgAwIBAgIUMjmrsWIuKwx7IY91T+uJbt9W/JwwDQYJKoZIhvcNAQEL
BQAwHDEaMBgGA1UEAwwRdGVzdDEuZXhhbXBsZS5jb20wHhcNMjUwNzI1MTgwMDE1
WhcNMjYwNzI1MTgwMDE1WjAcMRowGAYDVQQDDBF0ZXN0MS5leGFtcGxlLmNvbTCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMH46nyVilIPr60Y5a6eRq0f
AToieLr1TOc5PNdXRTsksKKkwi3FN78+Kaq2InI81qZT+3EXL+tQZxajH7wT2XCW
WoC5vACneZgEqEw7DShN0i981Q1frph0VWnfeuMFyeF1k6NRFttunxpuMw/hUVD1
vw7bAQVvPs1v5b7z9i4eagNDPxN+NRc/Ha+izG2lR+KIWppddw2SxNPqDqE8Pp/X
ghx+tXV6YBKAUKHKswCNo/ei+PfTP5zaBDujh5bpW7zEcEbO1MqhaK0udRoDN/ga
1tLic3qgwQkWuM5GaR0gD5akwxQn7xTq3QJnS4eZlXUvOuREaAvr61Hzsd5oXNUC
AwEAAaNTMFEwHQYDVR0OBBYEFKBcc0a/E7HxgA0EKYxFpvhVIEywMB8GA1UdIwQY
MBaAFKBcc0a/E7HxgA0EKYxFpvhVIEywMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZI
hvcNAQELBQADggEBAGu6NdXiEeEYOS85Z4SHnz/tJ0k/1sVLlv6DGBTdntr6BEHZ
bRs90sI5Xp/z+mGV16wAT6svezAZxSbrToVyydxMiXQhzbujR0068IrrQVSj3mi8
UNMYzx88f/jClo2WuYP7A74GYR36RVBbxD08O8A+jTArZwiv0iN81+NTmeg4ZQBy
XSDMDmZZx+ojBtchD2hX/cxaajUHf4udNqNgB/rHYaHFt3GZ8rnZvuahVIMQBBMz
mS5eN9puVa/irPcDL4wouoE3YalPzD0f4kJptJXb4t8ztIe03UIeKQ8sJMqBDykx
69hz1v4rzQFhf0TH/DK8wScx7zpirNuXS5dV2ck=
-----END CERTIFICATE-----`

	testCert2PEM := `-----BEGIN CERTIFICATE-----
MIIDGTCCAgGgAwIBAgIUPeH3t55Pw/sWqlpNyLINBAkB9VYwDQYJKoZIhvcNAQEL
BQAwHDEaMBgGA1UEAwwRdGVzdDIuZXhhbXBsZS5jb20wHhcNMjUwNzI1MTgwMDMy
WhcNMjYwNzI1MTgwMDMyWjAcMRowGAYDVQQDDBF0ZXN0Mi5leGFtcGxlLmNvbTCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALPjE7067AzjlvV2MRSszo+L
HeF5/UGaPdqBsyx9gTSWNeCJkbR1pQkjdZ5y4qFCIDLE27yWMjSb/+Ffvi1cuRea
HBjcBMVdZwzGtIaZEOmnYnPfk7kQUvBwNYHd32ako7L2TdQmsTN0NSlkxkT4aYfT
TepnsK4QDYv068w9a3oS6CPbBxBpa+3z6DBNY0Im9SKbems3uSWyVH3PAfFbf6ct
vXhLmKCyWg0urwnY57Py294pfDEsudQ3LC5m/JOwq7Wjzpc25MLK1Iiv7ujcqOrv
pCvzt2eimX3tWZFSAFPykRPTabXzhkpqlLaNQYznFn52eQB3JrPhKsK2bhRLHUsC
AwEAAaNTMFEwHQYDVR0OBBYEFDtob+IoH7z5+8JwcZ6ZOHie1/7DMB8GA1UdIwQY
MBaAFDtob+IoH7z5+8JwcZ6ZOHie1/7DMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZI
hvcNAQELBQADggEBAAZfy3HfudwGxkolP5KZAW7RNF6O56+4+Rgrbayms+iPpeof
tvfO1uC960ZexE2nUOUZ/gnZhPmyX9DMQVOiWeK982iz+qMacvC1XdcTlXERl4Q9
Q07o/baloPsxGxAdRM/3n6gi+lIBxK8YkD1R/B/J2fpIIfLKn49ejdffXHBEH3Jr
KkK+pApnwKCyKNGkbvG+iArNmowO5XzYbKsdMP9t/RKbLmqOndkLlyEuDlnRDR46
zY2hMuGsVROLFABoev8oxubZXekjgvJ305zqfQHZ1ae+6bxQD5cOa2NNttsmBkFH
ij7fULsnBQV7F7D++VuXZtfHz/P5xsHC20wjuY8=
-----END CERTIFICATE-----`

	// Serial numbers extracted from the test certificates using OpenSSL
	testCert1SerialDecimal := "286735637578885560113881828566553888840545139868"
	testCert2SerialDecimal := "353287683906654082246724919568538276704972567894"

	testPrivateKeyPEM := `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDB+Op8lYpSD6+t
GOWunkatHwE6Ini69UznOTzXV0U7JLCipMItxTe/PimqtiJyPNamU/txFy/rUGcW
ox+8E9lwllqAubwAp3mYBKhMOw0oTdIvfNUNX66YdFVp33rjBcnhdZOjURbbbp8a
bjMP4VFQ9b8O2wEFbz7Nb+W+8/YuHmoDQz8TfjUXPx2vosxtpUfiiFqaXXcNksTT
6g6hPD6f14IcfrV1emASgFChyrMAjaP3ovj30z+c2gQ7o4eW6Vu8xHBGztTKoWit
LnUaAzf4GtbS4nN6oMEJFrjORmkdIA+WpMMUJ+8U6t0CZ0uHmZV1LzrkRGgL6+tR
87HeaFzVAgMBAAECggEACoU4HpKzwlikdBR2HJ7sAWa9l8wX1TgJuD7D/H/4usj2
JZQfDgF00RQLiX1AaAbBs5q5d/xEdpRRSwqE5ZwD/pXBCdtJBZYPw7147U2EnKt/
22B/Y67NVup0WX4r+ZUuSmKoo8J5pWQeD/5rGZDkaqoxdxpMt2E9vEG16cHsl7qp
NawklW7iRj2mf+RDCalRXPAMV+fWAEJBcXFnsTjYkcH1Na4lmN8nhvG3Una/+W5O
eRF670cd+RMPaU7SZMcqX+7iKZAlk/iKZSRfN80WFFF8+rc0eyHSrhelm6dQEDI6
auYIXVHi+G6KBYpF41WU0jwscOL3oSGP7XTxsEAiHwKBgQD74vmu99cRwr+OrdRe
TushBAvSlctmnd/Sskz4WkEUS3ZsIAFPJbb8W6GDQkE90gzQF2yC20lzBjWDnxr+
QtVO731bOElCgrJSttjIPXQ691Xr/6mrqfpCRNV+lYFvhGDdURuZAC0NK6/x4XbK
Wzoa4qD96o8rV6/35INqWo5I4wKBgQDFI9O3raR+iUJlKNvwcsAgupcWQ26sI8Ud
3rd5HhH5oryOLhY+vAd+NgrlalXANwR9WeVv5iiNYDphQXINHtByarvbqNdgplht
PS851Ljp0PEwYhAcPF9aAXxFQWG2wps2p+Jzscv5w2HZcRP69CmYeX4XvGyjb3nZ
WrjnB0SI5wKBgFP6lrhJFUFspqURO469zRLS4IYzPv9Vf3wlyhe7L5tulWrzOLyE
nH+CpVS30DymPXNbe+gc6F4bIdhiQYOoEKoimq7BE1vDa2S8ZYZNRuUp9VGbiZwc
Lb3OaUes3NyrTAg9tG/MaTjM6fpA63QH+lVgXcCKZXVT5O1HGLFqw6l1AoGAI3Gw
lBqlM5bsGBIDkTSgdIH3vin7kPmRbDBp3l3Yr4Bh1FJW74qQ8lE3Hk5DAp8hsIPk
K30/F0QQ2wGQRumeYqPsCK9PofHmfiV9AzHK2UcWxjMrYFg+cIlJ1Y3OyrQsgeQn
Y9O4r7xAMH8TL5CMlfxp/kyDX9MgHkMgcXEuEksCgYAp44cvIJ32ZXnkhqOGqKqx
ROnKgP+IERYFaGuZZjbLszYGZim/XEltYcprTKFCMbM8t2ACnosaLsiryqEPVPae
Yv2WDpgiXITjqQ7QNOSl31sWtvreWlbD7WIuKF6IhyYcGeK5GWMVrzDgtVI8Mvri
YEd6GuL9bCWqfXw1cHbBKg==
-----END PRIVATE KEY-----`

	tests := []struct {
		name              string
		setupObjects      []client.Object
		fastlyCertificate *fastly.CustomTLSCertificate
		expectedStale     bool
		expectedError     string
	}{
		{
			name: "certificate is not stale - serial numbers match",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
						DNSNames:   []string{"test1.example.com"},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCert1PEM),
					},
				},
			},
			fastlyCertificate: &fastly.CustomTLSCertificate{
				ID:           "cert-123",
				Name:         "test-certificate",
				SerialNumber: testCert1SerialDecimal, // Matches the certificate in the secret
			},
			expectedStale: false,
		},
		{
			name: "certificate is stale - serial numbers differ",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
						DNSNames:   []string{"test1.example.com"},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCert1PEM), // Has testCert1SerialDecimal
					},
				},
			},
			fastlyCertificate: &fastly.CustomTLSCertificate{
				ID:           "cert-123",
				Name:         "test-certificate",
				SerialNumber: testCert2SerialDecimal, // Different serial number
			},
			expectedStale: true,
		},
		{
			name: "certificate with different local certificate - stale",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
						DNSNames:   []string{"test2.example.com"},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCert2PEM), // Has testCert2SerialDecimal
					},
				},
			},
			fastlyCertificate: &fastly.CustomTLSCertificate{
				ID:           "cert-456",
				Name:         "test-certificate",
				SerialNumber: testCert1SerialDecimal, // Different from testCert2SerialDecimal
			},
			expectedStale: true,
		},
		{
			name: "certificate with same local certificate - not stale",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
						DNSNames:   []string{"test2.example.com"},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCert2PEM), // Has testCert2SerialDecimal
					},
				},
			},
			fastlyCertificate: &fastly.CustomTLSCertificate{
				ID:           "cert-456",
				Name:         "test-certificate",
				SerialNumber: testCert2SerialDecimal, // Matches testCert2SerialDecimal
			},
			expectedStale: false,
		},
		{
			name:          "error getting certificate from context",
			setupObjects:  []client.Object{}, // No objects - will cause getCertificateAndTLSSecretFromSubject to fail
			expectedError: "failed to get TLS secret from context",
		},
		{
			name: "error getting cert PEM from secret",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						// Missing tls.crt - will cause getCertPEMForSecret to fail
					},
				},
			},
			fastlyCertificate: &fastly.CustomTLSCertificate{
				ID:           "cert-123",
				Name:         "test-certificate",
				SerialNumber: testCert1SerialDecimal,
			},
			expectedError: "failed to get cert PEM for secret",
		},
		{
			name: "invalid PEM data",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte("invalid-pem-data"), // Invalid PEM
					},
				},
			},
			fastlyCertificate: &fastly.CustomTLSCertificate{
				ID:           "cert-123",
				Name:         "test-certificate",
				SerialNumber: testCert1SerialDecimal,
			},
			expectedError: "failed to decode PEM block",
		},
		{
			name: "unparseable certificate",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nVGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZSBidXQgaXMgdmFsaWQgYmFzZTY0Cg==\n-----END CERTIFICATE-----"), // Valid PEM encoding but invalid cert data
					},
				},
			},
			fastlyCertificate: &fastly.CustomTLSCertificate{
				ID:           "cert-123",
				Name:         "test-certificate",
				SerialNumber: testCert1SerialDecimal,
			},
			expectedError: "failed to parse certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake k8s client with test objects
			scheme := runtime.NewScheme()
			_ = cmv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.setupObjects...).
				Build()

			// Create Logic instance
			logic := &Logic{}

			// Create test context with fake K8s client
			ctx := createTestContext()
			ctx.Client = &k8sutil.ContextClient{
				SchemedClient: k8sutil.SchemedClient{
					Client: fakeClient,
				},
				Context:   context.Background(),
				Namespace: "test-namespace",
			}

			// Call the function under test
			result, err := logic.isFastlyCertificateStale(ctx, tt.fastlyCertificate)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("isFastlyCertificateStale() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("isFastlyCertificateStale() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return // Don't check result if we expected an error
			}

			if err != nil {
				t.Errorf("isFastlyCertificateStale() unexpected error = %v", err)
				return
			}

			// Check result
			if result != tt.expectedStale {
				t.Errorf("isFastlyCertificateStale() = %v, want %v", result, tt.expectedStale)
			}
		})
	}
}

func TestLogic_createFastlyCertificate(t *testing.T) {
	// Test certificate PEM data generated with OpenSSL
	testCertPEM := `-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUF9ZX7/+b9LAOz6pC/skiX020488wDQYJKoZIhvcNAQEL
BQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNTA3MjUxODU1MTFaFw0yNjA3MjUx
ODU1MTFaMCcxJTAjBgNVBAMMHHRlc3QtY2VydGlmaWNhdGUuZXhhbXBsZS5jb20w
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCYp0K+SBuSoZ8JIkeAcAYY
nQuNF8RTxAlj9SqPj6M0/H4b0BwS3vZAlIpxmQ7ZVE84iQafdOLR6eatulNVuV14
9Ab7rT/aGWH6lH70x8RmoOXMVY040CXV76je+L6nm+ZN0Fv02zwL0NgRNfO3utLr
xW9T29gka3Bvko/Z87NtUKk+M+CIWK7TYjvMulDRIUI8YEJZdNKfwR/5vemOjzMT
hApgvkvglhXl9xJMJ/Eb4Sq30Lt0uRP11a4BUJl6b+jujykQEXyRMxq4zLncyhLk
Z1Sxt5wmBXlHwO9Chcgk9XfjZIt8IeZLiEmjgAHljVvMz4HpgwsknVr/bK/LbsER
AgMBAAGjQjBAMB0GA1UdDgQWBBQ8asgD+X8GoDfh1HaExrbjErroOjAfBgNVHSME
GDAWgBQYfGMYbFe1HnqxOa/HoU/u3GqKWzANBgkqhkiG9w0BAQsFAAOCAQEATB9M
eIlYV8lO2nZoyMPRf73njSdPYu0trD4aNQxSA3T0mt+dfszmy+kJpsAWKQ8sZodR
jfNVzo6yJlcOUD7AJaspAsmUsaN1USghnVbO/BAuXomptBFlSLGkRRxjUKzqygOw
0X4HDy0j/NDYW+Ifi8MOdAB6gNLUlRlmN6181Nrv1jzKbM9OGPHyElby1pRWP9CY
8ihOYhTjoPht2UflMNbptCtPH6yNrj/sxZXhCdXZNPMY3wdPdQY7TBtjBiRUzvat
/mjBLStI+NrwO6iYq6IAXWWo2MwPwgs54f3uYJ+OyU1qQX5vRp6QU5ei5KI7uuYc
TC0Xee/Aqtvr7zx4QQ==
-----END CERTIFICATE-----`

	testCACertPEM := `-----BEGIN CERTIFICATE-----
MIIDBTCCAe2gAwIBAgIUGcNQkfIBN+AM5f6Yp3L8fDppnU4wDQYJKoZIhvcNAQEL
BQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNTA3MjUxODU0MzdaFw0yNjA3MjUx
ODU0MzdaMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQC8m/rIYHQggrJs2NsJMDHsyKLw52T6MJH/QVRfjhIXuzkBl9N9
BZ9+DCgd2feXYRnOBbYe10YgjrK+TxyMEbzMkfW1Nat/kyZRY/aSXHfYaptJXU4X
qixyYkwir8qQaGrk527xIiXVf9PdVjeUeo5Beedic+AuOA+flocnLbvMz2K83k5j
LHTODO0A+cKiL1WSDPSQ7R4twtLxOo3/WcBv7nFjn7hSuQm6RuXtiGLCA5/965Vu
Kc8kcGudAfDHjk+U/9FHakRfEcjPANlVHQDPIX6lBosAxXEdKYVReOIb/FfhxblX
8o8qimMEdv6QthWoChltcTn933MHTP4VZ2OHAgMBAAGjUzBRMB0GA1UdDgQWBBQY
fGMYbFe1HnqxOa/HoU/u3GqKWzAfBgNVHSMEGDAWgBQYfGMYbFe1HnqxOa/HoU/u
3GqKWzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCD9qcLBMam
IdV3EIre1HiUhiw+QkWIS5iPBWoPHZ5KkvT4Jd1w7ykS/HtkdKqeoQCnuspbBVma
+3BgjcpnMI1UygKbjIw0waieeTuBwVVmhhjHQWyDjhejfLHYo88IJdmG7NbsShdj
D/HPhxGyDFvaAlGSNSG3tXmiNCfEyAKpxO5a3h+grkoQeFIGnaDxvTesWct/kEXN
W3D8yxXbf1pVSDu/n8psU4UehElQSUJ99OAE/r8ZAaz4FNk7uxUbMQXuutgcQpZ6
5G6IEoBindfwE0kPTZjWjIfOwezPAsweqTyztP5kcHgTwEMLu6rUXA9fMSXR+0bg
Obq/T4m2BUjO
-----END CERTIFICATE-----`

	testPrivateKeyPEM := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCYp0K+SBuSoZ8J
IkeAcAYYnQuNF8RTxAlj9SqPj6M0/H4b0BwS3vZAlIpxmQ7ZVE84iQafdOLR6eat
ulNVuV149Ab7rT/aGWH6lH70x8RmoOXMVY040CXV76je+L6nm+ZN0Fv02zwL0NgR
NfO3utLrxW9T29gka3Bvko/Z87NtUKk+M+CIWK7TYjvMulDRIUI8YEJZdNKfwR/5
vemOjzMThApgvkvglhXl9xJMJ/Eb4Sq30Lt0uRP11a4BUJl6b+jujykQEXyRMxq4
zLncyhLkZ1Sxt5wmBXlHwO9Chcgk9XfjZIt8IeZLiEmjgAHljVvMz4HpgwsknVr/
bK/LbsERAgMBAAECggEANKj+jUWyvVKj2jLJF7WNZNBIO9QHFh56XtEkbYHPe2fe
2RlhleD0cjLLz4RNawt6iLY8YqWf2Wom+addOCVJ6X/FKO0LKeG3uwmfAjInvn+i
xmp83Sxw4OxcBQ8qNgfB2vYVwtIeVLUm1EkYWjlIqaziSrt8RJQLpXGZzkYTj5HL
bGJjSRhfH7CFXAMNgvw2dKCsZtdWlLYtcE3saG0far9hSsvyTaPid3x28E8/jW8Z
oLpO2fPnyLKpLoc88quXyaM1rvRcDLCEanA2GNpM44l57eN5pK6npsjClYlMfoLV
yxoBwbwF+K1xKoem9PVCvHVusu1HciE+LFe47BJwQQKBgQDF6M9F76MxDjkPhZkL
n63n8U5+2SkOmT8uKyk9MBHrYa/QqXBljcIiB8LEkWBYGiDdKhFIBsPh7rhqlY/5
L4DdWGvgwa+ERKTTf78YTtPPXufH8dNp0HFqrzckPT/rkzn25zHwYSW/TBEJZ/yU
RCTW1aIkq23QFeBEWUpjyBws0wKBgQDFddkaYDtJUcGqO81HTIRGt4Mq3evFs2KT
tC6HKAdteJdlQ5Ca2KVjtIvUMqW4NNuUk4A5xIcz4MSlyQ1+2CdFrzJT4Hofa8G4
JuIkn1mp6OQhaSNXYfxGJ6lkfrTmFUXfZoyvcflY8u0VkO2UcLcP6Dp2sYltbkzw
FgiCr09cCwKBgQCA7MGiGJMh0NchInHp3ZLHpy3wen1BkllTNTC/OIJj6RZEgyzC
K0/NJWse7Glr21GPYekyF54hn4apgFbzCJwVFZXpK6OwMZuCYBTXu/pFe9jYKtQD
eZN44T21sOTkDNvU2RVyN4cEkIQEsaYb3Cx3e2IOK1L1HFsli1lnmSOpmwKBgEd4
bVlfpXXXUrq0JIv/BQ23lJFqe9E2KaL+n6yp725PLLUpbGivq8VX7xiiMFtpPmUb
sli2ap17aJH9IJZd1HEjhZrYcDt5PEfUQxwwVTrroc76CCGzxKT77BMEzaNN5dmD
e75xCWiJnQimSWfmGEx4qNiXT/+84bowr2nl3FqbAoGAWgLiK/ZjWBQA9j8EPkJc
Q6XCVFB/FTkoCyYxLzL/pVKaw16xi+UehzHeC7GcPidu2trH9ikW6v1i5lxKl8Y+
p/Xa4rAIUbRxNAL/KehpylhAZGZRL4iueGDGz/oLo3mj8G9nwUW5xcDVfU7TDHR7
rI/pIULoTkGajE0uXlIlG0k=
-----END PRIVATE KEY-----`

	tests := []struct {
		name                       string
		setupObjects               []client.Object // K8s objects to create in fake client
		fastlyAPIShouldNotBeCalled bool            // If true, fail test if API is called
		fastlyAPIError             string          // If set, return this error from API
		hackLocalReconciliation    bool            // Value for AllowUntrustedRoot
		expectedError              string
		expectFastlyClientCall     bool
		expectedFastlyInput        *fastly.CreateCustomTLSCertificateInput
	}{
		{
			name: "successful certificate creation - production mode",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
					},
				},
			},
			hackLocalReconciliation: false,
			expectFastlyClientCall:  true,
			expectedFastlyInput: &fastly.CreateCustomTLSCertificateInput{
				CertBlob:           testCertPEM,
				Name:               "test-certificate",
				AllowUntrustedRoot: false,
			},
		},
		{
			name: "successful certificate creation - local development mode with CA chain",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
						"ca.crt":  []byte(testCACertPEM), // Required for local reconciliation
					},
				},
			},
			hackLocalReconciliation: true,
			expectFastlyClientCall:  true,
			expectedFastlyInput: &fastly.CreateCustomTLSCertificateInput{
				CertBlob:           testCertPEM + testCACertPEM, // Should be concatenated
				Name:               "test-certificate",
				AllowUntrustedRoot: true,
			},
		},
		{
			name:                       "certificate not found",
			setupObjects:               []client.Object{}, // No objects - certificate missing
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get TLS secret from context",
			expectFastlyClientCall:     false,
		},
		{
			name: "secret not found",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret", // This secret doesn't exist
					},
				},
				// No secret object
			},
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get TLS secret from context",
			expectFastlyClientCall:     false,
		},
		{
			name: "secret missing tls.crt",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						// Note: tls.crt is missing
					},
				},
			},
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get CertPEM for Fastly certificate",
			expectFastlyClientCall:     false,
		},
		{
			name: "fastly api error",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
					},
				},
			},
			fastlyAPIError:         "fastly api connection failed",
			expectedError:          "failed to create Fastly certificate: fastly api connection failed",
			expectFastlyClientCall: true,
		},
	}

	// Helper function to create mock Fastly client based on raw parameters
	setupFastlyClient := func(t *testing.T, shouldNotBeCalled bool, apiError string) *MockFastlyClient {
		return &MockFastlyClient{
			CreateCustomTLSCertificateFunc: func(ctx context.Context, input *fastly.CreateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error) {
				if shouldNotBeCalled {
					t.Error("CreateCustomTLSCertificate should not be called in this test case")
					return nil, nil
				}

				if apiError != "" {
					return nil, errors.New(apiError)
				}

				// Success case
				return &fastly.CustomTLSCertificate{ID: "new-cert-123"}, nil
			},
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Fastly client mock with call tracking
			var actualFastlyInput *fastly.CreateCustomTLSCertificateInput
			mockFastlyClient := setupFastlyClient(t, tt.fastlyAPIShouldNotBeCalled, tt.fastlyAPIError)

			// Wrap the original function to capture input
			originalFunc := mockFastlyClient.CreateCustomTLSCertificateFunc
			mockFastlyClient.CreateCustomTLSCertificateFunc = func(ctx context.Context, input *fastly.CreateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error) {
				actualFastlyInput = input
				return originalFunc(ctx, input)
			}

			// Create fake k8s client with test objects
			scheme := runtime.NewScheme()
			_ = cmv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.setupObjects...).
				Build()

			// Create Logic instance
			logic := &Logic{
				FastlyClient: mockFastlyClient,
			}

			// Create test context with fake K8s client
			ctx := createTestContext()
			ctx.Client = &k8sutil.ContextClient{
				SchemedClient: k8sutil.SchemedClient{
					Client: fakeClient,
				},
				Context:   context.Background(),
				Namespace: "test-namespace",
			}
			// Set the hack flag for testing AllowUntrustedRoot
			ctx.Config.HackFastlyCertificateSyncLocalReconciliation = tt.hackLocalReconciliation

			// Call the function
			err := logic.createFastlyCertificate(ctx)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("createFastlyCertificate() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("createFastlyCertificate() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("createFastlyCertificate() unexpected error = %v", err)
				}
			}

			// Check if Fastly client was called as expected
			if tt.expectFastlyClientCall {
				if actualFastlyInput == nil {
					t.Error("createFastlyCertificate() expected Fastly CreateCustomTLSCertificate to be called, but it wasn't")
				} else if tt.expectedFastlyInput != nil {
					// Verify the input to CreateCustomTLSCertificate
					if actualFastlyInput.CertBlob != tt.expectedFastlyInput.CertBlob {
						t.Errorf("createFastlyCertificate() Fastly input CertBlob = %q, want %q", actualFastlyInput.CertBlob, tt.expectedFastlyInput.CertBlob)
					}
					if actualFastlyInput.Name != tt.expectedFastlyInput.Name {
						t.Errorf("createFastlyCertificate() Fastly input Name = %q, want %q", actualFastlyInput.Name, tt.expectedFastlyInput.Name)
					}
					if actualFastlyInput.AllowUntrustedRoot != tt.expectedFastlyInput.AllowUntrustedRoot {
						t.Errorf("createFastlyCertificate() Fastly input AllowUntrustedRoot = %v, want %v", actualFastlyInput.AllowUntrustedRoot, tt.expectedFastlyInput.AllowUntrustedRoot)
					}
				}
			} else {
				if actualFastlyInput != nil {
					t.Error("createFastlyCertificate() expected Fastly CreateCustomTLSCertificate NOT to be called, but it was")
				}
			}
		})
	}
}

func TestLogic_updateFastlyCertificate(t *testing.T) {
	// Reuse the same OpenSSL-generated certificates from createFastlyCertificate test
	testCertPEM := `-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUF9ZX7/+b9LAOz6pC/skiX020488wDQYJKoZIhvcNAQEL
BQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNTA3MjUxODU1MTFaFw0yNjA3MjUx
ODU1MTFaMCcxJTAjBgNVBAMMHHRlc3QtY2VydGlmaWNhdGUuZXhhbXBsZS5jb20w
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCYp0K+SBuSoZ8JIkeAcAYY
nQuNF8RTxAlj9SqPj6M0/H4b0BwS3vZAlIpxmQ7ZVE84iQafdOLR6eatulNVuV14
9Ab7rT/aGWH6lH70x8RmoOXMVY040CXV76je+L6nm+ZN0Fv02zwL0NgRNfO3utLr
xW9T29gka3Bvko/Z87NtUKk+M+CIWK7TYjvMulDRIUI8YEJZdNKfwR/5vemOjzMT
hApgvkvglhXl9xJMJ/Eb4Sq30Lt0uRP11a4BUJl6b+jujykQEXyRMxq4zLncyhLk
Z1Sxt5wmBXlHwO9Chcgk9XfjZIt8IeZLiEmjgAHljVvMz4HpgwsknVr/bK/LbsER
AgMBAAGjQjBAMB0GA1UdDgQWBBQ8asgD+X8GoDfh1HaExrbjErroOjAfBgNVHSME
GDAWgBQYfGMYbFe1HnqxOa/HoU/u3GqKWzANBgkqhkiG9w0BAQsFAAOCAQEATB9M
eIlYV8lO2nZoyMPRf73njSdPYu0trD4aNQxSA3T0mt+dfszmy+kJpsAWKQ8sZodR
jfNVzo6yJlcOUD7AJaspAsmUsaN1USghnVbO/BAuXomptBFlSLGkRRxjUKzqygOw
0X4HDy0j/NDYW+Ifi8MOdAB6gNLUlRlmN6181Nrv1jzKbM9OGPHyElby1pRWP9CY
8ihOYhTjoPht2UflMNbptCtPH6yNrj/sxZXhCdXZNPMY3wdPdQY7TBtjBiRUzvat
/mjBLStI+NrwO6iYq6IAXWWo2MwPwgs54f3uYJ+OyU1qQX5vRp6QU5ei5KI7uuYc
TC0Xee/Aqtvr7zx4QQ==
-----END CERTIFICATE-----`

	testCACertPEM := `-----BEGIN CERTIFICATE-----
MIIDBTCCAe2gAwIBAgIUGcNQkfIBN+AM5f6Yp3L8fDppnU4wDQYJKoZIhvcNAQEL
BQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNTA3MjUxODU0MzdaFw0yNjA3MjUx
ODU0MzdaMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQC8m/rIYHQggrJs2NsJMDHsyKLw52T6MJH/QVRfjhIXuzkBl9N9
BZ9+DCgd2feXYRnOBbYe10YgjrK+TxyMEbzMkfW1Nat/kyZRY/aSXHfYaptJXU4X
qixyYkwir8qQaGrk527xIiXVf9PdVjeUeo5Beedic+AuOA+flocnLbvMz2K83k5j
LHTODO0A+cKiL1WSDPSQ7R4twtLxOo3/WcBv7nFjn7hSuQm6RuXtiGLCA5/965Vu
Kc8kcGudAfDHjk+U/9FHakRfEcjPANlVHQDPIX6lBosAxXEdKYVReOIb/FfhxblX
8o8qimMEdv6QthWoChltcTn933MHTP4VZ2OHAgMBAAGjUzBRMB0GA1UdDgQWBBQY
fGMYbFe1HnqxOa/HoU/u3GqKWzAfBgNVHSMEGDAWgBQYfGMYbFe1HnqxOa/HoU/u
3GqKWzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCD9qcLBMam
IdV3EIre1HiUhiw+QkWIS5iPBWoPHZ5KkvT4Jd1w7ykS/HtkdKqeoQCnuspbBVma
+3BgjcpnMI1UygKbjIw0waieeTuBwVVmhhjHQWyDjhejfLHYo88IJdmG7NbsShdj
D/HPhxGyDFvaAlGSNSG3tXmiNCfEyAKpxO5a3h+grkoQeFIGnaDxvTesWct/kEXN
W3D8yxXbf1pVSDu/n8psU4UehElQSUJ99OAE/r8ZAaz4FNk7uxUbMQXuutgcQpZ6
5G6IEoBindfwE0kPTZjWjIfOwezPAsweqTyztP5kcHgTwEMLu6rUXA9fMSXR+0bg
Obq/T4m2BUjO
-----END CERTIFICATE-----`

	testPrivateKeyPEM := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCYp0K+SBuSoZ8J
IkeAcAYYnQuNF8RTxAlj9SqPj6M0/H4b0BwS3vZAlIpxmQ7ZVE84iQafdOLR6eat
ulNVuV149Ab7rT/aGWH6lH70x8RmoOXMVY040CXV76je+L6nm+ZN0Fv02zwL0NgR
NfO3utLrxW9T29gka3Bvko/Z87NtUKk+M+CIWK7TYjvMulDRIUI8YEJZdNKfwR/5
vemOjzMThApgvkvglhXl9xJMJ/Eb4Sq30Lt0uRP11a4BUJl6b+jujykQEXyRMxq4
zLncyhLkZ1Sxt5wmBXlHwO9Chcgk9XfjZIt8IeZLiEmjgAHljVvMz4HpgwsknVr/
bK/LbsERAgMBAAECggEANKj+jUWyvVKj2jLJF7WNZNBIO9QHFh56XtEkbYHPe2fe
2RlhleD0cjLLz4RNawt6iLY8YqWf2Wom+addOCVJ6X/FKO0LKeG3uwmfAjInvn+i
xmp83Sxw4OxcBQ8qNgfB2vYVwtIeVLUm1EkYWjlIqaziSrt8RJQLpXGZzkYTj5HL
bGJjSRhfH7CFXAMNgvw2dKCsZtdWlLYtcE3saG0far9hSsvyTaPid3x28E8/jW8Z
oLpO2fPnyLKpLoc88quXyaM1rvRcDLCEanA2GNpM44l57eN5pK6npsjClYlMfoLV
yxoBwbwF+K1xKoem9PVCvHVusu1HciE+LFe47BJwQQKBgQDF6M9F76MxDjkPhZkL
n63n8U5+2SkOmT8uKyk9MBHrYa/QqXBljcIiB8LEkWBYGiDdKhFIBsPh7rhqlY/5
L4DdWGvgwa+ERKTTf78YTtPPXufH8dNp0HFqrzckPT/rkzn25zHwYSW/TBEJZ/yU
RCTW1aIkq23QFeBEWUpjyBws0wKBgQDFddkaYDtJUcGqO81HTIRGt4Mq3evFs2KT
tC6HKAdteJdlQ5Ca2KVjtIvUMqW4NNuUk4A5xIcz4MSlyQ1+2CdFrzJT4Hofa8G4
JuIkn1mp6OQhaSNXYfxGJ6lkfrTmFUXfZoyvcflY8u0VkO2UcLcP6Dp2sYltbkzw
FgiCr09cCwKBgQCA7MGiGJMh0NchInHp3ZLHpy3wen1BkllTNTC/OIJj6RZEgyzC
K0/NJWse7Glr21GPYekyF54hn4apgFbzCJwVFZXpK6OwMZuCYBTXu/pFe9jYKtQD
eZN44T21sOTkDNvU2RVyN4cEkIQEsaYb3Cx3e2IOK1L1HFsli1lnmSOpmwKBgEd4
bVlfpXXXUrq0JIv/BQ23lJFqe9E2KaL+n6yp725PLLUpbGivq8VX7xiiMFtpPmUb
sli2ap17aJH9IJZd1HEjhZrYcDt5PEfUQxwwVTrroc76CCGzxKT77BMEzaNN5dmD
e75xCWiJnQimSWfmGEx4qNiXT/+84bowr2nl3FqbAoGAWgLiK/ZjWBQA9j8EPkJc
Q6XCVFB/FTkoCyYxLzL/pVKaw16xi+UehzHeC7GcPidu2trH9ikW6v1i5lxKl8Y+
p/Xa4rAIUbRxNAL/KehpylhAZGZRL4iueGDGz/oLo3mj8G9nwUW5xcDVfU7TDHR7
rI/pIULoTkGajE0uXlIlG0k=
-----END PRIVATE KEY-----`

	tests := []struct {
		name                          string
		setupObjects                  []client.Object              // K8s objects to create in fake client
		mockExistingFastlyCertificate *fastly.CustomTLSCertificate // What getFastlyCertificateMatchingSubject returns
		getFastlyCertificateError     string                       // Error from getFastlyCertificateMatchingSubject
		fastlyAPIShouldNotBeCalled    bool                         // If true, fail test if UpdateCustomTLSCertificate is called
		fastlyAPIError                string                       // If set, return this error from UpdateCustomTLSCertificate
		hackLocalReconciliation       bool                         // Value for AllowUntrustedRoot
		expectedError                 string
		expectFastlyUpdateCall        bool
		expectedFastlyUpdateInput     *fastly.UpdateCustomTLSCertificateInput
	}{
		{
			name: "successful certificate update - production mode",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
					},
				},
			},
			mockExistingFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "existing-cert-123",
				Name: "test-certificate",
			},
			hackLocalReconciliation: false,
			expectFastlyUpdateCall:  true,
			expectedFastlyUpdateInput: &fastly.UpdateCustomTLSCertificateInput{
				CertBlob:           testCertPEM,
				Name:               "test-certificate",
				ID:                 "existing-cert-123",
				AllowUntrustedRoot: false,
			},
		},
		{
			name: "successful certificate update - local development mode with CA chain",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
						"ca.crt":  []byte(testCACertPEM), // Required for local reconciliation
					},
				},
			},
			mockExistingFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "existing-cert-456",
				Name: "test-certificate",
			},
			hackLocalReconciliation: true,
			expectFastlyUpdateCall:  true,
			expectedFastlyUpdateInput: &fastly.UpdateCustomTLSCertificateInput{
				CertBlob:           testCertPEM + testCACertPEM, // Should be concatenated
				Name:               "test-certificate",
				ID:                 "existing-cert-456",
				AllowUntrustedRoot: true,
			},
		},
		{
			name:                       "certificate not found in kubernetes",
			setupObjects:               []client.Object{}, // No objects - certificate missing
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get TLS secret from context",
			expectFastlyUpdateCall:     false,
		},
		{
			name: "secret not found in kubernetes",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret", // This secret doesn't exist
					},
				},
				// No secret object
			},
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get TLS secret from context",
			expectFastlyUpdateCall:     false,
		},
		{
			name: "secret missing tls.crt",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						// Note: tls.crt is missing
					},
				},
			},
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get CertPEM for Fastly certificate",
			expectFastlyUpdateCall:     false,
		},
		{
			name: "local development mode missing ca.crt",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
						// Note: ca.crt is missing but required for local reconciliation
					},
				},
			},
			hackLocalReconciliation:    true,
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get CertPEM for Fastly certificate",
			expectFastlyUpdateCall:     false,
		},
		{
			name: "fastly certificate not found",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
					},
				},
			},
			mockExistingFastlyCertificate: nil, // Certificate not found in Fastly
			fastlyAPIShouldNotBeCalled:    true,
			expectedError:                 "fastly certificate not found",
			expectFastlyUpdateCall:        false,
		},
		{
			name: "error getting fastly certificate",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
					},
				},
			},
			getFastlyCertificateError:  "fastly list certificates failed",
			fastlyAPIShouldNotBeCalled: true,
			expectedError:              "failed to get Fastly certificate matching subject",
			expectFastlyUpdateCall:     false,
		},
		{
			name: "fastly api update error",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.key": []byte(testPrivateKeyPEM),
						"tls.crt": []byte(testCertPEM),
					},
				},
			},
			mockExistingFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "existing-cert-789",
				Name: "test-certificate",
			},
			fastlyAPIError:         "fastly update api connection failed",
			expectedError:          "failed to update Fastly certificate: fastly update api connection failed",
			expectFastlyUpdateCall: true,
		},
	}

	// Helper function to create logic with mocked Fastly API calls
	createLogicWithMocks := func(t *testing.T, mockCert *fastly.CustomTLSCertificate, getCertError string, shouldNotCallUpdate bool, updateError string) (*Logic, **fastly.UpdateCustomTLSCertificateInput) {
		var actualUpdateInput *fastly.UpdateCustomTLSCertificateInput

		mockFastlyClient := &MockFastlyClient{
			// Mock ListCustomTLSCertificates to control what getFastlyCertificateMatchingSubject finds
			ListCustomTLSCertificatesFunc: func(ctx context.Context, input *fastly.ListCustomTLSCertificatesInput) ([]*fastly.CustomTLSCertificate, error) {
				if getCertError != "" {
					return nil, errors.New(getCertError)
				}

				// Return the mock certificate if it exists, otherwise empty list
				// Only return on first page to simulate simple case
				if input.PageNumber == 1 && mockCert != nil {
					return []*fastly.CustomTLSCertificate{mockCert}, nil
				}
				return []*fastly.CustomTLSCertificate{}, nil
			},
			UpdateCustomTLSCertificateFunc: func(ctx context.Context, input *fastly.UpdateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error) {
				if shouldNotCallUpdate {
					t.Error("UpdateCustomTLSCertificate should not be called in this test case")
					return nil, nil
				}

				actualUpdateInput = input

				if updateError != "" {
					return nil, errors.New(updateError)
				}

				// Success case
				return &fastly.CustomTLSCertificate{ID: input.ID}, nil
			},
		}

		logic := &Logic{
			FastlyClient: mockFastlyClient,
		}

		return logic, &actualUpdateInput
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create logic with mocked methods
			logic, actualUpdateInputPtr := createLogicWithMocks(t, tt.mockExistingFastlyCertificate, tt.getFastlyCertificateError, tt.fastlyAPIShouldNotBeCalled, tt.fastlyAPIError)

			// Create fake k8s client with test objects
			scheme := runtime.NewScheme()
			_ = cmv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.setupObjects...).
				Build()

			// Create test context with fake K8s client
			ctx := createTestContext()
			ctx.Client = &k8sutil.ContextClient{
				SchemedClient: k8sutil.SchemedClient{
					Client: fakeClient,
				},
				Context:   context.Background(),
				Namespace: "test-namespace",
			}
			// Set the hack flag for testing AllowUntrustedRoot
			ctx.Config.HackFastlyCertificateSyncLocalReconciliation = tt.hackLocalReconciliation

			// Call the function
			err := logic.updateFastlyCertificate(ctx)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("updateFastlyCertificate() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("updateFastlyCertificate() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("updateFastlyCertificate() unexpected error = %v", err)
				}
			}

			// Check if Fastly client was called as expected
			actualUpdateInput := *actualUpdateInputPtr
			if tt.expectFastlyUpdateCall {
				if actualUpdateInput == nil {
					t.Error("updateFastlyCertificate() expected Fastly UpdateCustomTLSCertificate to be called, but it wasn't")
				} else if tt.expectedFastlyUpdateInput != nil {
					// Verify the input to UpdateCustomTLSCertificate
					if actualUpdateInput.CertBlob != tt.expectedFastlyUpdateInput.CertBlob {
						t.Errorf("updateFastlyCertificate() Fastly input CertBlob = %q, want %q", actualUpdateInput.CertBlob, tt.expectedFastlyUpdateInput.CertBlob)
					}
					if actualUpdateInput.Name != tt.expectedFastlyUpdateInput.Name {
						t.Errorf("updateFastlyCertificate() Fastly input Name = %q, want %q", actualUpdateInput.Name, tt.expectedFastlyUpdateInput.Name)
					}
					if actualUpdateInput.ID != tt.expectedFastlyUpdateInput.ID {
						t.Errorf("updateFastlyCertificate() Fastly input ID = %q, want %q", actualUpdateInput.ID, tt.expectedFastlyUpdateInput.ID)
					}
					if actualUpdateInput.AllowUntrustedRoot != tt.expectedFastlyUpdateInput.AllowUntrustedRoot {
						t.Errorf("updateFastlyCertificate() Fastly input AllowUntrustedRoot = %v, want %v", actualUpdateInput.AllowUntrustedRoot, tt.expectedFastlyUpdateInput.AllowUntrustedRoot)
					}
				}
			} else {
				if actualUpdateInput != nil {
					t.Error("updateFastlyCertificate() expected Fastly UpdateCustomTLSCertificate NOT to be called, but it was")
				}
			}
		})
	}
}

// Helper function to generate a full page of certificates
func generateCertPage(pageNum int, count int) []*fastly.CustomTLSCertificate {
	certs := make([]*fastly.CustomTLSCertificate, count)
	for i := 0; i < count; i++ {
		certs[i] = &fastly.CustomTLSCertificate{
			ID:   fmt.Sprintf("cert%d%d", pageNum, i),
			Name: fmt.Sprintf("certificate-%d%d", pageNum, i),
		}
	}
	return certs
}

// Helper function to generate a full page with a specific certificate at the end
func generateCertPageWithMatch(pageNum int, matchID, matchName string) []*fastly.CustomTLSCertificate {
	certs := make([]*fastly.CustomTLSCertificate, defaultFastlyPageSize)
	for i := 0; i < defaultFastlyPageSize-1; i++ {
		certs[i] = &fastly.CustomTLSCertificate{
			ID:   fmt.Sprintf("cert%d%d", pageNum, i),
			Name: fmt.Sprintf("certificate-%d%d", pageNum, i),
		}
	}
	// Last certificate matches
	certs[defaultFastlyPageSize-1] = &fastly.CustomTLSCertificate{ID: matchID, Name: matchName}
	return certs
}

func TestLogic_getFastlyCertificateMatchingSubject(t *testing.T) {
	tests := []struct {
		name                   string
		setupObjects           []client.Object
		mockFastlyCertificates [][]*fastly.CustomTLSCertificate // Support for pagination testing
		fastlyAPIError         error
		expectedCertificate    *fastly.CustomTLSCertificate // What should be returned
		expectedError          string
		expectedPageRequests   int // Number of page requests expected
	}{
		{
			name: "certificate found in fastly - single page",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificates: [][]*fastly.CustomTLSCertificate{
				// Page 1
				{
					{ID: "cert1", Name: "other-certificate"},
					{ID: "cert2", Name: "test-certificate"}, // This matches
					{ID: "cert3", Name: "another-certificate"},
				},
			},
			expectedCertificate:  &fastly.CustomTLSCertificate{ID: "cert2", Name: "test-certificate"},
			expectedPageRequests: 1,
		},
		{
			name: "certificate found in fastly - multiple pages",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificates: [][]*fastly.CustomTLSCertificate{
				// Page 1 - full page (20 certificates)
				generateCertPage(1, defaultFastlyPageSize),
				// Page 2 - partial page with matching certificate
				{
					{ID: "cert21", Name: "some-other-certificate"},
					{ID: "cert22", Name: "test-certificate"}, // This matches
					{ID: "cert23", Name: "final-certificate"},
				},
			},
			expectedCertificate:  &fastly.CustomTLSCertificate{ID: "cert22", Name: "test-certificate"},
			expectedPageRequests: 2,
		},
		{
			name: "certificate not found in fastly",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificates: [][]*fastly.CustomTLSCertificate{
				// Page 1
				{
					{ID: "cert1", Name: "other-certificate"},
					{ID: "cert2", Name: "different-certificate"},
					{ID: "cert3", Name: "another-certificate"},
				},
			},
			expectedCertificate:  nil, // Not found
			expectedPageRequests: 1,
		},
		{
			name: "no certificates in fastly",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificates: [][]*fastly.CustomTLSCertificate{
				// Page 1 - empty
				{},
			},
			expectedCertificate:  nil, // Not found
			expectedPageRequests: 1,
		},
		{
			name: "certificate found on first page even with multiple pages",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificates: [][]*fastly.CustomTLSCertificate{
				// Page 1 - full page with match at the end
				generateCertPageWithMatch(1, "matching-cert", "test-certificate"),
				// Page 2 - shouldn't be requested because we find match on page 1
				{
					{ID: "cert21", Name: "should-not-be-reached"},
				},
			},
			expectedCertificate:  &fastly.CustomTLSCertificate{ID: "matching-cert", Name: "test-certificate"},
			expectedPageRequests: 2, // Will request page 2 since page 1 was full, even though match is on page 1
		},
		{
			name: "multiple matching certificates - returns first found",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificates: [][]*fastly.CustomTLSCertificate{
				// Page 1
				{
					{ID: "cert1", Name: "other-certificate"},
					{ID: "cert2", Name: "test-certificate"}, // First match
					{ID: "cert3", Name: "test-certificate"}, // Second match (should not be returned)
				},
			},
			expectedCertificate:  &fastly.CustomTLSCertificate{ID: "cert2", Name: "test-certificate"}, // Returns first found
			expectedPageRequests: 1,
		},
		{
			name:          "kubernetes certificate not found",
			setupObjects:  []client.Object{}, // No certificate object
			expectedError: "failed to get certificate of name",
		},
		{
			name: "fastly api error",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			fastlyAPIError:       errors.New("fastly connection failed"),
			expectedError:        "failed to list Fastly certificates",
			expectedPageRequests: 1,
		},
		{
			name: "certificate found after pagination through empty pages",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificates: [][]*fastly.CustomTLSCertificate{
				// Page 1 - full page but no matches
				generateCertPage(1, defaultFastlyPageSize),
				// Page 2 - full page but no matches
				generateCertPage(2, defaultFastlyPageSize),
				// Page 3 - partial page with match
				{
					{ID: "final-cert", Name: "test-certificate"}, // This matches
				},
			},
			expectedCertificate:  &fastly.CustomTLSCertificate{ID: "final-cert", Name: "test-certificate"},
			expectedPageRequests: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track API calls
			var actualPageRequests int

			// Create mock Fastly client
			mockFastlyClient := &MockFastlyClient{
				ListCustomTLSCertificatesFunc: func(ctx context.Context, input *fastly.ListCustomTLSCertificatesInput) ([]*fastly.CustomTLSCertificate, error) {
					actualPageRequests++

					if tt.fastlyAPIError != nil {
						return nil, tt.fastlyAPIError
					}

					// Handle pagination testing with mockFastlyCertificates
					if len(tt.mockFastlyCertificates) > 0 {
						pageIndex := input.PageNumber - 1 // Convert to 0-based index
						if pageIndex < len(tt.mockFastlyCertificates) {
							return tt.mockFastlyCertificates[pageIndex], nil
						}
						return []*fastly.CustomTLSCertificate{}, nil // Empty page for out-of-range requests
					}

					// Default empty response
					return []*fastly.CustomTLSCertificate{}, nil
				},
			}

			// Create fake k8s client with test objects
			scheme := runtime.NewScheme()
			_ = cmv1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.setupObjects...).
				Build()

			// Create Logic instance
			logic := &Logic{
				FastlyClient: mockFastlyClient,
			}

			// Create test context with fake K8s client
			ctx := createTestContext()
			ctx.Client = &k8sutil.ContextClient{
				SchemedClient: k8sutil.SchemedClient{
					Client: fakeClient,
				},
				Context:   context.Background(),
				Namespace: "test-namespace",
			}

			// Call the actual function
			result, err := logic.getFastlyCertificateMatchingSubject(ctx)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("getFastlyCertificateMatchingSubject() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("getFastlyCertificateMatchingSubject() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return // Don't check result if we expected an error
			}

			if err != nil {
				t.Errorf("getFastlyCertificateMatchingSubject() unexpected error = %v", err)
				return
			}

			// Check result
			if tt.expectedCertificate == nil {
				if result != nil {
					t.Errorf("getFastlyCertificateMatchingSubject() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("getFastlyCertificateMatchingSubject() = nil, want certificate with ID %s", tt.expectedCertificate.ID)
				} else {
					if result.ID != tt.expectedCertificate.ID {
						t.Errorf("getFastlyCertificateMatchingSubject() certificate ID = %s, want %s", result.ID, tt.expectedCertificate.ID)
					}
					if result.Name != tt.expectedCertificate.Name {
						t.Errorf("getFastlyCertificateMatchingSubject() certificate Name = %s, want %s", result.Name, tt.expectedCertificate.Name)
					}
				}
			}

			// Check API call expectations
			if tt.expectedPageRequests > 0 {
				if actualPageRequests != tt.expectedPageRequests {
					t.Errorf("getFastlyCertificateMatchingSubject() made %d page requests, want %d", actualPageRequests, tt.expectedPageRequests)
				}
			}
		})
	}
}

func TestLogic_getFastlyTLSActivationState(t *testing.T) {
	tests := []struct {
		name                        string
		setupObjects                []client.Object                             // K8s objects to create in fake client
		mockFastlyCertificate       *fastly.CustomTLSCertificate                // What getFastlyCertificateMatchingSubject returns
		getFastlyCertificateError   string                                      // Error from getFastlyCertificateMatchingSubject
		mockActivationMap           map[string]map[string]*fastly.TLSActivation // What getFastlyDomainAndConfigurationToActivationMap returns
		getActivationMapError       string                                      // Error from getFastlyDomainAndConfigurationToActivationMap
		expectedTLSConfigurationIds []string                                    // TLS configuration IDs in the subject
		expectedMissingActivations  []TLSActivationData
		expectedExtraActivationIDs  []string
		expectedError               string
	}{
		{
			name: "no certificate exists in fastly - returns empty results",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificate:       nil, // No certificate found
			expectedTLSConfigurationIds: []string{"config1", "config2"},
			expectedMissingActivations:  []TLSActivationData{},
			expectedExtraActivationIDs:  []string{},
		},
		{
			name: "certificate exists but no domains - returns empty results",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:      "cert-123",
				Name:    "test-certificate",
				Domains: []*fastly.TLSDomain{}, // No domains
			},
			mockActivationMap:           map[string]map[string]*fastly.TLSActivation{},
			expectedTLSConfigurationIds: []string{"config1", "config2"},
			expectedMissingActivations:  []TLSActivationData{},
			expectedExtraActivationIDs:  []string{},
		},
		{
			name: "certificate exists with domains but no expected configurations - returns empty results",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "cert-123",
				Name: "test-certificate",
				Domains: []*fastly.TLSDomain{
					{ID: "domain1"},
					{ID: "domain2"},
				},
			},
			mockActivationMap:           map[string]map[string]*fastly.TLSActivation{},
			expectedTLSConfigurationIds: []string{}, // No expected configurations
			expectedMissingActivations:  []TLSActivationData{},
			expectedExtraActivationIDs:  []string{},
		},
		{
			name: "missing activations - some combinations don't exist",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "cert-123",
				Name: "test-certificate",
				Domains: []*fastly.TLSDomain{
					{ID: "domain1"},
					{ID: "domain2"},
				},
			},
			mockActivationMap: map[string]map[string]*fastly.TLSActivation{
				// domain1 has config1 but missing config2
				"domain1": {
					"config1": {ID: "activation1", Domain: &fastly.TLSDomain{ID: "domain1"}, Configuration: &fastly.TLSConfiguration{ID: "config1"}},
				},
				// domain2 has no configurations at all
				"domain2": {},
			},
			expectedTLSConfigurationIds: []string{"config1", "config2"},
			expectedMissingActivations: []TLSActivationData{
				// Missing: domain1 + config2
				{
					Certificate:   &fastly.CustomTLSCertificate{ID: "cert-123", Name: "test-certificate", Domains: []*fastly.TLSDomain{{ID: "domain1"}, {ID: "domain2"}}},
					Configuration: &fastly.TLSConfiguration{ID: "config2"},
					Domain:        &fastly.TLSDomain{ID: "domain1"},
				},
				// Missing: domain2 + config1
				{
					Certificate:   &fastly.CustomTLSCertificate{ID: "cert-123", Name: "test-certificate", Domains: []*fastly.TLSDomain{{ID: "domain1"}, {ID: "domain2"}}},
					Configuration: &fastly.TLSConfiguration{ID: "config1"},
					Domain:        &fastly.TLSDomain{ID: "domain2"},
				},
				// Missing: domain2 + config2
				{
					Certificate:   &fastly.CustomTLSCertificate{ID: "cert-123", Name: "test-certificate", Domains: []*fastly.TLSDomain{{ID: "domain1"}, {ID: "domain2"}}},
					Configuration: &fastly.TLSConfiguration{ID: "config2"},
					Domain:        &fastly.TLSDomain{ID: "domain2"},
				},
			},
			expectedExtraActivationIDs: []string{}, // domain1+config1 gets removed from map since it's expected
		},
		{
			name: "extra activations - some activations exist but aren't expected",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "cert-123",
				Name: "test-certificate",
				Domains: []*fastly.TLSDomain{
					{ID: "domain1"},
				},
			},
			mockActivationMap: map[string]map[string]*fastly.TLSActivation{
				"domain1": {
					"config1": {ID: "activation1", Domain: &fastly.TLSDomain{ID: "domain1"}, Configuration: &fastly.TLSConfiguration{ID: "config1"}},
					"config3": {ID: "activation3", Domain: &fastly.TLSDomain{ID: "domain1"}, Configuration: &fastly.TLSConfiguration{ID: "config3"}}, // Extra - not expected
				},
			},
			expectedTLSConfigurationIds: []string{"config1"},     // Only expect config1
			expectedMissingActivations:  []TLSActivationData{},   // No missing activations
			expectedExtraActivationIDs:  []string{"activation3"}, // config3 activation should be deleted
		},
		{
			name: "mixed scenario - both missing and extra activations",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "cert-123",
				Name: "test-certificate",
				Domains: []*fastly.TLSDomain{
					{ID: "domain1"},
					{ID: "domain2"},
				},
			},
			mockActivationMap: map[string]map[string]*fastly.TLSActivation{
				"domain1": {
					"config1": {ID: "activation1", Domain: &fastly.TLSDomain{ID: "domain1"}, Configuration: &fastly.TLSConfiguration{ID: "config1"}}, // Expected - will be kept
					"config3": {ID: "activation3", Domain: &fastly.TLSDomain{ID: "domain1"}, Configuration: &fastly.TLSConfiguration{ID: "config3"}}, // Extra - should be deleted
				},
				"domain2": {
					"config4": {ID: "activation4", Domain: &fastly.TLSDomain{ID: "domain2"}, Configuration: &fastly.TLSConfiguration{ID: "config4"}}, // Extra - should be deleted
				},
			},
			expectedTLSConfigurationIds: []string{"config1", "config2"},
			expectedMissingActivations: []TLSActivationData{
				// Missing: domain1 + config2
				{
					Certificate:   &fastly.CustomTLSCertificate{ID: "cert-123", Name: "test-certificate", Domains: []*fastly.TLSDomain{{ID: "domain1"}, {ID: "domain2"}}},
					Configuration: &fastly.TLSConfiguration{ID: "config2"},
					Domain:        &fastly.TLSDomain{ID: "domain1"},
				},
				// Missing: domain2 + config1
				{
					Certificate:   &fastly.CustomTLSCertificate{ID: "cert-123", Name: "test-certificate", Domains: []*fastly.TLSDomain{{ID: "domain1"}, {ID: "domain2"}}},
					Configuration: &fastly.TLSConfiguration{ID: "config1"},
					Domain:        &fastly.TLSDomain{ID: "domain2"},
				},
				// Missing: domain2 + config2
				{
					Certificate:   &fastly.CustomTLSCertificate{ID: "cert-123", Name: "test-certificate", Domains: []*fastly.TLSDomain{{ID: "domain1"}, {ID: "domain2"}}},
					Configuration: &fastly.TLSConfiguration{ID: "config2"},
					Domain:        &fastly.TLSDomain{ID: "domain2"},
				},
			},
			expectedExtraActivationIDs: []string{"activation3", "activation4"}, // Both extra activations should be deleted
		},
		{
			name: "all activations exist correctly - no changes needed",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "cert-123",
				Name: "test-certificate",
				Domains: []*fastly.TLSDomain{
					{ID: "domain1"},
				},
			},
			mockActivationMap: map[string]map[string]*fastly.TLSActivation{
				"domain1": {
					"config1": {ID: "activation1", Domain: &fastly.TLSDomain{ID: "domain1"}, Configuration: &fastly.TLSConfiguration{ID: "config1"}},
					"config2": {ID: "activation2", Domain: &fastly.TLSDomain{ID: "domain1"}, Configuration: &fastly.TLSConfiguration{ID: "config2"}},
				},
			},
			expectedTLSConfigurationIds: []string{"config1", "config2"},
			expectedMissingActivations:  []TLSActivationData{}, // All activations exist
			expectedExtraActivationIDs:  []string{},            // No extra activations
		},
		{
			name: "error getting fastly certificate",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			getFastlyCertificateError:   "fastly api connection failed",
			expectedTLSConfigurationIds: []string{"config1"},
			expectedError:               "failed to get Fastly certificate matching subject",
		},
		{
			name: "error getting activation map",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
				},
			},
			mockFastlyCertificate: &fastly.CustomTLSCertificate{
				ID:   "cert-123",
				Name: "test-certificate",
				Domains: []*fastly.TLSDomain{
					{ID: "domain1"},
				},
			},
			getActivationMapError:       "fastly activation list failed",
			expectedTLSConfigurationIds: []string{"config1"},
			expectedError:               "failed to get Fastly domain and configuration to activation map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Fastly client
			mockFastlyClient := &MockFastlyClient{
				// Mock ListCustomTLSCertificates to control what getFastlyCertificateMatchingSubject finds
				ListCustomTLSCertificatesFunc: func(ctx context.Context, input *fastly.ListCustomTLSCertificatesInput) ([]*fastly.CustomTLSCertificate, error) {
					if tt.getFastlyCertificateError != "" {
						return nil, errors.New(tt.getFastlyCertificateError)
					}

					// Return the mock certificate if it exists, otherwise empty list
					if input.PageNumber == 1 && tt.mockFastlyCertificate != nil {
						return []*fastly.CustomTLSCertificate{tt.mockFastlyCertificate}, nil
					}
					return []*fastly.CustomTLSCertificate{}, nil
				},
				// Mock ListTLSActivations to control what getFastlyDomainAndConfigurationToActivationMap returns
				ListTLSActivationsFunc: func(ctx context.Context, input *fastly.ListTLSActivationsInput) ([]*fastly.TLSActivation, error) {
					if tt.getActivationMapError != "" {
						return nil, errors.New(tt.getActivationMapError)
					}

					// Convert the map back to a flat list for the mock API response
					var activations []*fastly.TLSActivation
					for _, configToActivation := range tt.mockActivationMap {
						for _, activation := range configToActivation {
							activations = append(activations, activation)
						}
					}

					// Only return on first page to simulate simple case
					if input.PageNumber == 1 {
						return activations, nil
					}
					return []*fastly.TLSActivation{}, nil
				},
			}

			// Create fake k8s client with test objects
			scheme := runtime.NewScheme()
			_ = cmv1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.setupObjects...).
				Build()

			// Create Logic instance
			logic := &Logic{
				FastlyClient: mockFastlyClient,
			}

			// Create test context with fake K8s client and expected TLS configuration IDs
			ctx := createTestContext()
			ctx.Client = &k8sutil.ContextClient{
				SchemedClient: k8sutil.SchemedClient{
					Client: fakeClient,
				},
				Context:   context.Background(),
				Namespace: "test-namespace",
			}
			ctx.Subject.Spec.TLSConfigurationIds = tt.expectedTLSConfigurationIds

			// Call the function under test
			missingActivations, extraActivationIDs, err := logic.getFastlyTLSActivationState(ctx)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("getFastlyTLSActivationState() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("getFastlyTLSActivationState() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return // Don't check results if we expected an error
			}

			if err != nil {
				t.Errorf("getFastlyTLSActivationState() unexpected error = %v", err)
				return
			}

			// Check missing activations result
			if len(missingActivations) != len(tt.expectedMissingActivations) {
				t.Errorf("getFastlyTLSActivationState() returned %d missing activations, want %d", len(missingActivations), len(tt.expectedMissingActivations))
			} else {
				// Verify each missing activation
				for i, expected := range tt.expectedMissingActivations {
					if i >= len(missingActivations) {
						t.Errorf("getFastlyTLSActivationState() missing activation %d not found", i)
						continue
					}
					actual := missingActivations[i]
					if actual.Certificate.ID != expected.Certificate.ID {
						t.Errorf("getFastlyTLSActivationState() missing activation %d certificate ID = %s, want %s", i, actual.Certificate.ID, expected.Certificate.ID)
					}
					if actual.Configuration.ID != expected.Configuration.ID {
						t.Errorf("getFastlyTLSActivationState() missing activation %d configuration ID = %s, want %s", i, actual.Configuration.ID, expected.Configuration.ID)
					}
					if actual.Domain.ID != expected.Domain.ID {
						t.Errorf("getFastlyTLSActivationState() missing activation %d domain ID = %s, want %s", i, actual.Domain.ID, expected.Domain.ID)
					}
				}
			}

			// Check extra activation IDs result
			if len(extraActivationIDs) != len(tt.expectedExtraActivationIDs) {
				t.Errorf("getFastlyTLSActivationState() returned %d extra activation IDs, want %d", len(extraActivationIDs), len(tt.expectedExtraActivationIDs))
			} else {
				// Convert both slices to maps for easier comparison (order doesn't matter)
				actualMap := make(map[string]bool)
				for _, id := range extraActivationIDs {
					actualMap[id] = true
				}
				expectedMap := make(map[string]bool)
				for _, id := range tt.expectedExtraActivationIDs {
					expectedMap[id] = true
				}

				// Check that all expected IDs are present
				for expectedID := range expectedMap {
					if !actualMap[expectedID] {
						t.Errorf("getFastlyTLSActivationState() missing expected extra activation ID %s", expectedID)
					}
				}

				// Check that no unexpected IDs are present
				for actualID := range actualMap {
					if !expectedMap[actualID] {
						t.Errorf("getFastlyTLSActivationState() unexpected extra activation ID %s", actualID)
					}
				}
			}
		})
	}
}
