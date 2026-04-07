// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/internal/ipaddress"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("IPAddressController", func() {
	ns := SetupNamespace(&k8sClient)
	network := SetupNetwork(ns)

	It("should correctly protect IP addresses in use", func(ctx SpecContext) {
		By("creating a NAT gateway")
		natGateway := &v1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "nat-gateway",
			},
			Spec: v1alpha1.NATGatewaySpec{
				IPFamily: corev1.IPv4Protocol,
				NetworkRef: corev1.LocalObjectReference{
					Name: network.GetName(),
				},
				PortsPerNetworkInterface: 1024,
				IPs: []v1alpha1.NATGatewayIP{
					{
						Name: "ip-0",
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, natGateway)).To(Succeed())

		By("getting the created IP address")
		address := natGateway.Spec.IPs[0].IP
		ipAddress := &v1alpha1.IPAddress{
			ObjectMeta: metav1.ObjectMeta{
				Name: address.String(),
			},
		}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: address.String()}, ipAddress)).To(Succeed())

		By("inspecting the created IP address")
		Expect(ipAddress.Finalizers).To(ConsistOf(ipaddress.ProtectionFinalizer))

		By("deleting the IP address")
		Expect(k8sClient.Delete(ctx, ipAddress)).To(Succeed())

		By("asserting the IP address is still there as the NAT gateway still exists")
		Consistently(komega.Get(ipAddress)).Should(Succeed())

		By("deleting the NAT gateway")
		Expect(k8sClient.Delete(ctx, natGateway)).To(Succeed())

		By("asserting the NAT gateway is gone")
		Eventually(komega.Get(natGateway)).Should(Satisfy(apierrors.IsNotFound))

		By("waiting for the IP address to be gone")
		Eventually(komega.Get(ipAddress)).Should(Satisfy(apierrors.IsNotFound))
	})
})
