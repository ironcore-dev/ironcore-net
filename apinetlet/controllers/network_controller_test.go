// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"strconv"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
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
	ns1 := SetupNamespace(&k8sClient)
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
		apiNetNetwork := &apinetv1alpha1.Network{
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
		apiNetNetwork := &apinetv1alpha1.Network{
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
			Spec: apinetv1alpha1.NetworkSpec{},
		}
		Expect(k8sClient.Create(ctx, apiNetNetwork)).To(Succeed())

		By("waiting for the apinet network to be gone")
		Eventually(Get(apiNetNetwork)).Should(Satisfy(apierrors.IsNotFound))
	})

	It("should update peeredIDs if two networks from different namespaces peers each other correctly", func(ctx SpecContext) {
		By("creating a network network-1")
		network1 := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "network-1",
			},
			Spec: networkingv1alpha1.NetworkSpec{
				Peerings: []networkingv1alpha1.NetworkPeering{
					{
						Name: "peering-1",
						NetworkRef: networkingv1alpha1.NetworkPeeringNetworkRef{
							Name:      "network-2",
							Namespace: ns1.Name,
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, network1)).To(Succeed())

		By("creating a network network-2")
		network2 := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns1.Name,
				Name:      "network-2",
			},
			Spec: networkingv1alpha1.NetworkSpec{
				Peerings: []networkingv1alpha1.NetworkPeering{
					{
						Name: "peering-2",
						NetworkRef: networkingv1alpha1.NetworkPeeringNetworkRef{
							Name:      "network-1",
							Namespace: ns.Name,
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, network2)).To(Succeed())

		By("waiting for the corresponding APINet networks to be created")
		apiNetNetwork1 := &apinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(network1.UID),
			},
		}
		Eventually(Get(apiNetNetwork1)).Should(Succeed())

		apiNetNetwork2 := &apinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(network2.UID),
			},
		}
		Eventually(Get(apiNetNetwork2)).Should(Succeed())

		By("inspecting the created apinet networks")
		Expect(apiNetNetwork1.Labels).To(Equal(apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network1)))
		Expect(apiNetNetwork1.Spec.ID).NotTo(BeEmpty())

		Expect(apiNetNetwork2.Labels).To(Equal(apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network2)))
		Expect(apiNetNetwork2.Spec.ID).NotTo(BeEmpty())

		By("patching networks with peeringClaimRefs")
		baseNetwork1 := network1.DeepCopy()
		network1.Spec.PeeringClaimRefs = []networkingv1alpha1.NetworkPeeringClaimRef{{
			Name:      network2.Name,
			Namespace: network2.Namespace,
			UID:       network2.UID,
		}}
		network1.Status.Peerings = []networkingv1alpha1.NetworkPeeringStatus{{
			Name:  network1.Spec.Peerings[0].Name,
			State: networkingv1alpha1.NetworkPeeringStatePending,
		}}
		Expect(k8sClient.Patch(ctx, network1, client.MergeFrom(baseNetwork1))).To(Succeed())

		baseNetwork2 := network2.DeepCopy()
		network2.Spec.PeeringClaimRefs = []networkingv1alpha1.NetworkPeeringClaimRef{{
			Name:      network1.Name,
			Namespace: network1.Namespace,
			UID:       network1.UID,
		}}
		network2.Status.Peerings = []networkingv1alpha1.NetworkPeeringStatus{{
			Name:  network2.Spec.Peerings[0].Name,
			State: networkingv1alpha1.NetworkPeeringStatePending,
		}}
		Expect(k8sClient.Patch(ctx, network2, client.MergeFrom(baseNetwork2))).To(Succeed())

		By("ensuring apinet network spec peerings are updated")
		Eventually(Object(apiNetNetwork1)).Should(SatisfyAll(
			HaveField("Spec.Peerings", ConsistOf(apinetv1alpha1.NetworkPeering{
				Name: network1.Spec.Peerings[0].Name,
				ID:   apiNetNetwork2.Spec.ID,
			})),
		))

		Eventually(Object(apiNetNetwork2)).Should(SatisfyAll(
			HaveField("Spec.Peerings", ConsistOf(apinetv1alpha1.NetworkPeering{
				Name: network2.Spec.Peerings[0].Name,
				ID:   apiNetNetwork1.Spec.ID,
			})),
		))

		By("patching apinet network peering status")
		apiNetNetwork2ID, _ := strconv.Atoi(apiNetNetwork2.Spec.ID)
		Eventually(UpdateStatus(apiNetNetwork1, func() {
			apiNetNetwork1.Status.Peerings = []apinetv1alpha1.NetworkPeeringStatus{{
				ID:    int32(apiNetNetwork2ID),
				State: apinetv1alpha1.NetworkPeeringStateReady,
			}}
		})).Should(Succeed())

		apiNetNetwork1ID, _ := strconv.Atoi(apiNetNetwork1.Spec.ID)
		Eventually(UpdateStatus(apiNetNetwork2, func() {
			apiNetNetwork2.Status.Peerings = []apinetv1alpha1.NetworkPeeringStatus{{
				ID:    int32(apiNetNetwork1ID),
				State: apinetv1alpha1.NetworkPeeringStateReady,
			}}
		})).Should(Succeed())

		By("ensuring ironcore networks peering status is updated")
		Eventually(Object(network1)).Should(SatisfyAll(
			HaveField("Status.Peerings", ConsistOf(networkingv1alpha1.NetworkPeeringStatus{
				Name:  network1.Spec.Peerings[0].Name,
				State: networkingv1alpha1.NetworkPeeringStateReady,
			})),
		))

		Eventually(Object(network2)).Should(SatisfyAll(
			HaveField("Status.Peerings", ConsistOf(networkingv1alpha1.NetworkPeeringStatus{
				Name:  network2.Spec.Peerings[0].Name,
				State: networkingv1alpha1.NetworkPeeringStateReady,
			})),
		))

		By("deleting the networks")
		Expect(k8sClient.Delete(ctx, network1)).To(Succeed())
		Expect(k8sClient.Delete(ctx, network2)).To(Succeed())

		By("waiting for networks to be gone")
		Eventually(Get(network1)).Should(Satisfy(apierrors.IsNotFound))
		Eventually(Get(network2)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding apinet network is gone as well")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiNetNetwork1), apiNetNetwork1)).To(Satisfy(apierrors.IsNotFound))
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiNetNetwork2), apiNetNetwork2)).To(Satisfy(apierrors.IsNotFound))
	})
})
