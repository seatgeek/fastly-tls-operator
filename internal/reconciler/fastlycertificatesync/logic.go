package fastlycertificatesync

import (
	"context"
	"fmt"
	"reflect"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/fastly-operator/api/v1alpha1"
	"github.com/fastly/go-fastly/v11/fastly"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/genrec"
	rm "github.com/seatgeek/k8s-reconciler-generic/pkg/resourcemanager"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:rbac:groups=platform.seatgeek.io,resources=fastlycertificatesyncs,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups=platform.seatgeek.io,resources=fastlycertificatesyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.seatgeek.io,resources=fastlycertificatesyncs/finalizers,verbs=update
// +kubebuilder:rbac:groups="cert-manager.io",resources=certificaterequests;certificates,verbs=*
// +kubebuilder:rbac:groups="",resources=secrets,verbs=*

type Context = genrec.Context[*v1alpha1.FastlyCertificateSync, *Config]

type (
	CertificateStatus  string
	TLSActivationState string
)

const (
	CertificateStatusMissing CertificateStatus = "Missing"
	CertificateStatusStale   CertificateStatus = "Stale"
	CertificateStatusSynced  CertificateStatus = "Synced"
)

const (
	TLSActivationStateMissing TLSActivationState = "Missing"
	TLSActivationStateExtra   TLSActivationState = "Extra"
	TLSActivationStateSynced  TLSActivationState = "Synced"
)

type TLSActivationData struct {
	Certificate   *fastly.CustomTLSCertificate
	Configuration *fastly.TLSConfiguration
	Domain        *fastly.TLSDomain
}

type ObservedState struct {
	PrivateKeyUploaded       bool
	CertificateStatus        CertificateStatus
	UnusedPrivateKeyIDs      []string
	MissingTLSActivationData []TLSActivationData
	ExtraTLSActivationIDs    []string
}

type Logic struct {
	genrec.WithoutFinalizationMixin[*v1alpha1.FastlyCertificateSync, *Config]
	rm.ResourceManager[*Context]
	Config       RuntimeConfig
	FastlyClient FastlyClientInterface
	// For the following state, we make sure that:
	// * Always reset state at the beginning of `ObserveResources`
	// * Only set state during `ObserveResources`
	// * Only read state during `ApplyUnmanaged`
	ObservedState                 ObservedState
	SubjectReadyForReconciliation bool
}

func (l *Logic) NewSubject() *v1alpha1.FastlyCertificateSync {
	return &v1alpha1.FastlyCertificateSync{}
}

func (l *Logic) GetConfig(nn types.NamespacedName) *Config {
	return &Config{RuntimeConfig: l.Config}
}

func (l *Logic) FillDefaults(c *Context) error {
	return nil
}

func (l *Logic) IsStatusEqual(a, b *v1alpha1.FastlyCertificateSync) bool {
	return reflect.DeepEqual(a.Status, b.Status)
}

func (l *Logic) IsSubjectNil(subj *v1alpha1.FastlyCertificateSync) bool {
	return subj == nil
}

func (l *Logic) ResourceIssues(_ client.Object) (facts []string) {
	// TODO report any semantic or state issues for a watched object
	return
}

func (l *Logic) ExtraLabelsForObject(context *Context, tier, suffix string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "fastly-operator",
	}
}

func (l *Logic) ExtraAnnotationsForObject(_ *Context, _, _ string) map[string]string {
	return nil
}

func (l *Logic) ConfigureController(cb *builder.Builder, cluster cluster.Cluster) error {
	if err := l.RegisterOwnedTypes(cb); err != nil {
		return err
	}

	cb.Owns(&v1alpha1.FastlyCertificateSync{})

	watchOpts := builder.WithPredicates() // NOTE: we care about `.status` field updates on Certificates, so don't drop those events

	// watch all Certificates - re-reconcile the FastlyCertificateSync resources that reference them
	cb.Watches(&cmv1.Certificate{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
		res := []reconcile.Request{}

		// discard certificate if it is not annotated for fastly-certificate-sync
		if sync, ok := object.GetAnnotations()["platform.seatgeek.io/enable-fastly-sync"]; !ok || sync != "true" {
			ctrl.Log.V(5).Info("certificate is not annotated for fastly-certificate-sync, skipping reconciliation", "certificate_name", object.GetName(), "certificate_namespace", object.GetNamespace())
			return res
		}

		all := v1alpha1.FastlyCertificateSyncList{}

		if err := cluster.GetClient().List(ctx, &all, &client.ListOptions{Namespace: kmetav1.NamespaceAll}); err != nil {
			ctrl.Log.Error(err, "could not list FastlyCertificateSync resources to reconcile while watching Certificates")
		}

		// attempt to match a fastlyCertificateSync
		for _, fastlyCertificateSync := range all.Items {
			// reconcile fastlyCertificateSync resources that are referenced by the watched certificate
			if (object.GetName() == fastlyCertificateSync.Spec.CertificateName) && (object.GetNamespace() == fastlyCertificateSync.GetNamespace()) {
				res = append(res, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      fastlyCertificateSync.GetName(),
						Namespace: fastlyCertificateSync.GetNamespace(),
					},
				})
			}
		}

		return res
	}), watchOpts)

	ctrl.Log.Info("Configured controller", "controller", "fastlycertificatesync")

	return nil
}

func (l *Logic) Reconcile(ctx *Context) (ctrl.Result, error) {
	// TODO: Implement the actual reconciliation logic
	// For now, just log that we're reconciling
	ctx.Log.Info("reconciling FastlyCertificateSync", "name", ctx.Subject.Name, "namespace", ctx.Subject.Namespace)

	return ctrl.Result{}, nil
}

func (l *Logic) Validate(svc *v1alpha1.FastlyCertificateSync) error {
	// TODO: Implement validation logic
	return nil
}

func (l *Logic) ObserveResources(ctx *Context) (genrec.Resources, error) {
	ctx.Log.Info("observing resources for FastlyCertificateSync", "name", ctx.Subject.Name, "namespace", ctx.Subject.Namespace)

	// Allow `ApplyUnmanaged` to differentiate between:
	// * A subject that isn't ready for reconciliation (certificate and secret not available)
	// * An initially empty ObservedState indicating that we want to start taking action.
	l.SubjectReadyForReconciliation = false

	// Always start with fresh observation state, avoid sharing data between reconciliations
	l.ObservedState = ObservedState{}

	if !isSubjectReadyForReconciliation(ctx) {
		// Requeue after 30s to allow the certificate to be created and ready for reconciliation
		ctx.Log.Info("Requeueing in 30s")
		ctx.SetRequeue(30 * time.Second)

		return genrec.Resources{}, nil
	}

	l.SubjectReadyForReconciliation = true

	// Begin observation
	// First, the private key must exist in Fastly
	fastlyPrivateKeyExists, err := l.getFastlyPrivateKeyExists(ctx)
	if err != nil {
		return genrec.Resources{}, err
	}
	l.ObservedState.PrivateKeyUploaded = fastlyPrivateKeyExists

	// Second, the certificate must be present and up to date (synced) in Fastly
	fastlyCertificateStatus, err := l.getFastlyCertificateStatus(ctx)
	if err != nil {
		return genrec.Resources{}, err
	}
	l.ObservedState.CertificateStatus = fastlyCertificateStatus

	// Third, TLS activations must be present for all desired configurations
	missingTLSActivationData, extraTLSActivationIDs, err := l.getFastlyTLSActivationState(ctx)
	if err != nil {
		return genrec.Resources{}, err
	}
	l.ObservedState.MissingTLSActivationData = missingTLSActivationData
	l.ObservedState.ExtraTLSActivationIDs = extraTLSActivationIDs

	// Lastly, unused private keys must be removed from Fastly
	unusedPrivateKeyIDs, err := l.getFastlyUnusedPrivateKeyIDs(ctx)
	if err != nil {
		return genrec.Resources{}, err
	}
	l.ObservedState.UnusedPrivateKeyIDs = unusedPrivateKeyIDs

	return genrec.Resources{}, nil
}

func (l *Logic) ApplyUnmanaged(ctx *Context) error {
	if !l.SubjectReadyForReconciliation {
		ctx.Log.Info("Subject is not ready for reconciliation, skipping")
		return nil
	}

	ctx.Log.Info("applying unmanaged FastlyCertificateSync", "name", ctx.Subject.Name, "namespace", ctx.Subject.Namespace)

	if !l.ObservedState.PrivateKeyUploaded {
		ctx.Log.Info("Private key is not uploaded, doing that now...")

		if err := l.createFastlyPrivateKey(ctx); err != nil {
			return fmt.Errorf("failed to create Fastly private key: %w", err)
		}

		// Requeue immediately after altering state
		ctx.Log.Info("Requeueing...")
		ctx.SetRequeue(0)

		return nil
	}

	if l.ObservedState.CertificateStatus == CertificateStatusMissing {
		ctx.Log.Info("Certificate is missing, creating new certificate in Fastly")
		if err := l.createFastlyCertificate(ctx); err != nil {
			return fmt.Errorf("failed to create Fastly certificate: %w", err)
		}

		ctx.Log.Info("Requeueing...")
		ctx.SetRequeue(0)

		return nil
	}

	if l.ObservedState.CertificateStatus == CertificateStatusStale {
		ctx.Log.Info("Certificate is stale, updating certificate in Fastly")
		if err := l.updateFastlyCertificate(ctx); err != nil {
			return fmt.Errorf("failed to update Fastly certificate: %w", err)
		}

		ctx.Log.Info("Requeueing...")
		ctx.SetRequeue(0)
		return nil
	}

	if len(l.ObservedState.MissingTLSActivationData) > 0 {
		ctx.Log.Info("Missing TLS activations found, creating them in Fastly")
		if err := l.createMissingFastlyTLSActivations(ctx); err != nil {
			return fmt.Errorf("failed to create Fastly TLS activations: %w", err)
		}

		ctx.Log.Info("Requeueing...")
		ctx.SetRequeue(0)
		return nil
	}

	if len(l.ObservedState.ExtraTLSActivationIDs) > 0 {
		ctx.Log.Info("Extra TLS activations found, deleting them from Fastly")
		if err := l.deleteExtraFastlyTLSActivations(ctx); err != nil {
			return fmt.Errorf("failed to delete Fastly TLS activations: %w", err)
		}

		ctx.Log.Info("Requeueing...")
		ctx.SetRequeue(0)
		return nil
	}

	if len(l.ObservedState.UnusedPrivateKeyIDs) > 0 {
		ctx.Log.Info("Unused private keys found, deleting them from Fastly")
		l.clearFastlyUnusedPrivateKeys(ctx)

		ctx.Log.Info("Requeueing...")
		ctx.SetRequeue(0)
		return nil
	}

	return nil
}

func (l *Logic) Finalize(ctx *Context) (genrec.FinalizationAction, error) {
	// TODO: Implement finalization logic
	// Return Continue to indicate finalization should continue
	return genrec.FinalizationCompleted, nil
}
