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
	goflag "flag"
	"fmt"
	"os"

	"github.com/onmetal/controller-utils/configutils"
	metalnetv1alpha1 "github.com/onmetal/metalnet/api/v1alpha1"
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	metalnetletconfig "github.com/onmetal/onmetal-api-net/metalnetlet/client/config"
	"github.com/onmetal/onmetal-api-net/metalnetlet/controllers"
	"github.com/onmetal/onmetal-api/utils/client/config"
	flag "github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
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

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(metalnetv1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var name string
	var nodeLabels map[string]string

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	var configOptions config.GetConfigOptions
	var metalnetKubeconfig string
	var metalnetNamespace string

	flag.StringVar(&name, "name", "", "The name of the partition the metalnetlet represents (required).")
	flag.StringToStringVar(&nodeLabels, "node-label", nodeLabels, "Additional labels to add to the nodes.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	configOptions.BindFlags(flag.CommandLine)
	flag.StringVar(&metalnetKubeconfig, "metalnet-kubeconfig", "", "Metalnet kubeconfig to use.")
	flag.StringVar(&metalnetNamespace, "metalnet-namespace", corev1.NamespaceDefault, "Metalnet namespace to use.")

	opts := zap.Options{
		Development: true,
	}
	goFlags := goflag.NewFlagSet(os.Args[0], goflag.ExitOnError)
	opts.BindFlags(goFlags)
	flag.CommandLine.AddGoFlagSet(goFlags)
	flag.Parse()

	ctx := ctrl.SetupSignalHandler()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if name == "" {
		setupLog.Error(fmt.Errorf("must specify name"), "invalid configuration")
		os.Exit(1)
	}

	getter := metalnetletconfig.NewGetterOrDie(name)
	cfg, cfgCtrl, err := getter.GetConfig(ctx, &configOptions)
	if err != nil {
		setupLog.Error(err, "unable to load kubeconfig")
		os.Exit(1)
	}

	metalnetCfg, err := configutils.GetConfig(configutils.Kubeconfig(metalnetKubeconfig))
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
		LeaderElectionID:       "bf12dae0.metalnetlet.apinet.api.onmetal.de",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if err := config.SetupControllerWithManager(mgr, cfgCtrl); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Config")
		os.Exit(1)
	}

	metalnetCluster, err := cluster.New(metalnetCfg, func(options *cluster.Options) {
		options.Scheme = scheme
	})
	if err != nil {
		setupLog.Error(err, "unable to create metalnet cluster")
		os.Exit(1)
	}

	if err := mgr.Add(metalnetCluster); err != nil {
		setupLog.Error(err, "unable to add cluster", "cluster", "APINet")
		os.Exit(1)
	}

	if err := (&controllers.InstanceReconciler{
		Client:            mgr.GetClient(),
		MetalnetClient:    metalnetCluster.GetClient(),
		PartitionName:     name,
		MetalnetNamespace: metalnetNamespace,
	}).SetupWithManager(mgr, metalnetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Instance")
		os.Exit(1)
	}

	if err := (&controllers.MetalnetNodeReconciler{
		Client:         mgr.GetClient(),
		MetalnetClient: metalnetCluster.GetClient(),
		PartitionName:  name,
		NodeLabels:     nodeLabels,
	}).SetupWithManager(mgr, metalnetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MetalnetNode")
		os.Exit(1)
	}

	if err := (&controllers.NetworkReconciler{
		Client:            mgr.GetClient(),
		MetalnetClient:    metalnetCluster.GetClient(),
		PartitionName:     name,
		MetalnetNamespace: metalnetNamespace,
	}).SetupWithManager(mgr, metalnetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Network")
		os.Exit(1)
	}

	if err = (&controllers.NetworkInterfaceReconciler{
		Client:            mgr.GetClient(),
		MetalnetClient:    metalnetCluster.GetClient(),
		PartitionName:     name,
		MetalnetNamespace: metalnetNamespace,
	}).SetupWithManager(mgr, metalnetCluster.GetCache()); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Network")
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
