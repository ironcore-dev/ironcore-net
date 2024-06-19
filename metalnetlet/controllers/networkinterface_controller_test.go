// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"net/netip"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore/utils/generic"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkInterfaceController", func() {
	ns := SetupNamespace(&k8sClient)
	metalnetNs := SetupNamespace(&k8sClient)
	SetupTest(metalnetNs)

	metalnetNode := SetupMetalnetNode()
	network := SetupNetwork(ns)

	It("should create a metalnet network interface for a network interface", func(ctx SpecContext) {
		By("creating a network")

		By("creating a network interface")
		nic := &v1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nic-",
				Labels: map[string]string{
					"app": "target",
				},
			},
			Spec: v1alpha1.NetworkInterfaceSpec{
				NodeRef: corev1.LocalObjectReference{
					Name: PartitionNodeName(partitionName, metalnetNode.Name),
				},
				NetworkRef: corev1.LocalObjectReference{
					Name: network.Name,
				},
				IPs: []net.IP{
					net.MustParseIP("10.0.0.1"),
				},
			},
		}
		Expect(k8sClient.Create(ctx, nic)).To(Succeed())

		By("creating a load balancer")
		loadBalancer := &v1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "lb-",
			},
			Spec: v1alpha1.LoadBalancerSpec{
				Type:       v1alpha1.LoadBalancerTypePublic,
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				IPs:        []v1alpha1.LoadBalancerIP{{IPFamily: corev1.IPv4Protocol, Name: "ip-1"}},
				Selector:   &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				Template: v1alpha1.InstanceTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"foo": "bar"},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, loadBalancer)).To(Succeed())

		By("creating a load balancer routing")
		loadBalancerRouting := &v1alpha1.LoadBalancerRouting{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      loadBalancer.Name,
			},
			Destinations: []v1alpha1.LoadBalancerDestination{
				{
					IP: net.MustParseIP("10.0.0.1"),
					TargetRef: &v1alpha1.LoadBalancerTargetRef{
						UID:     nic.UID,
						Name:    nic.Name,
						NodeRef: corev1.LocalObjectReference{Name: PartitionNodeName(partitionName, metalnetNode.Name)},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, loadBalancerRouting)).To(Succeed())

		By("creating a network policy rule")
		np := &v1alpha1.NetworkPolicyRule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-policy-",
			},
			NetworkRef: v1alpha1.LocalUIDReference{Name: network.Name, UID: network.UID},
			Targets: []v1alpha1.TargetNetworkInterface{
				{
					IP: net.MustParseIP("10.0.0.1"),
					TargetRef: &v1alpha1.LocalUIDReference{
						UID:  nic.UID,
						Name: nic.Name,
					},
				},
			},
			Priority: generic.Pointer(int32(3000)),
			IngressRules: []v1alpha1.Rule{
				{
					CIDRBlock: []v1alpha1.IPBlock{
						{
							CIDR: net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.0/24")},
							Except: []net.IPPrefix{
								{Prefix: netip.MustParsePrefix("192.168.2.100/32")},
							},
						},
						{
							CIDR: net.IPPrefix{Prefix: netip.MustParsePrefix("2001:db8::/64")},
							Except: []net.IPPrefix{
								{Prefix: netip.MustParsePrefix("2001:db8::1234/128")},
							},
						},
					},
					ObjectIPs: []v1alpha1.ObjectIP{
						{
							Prefix: net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.2.0/24")},
						},
					},
					NetworkPolicyPorts: []v1alpha1.NetworkPolicyPort{
						{
							Protocol: generic.Pointer(corev1.ProtocolTCP),
							Port:     8080,
							EndPort:  generic.Pointer(int32(8090)),
						},
					},
				},
			},
			EgressRules: []v1alpha1.Rule{
				{
					CIDRBlock: []v1alpha1.IPBlock{
						{
							CIDR: net.IPPrefix{Prefix: netip.MustParsePrefix("10.0.0.0/16")},
						},
					},
					ObjectIPs: []v1alpha1.ObjectIP{
						{
							Prefix: net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.178.60/32")},
						},
						{
							Prefix: net.IPPrefix{Prefix: netip.MustParsePrefix("2001:db8:5678:abcd::60/128")},
						},
					},
					NetworkPolicyPorts: []v1alpha1.NetworkPolicyPort{
						{
							Protocol: generic.Pointer(corev1.ProtocolTCP),
							Port:     8095,
						},
						{
							Protocol: generic.Pointer(corev1.ProtocolTCP),
							Port:     9000,
							EndPort:  generic.Pointer(int32(9010)),
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, np)).To(Succeed())

		By("waiting for the network interface to have a finalizer")
		Eventually(Object(nic)).Should(HaveField("Finalizers", []string{PartitionFinalizer(partitionName)}))

		By("waiting for the metalnet network interface to be present with the expected values")
		metalnetNic := &metalnetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metalnetNs.Name,
				Name:      string(nic.UID),
			},
		}

		Eventually(Object(metalnetNic)).Should(SatisfyAll(
			HaveField("Spec.NetworkRef", Equal(corev1.LocalObjectReference{Name: string(network.UID)})),
			HaveField("Spec.IPFamilies", ConsistOf(corev1.IPv4Protocol)),
			HaveField("Spec.IPs", ConsistOf(metalnetv1alpha1.MustParseIP("10.0.0.1"))),
			HaveField("Spec.LoadBalancerTargets", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"Prefix": Equal(netip.PrefixFrom(loadBalancer.Spec.IPs[0].IP.Addr, 32)),
				}),
			)),
			HaveField("Spec.NodeName", Equal(&metalnetNode.Name)),
			HaveField("Spec.FirewallRules", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionIngress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv4Protocol),
					"SourcePrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("192.168.1.0/24")),
					})),
					"DestinationPrefix": BeNil(),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    PointTo(Equal(int32(8080))),
							"EndSrcPort": Equal(int32(8090)),
							"DstPort":    BeNil(),
							"EndDstPort": BeEquivalentTo(0),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionIngress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionDeny),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv4Protocol),
					"SourcePrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("192.168.2.100/32")),
					})),
					"DestinationPrefix": BeNil(),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    PointTo(Equal(int32(8080))),
							"EndSrcPort": Equal(int32(8090)),
							"DstPort":    BeNil(),
							"EndDstPort": BeEquivalentTo(0),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionIngress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv6Protocol),
					"SourcePrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("2001:db8::/64")),
					})),
					"DestinationPrefix": BeNil(),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    PointTo(Equal(int32(8080))),
							"EndSrcPort": Equal(int32(8090)),
							"DstPort":    BeNil(),
							"EndDstPort": BeEquivalentTo(0),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionIngress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionDeny),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv6Protocol),
					"SourcePrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("2001:db8::1234/128")),
					})),
					"DestinationPrefix": BeNil(),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    PointTo(Equal(int32(8080))),
							"EndSrcPort": Equal(int32(8090)),
							"DstPort":    BeNil(),
							"EndDstPort": BeEquivalentTo(0),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionIngress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv4Protocol),
					"SourcePrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("192.168.2.0/24")),
					})),
					"DestinationPrefix": BeNil(),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    PointTo(Equal(int32(8080))),
							"EndSrcPort": Equal(int32(8090)),
							"DstPort":    BeNil(),
							"EndDstPort": BeEquivalentTo(0),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionEgress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv4Protocol),
					"SourcePrefix":   BeNil(),
					"DestinationPrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("10.0.0.0/16")),
					})),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    BeNil(),
							"EndSrcPort": BeEquivalentTo(0),
							"DstPort":    PointTo(Equal(int32(8095))),
							"EndDstPort": BeEquivalentTo(0),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionEgress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv4Protocol),
					"SourcePrefix":   BeNil(),
					"DestinationPrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("10.0.0.0/16")),
					})),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    BeNil(),
							"EndSrcPort": BeEquivalentTo(0),
							"DstPort":    PointTo(Equal(int32(9000))),
							"EndDstPort": Equal(int32(9010)),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionEgress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv4Protocol),
					"SourcePrefix":   BeNil(),
					"DestinationPrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("192.168.178.60/32")),
					})),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    BeNil(),
							"EndSrcPort": BeEquivalentTo(0),
							"DstPort":    PointTo(Equal(int32(8095))),
							"EndDstPort": BeEquivalentTo(0),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionEgress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv4Protocol),
					"SourcePrefix":   BeNil(),
					"DestinationPrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("192.168.178.60/32")),
					})),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    BeNil(),
							"EndSrcPort": BeEquivalentTo(0),
							"DstPort":    PointTo(Equal(int32(9000))),
							"EndDstPort": Equal(int32(9010)),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionEgress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv6Protocol),
					"SourcePrefix":   BeNil(),
					"DestinationPrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("2001:db8:5678:abcd::60/128")),
					})),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    BeNil(),
							"EndSrcPort": BeEquivalentTo(0),
							"DstPort":    PointTo(Equal(int32(8095))),
							"EndDstPort": BeEquivalentTo(0),
						})),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"FirewallRuleID": Not(BeEmpty()),
					"Direction":      Equal(metalnetv1alpha1.FirewallRuleDirectionEgress),
					"Action":         Equal(metalnetv1alpha1.FirewallRuleActionAccept),
					"Priority":       PointTo(Equal(int32(3000))),
					"IpFamily":       Equal(corev1.IPv6Protocol),
					"SourcePrefix":   BeNil(),
					"DestinationPrefix": PointTo(MatchFields(IgnoreExtras, Fields{
						"Prefix": Equal(netip.MustParsePrefix("2001:db8:5678:abcd::60/128")),
					})),
					"ProtocolMatch": PointTo(MatchFields(IgnoreExtras, Fields{
						"ProtocolType": PointTo(Equal(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)),
						"PortRange": PointTo(MatchFields(IgnoreExtras, Fields{
							"SrcPort":    BeNil(),
							"EndSrcPort": BeEquivalentTo(0),
							"DstPort":    PointTo(Equal(int32(9000))),
							"EndDstPort": Equal(int32(9010)),
						})),
					})),
				}),
			)),
		))

		By("updating the metalnet network interface's status")
		Eventually(UpdateStatus(metalnetNic, func() {
			metalnetNic.Status.State = metalnetv1alpha1.NetworkInterfaceStateReady
			metalnetNic.Status.PCIAddress = &metalnetv1alpha1.PCIAddress{
				Domain:   "06",
				Bus:      "0000",
				Slot:     "3",
				Function: "00",
			}
		})).Should(Succeed())

		By("waiting for the network interface to reflect the status values")
		Eventually(Object(nic)).Should(HaveField("Status", v1alpha1.NetworkInterfaceStatus{
			State: v1alpha1.NetworkInterfaceStateReady,
			PCIAddress: &v1alpha1.PCIAddress{
				Domain:   "06",
				Bus:      "0000",
				Slot:     "3",
				Function: "00",
			},
		}))

		By("deleting the network interface")
		Expect(k8sClient.Delete(ctx, nic)).To(Succeed())

		By("waiting for the metalnet network interface to be gone")
		Eventually(Get(metalnetNic)).Should(Satisfy(apierrors.IsNotFound))
	})
})
