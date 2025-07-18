package fastlycertificatesync

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fastly/go-fastly/v10/fastly"
	"github.com/go-logr/logr"
)

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
		name           string
		mockResponse   []fastly.PrivateKey
		mockStatusCode int
		mockError      bool
		expectedIDs    []string
		expectedError  string
	}{
		{
			name: "successful call with multiple keys",
			mockResponse: []fastly.PrivateKey{
				{ID: "key1"},
				{ID: "key2"},
				{ID: "key3"},
			},
			mockStatusCode: 200,
			mockError:      false,
			expectedIDs:    []string{"key1", "key2", "key3"},
			expectedError:  "",
		},
		{
			name:           "successful call with no keys",
			mockResponse:   []fastly.PrivateKey{},
			mockStatusCode: 200,
			mockError:      false,
			expectedIDs:    []string{},
			expectedError:  "",
		},
		{
			name: "successful call with single key",
			mockResponse: []fastly.PrivateKey{
				{ID: "single-key"},
			},
			mockStatusCode: 200,
			mockError:      false,
			expectedIDs:    []string{"single-key"},
			expectedError:  "",
		},
		{
			name:           "api call returns 500",
			mockResponse:   nil,
			mockStatusCode: 500,
			mockError:      true,
			expectedIDs:    nil,
			expectedError:  "failed to list Fastly private keys:",
		},
		{
			name:           "api call returns 404",
			mockResponse:   nil,
			mockStatusCode: 404,
			mockError:      true,
			expectedIDs:    nil,
			expectedError:  "failed to list Fastly private keys:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request path and query params
				if r.URL.Path != "/tls/private_keys" {
					t.Errorf("Expected path /tls/private_keys, got %s", r.URL.Path)
				}

				// Verify filter[in_use]=false is set
				if r.URL.Query().Get("filter[in_use]") != "false" {
					t.Errorf("Expected filter[in_use]=false, got %s", r.URL.Query().Get("filter[in_use]"))
				}

				if tt.mockError {
					w.WriteHeader(tt.mockStatusCode)
					w.Write([]byte("Internal Server Error"))
					return
				}

				// Create JSONAPI response structure
				data := make([]map[string]interface{}, len(tt.mockResponse))
				for i, key := range tt.mockResponse {
					data[i] = map[string]interface{}{
						"type": "tls_private_key",
						"id":   key.ID,
						"attributes": map[string]interface{}{
							"name": key.Name,
						},
					}
				}

				response := map[string]interface{}{
					"data": data,
				}

				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(tt.mockStatusCode)
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create fastly client pointed at test server
			client, err := fastly.NewClientForEndpoint("test-key", server.URL)
			if err != nil {
				t.Fatalf("Failed to create fastly client: %v", err)
			}

			// Create Logic instance with real client pointed at mock server
			logic := &Logic{
				FastlyClient: client,
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
					t.Errorf("getFastlyUnusedPrivateKeyIDs() error = nil, want error containing %q", tt.expectedError)
				} else if len(tt.expectedError) > 0 && len(err.Error()) > 0 {
					// Check if error contains expected substring
					if len(err.Error()) < len(tt.expectedError) || err.Error()[:len(tt.expectedError)] != tt.expectedError {
						t.Errorf("getFastlyUnusedPrivateKeyIDs() error = %q, want error starting with %q", err.Error(), tt.expectedError)
					}
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
		deleteResponses     map[string]int // Map of keyID -> HTTP status code
		expectedDeletedKeys []string
	}{
		{
			name:                "successful deletion of multiple keys",
			unusedPrivateKeyIDs: []string{"key1", "key2", "key3"},
			deleteResponses: map[string]int{
				"key1": 200,
				"key2": 200,
				"key3": 200,
			},
			expectedDeletedKeys: []string{"key1", "key2", "key3"},
		},
		{
			name:                "no keys to delete",
			unusedPrivateKeyIDs: []string{},
			deleteResponses:     map[string]int{},
			expectedDeletedKeys: []string{},
		},
		{
			name:                "successful deletion of single key",
			unusedPrivateKeyIDs: []string{"single-key"},
			deleteResponses: map[string]int{
				"single-key": 200,
			},
			expectedDeletedKeys: []string{"single-key"},
		},
		{
			name:                "some deletions fail - should continue",
			unusedPrivateKeyIDs: []string{"key1", "key2", "key3"},
			deleteResponses: map[string]int{
				"key1": 500, // fails
				"key2": 200, // succeeds
				"key3": 404, // fails
			},
			expectedDeletedKeys: []string{"key1", "key2", "key3"}, // all attempted
		},
		{
			name:                "all deletions fail - should continue",
			unusedPrivateKeyIDs: []string{"key1", "key2"},
			deleteResponses: map[string]int{
				"key1": 500,
				"key2": 500,
			},
			expectedDeletedKeys: []string{"key1", "key2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletedKeys := []string{}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE method, got %s", r.Method)
				}

				// Extract key ID from path like /tls/private_keys/{id}
				path := r.URL.Path
				if len(path) < 18 { // "/tls/private_keys/" is 18 chars
					t.Errorf("Invalid path: %s", path)
					return
				}
				keyID := path[18:] // Extract ID after "/tls/private_keys/"

				// Track that this key was attempted to be deleted
				deletedKeys = append(deletedKeys, keyID)

				// Return the configured response for this key
				if statusCode, exists := tt.deleteResponses[keyID]; exists {
					w.WriteHeader(statusCode)
					if statusCode != 200 {
						w.Write([]byte("Error"))
					}
				} else {
					w.WriteHeader(404)
				}
			}))
			defer server.Close()

			// Create fastly client pointed at test server
			client, err := fastly.NewClientForEndpoint("test-key", server.URL)
			if err != nil {
				t.Fatalf("Failed to create fastly client: %v", err)
			}

			// Create Logic instance with real client pointed at mock server
			logic := &Logic{
				FastlyClient: client,
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
			if len(deletedKeys) != len(tt.expectedDeletedKeys) {
				t.Errorf("clearFastlyUnusedPrivateKeys() made %d delete calls, want %d",
					len(deletedKeys), len(tt.expectedDeletedKeys))
			}

			// Verify each expected call was made
			for i, expectedID := range tt.expectedDeletedKeys {
				if i >= len(deletedKeys) {
					t.Errorf("clearFastlyUnusedPrivateKeys() missing delete call %d for key %s", i, expectedID)
				} else if deletedKeys[i] != expectedID {
					t.Errorf("clearFastlyUnusedPrivateKeys() delete call %d = %s, want %s",
						i, deletedKeys[i], expectedID)
				}
			}
		})
	}
}
