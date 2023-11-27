// Copyright 2022 IronCore authors
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
	"net/netip"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	metalnetv1alpha1 "github.com/onmetal/metalnet/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	It("should create a metalnet network for a network", func(ctx SpecContext) {
		By("creating a network")

		By("creating a network interface")
		nic := &v1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nic-",
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

		By("waiting for the network interface to have a finalizer")
		Eventually(Object(nic)).Should(HaveField("Finalizers", []string{PartitionFinalizer(partitionName)}))

		By("waiting for the metalnet network interface to be present with the expected values")
		metalnetNic := &metalnetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metalnetNs.Name,
				Name:      string(nic.UID),
			},
		}
		Eventually(Object(metalnetNic)).Should(HaveField("Spec", metalnetv1alpha1.NetworkInterfaceSpec{
			NetworkRef: corev1.LocalObjectReference{Name: string(network.UID)},
			IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
			IPs:        []metalnetv1alpha1.IP{metalnetv1alpha1.MustParseIP("10.0.0.1")},
			LoadBalancerTargets: []metalnetv1alpha1.IPPrefix{
				{Prefix: netip.PrefixFrom(loadBalancer.Spec.IPs[0].IP.Addr, 32)},
			},
			NodeName: &metalnetNode.Name,
		}))

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
