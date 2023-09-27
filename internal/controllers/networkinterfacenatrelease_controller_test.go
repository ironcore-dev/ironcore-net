// Copyright 2022 OnMetal authors
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
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	"github.com/onmetal/onmetal-api-net/apimachinery/api/net"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkInterfaceNATReleaseReconciler", func() {
	ns := SetupNamespace(&k8sClient)
	network := SetupNetwork(ns)

	It("should not release NATs that exist", func(ctx SpecContext) {
		By("creating a network interface")
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

		By("creating a NAT gateway")
		natGateway := &v1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nat-gateway-",
			},
			Spec: v1alpha1.NATGatewaySpec{
				IPFamily:                 corev1.IPv4Protocol,
				NetworkRef:               corev1.LocalObjectReference{Name: network.Name},
				IPs:                      []v1alpha1.NATGatewayIP{{Name: "ip-1"}},
				PortsPerNetworkInterface: 64,
			},
		}
		Expect(k8sClient.Create(ctx, natGateway)).To(Succeed())

		By("waiting for the network interface to have a NAT")
		nat := v1alpha1.NetworkInterfaceNAT{
			IPFamily: corev1.IPv4Protocol,
			ClaimRef: v1alpha1.NetworkInterfaceNATClaimRef{
				Name: natGateway.Name,
				UID:  natGateway.UID,
			},
		}
		Eventually(Object(nic)).Should(HaveField("Spec.NATs", ConsistOf(nat)))

		By("ensuring it stays that way")
		Consistently(Object(nic)).Should(HaveField("Spec.NATs", ConsistOf(nat)))
	})

	It("should release a NAT address of a non-existent NAT gateway", func(ctx SpecContext) {
		By("creating a network interface")
		nic := &v1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nic-",
			},
			Spec: v1alpha1.NetworkInterfaceSpec{
				NodeRef:    corev1.LocalObjectReference{Name: "my-node"},
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				IPs:        []net.IP{net.MustParseIP("10.0.0.1")},
				NATs: []v1alpha1.NetworkInterfaceNAT{
					{
						IPFamily: corev1.IPv4Protocol,
						ClaimRef: v1alpha1.NetworkInterfaceNATClaimRef{
							Name: "should-not-exist",
							UID:  "should-not-exist",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, nic)).To(Succeed())

		By("waiting for the NAT to be released")
		Eventually(Object(nic)).Should(HaveField("Spec.NATs", BeEmpty()))
	})
})
