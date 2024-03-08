// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
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

var _ = Describe("NetworkPeeringController", func() {
	ns := SetupNamespace(&k8sClient)
	ns1 := SetupNamespace(&k8sClient)
	apiNetNs := SetupNamespace(&k8sClient)
	SetupTest(apiNetNs)

	It("should peer networks in the same namespace referencing each other", func(ctx SpecContext) {
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
							Namespace: ns.Name,
						},
					},
					{
						Name: "peering-2",
						NetworkRef: networkingv1alpha1.NetworkPeeringNetworkRef{
							Name: "network-3",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, network1)).To(Succeed())

		By("creating a network network-2")
		network2 := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
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

		By("creating a network network-3")
		network3 := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "network-3",
			},
			Spec: networkingv1alpha1.NetworkSpec{
				Peerings: []networkingv1alpha1.NetworkPeering{
					{
						Name: "peering-3",
						NetworkRef: networkingv1alpha1.NetworkPeeringNetworkRef{
							Name:      "network-1",
							Namespace: ns.Name,
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, network3)).To(Succeed())

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

		apiNetNetwork3 := &apinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(network3.UID),
			},
		}
		Eventually(Get(apiNetNetwork3)).Should(Succeed())

		By("inspecting the created apinet networks")
		Expect(apiNetNetwork1.Labels).To(Equal(
			apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network1),
		))
		Expect(apiNetNetwork1.Spec.ID).NotTo(BeEmpty())

		Expect(apiNetNetwork2.Labels).To(Equal(
			apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network2),
		))
		Expect(apiNetNetwork2.Spec.ID).NotTo(BeEmpty())

		By("waiting for networks to reference each other")
		Eventually(Object(network1)).
			Should(SatisfyAll(
				HaveField("Spec.ProviderID", Equal(provider.GetNetworkID(
					apiNetNetwork1.Namespace,
					apiNetNetwork1.Name,
					apiNetNetwork1.Spec.ID,
					apiNetNetwork1.UID,
				))),
				HaveField("Spec.PeeringClaimRefs", ConsistOf(networkingv1alpha1.NetworkPeeringClaimRef{
					Namespace: network2.Namespace,
					Name:      network2.Name,
					UID:       network2.UID,
				}, networkingv1alpha1.NetworkPeeringClaimRef{
					Namespace: network3.Namespace,
					Name:      network3.Name,
					UID:       network3.UID,
				})),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
				HaveField("Status.Peerings", ConsistOf(networkingv1alpha1.NetworkPeeringStatus{
					Name: network1.Spec.Peerings[0].Name,
				}, networkingv1alpha1.NetworkPeeringStatus{
					Name: network1.Spec.Peerings[1].Name,
				})),
			))

		Eventually(Object(network2)).
			Should(SatisfyAll(
				HaveField("Spec.ProviderID", Equal(provider.GetNetworkID(
					apiNetNetwork2.Namespace,
					apiNetNetwork2.Name,
					apiNetNetwork2.Spec.ID,
					apiNetNetwork2.UID,
				))),
				HaveField("Spec.PeeringClaimRefs", ConsistOf(networkingv1alpha1.NetworkPeeringClaimRef{
					Namespace: network1.Namespace,
					Name:      network1.Name,
					UID:       network1.UID,
				})),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
				HaveField("Status.Peerings", ConsistOf(networkingv1alpha1.NetworkPeeringStatus{
					Name: network2.Spec.Peerings[0].Name,
				})),
			))

		Eventually(Object(network3)).
			Should(SatisfyAll(
				HaveField("Spec.ProviderID", Equal(provider.GetNetworkID(
					apiNetNetwork3.Namespace,
					apiNetNetwork3.Name,
					apiNetNetwork3.Spec.ID,
					apiNetNetwork3.UID,
				))),
				HaveField("Spec.PeeringClaimRefs", ConsistOf(networkingv1alpha1.NetworkPeeringClaimRef{
					Namespace: network1.Namespace,
					Name:      network1.Name,
					UID:       network1.UID,
				})),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
				HaveField("Status.Peerings", ConsistOf(networkingv1alpha1.NetworkPeeringStatus{
					Name: network3.Spec.Peerings[0].Name,
				})),
			))

		By("ensuring apinet network peeredIDs are updated")
		Eventually(Object(apiNetNetwork1)).Should(SatisfyAll(
			HaveField("Spec.PeeredIDs", ConsistOf(apiNetNetwork2.Spec.ID, apiNetNetwork3.Spec.ID)),
		))

		Eventually(Object(apiNetNetwork2)).Should(SatisfyAll(
			HaveField("Spec.PeeredIDs", ConsistOf(apiNetNetwork1.Spec.ID)),
		))

		Eventually(Object(apiNetNetwork2)).Should(SatisfyAll(
			HaveField("Spec.PeeredIDs", ConsistOf(apiNetNetwork1.Spec.ID)),
		))

		By("deleting the networks")
		Expect(k8sClient.Delete(ctx, network1)).To(Succeed())
		Expect(k8sClient.Delete(ctx, network2)).To(Succeed())
		Expect(k8sClient.Delete(ctx, network3)).To(Succeed())

		By("waiting for networks to be gone")
		Eventually(Get(network1)).Should(Satisfy(apierrors.IsNotFound))
		Eventually(Get(network2)).Should(Satisfy(apierrors.IsNotFound))
		Eventually(Get(network3)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding apinet network is gone as well")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiNetNetwork1), apiNetNetwork1)).To(Satisfy(apierrors.IsNotFound))
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiNetNetwork2), apiNetNetwork2)).To(Satisfy(apierrors.IsNotFound))
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiNetNetwork3), apiNetNetwork3)).To(Satisfy(apierrors.IsNotFound))
	})

	It("should peer two networks from different namespaces if they reference each other correctly", func(ctx SpecContext) {
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
		Expect(apiNetNetwork1.Labels).To(Equal(
			apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network1),
		))
		Expect(apiNetNetwork1.Spec.ID).NotTo(BeEmpty())

		Expect(apiNetNetwork2.Labels).To(Equal(
			apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network2),
		))
		Expect(apiNetNetwork2.Spec.ID).NotTo(BeEmpty())

		By("waiting for networks to reference each other")
		Eventually(Object(network1)).
			Should(SatisfyAll(
				HaveField("Spec.ProviderID", Equal(provider.GetNetworkID(
					apiNetNetwork1.Namespace,
					apiNetNetwork1.Name,
					apiNetNetwork1.Spec.ID,
					apiNetNetwork1.UID,
				))),
				HaveField("Spec.PeeringClaimRefs", ConsistOf(networkingv1alpha1.NetworkPeeringClaimRef{
					Namespace: network2.Namespace,
					Name:      network2.Name,
					UID:       network2.UID,
				})),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
				HaveField("Status.Peerings", ConsistOf(networkingv1alpha1.NetworkPeeringStatus{
					Name: network1.Spec.Peerings[0].Name,
				})),
			))

		Eventually(Object(network2)).
			Should(SatisfyAll(
				HaveField("Spec.ProviderID", Equal(provider.GetNetworkID(
					apiNetNetwork2.Namespace,
					apiNetNetwork2.Name,
					apiNetNetwork2.Spec.ID,
					apiNetNetwork2.UID,
				))),
				HaveField("Spec.PeeringClaimRefs", ConsistOf(networkingv1alpha1.NetworkPeeringClaimRef{
					Namespace: network1.Namespace,
					Name:      network1.Name,
					UID:       network1.UID,
				})),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
				HaveField("Status.Peerings", ConsistOf(networkingv1alpha1.NetworkPeeringStatus{
					Name: network2.Spec.Peerings[0].Name,
				})),
			))

		By("ensuring apinet network peeredIDs are updated")
		Eventually(Object(apiNetNetwork1)).Should(SatisfyAll(
			HaveField("Spec.PeeredIDs", ConsistOf(apiNetNetwork2.Spec.ID)),
		))

		Eventually(Object(apiNetNetwork2)).Should(SatisfyAll(
			HaveField("Spec.PeeredIDs", ConsistOf(apiNetNetwork1.Spec.ID)),
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

	It("should not peer two networks if they dont exactly reference each other", func(ctx SpecContext) {
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
				Namespace: ns.Name,
				Name:      "network-2",
			},
			Spec: networkingv1alpha1.NetworkSpec{
				Peerings: []networkingv1alpha1.NetworkPeering{
					{
						Name: "peering-2",
						NetworkRef: networkingv1alpha1.NetworkPeeringNetworkRef{
							Name:      "network-other",
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
		Expect(apiNetNetwork1.Labels).To(Equal(
			apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network1),
		))
		Expect(apiNetNetwork1.Spec.ID).NotTo(BeEmpty())

		Expect(apiNetNetwork2.Labels).To(Equal(
			apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), network2),
		))
		Expect(apiNetNetwork2.Spec.ID).NotTo(BeEmpty())

		By("ensuring both networks do not get peered")
		Eventually(Object(network1)).
			Should(SatisfyAll(
				HaveField("Spec.ProviderID", Equal(provider.GetNetworkID(
					apiNetNetwork1.Namespace,
					apiNetNetwork1.Name,
					apiNetNetwork1.Spec.ID,
					apiNetNetwork1.UID,
				))),
				HaveField("Spec.PeeringClaimRefs", BeEmpty()),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
				HaveField("Status.Peerings", BeEmpty()),
			))

		Eventually(Object(network2)).
			Should(SatisfyAll(
				HaveField("Spec.ProviderID", Equal(provider.GetNetworkID(
					apiNetNetwork2.Namespace,
					apiNetNetwork2.Name,
					apiNetNetwork2.Spec.ID,
					apiNetNetwork2.UID,
				))),
				HaveField("Spec.PeeringClaimRefs", BeEmpty()),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
				HaveField("Status.Peerings", BeEmpty()),
			))

		By("ensuring apinet network peeredIDs are empty")
		Eventually(Object(apiNetNetwork1)).Should(SatisfyAll(
			HaveField("Spec.PeeredIDs", BeEmpty()),
		))

		Eventually(Object(apiNetNetwork2)).Should(SatisfyAll(
			HaveField("Spec.PeeredIDs", BeEmpty()),
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
