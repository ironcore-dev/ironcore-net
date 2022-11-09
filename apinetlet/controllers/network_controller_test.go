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

package controllers

import (
	"strconv"

	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/apis/networking/v1alpha1"
	"github.com/onmetal/onmetal-api/testutils"
	mcmeta "github.com/onmetal/poollet/multicluster/meta"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkController", func() {
	ctx := testutils.SetupContext()
	ns := SetupTest(ctx)

	It("should allocate an apinet network", func() {
		By("creating a network")
		network := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("waiting for the corresponding apinet network to be created")
		apiNetNetwork := &onmetalapinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      string(network.UID),
			},
		}
		Eventually(Get(apiNetNetwork)).Should(Succeed())

		By("ensuring the network state is initialized")
		Eventually(func(g Gomega) {
			g.Expect(Get(network)).Should(Succeed())
			g.Expect(network.Status.State).Should(BeEquivalentTo(networkingv1alpha1.NetworkStatePending))
		})

		By("inspecting the created apinet network")
		Expect(mcmeta.IsControlledBy(clusterName, network, apiNetNetwork)).To(BeTrue())
		Expect(apiNetNetwork.Spec).To(Equal(onmetalapinetv1alpha1.NetworkSpec{}))

		By("asserting the network does not report a vni")
		Consistently(Object(network)).
			ShouldNot(HaveField("ObjectMeta.Annotations", HaveKey(onmetalapinetv1alpha1.OnmetalAPINetworkVNIAnnotation)))

		By("setting the network to allocated")
		const vni int32 = 4
		apiNetNetwork.Status.VNI = vni
		Expect(k8sClient.Status().Update(ctx, apiNetNetwork)).To(Succeed())

		By("waiting for the network to reflect the allocated vni")

		Eventually(func(g Gomega) {
			g.Expect(Get(network)).Should(Succeed())
			g.Expect(network.Status.State).Should(BeEquivalentTo(networkingv1alpha1.NetworkStateAvailable))
			g.Expect(network.Spec.ProviderID).Should(BeEquivalentTo(strconv.FormatInt(int64(vni), 10)))
		})

		By("deleting the network")
		Expect(k8sClient.Delete(ctx, network)).To(Succeed())

		By("waiting for it to be gone")
		Eventually(Get(network)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding apinet network is gone as well")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiNetNetwork), apiNetNetwork)).To(Satisfy(apierrors.IsNotFound))
	})

	It("should clean up dangling apinet networks", func() {
		By("creating a apinet network")
		apiNetNetwork := &onmetalapinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "apinet-network-",
			},
			Spec: onmetalapinetv1alpha1.NetworkSpec{},
		}
		mcmeta.SetOwnerReferences(apiNetNetwork, []mcmeta.OwnerReference{
			{
				ClusterName: clusterName,
				APIVersion:  networkingv1alpha1.SchemeGroupVersion.String(),
				Kind:        "Network",
				Namespace:   ns.Name,
				Name:        "some-name",
				UID:         "some-uid",
				Controller:  pointer.Bool(true),
			},
		})
		Expect(k8sClient.Create(ctx, apiNetNetwork)).To(Succeed())

		By("waiting for the apinet network to be gone")
		Eventually(Get(apiNetNetwork)).Should(Satisfy(apierrors.IsNotFound))
	})
})
