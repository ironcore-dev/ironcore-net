// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"errors"
	goflag "flag"
	"fmt"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"

	ironcorenetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	apinetletconfig "github.com/ironcore-dev/ironcore-net/apinetlet/client/config"
	"github.com/ironcore-dev/ironcore-net/apinetlet/controllers"
	ironcorenet "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned"
	apinetclient "github.com/ironcore-dev/ironcore-net/internal/client"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/client/config"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
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
	var secureMetrics bool
	var metricsCertPath, metricsCertName, metricsCertKey string
	var enableHTTP2 bool
	var enableLeaderElection bool
	var probeAddr string

	var configOptions config.GetConfigOptions
	var apiNetGetConfigOptions config.GetConfigOptions

	var apiNetNamespace string

	var watchNamespace string
	var watchFilterValue string

	var isNodeAffinityAware bool

	var tlsOpts []func(*tls.Config)

	var disableNetworkPeering bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the metrics.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&isNodeAffinityAware, "is-node-affinity-aware", false, "If set, will determine node affinity topology for loadbalancer daemonsets.")

	configOptions.BindFlags(flag.CommandLine)
	apiNetGetConfigOptions.BindFlags(flag.CommandLine, config.WithNamePrefix(apiNetFlagPrefix))

	flag.StringVar(&apiNetNamespace, "api-net-namespace", "", "api-net cluster namespace to manage all objects in.")

	flag.StringVar(&watchNamespace, "namespace", "", "Namespace that the controller watches to reconcile ironcore objects. If unspecified, the controller watches for ironcore objects across all namespaces.")
	flag.StringVar(&watchFilterValue, "watch-filter", "", fmt.Sprintf("label value that the controller watches to reconcile ironcore objects. Label key is always %s. If unspecified, the controller watches for all ironcore objects", commonv1alpha1.WatchLabel))
	flag.BoolVar(&disableNetworkPeering, "disable-network-peering", false,
		"Disable the metalnet based network peering. If set to true the network peering is handled externally.")

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

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/controller/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}
	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	//
	// TODO(user): If you enable certManager, uncomment the following lines:
	// - [METRICS-WITH-CERTS] at config/controller/default/kustomization.yaml to generate and use certificates
	// managed by cert-manager for the metrics server.
	// - [PROMETHEUS-WITH-CERTS] at config/controller/prometheus/kustomization.yaml for TLS certification.

	// Create watchers for metrics certificates
	var metricsCertWatcher *certwatcher.CertWatcher

	if len(metricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", metricsCertPath, "metrics-cert-name", metricsCertName, "metrics-cert-key", metricsCertKey)

		var err error
		metricsCertWatcher, err = certwatcher.New(
			filepath.Join(metricsCertPath, metricsCertName),
			filepath.Join(metricsCertPath, metricsCertKey),
		)
		if err != nil {
			setupLog.Error(err, "to initialize metrics certificate watcher", "error", err)
			os.Exit(1)
		}

		metricsServerOptions.TLSOpts = append(metricsServerOptions.TLSOpts, func(config *tls.Config) {
			config.GetCertificate = metricsCertWatcher.GetCertificate
		})
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
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
		Client:              mgr.GetClient(),
		APINetClient:        apiNetCluster.GetClient(),
		APINetInterface:     apiNetIface,
		APINetNamespace:     apiNetNamespace,
		WatchFilterValue:    watchFilterValue,
		IsNodeAffinityAware: isNodeAffinityAware,
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
		Client:                 mgr.GetClient(),
		APINetClient:           apiNetCluster.GetClient(),
		APINetNamespace:        apiNetNamespace,
		WatchFilterValue:       watchFilterValue,
		NetworkPeeringDisabled: disableNetworkPeering,
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

	if metricsCertWatcher != nil {
		setupLog.Info("Adding metrics certificate watcher to manager")
		if err := mgr.Add(metricsCertWatcher); err != nil {
			setupLog.Error(err, "unable to add metrics certificate watcher to manager")
			os.Exit(1)
		}
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
