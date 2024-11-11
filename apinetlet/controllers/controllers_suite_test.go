// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ironcore-dev/controller-utils/buildutils"
	"github.com/ironcore-dev/controller-utils/modutils"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	apinetclient "github.com/ironcore-dev/ironcore-net/internal/client"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	ironcorenet "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	envtestutils "github.com/ironcore-dev/ironcore/utils/envtest"
	"github.com/ironcore-dev/ironcore/utils/envtest/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg        *rest.Config
	k8sClient  client.Client
	testEnv    *envtest.Environment
	testEnvExt *envtestutils.EnvironmentExtensions
)

const (
	pollingInterval      = 50 * time.Millisecond
	eventuallyTimeout    = 3 * time.Second
	consistentlyDuration = 1 * time.Second
	apiServiceTimeout    = 1 * time.Minute
)

func TestControllers(t *testing.T) {
	SetDefaultConsistentlyPollingInterval(pollingInterval)
	SetDefaultEventuallyPollingInterval(pollingInterval)
	SetDefaultEventuallyTimeout(eventuallyTimeout)
	SetDefaultConsistentlyDuration(consistentlyDuration)

	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	var err error
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.29.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}
	testEnvExt = &envtestutils.EnvironmentExtensions{
		APIServiceDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "apiserver", "apiservice", "bases"),
		},
		ErrorIfAPIServicePathIsMissing: true,
	}
	ironcoreOpts := testEnvExt.AddAPIServerInstallOptions(envtestutils.APIServerInstallOptions{
		Paths: []string{
			modutils.Dir("github.com/ironcore-dev/ironcore", "config", "apiserver", "apiservice", "bases"),
		},
		ErrorIfPathMissing: true,
	})

	cfg, err = envtestutils.StartWithExtensions(testEnv, testEnvExt)
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	DeferCleanup(envtestutils.StopWithExtensions, testEnv, testEnvExt)

	Expect(networkingv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(ipamv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(apinetv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	// Init package-level k8sClient
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	SetClient(k8sClient)

	apiSrv, err := apiserver.New(cfg, apiserver.Options{
		MainPath:     "github.com/ironcore-dev/ironcore-net/cmd/apiserver",
		BuildOptions: []buildutils.BuildOption{buildutils.ModModeMod},
		ETCDServers:  []string{testEnv.ControlPlane.Etcd.URL.String()},
		Host:         testEnvExt.APIServiceInstallOptions.LocalServingHost,
		Port:         testEnvExt.APIServiceInstallOptions.LocalServingPort,
		CertDir:      testEnvExt.APIServiceInstallOptions.LocalServingCertDir,
		Args: apiserver.ProcessArgs{
			"public-prefix": []string{"10.0.0.0/24"},
		},
	})
	Expect(err).NotTo(HaveOccurred())

	Expect(apiSrv.Start()).To(Succeed())
	DeferCleanup(apiSrv.Stop)

	ironcoreAPISrv, err := apiserver.New(cfg, apiserver.Options{
		MainPath:     "github.com/ironcore-dev/ironcore/cmd/ironcore-apiserver",
		BuildOptions: []buildutils.BuildOption{buildutils.ModModeMod},
		ETCDServers:  []string{testEnv.ControlPlane.Etcd.URL.String()},
		Host:         ironcoreOpts.LocalServingHost,
		Port:         ironcoreOpts.LocalServingPort,
		CertDir:      ironcoreOpts.LocalServingCertDir,
	})
	Expect(err).NotTo(HaveOccurred())

	Expect(ironcoreAPISrv.Start()).To(Succeed())
	DeferCleanup(ironcoreAPISrv.Stop)

	Expect(envtestutils.WaitUntilAPIServicesReadyWithTimeout(apiServiceTimeout, testEnvExt, k8sClient, scheme.Scheme)).To(Succeed())
})

func SetupTest(apiNetNamespace *corev1.Namespace) *corev1.Namespace {
	BeforeEach(func(ctx SpecContext) {
		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
			Metrics: metricsserver.Options{
				BindAddress: "0",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(apinetletclient.SetupNetworkPolicyNetworkNameFieldIndexer(ctx, k8sManager.GetFieldIndexer())).To(Succeed())
		Expect(apinetclient.SetupNetworkInterfaceNetworkNameFieldIndexer(ctx, k8sManager.GetFieldIndexer())).To(Succeed())

		apiNetInterface := ironcorenet.NewForConfigOrDie(cfg)

		// register reconciler here
		Expect((&VirtualIPReconciler{
			Client:          k8sManager.GetClient(),
			APINetClient:    k8sManager.GetClient(),
			APINetInterface: apiNetInterface,
			APINetNamespace: apiNetNamespace.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		Expect((&NetworkReconciler{
			Client:          k8sManager.GetClient(),
			APINetClient:    k8sManager.GetClient(),
			APINetNamespace: apiNetNamespace.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		Expect((&NetworkInterfaceReconciler{
			Client:          k8sManager.GetClient(),
			APINetClient:    k8sManager.GetClient(),
			APINetNamespace: apiNetNamespace.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		Expect((&NATGatewayReconciler{
			Client:          k8sManager.GetClient(),
			APINetClient:    k8sManager.GetClient(),
			APINetInterface: apiNetInterface,
			APINetNamespace: apiNetNamespace.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		Expect((&LoadBalancerReconciler{
			Client:          k8sManager.GetClient(),
			APINetClient:    k8sManager.GetClient(),
			APINetInterface: apiNetInterface,
			APINetNamespace: apiNetNamespace.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		Expect((&NetworkPolicyReconciler{
			Client:          k8sManager.GetClient(),
			APINetClient:    k8sManager.GetClient(),
			APINetInterface: apiNetInterface,
			APINetNamespace: apiNetNamespace.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		mgrCtx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		go func() {
			defer GinkgoRecover()
			Expect(k8sManager.Start(mgrCtx)).To(Succeed(), "failed to start manager")
		}()

	})

	return apiNetNamespace
}
