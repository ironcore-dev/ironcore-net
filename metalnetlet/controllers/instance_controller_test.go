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
	metalnetv1alpha1 "github.com/onmetal/metalnet/api/v1alpha1"
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	"github.com/onmetal/onmetal-api-net/apimachinery/api/net"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
})
