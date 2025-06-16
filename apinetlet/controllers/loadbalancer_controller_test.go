// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"net/netip"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("LoadBalancerController", func() {
	ns := SetupNamespace(&k8sClient)
	apiNetNs := SetupNamespace(&k8sClient)
	SetupTest(apiNetNs)

	network, apiNetNetwork := SetupNetwork(ns, apiNetNs)

	It("should manage the APINet load balancer and its IPs", func(ctx SpecContext) {
		By("creating a load balancer")
		loadBalancer := &networkingv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "load-balancer-",
			},
			Spec: networkingv1alpha1.LoadBalancerSpec{
				Type:       networkingv1alpha1.LoadBalancerTypePublic,
				IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
			},
		}
		Expect(k8sClient.Create(ctx, loadBalancer)).To(Succeed())

		By("waiting for the APINet load balancer to exist")
		apiNetLoadBalancer := &v1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(loadBalancer.UID),
			},
		}
		Eventually(Object(apiNetLoadBalancer)).Should(SatisfyAll(
			HaveField("Labels", apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer)),
			HaveField("Spec", MatchFields(IgnoreExtras, Fields{
				"Type":       Equal(v1alpha1.LoadBalancerTypePublic),
				"NetworkRef": Equal(corev1.LocalObjectReference{Name: apiNetNetwork.Name}),
				"IPs": ConsistOf(MatchFields(IgnoreExtras, Fields{
					"IPFamily": Equal(corev1.IPv4Protocol),
					"Name":     Equal("ipv4"),
				})),
				"Selector": Equal(&metav1.LabelSelector{
					MatchLabels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
				}),
				"Template": Equal(v1alpha1.InstanceTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
					},
					Spec: v1alpha1.InstanceSpec{
						Affinity: &v1alpha1.Affinity{
							InstanceAntiAffinity: &v1alpha1.InstanceAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []v1alpha1.InstanceAffinityTerm{
									{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
										},
										TopologyKey: v1alpha1.TopologyZoneLabel,
									},
								},
							},
						},
					},
				}),
			}))),
		)
	})

	It("should manage the internal APINet load balancer and its discrete IPs", func(ctx SpecContext) {
		By("creating an internal load balancer")
		loadBalancer := &networkingv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "load-balancer-",
			},
			Spec: networkingv1alpha1.LoadBalancerSpec{
				Type:       networkingv1alpha1.LoadBalancerTypeInternal,
				IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				IPs: []networkingv1alpha1.IPSource{
					{
						Value: commonv1alpha1.MustParseNewIP("10.0.0.1"),
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, loadBalancer)).To(Succeed())

		By("waiting for the internal APINet load balancer to exist")
		apiNetLoadBalancer := &v1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(loadBalancer.UID),
			},
		}
		Eventually(Object(apiNetLoadBalancer)).Should(SatisfyAll(
			HaveField("Labels", apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer)),
			HaveField("Spec", MatchFields(IgnoreExtras, Fields{
				"Type":       Equal(v1alpha1.LoadBalancerTypeInternal),
				"NetworkRef": Equal(corev1.LocalObjectReference{Name: apiNetNetwork.Name}),
				"IPs": ConsistOf(MatchFields(IgnoreExtras, Fields{
					"IPFamily": Equal(corev1.IPv4Protocol),
					"IP":       Equal(net.MustParseIP("10.0.0.1")),
				})),
				"Selector": Equal(&metav1.LabelSelector{
					MatchLabels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
				}),
				"Template": Equal(v1alpha1.InstanceTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
					},
					Spec: v1alpha1.InstanceSpec{
						Affinity: &v1alpha1.Affinity{
							InstanceAntiAffinity: &v1alpha1.InstanceAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []v1alpha1.InstanceAffinityTerm{
									{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
										},
										TopologyKey: v1alpha1.TopologyZoneLabel,
									},
								},
							},
						},
					},
				}),
			}))),
		)
	})

	It("should manage the internal APINet load balancer and its ephemeral IPs", func(ctx SpecContext) {
		By("creating a new parent prefix")
		parentPrefix := &ipamv1alpha1.Prefix{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "load-balancer-ephemeral",
			},
			Spec: ipamv1alpha1.PrefixSpec{
				IPFamily: corev1.IPv4Protocol,
				Prefix:   commonv1alpha1.MustParseNewIPPrefix("10.0.0.1/24"),
			},
		}
		Expect(k8sClient.Create(ctx, parentPrefix)).To(Succeed())
		DeferCleanup(k8sClient.Delete, parentPrefix)

		By("creating an internal load balancer")
		loadBalancer := &networkingv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "load-balancer-ephemeral",
			},
			Spec: networkingv1alpha1.LoadBalancerSpec{
				Type:       networkingv1alpha1.LoadBalancerTypeInternal,
				IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				IPs: []networkingv1alpha1.IPSource{
					{
						Ephemeral: &networkingv1alpha1.EphemeralPrefixSource{
							PrefixTemplate: &ipamv1alpha1.PrefixTemplateSpec{
								Spec: ipamv1alpha1.PrefixSpec{
									IPFamily: corev1.IPv4Protocol,
									ParentRef: &corev1.LocalObjectReference{
										Name: "load-balancer-ephemeral",
									},
								},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, loadBalancer)).To(Succeed())
		DeferCleanup(k8sClient.Delete, loadBalancer)

		By("creating a new prefix")
		prefix := &ipamv1alpha1.Prefix{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "load-balancer-ephemeral-0",
			},
			Spec: ipamv1alpha1.PrefixSpec{
				IPFamily: corev1.IPv4Protocol,
				Prefix:   commonv1alpha1.MustParseNewIPPrefix("10.0.0.1/32"),
			},
		}
		Expect(controllerutil.SetControllerReference(loadBalancer, prefix, k8sClient.Scheme())).To(Succeed())
		Expect(k8sClient.Create(ctx, prefix)).To(Succeed())
		DeferCleanup(k8sClient.Delete, prefix)

		By("patching the prefix phase to allocated")
		Eventually(UpdateStatus(prefix, func() {
			prefix.Status.Phase = ipamv1alpha1.PrefixPhaseAllocated
		})).Should(Succeed())

		By("waiting for the internal APINet load balancer to exist")
		apiNetLoadBalancer := &v1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(loadBalancer.UID),
			},
		}
		Eventually(Object(apiNetLoadBalancer)).Should(SatisfyAll(
			HaveField("Labels", apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer)),
			HaveField("Spec", MatchFields(IgnoreExtras, Fields{
				"Type":       Equal(v1alpha1.LoadBalancerTypeInternal),
				"NetworkRef": Equal(corev1.LocalObjectReference{Name: apiNetNetwork.Name}),
				"IPs": ConsistOf(MatchFields(IgnoreExtras, Fields{
					"IPFamily": Equal(corev1.IPv4Protocol),
					"IP":       Equal(net.MustParseIP("10.0.0.1")),
				})),
				"Selector": Equal(&metav1.LabelSelector{
					MatchLabels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
				}),
				"Template": Equal(v1alpha1.InstanceTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
					},
					Spec: v1alpha1.InstanceSpec{
						Affinity: &v1alpha1.Affinity{
							InstanceAntiAffinity: &v1alpha1.InstanceAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []v1alpha1.InstanceAffinityTerm{
									{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
										},
										TopologyKey: v1alpha1.TopologyZoneLabel,
									},
								},
							},
						},
					},
				}),
			}))),
		)
	})

	It("should manage the APINet load balancer and its node affintity", func(ctx SpecContext) {
		By("creating a load balancer")
		loadBalancer := &networkingv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "load-balancer-",
			},
			Spec: networkingv1alpha1.LoadBalancerSpec{
				Type:       networkingv1alpha1.LoadBalancerTypePublic,
				IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
			},
		}
		Expect(k8sClient.Create(ctx, loadBalancer)).To(Succeed())
		DeferCleanup(k8sClient.Delete, loadBalancer)

		By("creating the load balancer routing")
		lbRouting := &networkingv1alpha1.LoadBalancerRouting{
			ObjectMeta: metav1.ObjectMeta{
				Name:      loadBalancer.Name,
				Namespace: loadBalancer.Namespace,
			},
			NetworkRef: commonv1alpha1.LocalUIDReference{
				Name: network.Name,
				UID:  network.UID,
			},
			Destinations: []networkingv1alpha1.LoadBalancerDestination{
				{
					IP: commonv1alpha1.IP{Addr: netip.MustParseAddr("192.168.0.1")},
					TargetRef: &networkingv1alpha1.LoadBalancerTargetRef{
						UID:        "first-nic-uid",
						Name:       "first-nic-name",
						ProviderID: "ironcore-net://namespace/first-apinet-nic-name/first-node-name/first-metalnet-nic-uid",
					},
				},
				{
					IP: commonv1alpha1.IP{Addr: netip.MustParseAddr("192.168.0.2")},
					TargetRef: &networkingv1alpha1.LoadBalancerTargetRef{
						UID:        "second-nic-uid",
						Name:       "second-nic-name",
						ProviderID: "ironcore-net://namespace/second-apinet-nic-name/second-node-name/second-metalnet-nic-uid",
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, lbRouting)).To(Succeed())
		DeferCleanup(k8sClient.Delete, lbRouting)

		By("waiting for the APINet load balancer to exist")
		apiNetLoadBalancer := &v1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(loadBalancer.UID),
			},
		}
		Eventually(Object(apiNetLoadBalancer)).Should(SatisfyAll(
			HaveField("Labels", apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer)),
			HaveField("Spec", MatchFields(IgnoreExtras, Fields{
				"Type":       Equal(v1alpha1.LoadBalancerTypePublic),
				"NetworkRef": Equal(corev1.LocalObjectReference{Name: apiNetNetwork.Name}),
				"IPs": ConsistOf(MatchFields(IgnoreExtras, Fields{
					"IPFamily": Equal(corev1.IPv4Protocol),
					"Name":     Equal("ipv4"),
				})),
				"Selector": Equal(&metav1.LabelSelector{
					MatchLabels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
				}),
				"Template": Equal(v1alpha1.InstanceTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
					},
					Spec: v1alpha1.InstanceSpec{
						Affinity: &v1alpha1.Affinity{
							InstanceAntiAffinity: &v1alpha1.InstanceAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []v1alpha1.InstanceAffinityTerm{
									{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), loadBalancer),
										},
										TopologyKey: v1alpha1.TopologyZoneLabel,
									},
								},
							},
							NodeAffinity: &v1alpha1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &v1alpha1.NodeSelector{
									NodeSelectorTerms: []v1alpha1.NodeSelectorTerm{
										{
											MatchFields: []v1alpha1.NodeSelectorRequirement{
												{
													Key:      "metadata.name",
													Operator: v1alpha1.NodeSelectorOpIn,
													Values:   []string{"first-node-name"},
												},
											},
										},
										{
											MatchFields: []v1alpha1.NodeSelectorRequirement{
												{
													Key:      "metadata.name",
													Operator: v1alpha1.NodeSelectorOpIn,
													Values:   []string{"second-node-name"},
												},
											},
										},
									},
								},
							},
						},
					},
				}),
			}))),
		)
	})
})
