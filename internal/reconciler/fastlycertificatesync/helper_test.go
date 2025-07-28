package fastlycertificatesync

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"
	"testing"

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

func TestIsSubjectReadyForReconciliation(t *testing.T) {
	tests := []struct {
		name           string
		setupObjects   []client.Object // K8s objects to create in fake client
		expectedResult bool
		description    string
	}{
		{
			name:           "error_getting_certificate_and_secret",
			setupObjects:   []client.Object{}, // No objects - should fail to find certificate
			expectedResult: false,
			description:    "Should return false when getCertificateAndTLSSecretFromSubject returns an error",
		},
		{
			name: "certificate_ready_condition_true",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
					Status: cmv1.CertificateStatus{
						Conditions: []cmv1.CertificateCondition{
							{
								Type:   cmv1.CertificateConditionReady,
								Status: cmmetav1.ConditionTrue,
							},
						},
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
			expectedResult: true,
			description:    "Should return true when certificate has ready condition with status True",
		},
		{
			name: "certificate_ready_condition_false",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
					Status: cmv1.CertificateStatus{
						Conditions: []cmv1.CertificateCondition{
							{
								Type:   cmv1.CertificateConditionReady,
								Status: cmmetav1.ConditionFalse,
							},
						},
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
			expectedResult: false,
			description:    "Should return false when certificate has ready condition with status False",
		},
		{
			name: "certificate_ready_condition_unknown",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
					Status: cmv1.CertificateStatus{
						Conditions: []cmv1.CertificateCondition{
							{
								Type:   cmv1.CertificateConditionReady,
								Status: cmmetav1.ConditionUnknown,
							},
						},
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
			expectedResult: false,
			description:    "Should return false when certificate has ready condition with status Unknown",
		},
		{
			name: "certificate_no_ready_condition",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
					Status: cmv1.CertificateStatus{
						Conditions: []cmv1.CertificateCondition{
							{
								Type:   cmv1.CertificateConditionIssuing,
								Status: cmmetav1.ConditionTrue,
							},
						},
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
			expectedResult: false,
			description:    "Should return false when certificate has no ready condition",
		},
		{
			name: "certificate_no_conditions_at_all",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
					Status: cmv1.CertificateStatus{
						Conditions: []cmv1.CertificateCondition{},
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
			expectedResult: false,
			description:    "Should return false when certificate has no conditions at all",
		},
		{
			name: "certificate_multiple_conditions_ready_true",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
					Status: cmv1.CertificateStatus{
						Conditions: []cmv1.CertificateCondition{
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
						},
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
			expectedResult: true,
			description:    "Should return true when certificate has multiple conditions including ready=true",
		},
		{
			name: "certificate_multiple_conditions_ready_false",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "test-secret",
					},
					Status: cmv1.CertificateStatus{
						Conditions: []cmv1.CertificateCondition{
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
						},
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
			expectedResult: false,
			description:    "Should return false when certificate has multiple conditions but ready=false",
		},
		{
			name: "certificate_exists_but_secret_missing",
			setupObjects: []client.Object{
				&cmv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-certificate",
						Namespace: "test-namespace",
					},
					Spec: cmv1.CertificateSpec{
						SecretName: "missing-secret", // This secret doesn't exist
					},
					Status: cmv1.CertificateStatus{
						Conditions: []cmv1.CertificateCondition{
							{
								Type:   cmv1.CertificateConditionReady,
								Status: cmmetav1.ConditionTrue,
							},
						},
					},
				},
				// No secret object - should fail when trying to get the secret
			},
			expectedResult: false,
			description:    "Should return false when certificate exists but referenced secret is missing",
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

			// Call the actual function under test
			result := isSubjectReadyForReconciliation(ctx)

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
			name: "empty_ca_crt_value_local_mode",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": dummyCertPEM,
					"tls.key": []byte("dummy-key-data"),
					"ca.crt":  {},
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

func TestGetPublicKeySHA1FromPEM(t *testing.T) {
	// TEST DATA EXPLANATION:
	// The following RSA private keys are real test keys generated specifically for testing purposes.
	// These are NOT production keys and were created solely for this test using `openssl genrsa <size>`.
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
			name: "valid_2048_bit_rsa_key",
			privateKeyPEM: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA9D/xVwc+b9nUzXokFkKmXoaLghXGnwGfcoj/trlvEmS40Gz2
YVO/Kf9DEBo5UKHtsTJ98lSxX/P2fm1Z9hz6fsKxZJPk3lMKhpo74DcOu0p/3TgI
YySckz3KmZ4euvAmQ03wUSj8FIsxuexlpL2TzQFB3tv4osvPZ5C0JGMp0UOffpYY
YmLD96BIznPEIcXTbFFq5RHhXV3ictNCKczkDb3Hzk13zkcf9VT5WUxtOsf94kfh
xbjQTpVSojXOKhUr/AXJAtP19c8IPs0B6cNa2AFWQWM0KoSEDBJy0fiXSmqQJyhS
wbmCjLyhnVtl+H1SwKUNXrk6sBBGB7nXBu8giQIDAQABAoIBAEQ8XOsoTewnmgjx
l4VUh3Ae/HiSJtQjOu1fkrj0ozArTWqFFmvoXp6X/p9QBDUfl+0KIx+BQ7B/0pxN
ZnWYcO7a634ixyzJXEZwbkvcddQjIwelcMpp3whPmftCrmkhUD87VekGny4KGRFN
FrRodhMux71ADP1GHSJczcbgoT0hsIRrGQ6BliwahLWQommRPOduHF/rIHeRkMh7
FgmhmYAB26OTrZWx4kzFHwJmCOejpLVBuf+txl8QbuKPhZYyTCj5/ZEuR9dtZPPv
ePY8/5X3I0vPFFFhqjrwpNGTsGG817yj0Aa8KFSPJd7kQV+y9z4adzPeEZkMNxdx
wcf+POUCgYEA+qjFaWacVc7SOAMWzSq8oJ8ztSVPz5gEQ26DBVtcu6FKLrtyg9pm
VOOA1a0g2LXf7dwzsBFTqwTJil+Plqg5A0WoQ4/EkW0ks5+gb7AYruaA80+Hyf0p
Sn5iaUzcj6pG6BeIAFlY8Bc+fkqlO3AhdEOy9I63fMaBdjqKvLXag08CgYEA+XQ1
+lbN35LELsyL/LV23kFrdCTpiSe24KJ3xZy1YcUmS5sw9IVM48DIMAodsQwidhvw
4rMZEV2q3m+qkC+wMavOgPkGkdOF0VadUzYoX8FAuM9mU3pFy6iwY96sHevSXxE2
aVSzo7BhCPdSe73rfrkZ5mNY7aVa9ruRjFvnCKcCgYA90JUukxGGz8Rj788Vta5i
5h/4UkVGarTSdFR3Y7qQwwvqTmvFPHzz/k7tYw6wotmgbSeKChvaFwokx8A/ZSj6
N5lxX+kX/BSK/5ivMnxD1bCDUF+qXnZqWpSmZ0AVZeaqofL2MxKN0w2kU4BAEj0N
0Qw252M0sDeJEpLYSviiXQKBgQC8CQ57Mx2izuYVBNjs1/jPVm7iMMTdP1OKBs3T
5umO1makjUoct7Ka54G/PJDfGW+MqkktCaX2wi1/2Jqwb1IYTxKtg4mhONnhT7Ht
vKA4ddsMtEHE4SFlgDXeQkZpk46TXM0wHsn+tICgmpXRcvrmHi9YzECHeqKT5BW1
wLzpdwKBgQCHg6TNvxsPBsAJOi1ctojH94v/+BGFrSIIf+vGF1+FBI9lC6npUHV+
hhBFOVY9O4otug51NwcSBSuKUP2IoqSmUCbb9clLxWe7z+9dVUyExApcpoasxUgi
bcnHmgKjOizDALU0wp6J8ytldGD5lkGudp2m/zpT2J4ylQiJvlRNOw==
-----END RSA PRIVATE KEY-----`,
			expectedSHA1: "7945c3df946cb5a63720dd4faa2317489b84f576",
		},
		{
			name: "pem_with_extra_whitespace_and_newlines",
			privateKeyPEM: `
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`,
			expectedSHA1: "1ccf8849ae82aaab5749d5c791a221354f182a73", // Same as first key since it's the same key with whitespace
		},
		{
			name:          "empty_input",
			privateKeyPEM: "",
			expectError:   true,
			errorContains: "failed to parse PEM block",
		},
		{
			name:          "nil_equivalent_empty_bytes",
			privateKeyPEM: "",
			expectError:   true,
			errorContains: "failed to parse PEM block",
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
		{
			name: "wrong_pem_block_type_public_key",
			privateKeyPEM: `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDSIX1v14YXhBhoXs4xMDFaqcw0
BzFGN9BUetq4xCX0RQjOgwutEVAQg+zqSwRzW0eQsNuWQBX0qFlNQSxtE5/Bt0mr
9Vh5VTePHAj+kLqAWYwzpRK/IN8oOndsvTNJQHhHWPcnopJTIB+ktuBJpqjDVn6t
HmXIj2hYA9/AQJ4BywIDAQAB
-----END PUBLIC KEY-----`,
			expectError:   true,
			errorContains: "failed to parse RSA private key",
		},
		{
			name: "wrong_pem_block_type_certificate",
			privateKeyPEM: `-----BEGIN CERTIFICATE-----
MIICljCCAX4CCQDKg+l5v7nBKTANBgkqhkiG9w0BAQsFADANMQswCQYDVQQGEwJV
UzAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMA0xCzAJBgNVBAYTAlVT
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAx7V3J2qK4V7Qe4UYbGzK
8pXyDjNHk9Ij2lGQk8p3LVEhQ4Xm7B9n1s8Z5cQgHxJtPl7V4LxB6J3R8Zt9Wq2K
9Xm7VXbZ2s1C7qPdR3VdZm8K1qYf2jQr6L5pN8sXc9lKxvPz4Rd2sGqJc8Y7VtZm
Xc3r1mKs8RzLp9D4GcV2qB7lQ4Rt8sLq6XqVj9vfKlM3cYp7C2wJ8gTdZp3QzVxN
7Rq4YsX8V5Z2wK6LmJ4X9qV2B8Fy1nP7jRzZgL3M5vQpKx8dV7C2Bx6ZfXr8LnJ1
sQm4Yc8RzM2N7VjK6Qp8Lf4XzWbQc5T1dYv8Mx6K9R7VzF3J4H8XwYpQ5D2BZ9Lz
KwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQABCDEFGHIJKLMNOPQRSTUVWXYZabcd
-----END CERTIFICATE-----`,
			expectError:   true,
			errorContains: "failed to parse RSA private key",
		},
		{
			name: "multiple_pem_blocks_should_use_first",
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
-----END RSA PRIVATE KEY-----
-----BEGIN RSA PRIVATE KEY-----
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
			expectedSHA1: "1ccf8849ae82aaab5749d5c791a221354f182a73", // Should use first PEM block
		},
		{
			name: "pem_with_comments_before_key",
			privateKeyPEM: `# This is a comment
# Another comment
-----BEGIN RSA PRIVATE KEY-----
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
			expectedSHA1: "1ccf8849ae82aaab5749d5c791a221354f182a73", // Same as first key
		},
		{
			name: "corrupted_rsa_key_structure",
			privateKeyPEM: `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDSIX1v14YXhBhoXs4xMDFaqcw0BzFGN9BUetq4xCX0RQjOgwut
CORRUPTED_DATA_HERE_THAT_BREAKS_ASN1_STRUCTURE
EVAQg+zqSwRzW0eQsNuWQBX0qFlNQSxtE5/Bt0mr9Vh5VTePHAj+kLqAWYwzpRK/
IN8oOndsvTNJQHhHWPcnopJTIB+ktuBJpqjDVn6tHmXIj2hYA9/AQJ4BywIDAQAB
-----END RSA PRIVATE KEY-----`,
			expectError:   true,
			errorContains: "failed to parse PEM block", // Base64 decode fails first
		},
		{
			name: "truncated_rsa_key",
			privateKeyPEM: `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDSIX1v14YXhBhoXs4xMDFaqcw0BzFGN9BUetq4xCX0RQjOgwut
EVAQg+zqSwRzW0eQsNuWQBX0qFlNQSxtE5/Bt0mr9Vh5VTePHAj+kLqAWYwzpRK/
-----END RSA PRIVATE KEY-----`,
			expectError:   true,
			errorContains: "failed to parse RSA private key",
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

	// Additional test for nil input (since Go treats nil and empty slices differently in some contexts)
	t.Run("nil_input", func(t *testing.T) {
		result, err := getPublicKeySHA1FromPEM(nil)
		if err == nil {
			t.Error("getPublicKeySHA1FromPEM() with nil input expected error but got nil")
		} else if !strings.Contains(err.Error(), "failed to parse PEM block") {
			t.Errorf("getPublicKeySHA1FromPEM() with nil input error = %v, want error containing %q", err, "failed to parse PEM block")
		}
		if result != "" {
			t.Errorf("getPublicKeySHA1FromPEM() with nil input result = %s, want empty string", result)
		}
	})
}
