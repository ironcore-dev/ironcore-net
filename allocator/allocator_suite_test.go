// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package allocator_test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	. "github.com/onmetal/onmetal-api-net/allocator"
	commonv1alpha1 "github.com/onmetal/onmetal-api/apis/common/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go4.org/netipx"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	cfg       *rest.Config
	testEnv   *envtest.Environment
	k8sClient client.Client
)

const (
	slowSpecThreshold    = 10 * time.Second
	eventuallyTimeout    = 3 * time.Second
	pollingInterval      = 50 * time.Millisecond
	consistentlyDuration = 1 * time.Second
)

var (
	ipv4Prefix = commonv1alpha1.MustParseIPPrefix("10.0.0.0/24")
)

func TestCore(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.SlowSpecThreshold = slowSpecThreshold
	SetDefaultConsistentlyPollingInterval(pollingInterval)
	SetDefaultEventuallyPollingInterval(pollingInterval)
	SetDefaultEventuallyTimeout(eventuallyTimeout)
	SetDefaultConsistentlyDuration(consistentlyDuration)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Allocator Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	var err error
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	DeferCleanup(testEnv.Stop)

	// Init package-level k8sClient
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	SetClient(k8sClient)
})

func SetupTest(ctx context.Context) (*corev1.Namespace, *SecretAllocator) {
	ns := &corev1.Namespace{}
	allocator := &SecretAllocator{}
	var cancelMgr context.CancelFunc

	BeforeEach(func() {
		var mgrCtx context.Context
		mgrCtx, cancelMgr = context.WithCancel(ctx)

		*ns = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-ns-",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed(), "failed to create test namespace")

		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme.Scheme,
			Host:               "127.0.0.1",
			MetricsBindAddress: "0",
		})
		Expect(err).NotTo(HaveOccurred())

		allocatorSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "allocator-",
			},
		}
		Expect(k8sClient.Create(ctx, allocatorSecret)).To(Succeed())

		alloc, err := NewSecretAllocator(cfg, Options{
			IPv4Set:   mustParseIPSet("10.0.0.0/24"),
			SecretKey: client.ObjectKeyFromObject(allocatorSecret),
		})
		Expect(err).NotTo(HaveOccurred())
		*allocator = *alloc

		go func() {
			defer GinkgoRecover()
			Expect(k8sManager.Start(mgrCtx)).To(Succeed(), "failed to start manager")
		}()
	})

	AfterEach(func() {
		if cancelMgr != nil {
			cancelMgr()
		}
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed(), "failed to delete test namespace")
	})

	return ns, allocator
}

func mustParseIPSet(prefixes ...string) *netipx.IPSet {
	var bldr netipx.IPSetBuilder
	for _, prefix := range prefixes {
		p := netip.MustParsePrefix(prefix)
		bldr.AddPrefix(p)
	}
	set, err := bldr.IPSet()
	utilruntime.Must(err)
	return set
}
