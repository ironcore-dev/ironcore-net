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
	"github.com/onmetal/onmetal-api/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NATGatewayController", func() {
	ctx := testutils.SetupContext()
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

		ipName := "ip"
		ipFamily := corev1.IPv4Protocol

		By("creating a nat gateway")
		natGateway := &networkingv1alpha1.NATGateway{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nat-gateway-ip-",
			},
			Spec: networkingv1alpha1.NATGatewaySpec{
				Type: networkingv1alpha1.NATGatewayTypePublic,
				IPFamilies: []corev1.IPFamily{
					ipFamily,
				},
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				IPs: []networkingv1alpha1.NATGatewayIP{
					{
						Name: ipName,
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, natGateway)).To(Succeed())

		By("waiting for the corresponding public ip to be created")
		publicIP := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      fmt.Sprintf("%s-%s-%s", natGateway.UID, ipName, strings.ToLower(string(ipFamily))),
			},
		}
		Eventually(Get(publicIP)).Should(Succeed())

		By("inspecting the created public ip")
		Expect(publicIP.Labels).To(Equal(map[string]string{
			apinetletv1alpha1.NATGatewayNamespaceLabel: natGateway.Namespace,
			apinetletv1alpha1.NATGatewayNameLabel:      natGateway.Name,
			apinetletv1alpha1.NATGatewayUIDLabel:       string(natGateway.UID),
		}))
		Expect(publicIP.Spec).To(Equal(onmetalapinetv1alpha1.PublicIPSpec{
			IPFamily: ipFamily,
		}))

		By("asserting the nat gateway does not get an ip address")
		Consistently(Object(natGateway)).Should(HaveField("Status.IPs", BeNil()))

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

		By("checking that nat gateway contains ip")
		Eventually(Object(natGateway)).Should(HaveField("Status.IPs",
			ContainElement(networkingv1alpha1.NATGatewayIPStatus{
				Name: ipName,
				IP:   *commonv1alpha1.MustParseNewIP("10.0.0.1"),
			}),
		))

		By("requesting one more ip")
		natGatewayBase := natGateway.DeepCopy()
		natGateway.Spec.IPs = append(natGateway.Spec.IPs, networkingv1alpha1.NATGatewayIP{
			Name: ipName + "-2",
		})
		Expect(k8sClient.Patch(ctx, natGateway, client.MergeFrom(natGatewayBase))).To(Succeed())

		By("waiting for the corresponding public ip to be created")
		publicIP2 := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      fmt.Sprintf("%s-%s-%s", natGateway.UID, ipName+"-2", strings.ToLower(string(ipFamily))),
			},
		}
		Eventually(Get(publicIP2)).Should(Succeed())

		By("patching the second public ip spec ips")
		basePublicIP2 := publicIP2.DeepCopy()
		publicIP2.Spec.IP = onmetalapinetv1alpha1.MustParseNewIP("10.0.0.2")
		Expect(k8sClient.Patch(ctx, publicIP2, client.MergeFrom(basePublicIP2))).To(Succeed())

		By("patching the second public ip status to allocated")
		basePublicIP2 = publicIP2.DeepCopy()
		onmetalapinetv1alpha1.SetPublicIPCondition(&publicIP2.Status.Conditions, onmetalapinetv1alpha1.PublicIPCondition{
			Type:   onmetalapinetv1alpha1.PublicIPAllocated,
			Status: corev1.ConditionTrue,
		})
		Expect(k8sClient.Status().Patch(ctx, publicIP2, client.MergeFrom(basePublicIP2))).To(Succeed())

		By("checking that nat gateway contains ip")
		Eventually(Object(natGateway)).Should(HaveField("Status.IPs",
			ContainElements(
				networkingv1alpha1.NATGatewayIPStatus{
					Name: ipName,
					IP:   *commonv1alpha1.MustParseNewIP("10.0.0.1"),
				},
				networkingv1alpha1.NATGatewayIPStatus{
					Name: ipName + "-2",
					IP:   *commonv1alpha1.MustParseNewIP("10.0.0.2"),
				}),
		))

		By("deleting the nat gateway")
		Expect(k8sClient.Delete(ctx, natGateway)).To(Succeed())

		By("waiting for it to be gone")
		Eventually(Get(natGateway)).Should(Satisfy(apierrors.IsNotFound))

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
					apinetletv1alpha1.NATGatewayNamespaceLabel: ns.Name,
					apinetletv1alpha1.NATGatewayNameLabel:      "some-name",
					apinetletv1alpha1.NATGatewayUIDLabel:       "some-uid",
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
