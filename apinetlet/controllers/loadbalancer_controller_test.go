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
	apinetletclient "github.com/onmetal/onmetal-api-net/apinetlet/client"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	. "github.com/onmetal/onmetal-api/utils/testing"
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
			}))),
		)
	})
})
