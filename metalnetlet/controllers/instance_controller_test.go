// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("LoadBalancerInstanceController", func() {
	ns := SetupNamespace(&k8sClient)
	metalnetNs := SetupNamespace(&k8sClient)
	SetupTest(metalnetNs)

	metalnetNode := SetupMetalnetNode()
	network := SetupNetwork(ns)

	It("should reconcile the metalnet load balancers for the load balancer instance", func(ctx SpecContext) {
		By("creating a load balancer instance")
		protocol := corev1.ProtocolTCP
		inst := &v1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "lb-",
			},
			Spec: v1alpha1.InstanceSpec{
				Type:             v1alpha1.InstanceTypeLoadBalancer,
				LoadBalancerType: v1alpha1.LoadBalancerTypePublic,
				NetworkRef:       corev1.LocalObjectReference{Name: network.Name},
				IPs: []net.IP{
					net.MustParseIP("10.0.0.1"),
					net.MustParseIP("10.0.0.2"),
				},
				NodeRef: &corev1.LocalObjectReference{
					Name: PartitionNodeName(partitionName, metalnetNode.Name),
				},
				LoadBalancerPorts: []v1alpha1.LoadBalancerPort{
					{
						Protocol: &protocol,
						Port:     1000,
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, inst)).To(Succeed())

		By("waiting for the metalnet load balancers to appear")
		metalnetLoadBalancerList := &metalnetv1alpha1.LoadBalancerList{}
		Eventually(ObjectList(metalnetLoadBalancerList)).Should(HaveField("Items", ConsistOf(
			HaveField("Spec.IP", metalnetv1alpha1.MustParseIP("10.0.0.1")),
			HaveField("Spec.IP", metalnetv1alpha1.MustParseIP("10.0.0.2")),
		)))
	})

	It("should recreate the metalnet load balancer if it gets deleted", func(ctx SpecContext) {
		By("creating a load balancer instance")
		protocol := corev1.ProtocolTCP
		inst := &v1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "lb-",
			},
			Spec: v1alpha1.InstanceSpec{
				Type:             v1alpha1.InstanceTypeLoadBalancer,
				LoadBalancerType: v1alpha1.LoadBalancerTypePublic,
				NetworkRef:       corev1.LocalObjectReference{Name: network.Name},
				IPs:              []net.IP{net.MustParseIP("10.0.0.1")},
				NodeRef: &corev1.LocalObjectReference{
					Name: PartitionNodeName(partitionName, metalnetNode.Name),
				},
				LoadBalancerPorts: []v1alpha1.LoadBalancerPort{
					{
						Protocol: &protocol,
						Port:     1000,
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, inst)).To(Succeed())

		By("waiting for the metalnet load balancer to appear")
		metalnetLoadBalancerList := &metalnetv1alpha1.LoadBalancerList{}
		Eventually(ObjectList(metalnetLoadBalancerList, client.InNamespace(metalnetNs.Name))).
			Should(HaveField("Items", HaveLen(1)))

		By("deleting the metalnet load balancer")
		metalnetLoadBalancer := metalnetLoadBalancerList.Items[0].DeepCopy()
		Expect(k8sClient.Delete(ctx, metalnetLoadBalancer)).To(Succeed())

		By("waiting for a new metalnet load balancer to be created")
		Eventually(ObjectList(metalnetLoadBalancerList, client.InNamespace(metalnetNs.Name))).
			Should(HaveField("Items", ContainElement(
				HaveField("UID", Not(Equal(metalnetLoadBalancer.UID))),
			)))
	})
})
