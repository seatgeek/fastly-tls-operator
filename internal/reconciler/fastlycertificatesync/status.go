package fastlycertificatesync

import (
	"fmt"

	"github.com/seatgeek/k8s-reconciler-generic/apiobjects"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/genrec"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (l *Logic) FillStatus(ctx *Context, obs genrec.Resources, ss apiobjects.SubjectStatus) error {
	res := &(ctx.Subject.Status)
	res.SubjectStatus = ss

	ctx.Log.Info("filling status")

	// Consider the FastlyCertificateSync ready when all observed state results in no actions.
	res.Ready = l.ObservedState.PrivateKeyUploaded &&
		l.ObservedState.CertificateStatus == CertificateStatusSynced &&
		len(l.ObservedState.MissingTLSActivationData) == 0 &&
		len(l.ObservedState.ExtraTLSActivationIDs) == 0 &&
		len(l.ObservedState.UnusedPrivateKeyIDs) == 0

	return l.FillStatusConditions(ctx,
		l.observePrivateKeyReadyCondition,
		l.observeCertificateReadyCondition,
		l.observeTLSActivationReadyCondition,
		l.observeCleanupRequiredCondition,
		l.observeReadyCondition,
	)
}

func (l *Logic) FillStatusConditions(ctx *Context, conditionGeneratorFuncs ...func(ctx *Context) (*kmetav1.Condition, error)) error {
	ctx.Subject.Status.Conditions = []kmetav1.Condition{}

	for _, fn := range conditionGeneratorFuncs {
		cnd, err := fn(ctx)
		if err != nil {
			ctx.Log.Error(err, "error generating condition", "namespace", ctx.Subject.Namespace, "name", ctx.Subject.Name)
		}
		if cnd == nil {
			continue
		}
		_ = apimeta.SetStatusCondition(&ctx.Subject.Status.Conditions, *cnd)
	}

	return nil
}

// observePrivateKeyReadyCondition generates the condition for private key upload status
func (l *Logic) observePrivateKeyReadyCondition(ctx *Context) (*kmetav1.Condition, error) {
	condition := &kmetav1.Condition{
		Type: "PrivateKeyReady",
	}

	if l.ObservedState.PrivateKeyUploaded {
		condition.Status = kmetav1.ConditionTrue
		condition.Reason = "PrivateKeyUploaded"
		condition.Message = "Private key has been successfully uploaded to Fastly"
	} else {
		condition.Status = kmetav1.ConditionFalse
		condition.Reason = "PrivateKeyMissing"
		condition.Message = "Private key needs to be uploaded to Fastly"
	}

	return condition, nil
}

// observeCertificateReadyCondition generates the condition for certificate synchronization status
func (l *Logic) observeCertificateReadyCondition(ctx *Context) (*kmetav1.Condition, error) {
	condition := &kmetav1.Condition{
		Type: "CertificateReady",
	}

	switch l.ObservedState.CertificateStatus {
	case CertificateStatusSynced:
		condition.Status = kmetav1.ConditionTrue
		condition.Reason = "CertificateSynced"
		condition.Message = "Certificate is up-to-date and synced with Fastly"
	case CertificateStatusStale:
		condition.Status = kmetav1.ConditionFalse
		condition.Reason = "CertificateStale"
		condition.Message = "Certificate exists in Fastly but is stale and needs to be updated"
	case CertificateStatusMissing:
		condition.Status = kmetav1.ConditionFalse
		condition.Reason = "CertificateMissing"
		condition.Message = "Certificate is missing from Fastly and needs to be created"
	default:
		condition.Status = kmetav1.ConditionUnknown
		condition.Reason = "CertificateStatusUnknown"
		condition.Message = "Certificate status could not be determined"
	}

	return condition, nil
}

// observeTLSActivationReadyCondition generates the condition for TLS activation status
func (l *Logic) observeTLSActivationReadyCondition(ctx *Context) (*kmetav1.Condition, error) {
	condition := &kmetav1.Condition{
		Type: "TLSActivationReady",
	}

	if len(l.ObservedState.MissingTLSActivationData) > 0 {
		condition.Status = kmetav1.ConditionFalse
		condition.Reason = "TLSActivationsMissing"
		condition.Message = fmt.Sprintf("Missing %d TLS activations that need to be created", len(l.ObservedState.MissingTLSActivationData))
	} else if len(l.ObservedState.ExtraTLSActivationIDs) > 0 {
		condition.Status = kmetav1.ConditionFalse
		condition.Reason = "TLSActivationsExtra"
		condition.Message = fmt.Sprintf("Found %d extra TLS activations that need to be removed", len(l.ObservedState.ExtraTLSActivationIDs))
	} else {
		condition.Status = kmetav1.ConditionTrue
		condition.Reason = "TLSActivationsSynced"
		condition.Message = "All TLS activations are properly configured"
	}

	return condition, nil
}

// observeCleanupRequiredCondition generates the condition for cleanup requirements
func (l *Logic) observeCleanupRequiredCondition(ctx *Context) (*kmetav1.Condition, error) {
	condition := &kmetav1.Condition{
		Type: "CleanupRequired",
	}

	if len(l.ObservedState.UnusedPrivateKeyIDs) > 0 {
		condition.Status = kmetav1.ConditionTrue
		condition.Reason = "UnusedPrivateKeysFound"
		condition.Message = fmt.Sprintf("Found %d unused private keys that should be cleaned up", len(l.ObservedState.UnusedPrivateKeyIDs))
	} else {
		condition.Status = kmetav1.ConditionFalse
		condition.Reason = "NoCleanupNeeded"
		condition.Message = "No unused private keys found"
	}

	return condition, nil
}

// observeReadyCondition generates the overall ready condition
func (l *Logic) observeReadyCondition(ctx *Context) (*kmetav1.Condition, error) {
	condition := &kmetav1.Condition{
		Type: "Ready",
	}

	// Ready when: private key uploaded, certificate synced, TLS activations synced, and no cleanup required
	if l.ObservedState.PrivateKeyUploaded &&
		l.ObservedState.CertificateStatus == CertificateStatusSynced &&
		len(l.ObservedState.MissingTLSActivationData) == 0 &&
		len(l.ObservedState.ExtraTLSActivationIDs) == 0 &&
		len(l.ObservedState.UnusedPrivateKeyIDs) == 0 {
		condition.Status = kmetav1.ConditionTrue
		condition.Reason = "FastlySyncComplete"
		condition.Message = "FastlyCertificateSync is ready and all components are synchronized"
	} else {
		condition.Status = kmetav1.ConditionFalse
		condition.Reason = "FastlySyncIncomplete"
		condition.Message = "FastlyCertificateSync is not ready - synchronization in progress"
	}

	return condition, nil
}
