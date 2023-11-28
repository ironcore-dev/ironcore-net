// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NATGatewayController", func() {
	ns := SetupNamespace(&k8sClient)
	apiNetNs := SetupNamespace(&k8sClient)
	SetupTest(apiNetNs)
	network, _ := SetupNetwork(ns, apiNetNs)

	It("should allocate a public ip", func(ctx SpecContext) {
		By("creating a nat gateway")
		natGateway := &networkingv1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nat-gateway-",
			},
			Spec: networkingv1alpha1.NATGatewaySpec{
				Type:       networkingv1alpha1.NATGatewayTypePublic,
				IPFamily:   corev1.IPv4Protocol,
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
			},
		}
		Expect(k8sClient.Create(ctx, natGateway)).To(Succeed())

		By("waiting for the APINet NAT gateway to be present")
		apiNetNATGateway := &v1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(natGateway.UID),
			},
		}
		Eventually(Object(apiNetNATGateway)).Should(
			HaveField("Labels", apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), natGateway)),
		)

		By("waiting for the APINet NAT gateway autoscaler to be present")
		apiNetNATGatewayAutoscaler := &v1alpha1.NATGatewayAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(natGateway.UID),
			},
		}
		Eventually(Object(apiNetNATGatewayAutoscaler)).Should(
			SatisfyAll(
				HaveField("Labels", apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), natGateway)),
				BeControlledBy(apiNetNATGateway),
			),
		)
	})
})
