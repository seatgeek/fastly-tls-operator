package fastlycertificatesync

import (
	"errors"
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
