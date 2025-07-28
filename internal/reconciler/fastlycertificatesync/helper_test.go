package fastlycertificatesync

import (
	"context"
	"errors"
	"strings"
	"testing"

	"bytes"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/fastly-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Mock function type for getCertificateAndTLSSecretFromSubject
type getCertificateAndTLSSecretFromSubjectFunc func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error)

// Helper to create a test context with necessary fields
func createTestContext() *Context {
	return &Context{
		Subject: &v1alpha1.FastlyCertificateSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cert-sync",
				Namespace: "test-namespace",
			},
			Spec: v1alpha1.FastlyCertificateSyncSpec{
				CertificateName: "test-certificate",
			},
		},
		Config: &Config{},
		Log:    logr.Discard(), // Use a no-op logger for tests
	}
}

// Helper to create a certificate with specific conditions
func createCertificateWithConditions(conditions []cmv1.CertificateCondition) *cmv1.Certificate {
	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-certificate",
			Namespace: "test-namespace",
		},
		Spec: cmv1.CertificateSpec{
			SecretName: "test-secret",
		},
		Status: cmv1.CertificateStatus{
			Conditions: conditions,
		},
	}
}

// Helper to create a mock secret
func createTestSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("test-cert-data"),
			"tls.key": []byte("test-key-data"),
		},
	}
}

// Test helper function that mimics isSubjectReadyForReconciliation but allows mocking
func isSubjectReadyForReconciliationWithMock(ctx *Context, mockGetCertAndSecret getCertificateAndTLSSecretFromSubjectFunc) bool {
	var certificate *cmv1.Certificate
	var err error
	if certificate, _, err = mockGetCertAndSecret(ctx); err != nil {
		ctx.Log.Info("Certificate and Secret not available, we will not reconcile this FastlyCertificateSync", "name", ctx.Subject.Name, "namespace", ctx.Subject.Namespace)
		return false
	}

	for _, condition := range certificate.Status.Conditions {
		if condition.Type == cmv1.CertificateConditionReady && condition.Status == cmmetav1.ConditionTrue {
			return true
		}
	}
	return false
}

func TestIsSubjectReadyForReconciliation(t *testing.T) {
	tests := []struct {
		name           string
		mockFunc       getCertificateAndTLSSecretFromSubjectFunc
		expectedResult bool
		description    string
	}{
		{
			name: "error_getting_certificate_and_secret",
			mockFunc: func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
				return nil, nil, errors.New("failed to get certificate")
			},
			expectedResult: false,
			description:    "Should return false when getCertificateAndTLSSecretFromSubject returns an error",
		},
		{
			name: "certificate_ready_condition_true",
			mockFunc: func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
				certificate := createCertificateWithConditions([]cmv1.CertificateCondition{
					{
						Type:   cmv1.CertificateConditionReady,
						Status: cmmetav1.ConditionTrue,
					},
				})
				secret := createTestSecret()
				return certificate, secret, nil
			},
			expectedResult: true,
			description:    "Should return true when certificate has ready condition with status True",
		},
		{
			name: "certificate_ready_condition_false",
			mockFunc: func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
				certificate := createCertificateWithConditions([]cmv1.CertificateCondition{
					{
						Type:   cmv1.CertificateConditionReady,
						Status: cmmetav1.ConditionFalse,
					},
				})
				secret := createTestSecret()
				return certificate, secret, nil
			},
			expectedResult: false,
			description:    "Should return false when certificate has ready condition with status False",
		},
		{
			name: "certificate_ready_condition_unknown",
			mockFunc: func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
				certificate := createCertificateWithConditions([]cmv1.CertificateCondition{
					{
						Type:   cmv1.CertificateConditionReady,
						Status: cmmetav1.ConditionUnknown,
					},
				})
				secret := createTestSecret()
				return certificate, secret, nil
			},
			expectedResult: false,
			description:    "Should return false when certificate has ready condition with status Unknown",
		},
		{
			name: "certificate_no_ready_condition",
			mockFunc: func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
				certificate := createCertificateWithConditions([]cmv1.CertificateCondition{
					{
						Type:   cmv1.CertificateConditionIssuing,
						Status: cmmetav1.ConditionTrue,
					},
				})
				secret := createTestSecret()
				return certificate, secret, nil
			},
			expectedResult: false,
			description:    "Should return false when certificate has no ready condition",
		},
		{
			name: "certificate_no_conditions_at_all",
			mockFunc: func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
				certificate := createCertificateWithConditions([]cmv1.CertificateCondition{})
				secret := createTestSecret()
				return certificate, secret, nil
			},
			expectedResult: false,
			description:    "Should return false when certificate has no conditions at all",
		},
		{
			name: "certificate_multiple_conditions_ready_true",
			mockFunc: func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
				certificate := createCertificateWithConditions([]cmv1.CertificateCondition{
					{
						Type:   cmv1.CertificateConditionIssuing,
						Status: cmmetav1.ConditionFalse,
					},
					{
						Type:   cmv1.CertificateConditionReady,
						Status: cmmetav1.ConditionTrue,
					},
					{
						Type:   "CustomCondition",
						Status: cmmetav1.ConditionUnknown,
					},
				})
				secret := createTestSecret()
				return certificate, secret, nil
			},
			expectedResult: true,
			description:    "Should return true when certificate has multiple conditions including ready=true",
		},
		{
			name: "certificate_multiple_conditions_ready_false",
			mockFunc: func(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
				certificate := createCertificateWithConditions([]cmv1.CertificateCondition{
					{
						Type:   cmv1.CertificateConditionIssuing,
						Status: cmmetav1.ConditionTrue,
					},
					{
						Type:   cmv1.CertificateConditionReady,
						Status: cmmetav1.ConditionFalse,
					},
					{
						Type:   "CustomCondition",
						Status: cmmetav1.ConditionTrue,
					},
				})
				secret := createTestSecret()
				return certificate, secret, nil
			},
			expectedResult: false,
			description:    "Should return false when certificate has multiple conditions but ready=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context
			ctx := createTestContext()

			// Run the test using our test helper that accepts a mock function
			result := isSubjectReadyForReconciliationWithMock(ctx, tt.mockFunc)

			// Check the result
			if result != tt.expectedResult {
				t.Errorf("isSubjectReadyForReconciliation() = %v, want %v. %s", result, tt.expectedResult, tt.description)
			}
		})
	}
}

func TestGetCertificateAndTLSSecretFromSubject(t *testing.T) {
	tests := []struct {
		name               string
		setupObjects       []client.Object // K8s objects to create in fake client
		expectedError      string          // Expected error message substring (empty if no error expected)
		expectedCertName   string          // Expected certificate name in result
		expectedSecretName string          // Expected secret name in result
		description        string          // Test case description
	}{
		{
			name: "success_certificate_and_secret_exist",
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
						"tls.key": []byte("test-key-data"),
					},
				},
			},
			expectedCertName:   "test-certificate",
			expectedSecretName: "test-secret",
			description:        "Should successfully return certificate and secret when both exist",
		},
		{
			name: "certificate_not_found",
			setupObjects: []client.Object{
				// No certificate object - should fail on first Get()
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.crt": []byte("test-cert-data"),
						"tls.key": []byte("test-key-data"),
					},
				},
			},
			expectedError: "failed to get certificate of name test-certificate and namespace test-namespace",
			description:   "Should return error when certificate does not exist",
		},
		{
			name: "secret_not_found",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "missing-secret", // This secret doesn't exist
					},
				},
				// No secret object matching the certificate's secretName
			},
			expectedError: "failed to get secret of name missing-secret and namespace test-namespace",
			description:   "Should return error when secret referenced by certificate does not exist",
		},
		{
			name: "certificate_and_secret_different_namespaces",
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
						Namespace: "different-namespace", // Different namespace
					},
					Data: map[string][]byte{
						"tls.crt": []byte("test-cert-data"),
						"tls.key": []byte("test-key-data"),
					},
				},
			},
			expectedError: "failed to get secret of name test-secret and namespace test-namespace",
			description:   "Should return error when secret is in different namespace than certificate",
		},
		{
			name: "certificate_in_different_namespace_than_subject",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "different-namespace", // Different from subject namespace
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "different-namespace",
					},
					Data: map[string][]byte{
						"tls.crt": []byte("test-cert-data"),
						"tls.key": []byte("test-key-data"),
					},
				},
			},
			expectedError: "failed to get certificate of name test-certificate and namespace test-namespace",
			description:   "Should use subject namespace to find certificate, not certificate's own namespace",
		},

		{
			name: "empty_certificate_spec",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						// SecretName is empty - will try to find secret with empty name
					},
				},
			},
			expectedError: "failed to get secret of name  and namespace test-namespace",
			description:   "Should return error when certificate has empty SecretName",
		},
		{
			name: "multiple_certificates_and_secrets",
			setupObjects: []client.Object{
				// Target certificate and secret
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
						"tls.key": []byte("test-key-data"),
					},
				},
				// Extra objects that should be ignored
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "other-secret",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"tls.crt": []byte("other-cert-data"),
						"tls.key": []byte("other-key-data"),
					},
				},
			},
			expectedCertName:   "test-certificate",
			expectedSecretName: "test-secret",
			description:        "Should find correct certificate and secret among multiple objects",
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
			certificate, secret, err := getCertificateAndTLSSecretFromSubject(ctx)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("getCertificateAndTLSSecretFromSubject() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("getCertificateAndTLSSecretFromSubject() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return // Don't check results if we expected an error
			}

			if err != nil {
				t.Errorf("getCertificateAndTLSSecretFromSubject() unexpected error = %v", err)
				return
			}

			// Check certificate result
			if certificate == nil {
				t.Errorf("getCertificateAndTLSSecretFromSubject() returned nil certificate")
				return
			}
			if certificate.Name != tt.expectedCertName {
				t.Errorf("getCertificateAndTLSSecretFromSubject() certificate name = %q, want %q", certificate.Name, tt.expectedCertName)
			}

			// Check secret result
			if secret == nil {
				t.Errorf("getCertificateAndTLSSecretFromSubject() returned nil secret")
				return
			}
			if secret.Name != tt.expectedSecretName {
				t.Errorf("getCertificateAndTLSSecretFromSubject() secret name = %q, want %q", secret.Name, tt.expectedSecretName)
			}

			// Verify certificate and secret namespaces are consistent
			if certificate.Namespace != secret.Namespace {
				t.Errorf("getCertificateAndTLSSecretFromSubject() certificate namespace %q != secret namespace %q", certificate.Namespace, secret.Namespace)
			}
		})
	}
}

func TestGetCertPEMForSecret(t *testing.T) {
	// Dummy PEM values for testing - actual format doesn't matter for these tests
	dummyCertPEM := []byte(`-----BEGIN CERTIFICATE-----
MIICertificateDataHere
-----END CERTIFICATE-----`)
	dummyCACertPEM := []byte(`-----BEGIN CERTIFICATE-----
MIICACertificateDataHere
-----END CERTIFICATE-----`)
	expectedCombinedPEM := append(dummyCertPEM, dummyCACertPEM...)

	tests := []struct {
		name                    string
		secret                  *corev1.Secret
		hackLocalReconciliation bool   // Value for HackFastlyCertificateSyncLocalReconciliation
		expectedPEM             []byte // Expected returned PEM data
		expectedError           string // Expected error message substring (empty if no error expected)
		description             string // Test case description
	}{
		{
			name: "production_mode_success_tls_cert_only",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": dummyCertPEM,
					"tls.key": []byte("dummy-key-data"),
				},
			},
			hackLocalReconciliation: false,
			expectedPEM:             dummyCertPEM,
			description:             "Should return only tls.crt in production mode",
		},
		{
			name: "local_mode_success_with_ca_cert",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": dummyCertPEM,
					"ca.crt":  dummyCACertPEM,
					"tls.key": []byte("dummy-key-data"),
				},
			},
			hackLocalReconciliation: true,
			expectedPEM:             expectedCombinedPEM,
			description:             "Should return combined tls.crt + ca.crt in local mode",
		},
		{
			name: "production_mode_success_ignores_ca_cert",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": dummyCertPEM,
					"ca.crt":  dummyCACertPEM, // Present but should be ignored in production mode
					"tls.key": []byte("dummy-key-data"),
				},
			},
			hackLocalReconciliation: false,
			expectedPEM:             dummyCertPEM, // Should only return tls.crt, not combined
			description:             "Should ignore ca.crt in production mode even when present",
		},
		{
			name: "missing_tls_crt_production_mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.key": []byte("dummy-key-data"),
					"ca.crt":  dummyCACertPEM,
				},
			},
			hackLocalReconciliation: false,
			expectedError:           "secret test-namespace/test-secret does not contain tls.crt",
			description:             "Should return error when tls.crt is missing in production mode",
		},
		{
			name: "missing_tls_crt_local_mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.key": []byte("dummy-key-data"),
					"ca.crt":  dummyCACertPEM,
				},
			},
			hackLocalReconciliation: true,
			expectedError:           "secret test-namespace/test-secret does not contain tls.crt",
			description:             "Should return error when tls.crt is missing in local mode",
		},
		{
			name: "local_mode_missing_ca_crt",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": dummyCertPEM,
					"tls.key": []byte("dummy-key-data"),
					// ca.crt is missing
				},
			},
			hackLocalReconciliation: true,
			expectedError:           "secret test-namespace/test-secret does not contain ca.crt",
			description:             "Should return error when ca.crt is missing in local mode",
		},
		{
			name: "empty_secret_data_production_mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{}, // Empty data
			},
			hackLocalReconciliation: false,
			expectedError:           "secret test-namespace/test-secret does not contain tls.crt",
			description:             "Should return error when secret has no data in production mode",
		},
		{
			name: "empty_secret_data_local_mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{}, // Empty data
			},
			hackLocalReconciliation: true,
			expectedError:           "secret test-namespace/test-secret does not contain tls.crt",
			description:             "Should return error when secret has no data in local mode",
		},
		{
			name: "nil_secret_data_production_mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: nil, // Nil data
			},
			hackLocalReconciliation: false,
			expectedError:           "secret test-namespace/test-secret does not contain tls.crt",
			description:             "Should return error when secret has nil data in production mode",
		},
		{
			name: "empty_tls_crt_value_production_mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": []byte{}, // Empty but present
					"tls.key": []byte("dummy-key-data"),
				},
			},
			hackLocalReconciliation: false,
			expectedPEM:             []byte{}, // Should return empty byte slice, not error
			description:             "Should return empty PEM when tls.crt is empty but present in production mode",
		},
		{
			name: "empty_ca_crt_value_local_mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": dummyCertPEM,
					"ca.crt":  []byte{}, // Empty but present
					"tls.key": []byte("dummy-key-data"),
				},
			},
			hackLocalReconciliation: true,
			expectedPEM:             dummyCertPEM, // Should return just tls.crt since ca.crt is empty
			description:             "Should handle empty ca.crt value in local mode",
		},
		{
			name: "secret_with_different_namespace",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-name",
					Namespace: "different-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": dummyCertPEM,
					"tls.key": []byte("dummy-key-data"),
				},
			},
			hackLocalReconciliation: false,
			expectedPEM:             dummyCertPEM,
			description:             "Should work with secrets from any namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context
			ctx := createTestContext()
			ctx.Config.HackFastlyCertificateSyncLocalReconciliation = tt.hackLocalReconciliation

			// Call the function under test
			result, err := getCertPEMForSecret(ctx, tt.secret)

			// Check error expectation
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("getCertPEMForSecret() expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("getCertPEMForSecret() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return // Don't check result if we expected an error
			}

			if err != nil {
				t.Errorf("getCertPEMForSecret() unexpected error = %v", err)
				return
			}

			// Check result
			if !bytes.Equal(result, tt.expectedPEM) {
				t.Errorf("getCertPEMForSecret() result = %q, want %q", result, tt.expectedPEM)
			}

			// Additional validation for local mode with CA cert
			if tt.hackLocalReconciliation && tt.expectedError == "" {
				// Verify the result contains both parts when expected
				if len(tt.expectedPEM) > len(dummyCertPEM) {
					// Should contain both cert and CA cert
					if !bytes.Contains(result, dummyCertPEM) {
						t.Errorf("getCertPEMForSecret() result should contain tls.crt data")
					}
					if len(tt.secret.Data["ca.crt"]) > 0 && !bytes.Contains(result, dummyCACertPEM) {
						t.Errorf("getCertPEMForSecret() result should contain ca.crt data")
					}
				}
			}
		})
	}
}
