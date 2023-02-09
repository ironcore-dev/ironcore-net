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
	"fmt"
	"strings"

	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	apinetletv1alpha1 "github.com/onmetal/onmetal-api-net/apinetlet/api/v1alpha1"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("LoadBalancerController", func() {
	ctx := SetupContext()
	ns := SetupTest(ctx)

	It("should allocate a public ip", func() {
		By("creating a network")
		network := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		ipFamily := corev1.IPv4Protocol

		By("creating a load balancer")
		loadBalancer := &networkingv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "load-balancer-",
			},
			Spec: networkingv1alpha1.LoadBalancerSpec{
				Type: networkingv1alpha1.LoadBalancerTypePublic,
				IPFamilies: []corev1.IPFamily{
					ipFamily,
				},
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
			},
		}
		Expect(k8sClient.Create(ctx, loadBalancer)).To(Succeed())

		By("waiting for the corresponding public ip to be created")
		publicIP := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      fmt.Sprintf("%s-%s", loadBalancer.UID, strings.ToLower(string(ipFamily))),
			},
		}
		Eventually(Get(publicIP)).Should(Succeed())

		By("inspecting the created public ip")
		Expect(publicIP.Labels).To(Equal(map[string]string{
			apinetletv1alpha1.LoadBalancerNamespaceLabel: loadBalancer.Namespace,
			apinetletv1alpha1.LoadBalancerNameLabel:      loadBalancer.Name,
			apinetletv1alpha1.LoadBalancerUIDLabel:       string(loadBalancer.UID),
		}))
		Expect(publicIP.Spec).To(Equal(onmetalapinetv1alpha1.PublicIPSpec{
			IPFamily: ipFamily,
		}))

		By("asserting the load balancer does not get an ip address")
		Consistently(Object(loadBalancer)).Should(HaveField("Status.IPs", BeNil()))

		By("patching the public ip spec ips")
		basePublicIP := publicIP.DeepCopy()
		publicIP.Spec.IP = onmetalapinetv1alpha1.MustParseNewIP("10.0.0.1")
		Expect(k8sClient.Patch(ctx, publicIP, client.MergeFrom(basePublicIP))).To(Succeed())

		By("patching the public ip status to allocated")
		basePublicIP = publicIP.DeepCopy()
		onmetalapinetv1alpha1.SetPublicIPCondition(&publicIP.Status.Conditions, onmetalapinetv1alpha1.PublicIPCondition{
			Type:   onmetalapinetv1alpha1.PublicIPAllocated,
			Status: corev1.ConditionTrue,
		})
		Expect(k8sClient.Status().Patch(ctx, publicIP, client.MergeFrom(basePublicIP))).To(Succeed())

		By("checking that load balancer contains ip")
		Eventually(Object(loadBalancer)).Should(HaveField("Status.IPs",
			ContainElement(*commonv1alpha1.MustParseNewIP("10.0.0.1")),
		))

		ipFamily2 := corev1.IPv6Protocol

		By("requesting further ip by adding another protocol")
		loadBalancerBase := loadBalancer.DeepCopy()
		loadBalancer.Spec.IPFamilies = append(loadBalancer.Spec.IPFamilies, ipFamily2)
		Expect(k8sClient.Patch(ctx, loadBalancer, client.MergeFrom(loadBalancerBase))).To(Succeed())

		By("waiting for the corresponding public ip to be created")
		publicIP2 := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      fmt.Sprintf("%s-%s", loadBalancer.UID, strings.ToLower(string(ipFamily2))),
			},
		}
		Eventually(Get(publicIP2)).Should(Succeed())

		By("patching the second public ip spec ips")
		basePublicIP2 := publicIP2.DeepCopy()
		publicIP2.Spec.IP = onmetalapinetv1alpha1.MustParseNewIP("::ffff:a00:2")
		Expect(k8sClient.Patch(ctx, publicIP2, client.MergeFrom(basePublicIP2))).To(Succeed())

		By("patching the second public ip status to allocated")
		basePublicIP2 = publicIP2.DeepCopy()
		onmetalapinetv1alpha1.SetPublicIPCondition(&publicIP2.Status.Conditions, onmetalapinetv1alpha1.PublicIPCondition{
			Type:   onmetalapinetv1alpha1.PublicIPAllocated,
			Status: corev1.ConditionTrue,
		})
		Expect(k8sClient.Status().Patch(ctx, publicIP2, client.MergeFrom(basePublicIP2))).To(Succeed())

		By("checking that load balancer contains ips")
		Eventually(Object(loadBalancer)).Should(HaveField("Status.IPs",
			ContainElements(
				*commonv1alpha1.MustParseNewIP("10.0.0.1"),
				*commonv1alpha1.MustParseNewIP("::ffff:a00:2"),
			),
		))

		By("deleting the load balancer")
		Expect(k8sClient.Delete(ctx, loadBalancer)).To(Succeed())

		By("waiting for it to be gone")
		Eventually(Get(loadBalancer)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding public ips are gone as well")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(publicIP), publicIP)).To(Satisfy(apierrors.IsNotFound))
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(publicIP2), publicIP2)).To(Satisfy(apierrors.IsNotFound))
	})

	It("should clean up dangling public ips", func() {
		By("creating a public ip")
		publicIP := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "public-ip-",
				Labels: map[string]string{
					apinetletv1alpha1.LoadBalancerNamespaceLabel: ns.Name,
					apinetletv1alpha1.LoadBalancerNameLabel:      "some-name",
					apinetletv1alpha1.LoadBalancerUIDLabel:       "some-uid",
				},
			},
			Spec: onmetalapinetv1alpha1.PublicIPSpec{
				IPFamily: corev1.IPv4Protocol,
			},
		}
		Expect(k8sClient.Create(ctx, publicIP)).To(Succeed())

		By("waiting for the public ip to be gone")
		Eventually(Get(publicIP)).Should(Satisfy(apierrors.IsNotFound))
	})
})
