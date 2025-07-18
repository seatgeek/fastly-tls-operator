package fastlycertificatesync

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fastly/go-fastly/v10/fastly"
)

// FastlyClientInterface defines the methods we need for testing
type FastlyClientInterface interface {
	ListPrivateKeys(input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error)
}

// Mock FastlyClient interface for testing
type MockFastlyClient struct {
	ListPrivateKeysFunc func(input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error)
}

func (m *MockFastlyClient) ListPrivateKeys(input *fastly.ListPrivateKeysInput) ([]*fastly.PrivateKey, error) {
	if m.ListPrivateKeysFunc != nil {
		return m.ListPrivateKeysFunc(input)
	}
	return nil, nil
}

// TestLogic wraps Logic with a FastlyClientInterface for testing
type TestLogic struct {
	client FastlyClientInterface
}

func (l *TestLogic) getFastlyUnusedPrivateKeyIDs(_ *Context) ([]string, error) {
	privateKeys, err := l.client.ListPrivateKeys(&fastly.ListPrivateKeysInput{FilterInUse: "false"})
	if err != nil {
		return nil, fmt.Errorf("failed to list Fastly private keys: %w", err)
	}

	unusedPrivateKeyIDs := []string{}
	for _, key := range privateKeys {
		unusedPrivateKeyIDs = append(unusedPrivateKeyIDs, key.ID)
	}
	return unusedPrivateKeyIDs, nil
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
					return tt.mockResponse, tt.mockError
				},
			}

			// Create test logic instance with mock client
			logic := &TestLogic{
				client: mockClient,
			}

			// Call function
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
