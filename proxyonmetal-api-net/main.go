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
	"github.com/onmetal/onmetal-api-net/allocator"
	netflag "github.com/onmetal/onmetal-api-net/flag"
	"github.com/onmetal/onmetal-api-net/proxyonmetal-api-net/controllers/networking"
	"github.com/onmetal/poollet/broker"
	brokercluster "github.com/onmetal/poollet/broker/cluster"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	networkingv1alpha1 "github.com/onmetal/onmetal-api/apis/networking/v1alpha1"
	flag "github.com/spf13/pflag"
	"inet.af/netaddr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	var targetKubeconfig string
	var allocatorKubeconfig string

	var allocatorSecretNamespace string
	var allocatorSecretName string

	var ipv4Prefixes []netaddr.IPPrefix
	var ipv6Prefixes []netaddr.IPPrefix

	var clusterName string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.StringVar(&targetKubeconfig, "target-kubeconfig", "", "Path pointing to the target kubeconfig.")
	flag.StringVar(&allocatorKubeconfig, "allocator-kubeconfig", "", "Path pointing to the allocator kubeconfig.")

	flag.StringVar(&allocatorSecretNamespace, "allocator-secret-namespace", os.Getenv("ALLOCATOR_SECRET_NAMESPACE"), "allocator-secret-namespace")
	flag.StringVar(&allocatorSecretName, "allocator-secret-name", "allocator", "allocator-secret-name")

	netflag.IPPrefixesVar(&ipv4Prefixes, "ipv4-prefixes", nil, "IPv4 prefixes to allocate from")
	netflag.IPPrefixesVar(&ipv6Prefixes, "ipv6-prefixes", nil, "IPv6 prefixes to allocate from")

	flag.StringVar(&clusterName, "cluster-name", "", "Name of the source cluster. Used for cross-cluster owner references.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(goflag.CommandLine)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if clusterName == "" {
		setupLog.Error(fmt.Errorf("empty cluster name"), "cluster-name needs to be specified")
		os.Exit(1)
	}

	cfg, err := configutils.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to load kubeconfig")
		os.Exit(1)
	}

	targetCfg, err := configutils.GetConfig(configutils.Kubeconfig(targetKubeconfig))
	if err != nil {
		setupLog.Error(err, "unable to load target kubeconfig")
		os.Exit(1)
	}

	allocatorCfg, err := configutils.GetConfig(configutils.Kubeconfig(allocatorKubeconfig))
	if err != nil {
		setupLog.Error(err, "unable to load allocator kubeconfig")
		os.Exit(1)
	}

	targetCluster, err := brokercluster.New(targetCfg,
		func(o *cluster.Options) {
			o.Scheme = scheme
		},
	)
	if err != nil {
		setupLog.Error(err, "could not create target cluster")
		os.Exit(1)
	}

	mgr, err := broker.NewManager(cfg, targetCluster, broker.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ea242330.api.onmetal.de",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ipv4Set, err := netflag.IPFamilySetFromPrefixes(corev1.IPv4Protocol, ipv4Prefixes)
	if err != nil {
		setupLog.Error(err, "invalid ipv4 prefixes")
		os.Exit(1)
	}

	ipv6Set, err := netflag.IPFamilySetFromPrefixes(corev1.IPv6Protocol, ipv6Prefixes)
	if err != nil {
		setupLog.Error(err, "invalid ipv6 prefixes")
		os.Exit(1)
	}

	alloc, err := allocator.NewSecretAllocator(allocatorCfg, allocator.Options{
		SecretKey: client.ObjectKey{
			Namespace: allocatorSecretNamespace,
			Name:      allocatorSecretName,
		},
		IPv4Set: ipv4Set,
		IPv6Set: ipv6Set,
	})
	if err != nil {
		setupLog.Error(err, "error setting up secret allocator")
		os.Exit(1)
	}

	if err = (&networking.ProxyVirtualIPReconciler{
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("virtualip"),
		Scheme:        mgr.GetScheme(),
		Allocator:     alloc,
		TargetClient:  mgr.GetTarget().GetClient(),
		ClusterName:   clusterName,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ProxyVirtualIP")
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
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
