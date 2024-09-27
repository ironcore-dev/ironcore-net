// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"net/netip"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ironcore-dev/controller-utils/buildutils"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetclient "github.com/ironcore-dev/ironcore-net/internal/client"
	ironcorenet "github.com/ironcore-dev/ironcore-net/internal/controllers/certificate/ironcore-net"
	"github.com/ironcore-dev/ironcore-net/internal/controllers/scheduler"
	"github.com/ironcore-dev/ironcore-net/utils/expectations"
	utilsenvtest "github.com/ironcore-dev/ironcore/utils/envtest"
	"github.com/ironcore-dev/ironcore/utils/envtest/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/lru"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg            *rest.Config
	k8sClient      client.Client
	mgrClient      client.Client
	testEnv        *envtest.Environment
	testEnvExt     *utilsenvtest.EnvironmentExtensions
	schedulerCache *scheduler.Cache
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

func PrefixV4() netip.Prefix {
	return netip.MustParsePrefix("10.0.0.0/24")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(GinkgoLogr)

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
	testEnvExt = &utilsenvtest.EnvironmentExtensions{
		APIServiceDirectoryPaths:       []string{filepath.Join("..", "..", "config", "apiserver", "apiservice", "bases")},
		ErrorIfAPIServicePathIsMissing: true,
	}

	cfg, err = utilsenvtest.StartWithExtensions(testEnv, testEnvExt)
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	DeferCleanup(utilsenvtest.StopWithExtensions, testEnv, testEnvExt)

	Expect(v1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

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
			"public-prefix": []string{PrefixV4().String()},
		},
	})
	Expect(err).NotTo(HaveOccurred())

	Expect(apiSrv.Start()).To(Succeed())
	DeferCleanup(apiSrv.Stop)

	Expect(utilsenvtest.WaitUntilAPIServicesReadyWithTimeout(apiServiceTimeout, testEnvExt, k8sClient, scheme.Scheme)).To(Succeed())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	mgrClient = k8sManager.GetClient()

	Expect((&IPAddressGCReconciler{
		Client:       k8sManager.GetClient(),
		APIReader:    k8sManager.GetAPIReader(),
		AbsenceCache: lru.New(100),
	}).SetupWithManager(k8sManager)).To(Succeed())

	Expect((&NetworkIDGCReconciler{
		Client:       k8sManager.GetClient(),
		APIReader:    k8sManager.GetAPIReader(),
		AbsenceCache: lru.New(100),
	}).SetupWithManager(k8sManager)).To(Succeed())

	Expect((&NATGatewayReconciler{
		Client:        k8sManager.GetClient(),
		EventRecorder: &record.FakeRecorder{},
	}).SetupWithManager(k8sManager)).To(Succeed())

	Expect((&LoadBalancerReconciler{
		Client: k8sManager.GetClient(),
	}).SetupWithManager(k8sManager)).To(Succeed())

	Expect((&CertificateApprovalReconciler{
		Client:      k8sManager.GetClient(),
		Recognizers: ironcorenet.Recognizers,
	}).SetupWithManager(k8sManager)).To(Succeed())

	Expect((&NATGatewayAutoscalerReconciler{
		Client: k8sManager.GetClient(),
	}).SetupWithManager(k8sManager)).To(Succeed())

	Expect((&DaemonSetReconciler{
		Client:       k8sManager.GetClient(),
		Expectations: expectations.New(),
	}).SetupWithManager(k8sManager)).To(Succeed())

	schedulerCache = scheduler.NewCache(k8sManager.GetLogger(), scheduler.DefaultCacheStrategy)
	Expect(k8sManager.Add(schedulerCache)).To(Succeed())

	Expect((&SchedulerReconciler{
		Client:        k8sManager.GetClient(),
		EventRecorder: &record.FakeRecorder{},
		Cache:         schedulerCache,
	}).SetupWithManager(k8sManager)).To(Succeed())

	Expect((&NetworkInterfaceNATReleaseReconciler{
		Client:       k8sManager.GetClient(),
		APIReader:    k8sManager.GetAPIReader(),
		AbsenceCache: lru.New(100),
	}).SetupWithManager(k8sManager)).To(Succeed())

	mgrCtx, cancel := context.WithCancel(context.Background())
	DeferCleanup(cancel)

	Expect(apinetclient.SetupNetworkInterfaceNetworkNameFieldIndexer(mgrCtx, k8sManager.GetFieldIndexer())).To(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(k8sManager.Start(mgrCtx)).To(Succeed(), "failed to start manager")
	}()
})
