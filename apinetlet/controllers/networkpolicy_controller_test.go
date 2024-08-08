// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"net/netip"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore/api/core/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/generic"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkPolicyController", func() {
	ns := SetupNamespace(&k8sClient)
	apiNetNs := SetupNamespace(&k8sClient)
	SetupTest(apiNetNs)

	network, apiNetNetwork := SetupNetwork(ns, apiNetNs)

	It("should manage and reconcile the APINet network policy and its rules without target apinet nic", func(ctx SpecContext) {
		By("creating an apinet nic for ingress")
		ingressApiNetNic := &apinetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    apiNetNs.Name,
				GenerateName: "apinet-nic-",
				Labels: map[string]string{
					"rule": "ingress",
				},
			},
			Spec: apinetv1alpha1.NetworkInterfaceSpec{
				NetworkRef: corev1.LocalObjectReference{Name: apiNetNetwork.Name},
				IPs:        []net.IP{net.MustParseIP("192.168.178.50")},
				NodeRef:    corev1.LocalObjectReference{Name: "test-node"},
			},
		}
		Expect(k8sClient.Create(ctx, ingressApiNetNic)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ingressApiNetNic)

		By("setting the ingress apinet nic to be ready")
		Eventually(UpdateStatus(ingressApiNetNic, func() {
			ingressApiNetNic.Status.State = apinetv1alpha1.NetworkInterfaceStateReady
		})).Should(Succeed())

		By("creating an apinet nic for egress")
		egressApiNetNic := &apinetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    apiNetNs.Name,
				GenerateName: "apinet-nic-",
				Labels: map[string]string{
					"rule": "egress",
				},
			},
			Spec: apinetv1alpha1.NetworkInterfaceSpec{
				NetworkRef: corev1.LocalObjectReference{Name: apiNetNetwork.Name},
				IPs:        []net.IP{net.MustParseIP("192.168.178.60")},
				NodeRef:    corev1.LocalObjectReference{Name: "test-node"},
			},
		}
		Expect(k8sClient.Create(ctx, egressApiNetNic)).To(Succeed())
		DeferCleanup(k8sClient.Delete, egressApiNetNic)

		By("setting the egress apinet nic to be ready")
		Eventually(UpdateStatus(egressApiNetNic, func() {
			egressApiNetNic.Status.State = apinetv1alpha1.NetworkInterfaceStateReady
		})).Should(Succeed())

		By("creating an ironcore network policy")
		np := &networkingv1alpha1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-policy-",
			},
			Spec: networkingv1alpha1.NetworkPolicySpec{
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				NetworkInterfaceSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "target",
					},
				},
				PolicyTypes: []networkingv1alpha1.PolicyType{networkingv1alpha1.PolicyTypeIngress, networkingv1alpha1.PolicyTypeEgress},
				Ingress: []networkingv1alpha1.NetworkPolicyIngressRule{
					{
						Ports: []networkingv1alpha1.NetworkPolicyPort{
							{
								Protocol: generic.Pointer(corev1.ProtocolTCP),
								Port:     80,
							},
							{
								Protocol: generic.Pointer(corev1.ProtocolUDP),
								Port:     8080,
								EndPort:  generic.Pointer(int32(8090)),
							},
						},
						From: []networkingv1alpha1.NetworkPolicyPeer{
							{
								ObjectSelector: &corev1alpha1.ObjectSelector{
									Kind: "NetworkInterface",
									LabelSelector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"rule": "ingress",
										},
									},
								},
							},
							{
								IPBlock: &networkingv1alpha1.IPBlock{
									CIDR: commonv1alpha1.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.0/24")},
									Except: []commonv1alpha1.IPPrefix{
										{Prefix: netip.MustParsePrefix("192.168.1.1/32")},
										{Prefix: netip.MustParsePrefix("192.168.1.2/32")}},
								},
							},
						},
					},
				},
				Egress: []networkingv1alpha1.NetworkPolicyEgressRule{
					{
						Ports: []networkingv1alpha1.NetworkPolicyPort{
							{
								Protocol: generic.Pointer(corev1.ProtocolTCP),
								Port:     443,
							},
						},
						To: []networkingv1alpha1.NetworkPolicyPeer{
							{
								ObjectSelector: &corev1alpha1.ObjectSelector{
									Kind: "NetworkInterface",
									LabelSelector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"rule": "egress",
										},
									},
								},
							},
							{
								IPBlock: &networkingv1alpha1.IPBlock{
									CIDR: commonv1alpha1.IPPrefix{Prefix: netip.MustParsePrefix("10.0.0.0/16")},
								},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, np)).To(Succeed())
		DeferCleanup(k8sClient.Delete, np)

		By("waiting for the APINet network policy to exist with correct specs")
		apiNetNP := &apinetv1alpha1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(np.UID),
			},
		}

		Eventually(Object(apiNetNP)).Should(SatisfyAll(
			HaveField("Labels", apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), np)),
			HaveField("Spec.NetworkRef", Equal(corev1.LocalObjectReference{Name: apiNetNetwork.Name})),
			HaveField("Spec.NetworkInterfaceSelector.MatchLabels", Equal(map[string]string{"app": "target"})),
			HaveField("Spec.PolicyTypes", ConsistOf(apinetv1alpha1.PolicyTypeIngress, apinetv1alpha1.PolicyTypeEgress)),
			HaveField("Spec.Ingress", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"Ports": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolTCP)),
							"Port":     Equal(int32(80)),
						}),
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolUDP)),
							"Port":     Equal(int32(8080)),
							"EndPort":  PointTo(Equal(int32(8090))),
						}),
					),
					"From": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"ObjectSelector": PointTo(MatchFields(IgnoreExtras, Fields{
								"Kind": Equal("NetworkInterface"),
								"LabelSelector": Equal(metav1.LabelSelector{
									MatchLabels: map[string]string{
										"rule": "ingress",
									},
								}),
							})),
						}),
						MatchFields(IgnoreExtras, Fields{
							"IPBlock": PointTo(MatchFields(IgnoreExtras, Fields{
								"CIDR": Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.0/24")}),
								"Except": ConsistOf(
									Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.1/32")}),
									Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.2/32")}),
								),
							})),
						}),
					),
				}),
			)),
			HaveField("Spec.Egress", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"Ports": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolTCP)),
							"Port":     Equal(int32(443)),
						}),
					),
					"To": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"ObjectSelector": PointTo(MatchFields(IgnoreExtras, Fields{
								"Kind": Equal("NetworkInterface"),
								"LabelSelector": Equal(metav1.LabelSelector{
									MatchLabels: map[string]string{
										"rule": "egress",
									},
								}),
							})),
						}),
						MatchFields(IgnoreExtras, Fields{
							"IPBlock": PointTo(MatchFields(IgnoreExtras, Fields{
								"CIDR": Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("10.0.0.0/16")}),
							})),
						}),
					),
				}),
			)),
		))

		By("waiting for the APINet network policy rule to exist with empty targets and other correct specs ")
		networkPolicyRule := &apinetv1alpha1.NetworkPolicyRule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNP.Namespace,
				Name:      apiNetNP.Name,
			},
		}

		Eventually(Object(networkPolicyRule)).Should(SatisfyAll(
			HaveField("NetworkRef", apinetv1alpha1.LocalUIDReference{
				Name: apiNetNetwork.Name,
				UID:  apiNetNetwork.UID,
			}),
			HaveField("Targets", BeEmpty()),
			HaveField("IngressRules", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"NetworkPolicyPorts": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolTCP)),
							"Port":     Equal(int32(80)),
							"EndPort":  BeNil(),
						}),
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolUDP)),
							"Port":     Equal(int32(8080)),
							"EndPort":  PointTo(Equal(int32(8090))),
						}),
					),
					"CIDRBlock": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"CIDR": Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.0/24")}),
							"Except": ConsistOf(
								Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.1/32")}),
								Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.2/32")}),
							)}),
					),
					"ObjectIPs": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"IPFamily": Equal(corev1.IPv4Protocol),
							"Prefix":   Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.178.50/32")})}),
					),
				}),
			)),
			HaveField("EgressRules", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"NetworkPolicyPorts": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolTCP)),
							"Port":     Equal(int32(443)),
							"EndPort":  BeNil(),
						}),
					),
					"CIDRBlock": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"CIDR":   Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("10.0.0.0/16")}),
							"Except": BeEmpty()}),
					),
					"ObjectIPs": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"IPFamily": Equal(corev1.IPv4Protocol),
							"Prefix":   Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.178.60/32")})}),
					),
				}),
			)),
		))
	})

	It("should manage and reconcile the APINet network policy and its rules with available target apinet nic ", func(ctx SpecContext) {
		By("creating a target apinet nic")
		targetApiNetNic1 := &apinetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    apiNetNs.Name,
				GenerateName: "apinet-nic-",
				Labels: map[string]string{
					"app": "target",
				},
			},
			Spec: apinetv1alpha1.NetworkInterfaceSpec{
				NetworkRef: corev1.LocalObjectReference{Name: apiNetNetwork.Name},
				IPs:        []net.IP{net.MustParseIP("192.168.178.1")},
				NodeRef:    corev1.LocalObjectReference{Name: "test-node"},
			},
		}
		Expect(k8sClient.Create(ctx, targetApiNetNic1)).To(Succeed())
		DeferCleanup(k8sClient.Delete, targetApiNetNic1)

		By("setting the target apinet nic to be ready")
		Eventually(UpdateStatus(targetApiNetNic1, func() {
			targetApiNetNic1.Status.State = apinetv1alpha1.NetworkInterfaceStateReady
		})).Should(Succeed())

		By("creating an apinet nic for ingress")
		ingressApiNetNic := &apinetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    apiNetNs.Name,
				GenerateName: "apinet-nic-",
				Labels: map[string]string{
					"rule": "ingress",
				},
			},
			Spec: apinetv1alpha1.NetworkInterfaceSpec{
				NetworkRef: corev1.LocalObjectReference{Name: apiNetNetwork.Name},
				IPs:        []net.IP{net.MustParseIP("192.168.178.50")},
				NodeRef:    corev1.LocalObjectReference{Name: "test-node"},
			},
		}
		Expect(k8sClient.Create(ctx, ingressApiNetNic)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ingressApiNetNic)

		By("setting the ingress apinet nic to be ready")
		Eventually(UpdateStatus(ingressApiNetNic, func() {
			ingressApiNetNic.Status.State = apinetv1alpha1.NetworkInterfaceStateReady
		})).Should(Succeed())

		By("creating an apinet nic for egress")
		egressApiNetNic := &apinetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    apiNetNs.Name,
				GenerateName: "apinet-nic-",
				Labels: map[string]string{
					"rule": "egress",
				},
			},
			Spec: apinetv1alpha1.NetworkInterfaceSpec{
				NetworkRef: corev1.LocalObjectReference{Name: apiNetNetwork.Name},
				IPs:        []net.IP{net.MustParseIP("192.168.178.60")},
				NodeRef:    corev1.LocalObjectReference{Name: "test-node"},
			},
		}
		Expect(k8sClient.Create(ctx, egressApiNetNic)).To(Succeed())
		DeferCleanup(k8sClient.Delete, egressApiNetNic)

		By("setting the egress apinet nic to be ready")
		Eventually(UpdateStatus(egressApiNetNic, func() {
			egressApiNetNic.Status.State = apinetv1alpha1.NetworkInterfaceStateReady
		})).Should(Succeed())

		By("creating an ironcore network policy")
		np := &networkingv1alpha1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-policy-",
			},
			Spec: networkingv1alpha1.NetworkPolicySpec{
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				NetworkInterfaceSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "target",
					},
				},
				PolicyTypes: []networkingv1alpha1.PolicyType{networkingv1alpha1.PolicyTypeIngress, networkingv1alpha1.PolicyTypeEgress},
				Ingress: []networkingv1alpha1.NetworkPolicyIngressRule{
					{
						Ports: []networkingv1alpha1.NetworkPolicyPort{
							{
								Protocol: generic.Pointer(corev1.ProtocolTCP),
								Port:     80,
							},
							{
								Protocol: generic.Pointer(corev1.ProtocolUDP),
								Port:     8080,
								EndPort:  generic.Pointer(int32(8090)),
							},
						},
						From: []networkingv1alpha1.NetworkPolicyPeer{
							{
								ObjectSelector: &corev1alpha1.ObjectSelector{
									Kind: "NetworkInterface",
									LabelSelector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"rule": "ingress",
										},
									},
								},
							},
							{
								IPBlock: &networkingv1alpha1.IPBlock{
									CIDR: commonv1alpha1.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.0/24")},
									Except: []commonv1alpha1.IPPrefix{
										{Prefix: netip.MustParsePrefix("192.168.1.1/32")},
										{Prefix: netip.MustParsePrefix("192.168.1.2/32")}},
								},
							},
						},
					},
				},
				Egress: []networkingv1alpha1.NetworkPolicyEgressRule{
					{
						Ports: []networkingv1alpha1.NetworkPolicyPort{
							{
								Protocol: generic.Pointer(corev1.ProtocolTCP),
								Port:     443,
							},
						},
						To: []networkingv1alpha1.NetworkPolicyPeer{
							{
								ObjectSelector: &corev1alpha1.ObjectSelector{
									Kind: "NetworkInterface",
									LabelSelector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"rule": "egress",
										},
									},
								},
							},
							{
								IPBlock: &networkingv1alpha1.IPBlock{
									CIDR: commonv1alpha1.IPPrefix{Prefix: netip.MustParsePrefix("10.0.0.0/16")},
								},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, np)).To(Succeed())
		DeferCleanup(k8sClient.Delete, np)

		By("waiting for the APINet network policy to exist with correct specs")
		apiNetNP := &apinetv1alpha1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(np.UID),
			},
		}

		Eventually(Object(apiNetNP)).Should(SatisfyAll(
			HaveField("Labels", apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), np)),
			HaveField("Spec.NetworkRef", Equal(corev1.LocalObjectReference{Name: apiNetNetwork.Name})),
			HaveField("Spec.NetworkInterfaceSelector.MatchLabels", Equal(map[string]string{"app": "target"})),
			HaveField("Spec.PolicyTypes", ConsistOf(apinetv1alpha1.PolicyTypeIngress, apinetv1alpha1.PolicyTypeEgress)),
			HaveField("Spec.Ingress", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"Ports": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolTCP)),
							"Port":     Equal(int32(80)),
						}),
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolUDP)),
							"Port":     Equal(int32(8080)),
							"EndPort":  PointTo(Equal(int32(8090))),
						}),
					),
					"From": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"ObjectSelector": PointTo(MatchFields(IgnoreExtras, Fields{
								"Kind": Equal("NetworkInterface"),
								"LabelSelector": Equal(metav1.LabelSelector{
									MatchLabels: map[string]string{
										"rule": "ingress",
									},
								}),
							})),
						}),
						MatchFields(IgnoreExtras, Fields{
							"IPBlock": PointTo(MatchFields(IgnoreExtras, Fields{
								"CIDR": Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.0/24")}),
								"Except": ConsistOf(
									Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.1/32")}),
									Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.2/32")}),
								),
							})),
						}),
					),
				}),
			)),
			HaveField("Spec.Egress", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"Ports": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolTCP)),
							"Port":     Equal(int32(443)),
						}),
					),
					"To": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"ObjectSelector": PointTo(MatchFields(IgnoreExtras, Fields{
								"Kind": Equal("NetworkInterface"),
								"LabelSelector": Equal(metav1.LabelSelector{
									MatchLabels: map[string]string{
										"rule": "egress",
									},
								}),
							})),
						}),
						MatchFields(IgnoreExtras, Fields{
							"IPBlock": PointTo(MatchFields(IgnoreExtras, Fields{
								"CIDR": Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("10.0.0.0/16")}),
							})),
						}),
					),
				}),
			)),
		))

		By("waiting for the APINet network policy rule to exist with correct specs")
		networkPolicyRule := &apinetv1alpha1.NetworkPolicyRule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNP.Namespace,
				Name:      apiNetNP.Name,
			},
		}

		Eventually(Object(networkPolicyRule)).Should(SatisfyAll(
			HaveField("NetworkRef", apinetv1alpha1.LocalUIDReference{
				Name: apiNetNetwork.Name,
				UID:  apiNetNetwork.UID,
			}),
			HaveField("Targets", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"IP": Equal(net.MustParseIP("192.168.178.1")),
					"TargetRef": PointTo(MatchFields(IgnoreExtras, Fields{
						"UID":  Equal(targetApiNetNic1.UID),
						"Name": Equal(targetApiNetNic1.Name),
					})),
				}),
			)),
			HaveField("IngressRules", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"NetworkPolicyPorts": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolTCP)),
							"Port":     Equal(int32(80)),
							"EndPort":  BeNil(),
						}),
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolUDP)),
							"Port":     Equal(int32(8080)),
							"EndPort":  PointTo(Equal(int32(8090))),
						}),
					),
					"CIDRBlock": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"CIDR": Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.0/24")}),
							"Except": ConsistOf(
								Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.1/32")}),
								Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.1.2/32")}),
							)}),
					),
					"ObjectIPs": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"IPFamily": Equal(corev1.IPv4Protocol),
							"Prefix":   Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.178.50/32")})}),
					),
				}),
			)),
			HaveField("EgressRules", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"NetworkPolicyPorts": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"Protocol": PointTo(Equal(corev1.ProtocolTCP)),
							"Port":     Equal(int32(443)),
							"EndPort":  BeNil(),
						}),
					),
					"CIDRBlock": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"CIDR":   Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("10.0.0.0/16")}),
							"Except": BeEmpty()}),
					),
					"ObjectIPs": ConsistOf(
						MatchFields(IgnoreExtras, Fields{
							"IPFamily": Equal(corev1.IPv4Protocol),
							"Prefix":   Equal(net.IPPrefix{Prefix: netip.MustParsePrefix("192.168.178.60/32")})}),
					),
				}),
			)),
		))

		By("creating another target apinet nic")
		targetApiNetNic2 := &apinetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    apiNetNs.Name,
				GenerateName: "apinet-nic-",
				Labels: map[string]string{
					"app": "target2",
				},
			},
			Spec: apinetv1alpha1.NetworkInterfaceSpec{
				NetworkRef: corev1.LocalObjectReference{Name: apiNetNetwork.Name},
				IPs:        []net.IP{net.MustParseIP("192.168.178.15")},
				NodeRef:    corev1.LocalObjectReference{Name: "test-node"},
			},
		}
		Expect(k8sClient.Create(ctx, targetApiNetNic2)).To(Succeed())
		DeferCleanup(k8sClient.Delete, targetApiNetNic2)

		By("setting the target apinet nic to be ready")
		Eventually(UpdateStatus(targetApiNetNic2, func() {
			targetApiNetNic2.Status.State = apinetv1alpha1.NetworkInterfaceStateReady
		})).Should(Succeed())

		By("updating the ironcore networkpolicy with new NetworkInterfaceSelector labels")
		Eventually(Update(np, func() {
			np.Spec.NetworkInterfaceSelector.MatchLabels = map[string]string{"app": "target2"}
		})).Should(Succeed())

		By("waiting for the APINet network policy rule to be updated with new target network interface")
		Eventually(Object(networkPolicyRule)).Should(SatisfyAll(
			HaveField("NetworkRef", apinetv1alpha1.LocalUIDReference{
				Name: apiNetNetwork.Name,
				UID:  apiNetNetwork.UID,
			}),
			HaveField("Targets", ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"IP": Equal(net.MustParseIP("192.168.178.15")),
					"TargetRef": PointTo(MatchFields(IgnoreExtras, Fields{
						"UID":  Equal(targetApiNetNic2.UID),
						"Name": Equal(targetApiNetNic2.Name),
					})),
				}),
			)),
		))

	})
})
