// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/generic"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("NATGatewayAutoscalerController", func() {
	ns := SetupNamespace(&k8sClient)
	network := SetupNetwork(ns)

	It("should add and remove public IPs depending on the demand", func(ctx SpecContext) {
		By("creating a NAT gateway")
		natGateway := &v1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nat-gateway-",
			},
			Spec: v1alpha1.NATGatewaySpec{
				IPFamily:                 corev1.IPv4Protocol,
				NetworkRef:               corev1.LocalObjectReference{Name: network.Name},
				PortsPerNetworkInterface: 64512,
			},
		}
		Expect(k8sClient.Create(ctx, natGateway)).To(Succeed())

		By("creating a NAT gateway autoscaler")
		natGatewayAutoscaler := &v1alpha1.NATGatewayAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nat-gateway-as-",
			},
			Spec: v1alpha1.NATGatewayAutoscalerSpec{
				NATGatewayRef: corev1.LocalObjectReference{
					Name: natGateway.Name,
				},
				MinPublicIPs: generic.Pointer[int32](1),
				MaxPublicIPs: generic.Pointer[int32](3),
			},
		}
		Expect(k8sClient.Create(ctx, natGatewayAutoscaler)).To(Succeed())
	})
})
