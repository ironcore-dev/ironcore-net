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
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	commonv1alpha1 "github.com/onmetal/onmetal-api/apis/common/v1alpha1"
	"github.com/onmetal/onmetal-api/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("PublicIPController", func() {
	ctx := testutils.SetupContext()
	ns := SetupTest(ctx)

	It("should allocate a public ip", func() {
		By("creating a public ip")
		publicIP := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "public-ip-",
			},
			Spec: onmetalapinetv1alpha1.PublicIPSpec{
				IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
			},
		}
		Expect(k8sClient.Create(ctx, publicIP)).To(Succeed())

		By("waiting for the public ip to be allocated")
		Eventually(Object(publicIP)).Should(BeAllocated())
	})

	It("should mark public ips as pending if they can't be allocated and allocate them as soon as there's space", func() {
		By("creating public ips until we run out of addresses")
		publicIPKeys := make([]client.ObjectKey, NoOfIPv4Addresses)
		for i := 0; i < NoOfIPv4Addresses; i++ {
			publicIP := &onmetalapinetv1alpha1.PublicIP{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "block-public-ip-",
				},
				Spec: onmetalapinetv1alpha1.PublicIPSpec{
					IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
				},
			}
			Expect(k8sClient.Create(ctx, publicIP)).To(Succeed())
			publicIPKeys[i] = client.ObjectKeyFromObject(publicIP)

			By("waiting for the public ip to be allocated")
			Eventually(Object(publicIP)).Should(BeAllocated())
		}

		By("creating another public ip")
		publicIP := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "public-ip-",
			},
			Spec: onmetalapinetv1alpha1.PublicIPSpec{
				IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
			},
		}
		Expect(k8sClient.Create(ctx, publicIP)).To(Succeed())

		By("waiting for the public ip to be marked as non-allocated")
		Eventually(Object(publicIP)).Should(BeNonAllocated())

		By("asserting it stays that way")
		Consistently(Object(publicIP)).Should(BeNonAllocated())

		By("deleting one of the original public ips")
		Expect(k8sClient.Delete(ctx, &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: publicIPKeys[0].Namespace,
				Name:      publicIPKeys[0].Name,
			},
		})).To(Succeed())

		By("waiting for the ip to be allocated")
		Eventually(Object(publicIP)).Should(BeAllocated())
	})
})

func BeNonAllocated() types.GomegaMatcher {
	return HaveField("Status", SatisfyAll(
		HaveField("IPs", BeEmpty()),
		HaveField("Conditions", ConsistOf(
			SatisfyAll(
				HaveField("Type", onmetalapinetv1alpha1.PublicIPAllocated),
				HaveField("Status", corev1.ConditionFalse),
			)),
		),
	))
}

func BeAllocated() types.GomegaMatcher {
	return HaveField("Status", SatisfyAll(
		HaveField("IPs", ConsistOf(Satisfy(func(ip commonv1alpha1.IP) bool {
			return ip.Is4() && ip.IsValid() && InitialAvailableIPs().Contains(ip.Addr)
		}))),
		HaveField("Conditions", ConsistOf(
			SatisfyAll(
				HaveField("Type", onmetalapinetv1alpha1.PublicIPAllocated),
				HaveField("Status", corev1.ConditionTrue),
			)),
		),
	))
}
