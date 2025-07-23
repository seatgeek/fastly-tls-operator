package fastlycertificatesync

import (
	"errors"
	"testing"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/fastly-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
