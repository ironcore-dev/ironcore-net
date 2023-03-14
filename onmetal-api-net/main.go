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
	"net/netip"
	"os"

	"github.com/onmetal/controller-utils/configutils"
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	netflag "github.com/onmetal/onmetal-api-net/flag"
	"github.com/onmetal/onmetal-api-net/onmetal-api-net/controllers"
	onmetalapinet "github.com/onmetal/onmetal-api-net/onmetal-api-net/controllers/certificate/onmetal-api-net"
	flag "github.com/spf13/pflag"

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

	utilruntime.Must(onmetalapinetv1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	var prefixes []netip.Prefix

	var minVNI, maxVNI int32

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	netflag.IPPrefixesVar(&prefixes, "prefixes", nil, "IP Prefixes to allocate from.")

	flag.Int32Var(&minVNI, "min-vni", controllers.DefaultMinVNI, "Default minimum vni to allocate.")
	flag.Int32Var(&maxVNI, "max-vni", controllers.DefaultMaxVNI, "Default maximum vni to allocate.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(goflag.CommandLine)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	cfg, err := configutils.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to load kubeconfig")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ff142330.apinet.api.onmetal.de",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	initialAvailableIPs, err := netflag.IPSetFromPrefixes(prefixes)
	if err != nil {
		setupLog.Error(err, "invalid ipv4 prefixes")
		os.Exit(1)
	}

	if err = (&controllers.PublicIPReconciler{
		EventRecorder:       mgr.GetEventRecorderFor("publicip"),
		Client:              mgr.GetClient(),
		APIReader:           mgr.GetAPIReader(),
		InitialAvailableIPs: initialAvailableIPs,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PublicIP")
		os.Exit(1)
	}

	if err = (&controllers.NetworkReconciler{
		EventRecorder: mgr.GetEventRecorderFor("network"),
		Client:        mgr.GetClient(),
		APIReader:     mgr.GetAPIReader(),
		MinVNI:        minVNI,
		MaxVNI:        maxVNI,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PublicIP")
		os.Exit(1)
	}

	if err = (&controllers.CertificateApprovalReconciler{
		Client:      mgr.GetClient(),
		Recognizers: onmetalapinet.Recognizers,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CertificateApproval")
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
