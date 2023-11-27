// Copyright 2022 IronCore authors
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
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	"github.com/ironcore-dev/ironcore-net/apinetlet/provider"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkController", func() {
	ns := SetupNamespace(&k8sClient)
	apiNetNs := SetupNamespace(&k8sClient)
	SetupTest(apiNetNs)

	It("should allocate an APINet network", func(ctx SpecContext) {
		By("creating a network")
		network := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("waiting for the corresponding APINet network to be created")
		apiNetNetwork := &v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(network.UID),
			},
		}
		Eventually(Get(apiNetNetwork)).Should(Succeed())

		By("inspecting the created apinet network")
		Expect(apiNetNetwork.Labels).To(Equal(
			apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network),
		))
		Expect(apiNetNetwork.Spec.ID).NotTo(BeEmpty())

		By("waiting for the network to reflect the allocated vni")
		Eventually(Object(network)).
			Should(SatisfyAll(
				HaveField("Spec.ProviderID", Equal(provider.GetNetworkID(
					apiNetNetwork.Namespace,
					apiNetNetwork.Name,
					apiNetNetwork.Spec.ID,
					apiNetNetwork.UID,
				))),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
			))

		By("deleting the network")
		Expect(k8sClient.Delete(ctx, network)).To(Succeed())

		By("waiting for it to be gone")
		Eventually(Get(network)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding apinet network is gone as well")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiNetNetwork), apiNetNetwork)).To(Satisfy(apierrors.IsNotFound))
	})

	It("should clean up dangling apinet networks", func(ctx SpecContext) {
		By("creating a apinet network")
		apiNetNetwork := &v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    apiNetNs.Name,
				GenerateName: "apinet-network-",
				Labels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(),
					&networkingv1alpha1.Network{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns.Name,
							Name:      "should-not-exist",
							UID:       "some-uid",
						},
					},
				),
			},
			Spec: v1alpha1.NetworkSpec{},
		}
		Expect(k8sClient.Create(ctx, apiNetNetwork)).To(Succeed())

		By("waiting for the apinet network to be gone")
		Eventually(Get(apiNetNetwork)).Should(Satisfy(apierrors.IsNotFound))
	})
})
