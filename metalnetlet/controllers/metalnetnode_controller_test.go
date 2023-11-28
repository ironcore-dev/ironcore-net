// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("MetalnetNodeController", func() {
	metalnetNs := SetupNamespace(&k8sClient)
	SetupTest(metalnetNs)

	metalnetNode := SetupMetalnetNode()

	It("should reconcile the metalnet nodes with the nodes", func(ctx SpecContext) {
		By("waiting for the metalnet node to have a finalizer")
		Eventually(Object(metalnetNode)).Should(HaveField("Finalizers", []string{metalnetNodeFinalizer}))

		By("waiting for the corresponding node to appear")
		node := &v1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: PartitionNodeName(partitionName, metalnetNode.Name),
			},
		}
		Eventually(Object(node)).Should(HaveField("Labels", map[string]string{
			"the":                           "node",
			v1alpha1.TopologyPartitionLabel: partitionName,
		}))

		By("deleting the metalnet node")
		Expect(k8sClient.Delete(ctx, metalnetNode)).To(Succeed())

		By("waiting for the node to be gone")
		Eventually(Get(node)).Should(Satisfy(apierrors.IsNotFound))
	})
})
