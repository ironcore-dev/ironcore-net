// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NATGatewayController", func() {
	ns := SetupNamespace(&k8sClient)
	network := SetupNetwork(ns)
	networkWithoutNAT := SetupNetwork(ns)

	It("should correctly reconcile the NAT gateway", func(ctx SpecContext) {
		By("creating a NAT gateway")
		natGateway := &v1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nat-gateway-",
			},
			Spec: v1alpha1.NATGatewaySpec{
				IPFamily:                 corev1.IPv4Protocol,
				NetworkRef:               corev1.LocalObjectReference{Name: network.Name},
				PortsPerNetworkInterface: 64,
				IPs: []v1alpha1.NATGatewayIP{
					{Name: "ip-1"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, natGateway)).To(Succeed())
		natGatewayIP := natGateway.Spec.IPs[0].IP

		ipaddressForNGW := &v1alpha1.IPAddress{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: natGatewayIP.String()}, ipaddressForNGW)
		Expect(err).NotTo(HaveOccurred())
		Expect(ipaddressForNGW.Spec.ClaimRef.Name).ToNot(BeEmpty())

		ipForNGW := &v1alpha1.IP{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: ipaddressForNGW.Spec.ClaimRef.Name, Namespace: ipaddressForNGW.Spec.ClaimRef.Namespace}, ipForNGW)
		Expect(err).NotTo(HaveOccurred())
		Expect(ipForNGW.Spec.ClaimRef).ToNot(BeNil())

		By("creating a network interface as potential target")
		nic := &v1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nic-",
			},
			Spec: v1alpha1.NetworkInterfaceSpec{
				NodeRef:    corev1.LocalObjectReference{Name: "my-node"},
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				IPs:        []net.IP{net.MustParseIP("10.0.0.1")},
			},
		}
		Expect(k8sClient.Create(ctx, nic)).To(Succeed())

		By("creating a network interface using network not claiming NAT Gateway")
		nic1 := &v1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nic-",
			},
			Spec: v1alpha1.NetworkInterfaceSpec{
				NodeRef:    corev1.LocalObjectReference{Name: "my-node"},
				NetworkRef: corev1.LocalObjectReference{Name: networkWithoutNAT.Name},
				IPs:        []net.IP{net.MustParseIP("10.0.0.2")},
			},
		}
		Expect(k8sClient.Create(ctx, nic1)).To(Succeed())

		By("waiting for the NAT table to be updated")
		natTable := &v1alpha1.NATTable{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      natGateway.Name,
			},
		}
		Eventually(Object(natTable)).Should(HaveField("IPs", ConsistOf(
			v1alpha1.NATIP{
				IP: natGatewayIP,
				Sections: []v1alpha1.NATIPSection{
					{
						IP:      net.MustParseIP("10.0.0.1"),
						Port:    1024,
						EndPort: 1087,
						TargetRef: &v1alpha1.NATTableIPTargetRef{
							UID:     nic.UID,
							Name:    nic.Name,
							NodeRef: nic.Spec.NodeRef,
						},
					},
				},
			},
		)))
	})
})
