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

	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go4.org/netipx"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

const (
	pollingInterval      = 50 * time.Millisecond
	eventuallyTimeout    = 3 * time.Second
	consistentlyDuration = 1 * time.Second
)

func TestControllers(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.SlowSpecThreshold = 10 * time.Second
	SetDefaultConsistentlyPollingInterval(pollingInterval)
	SetDefaultEventuallyPollingInterval(pollingInterval)
	SetDefaultEventuallyTimeout(eventuallyTimeout)
	SetDefaultConsistentlyDuration(consistentlyDuration)

	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

func InitialAvailableIPs() *netipx.IPSet {
	var sb netipx.IPSetBuilder
	sb.AddPrefix(netip.MustParsePrefix("10.0.0.0/31"))
	set, _ := sb.IPSet()
	return set
}

const (
	NoOfIPv4Addresses = 2

	MinVNI   = 1
	MaxVNI   = 2
	NoOfVNIs = (MaxVNI - MinVNI) + 1
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "onmetal-api-net", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = onmetalapinetv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	SetClient(k8sClient)
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func SetupTest(ctx context.Context) *corev1.Namespace {
	ns := &corev1.Namespace{}

	BeforeEach(func() {
		mgrCtx, cancel := context.WithCancel(ctx)
		DeferCleanup(cancel)

		*ns = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "testns-",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create test namespace")

		DeferCleanup(func() {
			Expect(k8sClient.DeleteAllOf(ctx, &onmetalapinetv1alpha1.Network{}, client.InNamespace(ns.Name))).To(Succeed(), "failed to delete networks")
			Expect(k8sClient.DeleteAllOf(ctx, &onmetalapinetv1alpha1.PublicIP{}, client.InNamespace(ns.Name))).To(Succeed(), "failed to delete public ips")
			Expect(k8sClient.Delete(ctx, ns)).To(Succeed(), "failed to delete test namespace")
		})

		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme.Scheme,
			Host:               "127.0.0.1",
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		// register reconciler here
		Expect((&PublicIPReconciler{
			EventRecorder:       &record.FakeRecorder{},
			Client:              k8sManager.GetClient(),
			APIReader:           k8sManager.GetAPIReader(),
			InitialAvailableIPs: InitialAvailableIPs(),
		}).SetupWithManager(k8sManager)).To(Succeed())

		Expect((&NetworkReconciler{
			EventRecorder: &record.FakeRecorder{},
			Client:        k8sManager.GetClient(),
			APIReader:     k8sManager.GetAPIReader(),
			MinVNI:        MinVNI,
			MaxVNI:        MaxVNI,
		}).SetupWithManager(k8sManager)).To(Succeed())

		go func() {
			defer GinkgoRecover()
			Expect(k8sManager.Start(mgrCtx)).To(Succeed(), "failed to start manager")
		}()
	})

	return ns
}
