package fastlycertificatesync

import (
	"testing"

	"github.com/fastly-operator/api/v1alpha1"
	"github.com/fastly/go-fastly/v11/fastly"
	"github.com/go-logr/logr"
	"github.com/seatgeek/k8s-reconciler-generic/apiobjects"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/genrec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLogic_FillStatus(t *testing.T) {
	tests := []struct {
		name               string
		observedState      ObservedState
		expectedReady      bool
		expectedConditions map[string]struct {
			status  metav1.ConditionStatus
			reason  string
			message string
		}
	}{
		{
			name: "initial_state_nothing_done",
			observedState: ObservedState{
				PrivateKeyUploaded:       false,
				CertificateStatus:        CertificateStatusMissing,
				UnusedPrivateKeyIDs:      []string{},
				MissingTLSActivationData: []TLSActivationData{},
				ExtraTLSActivationIDs:    []string{},
			},
			expectedReady: false,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"PrivateKeyReady": {
					status:  metav1.ConditionFalse,
					reason:  "PrivateKeyMissing",
					message: "Private key needs to be uploaded to Fastly",
				},
				"CertificateReady": {
					status:  metav1.ConditionFalse,
					reason:  "CertificateMissing",
					message: "Certificate is missing from Fastly and needs to be created",
				},
				"TLSActivationReady": {
					status:  metav1.ConditionTrue,
					reason:  "TLSActivationsSynced",
					message: "All TLS activations are properly configured",
				},
				"CleanupRequired": {
					status:  metav1.ConditionFalse,
					reason:  "NoCleanupNeeded",
					message: "No unused private keys found",
				},
				"Ready": {
					status:  metav1.ConditionFalse,
					reason:  "FastlySyncIncomplete",
					message: "FastlyCertificateSync is not ready - synchronization in progress",
				},
			},
		},
		{
			name: "private_key_uploaded_certificate_missing",
			observedState: ObservedState{
				PrivateKeyUploaded:       true,
				CertificateStatus:        CertificateStatusMissing,
				UnusedPrivateKeyIDs:      []string{},
				MissingTLSActivationData: []TLSActivationData{},
				ExtraTLSActivationIDs:    []string{},
			},
			expectedReady: false,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"PrivateKeyReady": {
					status:  metav1.ConditionTrue,
					reason:  "PrivateKeyUploaded",
					message: "Private key has been successfully uploaded to Fastly",
				},
				"CertificateReady": {
					status:  metav1.ConditionFalse,
					reason:  "CertificateMissing",
					message: "Certificate is missing from Fastly and needs to be created",
				},
				"Ready": {
					status:  metav1.ConditionFalse,
					reason:  "FastlySyncIncomplete",
					message: "FastlyCertificateSync is not ready - synchronization in progress",
				},
			},
		},
		{
			name: "private_key_uploaded_certificate_stale",
			observedState: ObservedState{
				PrivateKeyUploaded:       true,
				CertificateStatus:        CertificateStatusStale,
				UnusedPrivateKeyIDs:      []string{},
				MissingTLSActivationData: []TLSActivationData{},
				ExtraTLSActivationIDs:    []string{},
			},
			expectedReady: false,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"PrivateKeyReady": {
					status:  metav1.ConditionTrue,
					reason:  "PrivateKeyUploaded",
					message: "Private key has been successfully uploaded to Fastly",
				},
				"CertificateReady": {
					status:  metav1.ConditionFalse,
					reason:  "CertificateStale",
					message: "Certificate exists in Fastly but is stale and needs to be updated",
				},
				"Ready": {
					status:  metav1.ConditionFalse,
					reason:  "FastlySyncIncomplete",
					message: "FastlyCertificateSync is not ready - synchronization in progress",
				},
			},
		},
		{
			name: "private_key_and_certificate_synced_missing_tls_activations",
			observedState: ObservedState{
				PrivateKeyUploaded:  true,
				CertificateStatus:   CertificateStatusSynced,
				UnusedPrivateKeyIDs: []string{},
				MissingTLSActivationData: []TLSActivationData{
					{
						Certificate:   &fastly.CustomTLSCertificate{ID: "cert1"},
						Configuration: &fastly.TLSConfiguration{ID: "config1"},
						Domain:        &fastly.TLSDomain{ID: "domain1"},
					},
					{
						Certificate:   &fastly.CustomTLSCertificate{ID: "cert1"},
						Configuration: &fastly.TLSConfiguration{ID: "config2"},
						Domain:        &fastly.TLSDomain{ID: "domain2"},
					},
				},
				ExtraTLSActivationIDs: []string{},
			},
			expectedReady: false,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"PrivateKeyReady": {
					status:  metav1.ConditionTrue,
					reason:  "PrivateKeyUploaded",
					message: "Private key has been successfully uploaded to Fastly",
				},
				"CertificateReady": {
					status:  metav1.ConditionTrue,
					reason:  "CertificateSynced",
					message: "Certificate is up-to-date and synced with Fastly",
				},
				"TLSActivationReady": {
					status:  metav1.ConditionFalse,
					reason:  "TLSActivationsMissing",
					message: "Missing 2 TLS activations that need to be created",
				},
				"Ready": {
					status:  metav1.ConditionFalse,
					reason:  "FastlySyncIncomplete",
					message: "FastlyCertificateSync is not ready - synchronization in progress",
				},
			},
		},
		{
			name: "private_key_and_certificate_synced_extra_tls_activations",
			observedState: ObservedState{
				PrivateKeyUploaded:       true,
				CertificateStatus:        CertificateStatusSynced,
				UnusedPrivateKeyIDs:      []string{},
				MissingTLSActivationData: []TLSActivationData{},
				ExtraTLSActivationIDs:    []string{"activation1", "activation2", "activation3"},
			},
			expectedReady: false,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"PrivateKeyReady": {
					status:  metav1.ConditionTrue,
					reason:  "PrivateKeyUploaded",
					message: "Private key has been successfully uploaded to Fastly",
				},
				"CertificateReady": {
					status:  metav1.ConditionTrue,
					reason:  "CertificateSynced",
					message: "Certificate is up-to-date and synced with Fastly",
				},
				"TLSActivationReady": {
					status:  metav1.ConditionFalse,
					reason:  "TLSActivationsExtra",
					message: "Found 3 extra TLS activations that need to be removed",
				},
				"Ready": {
					status:  metav1.ConditionFalse,
					reason:  "FastlySyncIncomplete",
					message: "FastlyCertificateSync is not ready - synchronization in progress",
				},
			},
		},
		{
			name: "everything_synced_but_unused_private_keys_need_cleanup",
			observedState: ObservedState{
				PrivateKeyUploaded:       true,
				CertificateStatus:        CertificateStatusSynced,
				UnusedPrivateKeyIDs:      []string{"key1", "key2"},
				MissingTLSActivationData: []TLSActivationData{},
				ExtraTLSActivationIDs:    []string{},
			},
			expectedReady: false,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"PrivateKeyReady": {
					status:  metav1.ConditionTrue,
					reason:  "PrivateKeyUploaded",
					message: "Private key has been successfully uploaded to Fastly",
				},
				"CertificateReady": {
					status:  metav1.ConditionTrue,
					reason:  "CertificateSynced",
					message: "Certificate is up-to-date and synced with Fastly",
				},
				"TLSActivationReady": {
					status:  metav1.ConditionTrue,
					reason:  "TLSActivationsSynced",
					message: "All TLS activations are properly configured",
				},
				"CleanupRequired": {
					status:  metav1.ConditionTrue,
					reason:  "UnusedPrivateKeysFound",
					message: "Found 2 unused private keys that should be cleaned up",
				},
				"Ready": {
					status:  metav1.ConditionFalse,
					reason:  "FastlySyncIncomplete",
					message: "FastlyCertificateSync is not ready - synchronization in progress",
				},
			},
		},
		{
			name: "fully_ready_everything_synced",
			observedState: ObservedState{
				PrivateKeyUploaded:       true,
				CertificateStatus:        CertificateStatusSynced,
				UnusedPrivateKeyIDs:      []string{},
				MissingTLSActivationData: []TLSActivationData{},
				ExtraTLSActivationIDs:    []string{},
			},
			expectedReady: true,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"PrivateKeyReady": {
					status:  metav1.ConditionTrue,
					reason:  "PrivateKeyUploaded",
					message: "Private key has been successfully uploaded to Fastly",
				},
				"CertificateReady": {
					status:  metav1.ConditionTrue,
					reason:  "CertificateSynced",
					message: "Certificate is up-to-date and synced with Fastly",
				},
				"TLSActivationReady": {
					status:  metav1.ConditionTrue,
					reason:  "TLSActivationsSynced",
					message: "All TLS activations are properly configured",
				},
				"CleanupRequired": {
					status:  metav1.ConditionFalse,
					reason:  "NoCleanupNeeded",
					message: "No unused private keys found",
				},
				"Ready": {
					status:  metav1.ConditionTrue,
					reason:  "FastlySyncComplete",
					message: "FastlyCertificateSync is ready and all components are synchronized",
				},
			},
		},
		{
			name: "mixed_scenario_missing_and_extra_tls_activations",
			observedState: ObservedState{
				PrivateKeyUploaded:  true,
				CertificateStatus:   CertificateStatusSynced,
				UnusedPrivateKeyIDs: []string{},
				MissingTLSActivationData: []TLSActivationData{
					{
						Certificate:   &fastly.CustomTLSCertificate{ID: "cert1"},
						Configuration: &fastly.TLSConfiguration{ID: "config1"},
						Domain:        &fastly.TLSDomain{ID: "domain1"},
					},
				},
				ExtraTLSActivationIDs: []string{"activation1"},
			},
			expectedReady: false,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"TLSActivationReady": {
					status:  metav1.ConditionFalse,
					reason:  "TLSActivationsMissing", // Missing takes precedence in the condition logic
					message: "Missing 1 TLS activations that need to be created",
				},
				"Ready": {
					status:  metav1.ConditionFalse,
					reason:  "FastlySyncIncomplete",
					message: "FastlyCertificateSync is not ready - synchronization in progress",
				},
			},
		},
		{
			name: "complex_scenario_multiple_issues",
			observedState: ObservedState{
				PrivateKeyUploaded:  true,
				CertificateStatus:   CertificateStatusStale,
				UnusedPrivateKeyIDs: []string{"key1", "key2", "key3"},
				MissingTLSActivationData: []TLSActivationData{
					{
						Certificate:   &fastly.CustomTLSCertificate{ID: "cert1"},
						Configuration: &fastly.TLSConfiguration{ID: "config1"},
						Domain:        &fastly.TLSDomain{ID: "domain1"},
					},
				},
				ExtraTLSActivationIDs: []string{"activation1", "activation2"},
			},
			expectedReady: false,
			expectedConditions: map[string]struct {
				status  metav1.ConditionStatus
				reason  string
				message string
			}{
				"PrivateKeyReady": {
					status:  metav1.ConditionTrue,
					reason:  "PrivateKeyUploaded",
					message: "Private key has been successfully uploaded to Fastly",
				},
				"CertificateReady": {
					status:  metav1.ConditionFalse,
					reason:  "CertificateStale",
					message: "Certificate exists in Fastly but is stale and needs to be updated",
				},
				"TLSActivationReady": {
					status:  metav1.ConditionFalse,
					reason:  "TLSActivationsMissing",
					message: "Missing 1 TLS activations that need to be created",
				},
				"CleanupRequired": {
					status:  metav1.ConditionTrue,
					reason:  "UnusedPrivateKeysFound",
					message: "Found 3 unused private keys that should be cleaned up",
				},
				"Ready": {
					status:  metav1.ConditionFalse,
					reason:  "FastlySyncIncomplete",
					message: "FastlyCertificateSync is not ready - synchronization in progress",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context with minimal setup
			ctx := &Context{
				Subject: &v1alpha1.FastlyCertificateSync{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cert-sync",
						Namespace: "test-namespace",
					},
					Status: v1alpha1.FastlyCertificateSyncStatus{},
				},
				Config: &Config{},
				Log:    logr.Discard(),
			}

			// Create logic with the test observed state
			logic := &Logic{
				ObservedState: tt.observedState,
			}

			// Call FillStatus
			err := logic.FillStatus(ctx, genrec.Resources{}, apiobjects.SubjectStatus{})
			require.NoError(t, err)

			// Verify Ready field
			assert.Equal(t, tt.expectedReady, ctx.Subject.Status.Ready, "Ready field should match expected value")

			// Verify expected conditions are present with correct values
			for conditionType, expected := range tt.expectedConditions {
				t.Run("condition_"+conditionType, func(t *testing.T) {
					var actualCondition *metav1.Condition
					for i := range ctx.Subject.Status.Conditions {
						if ctx.Subject.Status.Conditions[i].Type == conditionType {
							actualCondition = &ctx.Subject.Status.Conditions[i]
							break
						}
					}

					require.NotNil(t, actualCondition, "Condition %s should be present", conditionType)
					assert.Equal(t, expected.status, actualCondition.Status, "Condition %s status should match", conditionType)
					assert.Equal(t, expected.reason, actualCondition.Reason, "Condition %s reason should match", conditionType)
					assert.Equal(t, expected.message, actualCondition.Message, "Condition %s message should match", conditionType)
				})
			}

			// Verify all conditions have LastTransitionTime set (basic validation)
			for _, condition := range ctx.Subject.Status.Conditions {
				assert.False(t, condition.LastTransitionTime.IsZero(), "Condition %s should have LastTransitionTime set", condition.Type)
			}
		})
	}
}

func TestLogic_FillStatusConditions_ErrorHandling(t *testing.T) {
	t.Run("condition_generator_returns_error", func(t *testing.T) {
		ctx := &Context{
			Subject: &v1alpha1.FastlyCertificateSync{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cert-sync",
					Namespace: "test-namespace",
				},
				Status: v1alpha1.FastlyCertificateSyncStatus{},
			},
			Log: logr.Discard(),
		}

		logic := &Logic{}

		errorConditionFunc := func(ctx *Context) (*metav1.Condition, error) {
			return nil, assert.AnError
		}

		validConditionFunc := func(ctx *Context) (*metav1.Condition, error) {
			return &metav1.Condition{
				Type:   "TestCondition",
				Status: metav1.ConditionTrue,
				Reason: "TestReason",
			}, nil
		}

		// This should not return an error even if one of the condition generators fails
		err := logic.FillStatusConditions(ctx, errorConditionFunc, validConditionFunc)
		assert.NoError(t, err, "FillStatusConditions should not return error even if condition generator fails")

		// Should still have the valid condition
		assert.Len(t, ctx.Subject.Status.Conditions, 1)
		assert.Equal(t, "TestCondition", ctx.Subject.Status.Conditions[0].Type)
	})

	t.Run("condition_generator_returns_nil", func(t *testing.T) {
		ctx := &Context{
			Subject: &v1alpha1.FastlyCertificateSync{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cert-sync",
					Namespace: "test-namespace",
				},
				Status: v1alpha1.FastlyCertificateSyncStatus{},
			},
			Log: logr.Discard(),
		}

		logic := &Logic{}

		nilConditionFunc := func(ctx *Context) (*metav1.Condition, error) {
			return nil, nil
		}

		validConditionFunc := func(ctx *Context) (*metav1.Condition, error) {
			return &metav1.Condition{
				Type:   "TestCondition",
				Status: metav1.ConditionTrue,
				Reason: "TestReason",
			}, nil
		}

		err := logic.FillStatusConditions(ctx, nilConditionFunc, validConditionFunc)
		assert.NoError(t, err)

		// Should only have the valid condition, nil condition should be skipped
		assert.Len(t, ctx.Subject.Status.Conditions, 1)
		assert.Equal(t, "TestCondition", ctx.Subject.Status.Conditions[0].Type)
	})
}

func TestLogic_ObserveConditionFunctions_Individual(t *testing.T) {
	t.Run("observePrivateKeyReadyCondition", func(t *testing.T) {
		ctx := &Context{
			Subject: &v1alpha1.FastlyCertificateSync{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			},
			Log: logr.Discard(),
		}

		tests := []struct {
			name                  string
			privateKeyUploaded    bool
			expectedStatus        metav1.ConditionStatus
			expectedReason        string
			expectedMessageSubstr string
		}{
			{
				name:                  "private_key_uploaded",
				privateKeyUploaded:    true,
				expectedStatus:        metav1.ConditionTrue,
				expectedReason:        "PrivateKeyUploaded",
				expectedMessageSubstr: "successfully uploaded",
			},
			{
				name:                  "private_key_not_uploaded",
				privateKeyUploaded:    false,
				expectedStatus:        metav1.ConditionFalse,
				expectedReason:        "PrivateKeyMissing",
				expectedMessageSubstr: "needs to be uploaded",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				logic := &Logic{
					ObservedState: ObservedState{
						PrivateKeyUploaded: tt.privateKeyUploaded,
					},
				}

				condition, err := logic.observePrivateKeyReadyCondition(ctx)
				require.NoError(t, err)
				require.NotNil(t, condition)

				assert.Equal(t, "PrivateKeyReady", condition.Type)
				assert.Equal(t, tt.expectedStatus, condition.Status)
				assert.Equal(t, tt.expectedReason, condition.Reason)
				assert.Contains(t, condition.Message, tt.expectedMessageSubstr)
			})
		}
	})

	t.Run("observeCertificateReadyCondition", func(t *testing.T) {
		ctx := &Context{
			Subject: &v1alpha1.FastlyCertificateSync{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			},
			Log: logr.Discard(),
		}

		tests := []struct {
			name                  string
			certificateStatus     CertificateStatus
			expectedStatus        metav1.ConditionStatus
			expectedReason        string
			expectedMessageSubstr string
		}{
			{
				name:                  "certificate_synced",
				certificateStatus:     CertificateStatusSynced,
				expectedStatus:        metav1.ConditionTrue,
				expectedReason:        "CertificateSynced",
				expectedMessageSubstr: "up-to-date and synced",
			},
			{
				name:                  "certificate_stale",
				certificateStatus:     CertificateStatusStale,
				expectedStatus:        metav1.ConditionFalse,
				expectedReason:        "CertificateStale",
				expectedMessageSubstr: "stale and needs to be updated",
			},
			{
				name:                  "certificate_missing",
				certificateStatus:     CertificateStatusMissing,
				expectedStatus:        metav1.ConditionFalse,
				expectedReason:        "CertificateMissing",
				expectedMessageSubstr: "missing from Fastly",
			},
			{
				name:                  "certificate_status_unknown",
				certificateStatus:     CertificateStatus("Unknown"),
				expectedStatus:        metav1.ConditionUnknown,
				expectedReason:        "CertificateStatusUnknown",
				expectedMessageSubstr: "could not be determined",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				logic := &Logic{
					ObservedState: ObservedState{
						CertificateStatus: tt.certificateStatus,
					},
				}

				condition, err := logic.observeCertificateReadyCondition(ctx)
				require.NoError(t, err)
				require.NotNil(t, condition)

				assert.Equal(t, "CertificateReady", condition.Type)
				assert.Equal(t, tt.expectedStatus, condition.Status)
				assert.Equal(t, tt.expectedReason, condition.Reason)
				assert.Contains(t, condition.Message, tt.expectedMessageSubstr)
			})
		}
	})
}
