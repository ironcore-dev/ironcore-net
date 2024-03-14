// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/networkid"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkController", func() {
	ns := SetupNamespace(&k8sClient)
	metalnetNs := SetupNamespace(&k8sClient)
	SetupTest(metalnetNs)

	It("should create a metalnet network for a network", func(ctx SpecContext) {
		By("creating a network")
		network := &v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
			Spec: v1alpha1.NetworkSpec{
				PeeredIDs: []string{"123456", "234567"},
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("parsing the VNI")
		vni, err := networkid.ParseVNI(network.Spec.ID)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the metalnet network to be created")
		metalnetNetwork := &metalnetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metalnetNs.Name,
				Name:      string(network.UID),
			},
		}
		Eventually(Object(metalnetNetwork)).Should(SatisfyAll(
			HaveField("Spec", metalnetv1alpha1.NetworkSpec{
				ID:        vni,
				PeeredIDs: []int32{123456, 234567},
			}),
		))
	})
})
