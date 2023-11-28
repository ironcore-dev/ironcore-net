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
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
})
