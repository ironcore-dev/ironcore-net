// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	goflag "flag"
	"os"

	"github.com/ironcore-dev/controller-utils/configutils"
	ironcorenetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetclient "github.com/ironcore-dev/ironcore-net/internal/client"

	"github.com/ironcore-dev/ironcore-net/internal/controllers"
	ironcorenet "github.com/ironcore-dev/ironcore-net/internal/controllers/certificate/ironcore-net"
	"github.com/ironcore-dev/ironcore-net/internal/controllers/scheduler"
	"github.com/ironcore-dev/ironcore-net/utils/expectations"
	flag "github.com/spf13/pflag"
	"k8s.io/utils/lru"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ironcorenetv1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(goflag.CommandLine)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	ctx := ctrl.SetupSignalHandler()

	cfg, err := configutils.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to load kubeconfig")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ff142330.apinet.ironcore.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.CertificateApprovalReconciler{
		Client:      mgr.GetClient(),
		Recognizers: ironcorenet.Recognizers,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CertificateApproval")
		os.Exit(1)
	}

	if err = (&controllers.DaemonSetReconciler{
		Client:       mgr.GetClient(),
		Expectations: expectations.New(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DaemonSet")
		os.Exit(1)
	}

	if err = (&controllers.IPAddressGCReconciler{
		Client:       mgr.GetClient(),
		APIReader:    mgr.GetAPIReader(),
		AbsenceCache: lru.New(500),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DaemonSet")
		os.Exit(1)
	}

	if err = (&controllers.LoadBalancerReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LoadBalancer")
		os.Exit(1)
	}

	if err = (&controllers.NetworkPolicyReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkPolicy")
		os.Exit(1)
	}

	if err = (&controllers.NATGatewayReconciler{
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("natgateways"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NATGateway")
		os.Exit(1)
	}

	if err = (&controllers.NATGatewayAutoscalerReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NATGatewayAutoscaler")
		os.Exit(1)
	}

	if err = (&controllers.NetworkIDGCReconciler{
		Client:       mgr.GetClient(),
		APIReader:    mgr.GetAPIReader(),
		AbsenceCache: lru.New(500),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkIDGC")
		os.Exit(1)
	}

	if err = (&controllers.NetworkInterfaceNATReleaseReconciler{
		Client:       mgr.GetClient(),
		APIReader:    mgr.GetAPIReader(),
		AbsenceCache: lru.New(500),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkInterfaceNATRelease")
	}

	schedulerCache := scheduler.NewCache(
		mgr.GetLogger().WithName("scheduler").WithName("cache"),
		scheduler.DefaultCacheStrategy,
	)
	if err = mgr.Add(schedulerCache); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SchedulerCache")
		os.Exit(1)
	}

	if err = (&controllers.SchedulerReconciler{
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("scheduler"),
		Cache:         schedulerCache,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Scheduler")
		os.Exit(1)
	}

	if err := apinetclient.SetupNetworkInterfaceNetworkNameFieldIndexer(ctx, mgr.GetFieldIndexer()); err != nil {
		setupLog.Error(err, "unable to setup field indexer", "field", apinetclient.NetworkInterfaceSpecNetworkRefNameField)
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
