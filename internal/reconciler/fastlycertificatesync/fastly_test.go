package fastlycertificatesync

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/fastly/go-fastly/v10/fastly"
	"github.com/go-logr/logr"
)

// MockFastlyClient implements FastlyClientInterface for testing
type MockFastlyClient struct {
	ListPrivateKeysFunc            func(input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error)
	CreatePrivateKeyFunc           func(input *fastly.CreatePrivateKeyInput) (*fastly.PrivateKey, error)
	DeletePrivateKeyFunc           func(input *fastly.DeletePrivateKeyInput) error
	ListCustomTLSCertificatesFunc  func(input *fastly.ListCustomTLSCertificatesInput) ([]*fastly.CustomTLSCertificate, error)
	CreateCustomTLSCertificateFunc func(input *fastly.CreateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error)
	UpdateCustomTLSCertificateFunc func(input *fastly.UpdateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error)
	ListTLSActivationsFunc         func(input *fastly.ListTLSActivationsInput) ([]*fastly.TLSActivation, error)
	CreateTLSActivationFunc        func(input *fastly.CreateTLSActivationInput) (*fastly.TLSActivation, error)
	DeleteTLSActivationFunc        func(input *fastly.DeleteTLSActivationInput) error

	// Track method calls
	DeletePrivateKeyCalls []string
}

func (m *MockFastlyClient) ListPrivateKeys(input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error) {
	if m.ListPrivateKeysFunc != nil {
		return m.ListPrivateKeysFunc(input)
	}
	return nil, nil
}

func (m *MockFastlyClient) CreatePrivateKey(input *fastly.CreatePrivateKeyInput) (*fastly.PrivateKey, error) {
	if m.CreatePrivateKeyFunc != nil {
		return m.CreatePrivateKeyFunc(input)
	}
	return nil, nil
}

func (m *MockFastlyClient) DeletePrivateKey(input *fastly.DeletePrivateKeyInput) error {
	// Track the call
	m.DeletePrivateKeyCalls = append(m.DeletePrivateKeyCalls, input.ID)

	if m.DeletePrivateKeyFunc != nil {
		return m.DeletePrivateKeyFunc(input)
	}
	return nil
}

func (m *MockFastlyClient) ListCustomTLSCertificates(input *fastly.ListCustomTLSCertificatesInput) ([]*fastly.CustomTLSCertificate, error) {
	if m.ListCustomTLSCertificatesFunc != nil {
		return m.ListCustomTLSCertificatesFunc(input)
	}
	return nil, nil
}

func (m *MockFastlyClient) CreateCustomTLSCertificate(input *fastly.CreateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error) {
	if m.CreateCustomTLSCertificateFunc != nil {
		return m.CreateCustomTLSCertificateFunc(input)
	}
	return nil, nil
}

func (m *MockFastlyClient) UpdateCustomTLSCertificate(input *fastly.UpdateCustomTLSCertificateInput) (*fastly.CustomTLSCertificate, error) {
	if m.UpdateCustomTLSCertificateFunc != nil {
		return m.UpdateCustomTLSCertificateFunc(input)
	}
	return nil, nil
}

func (m *MockFastlyClient) ListTLSActivations(input *fastly.ListTLSActivationsInput) ([]*fastly.TLSActivation, error) {
	if m.ListTLSActivationsFunc != nil {
		return m.ListTLSActivationsFunc(input)
	}
	return nil, nil
}

func (m *MockFastlyClient) CreateTLSActivation(input *fastly.CreateTLSActivationInput) (*fastly.TLSActivation, error) {
	if m.CreateTLSActivationFunc != nil {
		return m.CreateTLSActivationFunc(input)
	}
	return nil, nil
}

func (m *MockFastlyClient) DeleteTLSActivation(input *fastly.DeleteTLSActivationInput) error {
	if m.DeleteTLSActivationFunc != nil {
		return m.DeleteTLSActivationFunc(input)
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
				ListPrivateKeysFunc: func(input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error) {
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
				DeletePrivateKeyFunc: func(input *fastly.DeletePrivateKeyInput) error {
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
