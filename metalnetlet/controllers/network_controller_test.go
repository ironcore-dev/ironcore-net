// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore-net/networkid"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkController", func() {
	ns := SetupNamespace(&k8sClient)
	metalnetNs := SetupNamespace(&k8sClient)
	SetupTest(metalnetNs)

	It("should create metalnet networks for apinet networks with peerings", func(ctx SpecContext) {
		By("creating a apinet network-1")
		network1 := &apinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "network-1",
			},
		}
		Expect(k8sClient.Create(ctx, network1)).To(Succeed())

		By("creating a apinet network-2")
		network2 := &apinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "network-2",
			},
		}
		Expect(k8sClient.Create(ctx, network2)).To(Succeed())

		By("updating apinet networks spec with peerings")
		baseNetwork1 := network1.DeepCopy()
		network1.Spec.Peerings = []apinetv1alpha1.NetworkPeering{{
			Name: "peering-1",
			Prefixes: []apinetv1alpha1.PeeringPrefix{{
				Name:   "my-prefix",
				Prefix: net.MustParseNewIPPrefix("10.0.0.0/24")}},
			ID: network2.Spec.ID}}
		Expect(k8sClient.Patch(ctx, network1, client.MergeFrom(baseNetwork1))).To(Succeed())

		baseNetwork2 := network2.DeepCopy()
		network2.Spec.Peerings = []apinetv1alpha1.NetworkPeering{{
			Name: "peering-1",
			ID:   network1.Spec.ID}}
		Expect(k8sClient.Patch(ctx, network2, client.MergeFrom(baseNetwork2))).To(Succeed())

		By("parsing the VNI of network-1")
		network1Vni, err := networkid.ParseVNI(network1.Spec.ID)
		Expect(err).NotTo(HaveOccurred())

		By("parsing the VNI of network-2")
		network2Vni, err := networkid.ParseVNI(network2.Spec.ID)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the metalnet networks to be created")
		metalnetNetwork1 := &metalnetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metalnetNs.Name,
				Name:      string(network1.UID),
			},
		}
		Eventually(Object(metalnetNetwork1)).Should(SatisfyAll(
			HaveField("Spec", metalnetv1alpha1.NetworkSpec{
				ID: network1Vni,
				PeeredPrefixes: []metalnetv1alpha1.PeeredPrefix{
					{
						ID:       network2Vni,
						Prefixes: []metalnetv1alpha1.IPPrefix{metalnetv1alpha1.MustParseIPPrefix("10.0.0.0/24")}, // Add desired IPPrefixes here
					}},
				PeeredIDs: []int32{network2Vni},
			}),
		))

		metalnetNetwork2 := &metalnetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metalnetNs.Name,
				Name:      string(network2.UID),
			},
		}
		Eventually(Object(metalnetNetwork2)).Should(SatisfyAll(
			HaveField("Spec", metalnetv1alpha1.NetworkSpec{
				ID:        network2Vni,
				PeeredIDs: []int32{network1Vni},
			}),
		))

		By("updating status of metalnet network peerings")
		Eventually(UpdateStatus(metalnetNetwork1, func() {
			metalnetNetwork1.Status.Peerings = []metalnetv1alpha1.NetworkPeeringStatus{{
				ID:    network2Vni,
				State: metalnetv1alpha1.NetworkPeeringStateReady,
			}}
		})).Should(Succeed())

		Eventually(UpdateStatus(metalnetNetwork2, func() {
			metalnetNetwork2.Status.Peerings = []metalnetv1alpha1.NetworkPeeringStatus{{
				ID:    network1Vni,
				State: metalnetv1alpha1.NetworkPeeringStateReady,
			}}
		})).Should(Succeed())

		By("ensuring apinet network status peerings are also updated")
		Eventually(Object(network1)).Should(SatisfyAll(
			HaveField("Status.Peerings", []apinetv1alpha1.NetworkPeeringStatus{{
				ID:    network2Vni,
				State: apinetv1alpha1.NetworkPeeringStateReady,
			}}),
		))

		Eventually(Object(network2)).Should(SatisfyAll(
			HaveField("Status.Peerings", []apinetv1alpha1.NetworkPeeringStatus{{
				ID:    network1Vni,
				State: apinetv1alpha1.NetworkPeeringStateReady,
			}}),
		))

		By("deleting the networks")
		Expect(k8sClient.Delete(ctx, network1)).To(Succeed())
		Expect(k8sClient.Delete(ctx, network2)).To(Succeed())

		By("waiting for networks to be gone")
		Eventually(Get(network1)).Should(Satisfy(apierrors.IsNotFound))
		Eventually(Get(network2)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding apinet network is gone as well")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(metalnetNetwork1), metalnetNetwork1)).To(Satisfy(apierrors.IsNotFound))
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(metalnetNetwork2), metalnetNetwork2)).To(Satisfy(apierrors.IsNotFound))

	})
})
