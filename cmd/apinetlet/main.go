// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	goflag "flag"
	"fmt"
	"os"

	ironcorenetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	apinetletconfig "github.com/ironcore-dev/ironcore-net/apinetlet/client/config"
	"github.com/ironcore-dev/ironcore-net/apinetlet/controllers"
	"github.com/ironcore-dev/ironcore-net/client-go/ironcorenet"
	apinetclient "github.com/ironcore-dev/ironcore-net/internal/client"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/client/config"
	flag "github.com/spf13/pflag"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	apiNetFlagPrefix = "api-net-"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(networkingv1alpha1.AddToScheme(scheme))
	utilruntime.Must(ipamv1alpha1.AddToScheme(scheme))
	utilruntime.Must(ironcorenetv1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	var configOptions config.GetConfigOptions
	var apiNetGetConfigOptions config.GetConfigOptions

	var apiNetNamespace string

	var watchNamespace string
	var watchFilterValue string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	configOptions.BindFlags(flag.CommandLine)
	apiNetGetConfigOptions.BindFlags(flag.CommandLine, config.WithNamePrefix(apiNetFlagPrefix))

	flag.StringVar(&apiNetNamespace, "api-net-namespace", "", "api-net cluster namespace to manage all objects in.")

	flag.StringVar(&watchNamespace, "namespace", "", "Namespace that the controller watches to reconcile ironcore objects. If unspecified, the controller watches for ironcore objects across all namespaces.")
	flag.StringVar(&watchFilterValue, "watch-filter", "", fmt.Sprintf("label value that the controller watches to reconcile ironcore objects. Label key is always %s. If unspecified, the controller watches for all ironcore objects", commonv1alpha1.WatchLabel))

	opts := zap.Options{
		Development: true,
	}
	goFlags := goflag.NewFlagSet(os.Args[0], goflag.ExitOnError)
	opts.BindFlags(goFlags)
	flag.CommandLine.AddGoFlagSet(goFlags)
	flag.Parse()

	ctx := ctrl.SetupSignalHandler()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if apiNetNamespace == "" {
		setupLog.Error(errors.New("must specify --api-net-namespace"), "Invalid configuration")
		os.Exit(1)
	}

	if watchNamespace != "" {
		setupLog.Info("Watching ironcore objects only in namespace for reconciliation", "namespace", watchNamespace)
	}

	cfg, cfgCtrl, err := apinetletconfig.GetConfig(ctx, &configOptions)
	if err != nil {
		setupLog.Error(err, "unable to load kubeconfig")
		os.Exit(1)
	}

	apiNetGetter := apinetletconfig.NewAPINetGetterOrDie(apiNetNamespace)
	apiNetCfg, apiNetCfgCtrl, err := apiNetGetter.GetConfig(ctx, &apiNetGetConfigOptions)
	if err != nil {
		setupLog.Error(err, "unable to load api net kubeconfig")
		os.Exit(1)
	}

	var cacheDefaultNamespaces map[string]cache.Config
	if watchNamespace != "" {
		cacheDefaultNamespaces = map[string]cache.Config{
			watchNamespace: {},
		}
	}
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionConfig:   apiNetCfg,
		LeaderElectionID:       "fa89daf5.apinetlet.apinet.ironcore.dev",
		Cache: cache.Options{
			DefaultNamespaces: cacheDefaultNamespaces,
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if err := config.SetupControllerWithManager(mgr, cfgCtrl); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Config")
		os.Exit(1)
	}
	if err := config.SetupControllerWithManager(mgr, apiNetCfgCtrl); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "APINetConfig")
		os.Exit(1)
	}

	apiNetCluster, err := cluster.New(apiNetCfg, func(options *cluster.Options) {
		options.Scheme = scheme
		options.Cache.DefaultNamespaces = map[string]cache.Config{
			apiNetNamespace: {},
		}
	})
	if err != nil {
		setupLog.Error(err, "unable to create api net cluster")
		os.Exit(1)
	}

	apiNetIface, err := ironcorenet.NewForConfig(apiNetCfg)
	if err != nil {
		setupLog.Error(err, "unable to create api net interface")
		os.Exit(1)
	}

	if err := mgr.Add(apiNetCluster); err != nil {
		setupLog.Error(err, "unable to add cluster", "cluster", "APINet")
		os.Exit(1)
	}

	if err = (&controllers.LoadBalancerReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetInterface:  apiNetIface,
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LoadBalancer")
		os.Exit(1)
	}

	if err = (&controllers.NATGatewayReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetInterface:  apiNetIface,
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NATGateway")
		os.Exit(1)
	}

	if err = (&controllers.NetworkReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Network")
		os.Exit(1)
	}

	if err = (&controllers.NetworkInterfaceReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkInterface")
		os.Exit(1)
	}

	if err = (&controllers.NetworkPolicyReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetInterface:  apiNetIface,
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkPolicy")
		os.Exit(1)
	}

	if err = (&controllers.VirtualIPReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetInterface:  apiNetIface,
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualIP")
		os.Exit(1)
	}

	if err := apinetletclient.SetupNetworkPolicyNetworkNameFieldIndexer(ctx, apiNetCluster.GetFieldIndexer()); err != nil {
		setupLog.Error(err, "unable to setup field indexer", "field", apinetletclient.NetworkPolicyNetworkNameField)
		os.Exit(1)
	}
	if err := apinetclient.SetupNetworkInterfaceNetworkNameFieldIndexer(ctx, apiNetCluster.GetFieldIndexer()); err != nil {
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
