// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("LoadBalancerController", func() {
	ns := SetupNamespace(&k8sClient)
	network := SetupNetwork(ns)

	It("should reconcile the load balancer", func(ctx SpecContext) {
		By("creating a load balancer")
		loadBalancer := &v1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "load-balancer-",
			},
			Spec: v1alpha1.LoadBalancerSpec{
				Type:       v1alpha1.LoadBalancerTypePublic,
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				IPs: []v1alpha1.LoadBalancerIP{
					{
						Name:     "ip-1",
						IPFamily: corev1.IPv4Protocol,
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"foo": "bar"},
				},
				Template: v1alpha1.InstanceTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"foo": "bar"},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, loadBalancer)).To(Succeed())
		ips := v1alpha1.GetLoadBalancerIPs(loadBalancer)

		By("waiting for the load balancer to create a daemon set")
		daemonSet := &v1alpha1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      v1alpha1.LoadBalancerDaemonSetName(loadBalancer.Name),
			},
		}
		Eventually(Object(daemonSet)).Should(HaveField("Spec", v1alpha1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
			Template: v1alpha1.InstanceTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"foo": "bar"},
				},
				Spec: v1alpha1.InstanceSpec{
					Type:             v1alpha1.InstanceTypeLoadBalancer,
					LoadBalancerType: v1alpha1.LoadBalancerTypePublic,
					NetworkRef:       corev1.LocalObjectReference{Name: network.Name},
					IPs:              ips,
				},
			},
		}))
	})
})
