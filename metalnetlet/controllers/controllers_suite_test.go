/*
Copyright 2022 The Metal Authors.

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

package controllers

import (
	"context"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"github.com/onmetal/controller-utils/buildutils"
	"github.com/onmetal/controller-utils/modutils"
	metalnetv1alpha1 "github.com/onmetal/metalnet/api/v1alpha1"
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	utilsenvtest "github.com/onmetal/onmetal-api/utils/envtest"
	"github.com/onmetal/onmetal-api/utils/envtest/apiserver"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg        *rest.Config
	k8sClient  client.Client
	testEnv    *envtest.Environment
	testEnvExt *utilsenvtest.EnvironmentExtensions
)

const (
	pollingInterval      = 50 * time.Millisecond
	eventuallyTimeout    = 3 * time.Second
	consistentlyDuration = 1 * time.Second
	apiServiceTimeout    = 1 * time.Minute
)

const (
	partitionName = "test-metalnetlet"
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
		CRDDirectoryPaths: []string{
			filepath.Join(modutils.Dir("github.com/onmetal/metalnet", "config", "crd", "bases")),
		},
		ErrorIfCRDPathMissing: true,
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
	Expect(metalnetv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	SetClient(k8sClient)

	apiSrv, err := apiserver.New(cfg, apiserver.Options{
		MainPath:     "github.com/onmetal/onmetal-api-net/cmd/apiserver",
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
})

func SetupTest(metalnetNs *corev1.Namespace) {
	BeforeEach(func(ctx SpecContext) {
		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme.Scheme,
			Host:               "127.0.0.1",
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		// register reconciler here
		Expect((&NetworkReconciler{
			Client:            k8sManager.GetClient(),
			MetalnetClient:    k8sManager.GetClient(),
			PartitionName:     partitionName,
			MetalnetNamespace: metalnetNs.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		Expect((&MetalnetNodeReconciler{
			Client:         k8sManager.GetClient(),
			MetalnetClient: k8sManager.GetClient(),
			PartitionName:  partitionName,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		Expect((&NetworkInterfaceReconciler{
			Client:            k8sManager.GetClient(),
			MetalnetClient:    k8sManager.GetClient(),
			PartitionName:     partitionName,
			MetalnetNamespace: metalnetNs.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		Expect((&InstanceReconciler{
			Client:            k8sManager.GetClient(),
			MetalnetClient:    k8sManager.GetClient(),
			PartitionName:     partitionName,
			MetalnetNamespace: metalnetNs.Name,
		}).SetupWithManager(k8sManager, k8sManager.GetCache())).To(Succeed())

		mgrCtx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		go func() {
			defer GinkgoRecover()
			Expect(k8sManager.Start(mgrCtx)).To(Succeed(), "failed to start manager")
		}()
	})
}

func SetupMetalnetNode() *corev1.Node {
	return SetupObjectStruct[*corev1.Node](&k8sClient, func(node *corev1.Node) {
		*node = corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "node-",
			},
		}
	})
}

func SetupNetwork(ns *corev1.Namespace) *v1alpha1.Network {
	return SetupObjectStruct[*v1alpha1.Network](&k8sClient, func(network *v1alpha1.Network) {
		*network = v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
		}
	})
}
