/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	goflag "flag"
	"fmt"
	"os"

	"github.com/onmetal/controller-utils/configutils"
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	"github.com/onmetal/onmetal-api-net/apinetlet/controllers"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

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

	utilruntime.Must(networkingv1alpha1.AddToScheme(scheme))
	utilruntime.Must(onmetalapinetv1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	var apiNetKubeconfig string

	var apiNetNamespace string

	var watchNamespace string
	var watchFilterValue string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.StringVar(&apiNetKubeconfig, "api-net-kubeconfig", "", "Path pointing to the api-net kubeconfig.")
	flag.StringVar(&apiNetNamespace, "api-net-namespace", "", "api-net cluster namespace to manage all objects in.")

	flag.StringVar(&watchNamespace, "namespace", "", "Namespace that the controller watches to reconcile onmetal-api objects. If unspecified, the controller watches for onmetal-api objects across all namespaces.")
	flag.StringVar(&watchFilterValue, "watch-filter", "", fmt.Sprintf("label value that the controller watches to reconcile onmetal-api objects. Label key is always %s. If unspecified, the controller watches for all onmetal-api objects", commonv1alpha1.WatchLabel))

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(goflag.CommandLine)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	ctx := ctrl.SetupSignalHandler()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if apiNetNamespace == "" {
		setupLog.Error(errors.New("must specify --api-net-namespace"), "Invalid configuration")
		os.Exit(1)
	}

	if watchNamespace != "" {
		setupLog.Info("Watching onmetal-api objects only in namespace for reconciliation", "namespace", watchNamespace)
	}

	cfg, err := configutils.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to load kubeconfig")
		os.Exit(1)
	}

	apiNetCfg, err := configutils.GetConfig(configutils.Kubeconfig(apiNetKubeconfig))
	if err != nil {
		setupLog.Error(err, "unable to load api net kubeconfig")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "fa89daf5.apinetlet.apinet.api.onmetal.de",
		LeaderElectionConfig:   apiNetCfg,
		Namespace:              watchNamespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	apiNetCluster, err := cluster.New(apiNetCfg, func(options *cluster.Options) {
		options.Scheme = scheme
		options.Namespace = apiNetNamespace
	})
	if err != nil {
		setupLog.Error(err, "unable to create api net cluster")
		os.Exit(1)
	}

	if err := mgr.Add(apiNetCluster); err != nil {
		setupLog.Error(err, "unable to add cluster", "cluster", "APINet")
		os.Exit(1)
	}

	if err = (&controllers.VirtualIPReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualIP")
		os.Exit(1)
	}

	if err = (&controllers.NetworkReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Network")
		os.Exit(1)
	}

	if err = (&controllers.NATGatewayReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NATGateway")
		os.Exit(1)
	}

	if err = (&controllers.LoadBalancerReconciler{
		Client:           mgr.GetClient(),
		APINetClient:     apiNetCluster.GetClient(),
		APINetNamespace:  apiNetNamespace,
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(mgr, apiNetCluster); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LoadBalancer")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

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
