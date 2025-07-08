package fastlycertificatesync

import (
	"reflect"

	"github.com/fastly-operator/api/v1alpha1"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/genrec"
	rm "github.com/seatgeek/k8s-reconciler-generic/pkg/resourcemanager"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

//+kubebuilder:rbac:groups=platform.seatgeek.io,resources=fastlycertificatesyncs,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups=platform.seatgeek.io,resources=fastlycertificatesyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.seatgeek.io,resources=fastlycertificatesyncs/finalizers,verbs=update

var (
	msGK = schema.GroupKind{Group: "platform.seatgeek.io", Kind: "FastlyCertificateSync"}
)

type Context = genrec.Context[*v1alpha1.FastlyCertificateSync, *Config]

type Logic struct {
	genrec.WithoutFinalizationMixin[*v1alpha1.FastlyCertificateSync, *Config]
	rm.ResourceManager[*Context]
	Config RuntimeConfig
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
	if err := l.ResourceManager.RegisterOwnedTypes(cb); err != nil {
		return err
	}

	cb.Owns(&v1alpha1.FastlyCertificateSync{})

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
	// TODO: Implement resource observation
	return genrec.Resources{}, nil
}

func (l *Logic) ApplyUnmanaged(ctx *Context) error {
	// TODO: Implement unmanaged apply logic
	return nil
}

func (l *Logic) Finalize(ctx *Context) (genrec.FinalizationAction, error) {
	// TODO: Implement finalization logic
	// Return Continue to indicate finalization should continue
	return genrec.FinalizationCompleted, nil
}
