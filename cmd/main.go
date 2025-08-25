package main

import (
	"crypto/tls"
	"flag"
	"os"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/fastly-tls-operator/api/v1alpha1"
	"github.com/fastly/go-fastly/v11/fastly"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/transport"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	kconf "sigs.k8s.io/controller-runtime/pkg/client/config"
	crconfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/fastly-tls-operator/internal/reconciler/fastlycertificatesync"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/genrec"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(cmv1.AddToScheme(scheme))
}

type cliFlags struct {
	metricsAddr                                  string
	enableLeaderElection                         bool
	probeAddr                                    string
	leaderElectionID                             string
	syncPeriod                                   time.Duration
	webhookPort                                  int
	webhookCertDir                               string
	hackFastlyCertificateSyncLocalReconciliation bool
}

// BindFlags will parse the given flagset
func (c *cliFlags) BindFlags(fs *flag.FlagSet) {
	fs.StringVar(&(c.metricsAddr), "metrics-bind-address", c.metricsAddr, "The address the metric endpoint binds to.")
	fs.BoolVar(&(c.enableLeaderElection), "leader-election", c.enableLeaderElection,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	fs.StringVar(&(c.leaderElectionID), "leader-election-id", c.leaderElectionID,
		"The name of the resource that leader election will use for holding the leader lock.")
	fs.DurationVar(&(c.syncPeriod), "sync-period", c.syncPeriod, "Maximum delay between reconciles of any object.")
	fs.IntVar(&(c.webhookPort), "webhook-port", c.webhookPort, "Webhook bind port")
	fs.StringVar(&(c.webhookCertDir), "webhook-cert-dir", c.webhookCertDir,
		"Certs used to terminate TLS for webhook server")
	fs.BoolVar(&(c.hackFastlyCertificateSyncLocalReconciliation), "hack-fastly-certificate-sync-local-reconciliation",
		c.hackFastlyCertificateSyncLocalReconciliation, "Enable local reconciliation for Fastly certificate sync")
}

func main() {
	opts := cliFlags{
		metricsAddr:          ":8080",
		probeAddr:            ":8081",
		enableLeaderElection: true,
		leaderElectionID:     "fastly-tls-operator-leader-election",
		syncPeriod:           4 * time.Hour,
		webhookPort:          9443,
		webhookCertDir:       "/var/run/webhook-serving-certs",
		hackFastlyCertificateSyncLocalReconciliation: false,
	}

	opts.BindFlags(flag.CommandLine)
	zapOpts := zap.Options{}
	zapOpts.BindFlags(flag.CommandLine)
	bindKlogFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	setupLog.Info("initializing", "cluster", "fastly-tls-operator")

	config, err := kconf.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to get kubeconfig")
		os.Exit(1)
	}

	webhookOpts := webhook.Options{
		Port:     opts.webhookPort,
		CertName: "tls.crt",
		KeyName:  "tls.key",
		CertDir:  opts.webhookCertDir,
		TLSOpts:  []func(*tls.Config){},
	}

	config.WrapTransport = transport.DebugWrappers

	// populate the runtime config struct for the controller
	controllerRuntimeConfig := fastlycertificatesync.RuntimeConfig{
		HackFastlyCertificateSyncLocalReconciliation: opts.hackFastlyCertificateSyncLocalReconciliation,
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: opts.metricsAddr,
		},
		WebhookServer:          webhook.NewServer(webhookOpts),
		HealthProbeBindAddress: opts.probeAddr,
		LeaderElection:         opts.enableLeaderElection,
		LeaderElectionID:       opts.leaderElectionID,
		Cache: cache.Options{
			SyncPeriod: &(opts.syncPeriod),
		},
		Controller: crconfig.Controller{
			RecoverPanic:       &[]bool{true}[0],
			NeedLeaderElection: &opts.enableLeaderElection,
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	sc := k8sutil.SchemedClient{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}

	// setup FastlyCertificateSync controller
	if err = (&genrec.Reconciler[*v1alpha1.FastlyCertificateSync, *fastlycertificatesync.Config]{
		Logic: &fastlycertificatesync.Logic{
			ResourceManager: fastlycertificatesync.ResourceManager,
			Config:          controllerRuntimeConfig,
			FastlyClient: func() *fastly.Client {
				client, err := fastly.NewClient(os.Getenv("FASTLY_API_KEY"))
				if err != nil {
					setupLog.Error(err, "unable to create Fastly client")
					os.Exit(1)
				}
				return client
			}(),
		},
		Recorder:     mgr.GetEventRecorderFor("fastly-tls-operator"),
		Client:       sc,
		KeyNamespace: "platform.seatgeek.io",
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FastlyCertificateSync")
		os.Exit(1)
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	setupLog.Info("starting manager")
	if err = mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func bindKlogFlags(into *flag.FlagSet) {
	// zap, logr, and klog... all in one process, logging to the same stdio streams, using different formats.
	// in this function, we prefix all the klog CLI flags with `klog-` to avoid collisions.
	tmp := &flag.FlagSet{}
	klog.InitFlags(tmp)
	tmp.VisitAll(func(f *flag.Flag) {
		into.Var(f.Value, "klog-"+f.Name, f.Usage)
	})
}
