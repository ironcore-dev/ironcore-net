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

var _ = Describe("VirtualIPController", func() {
	ctx := SetupContext()
	ns := SetupTest(ctx)

	It("should allocate a public ip", func() {
		By("creating a virtual ip")
		virtualIP := &networkingv1alpha1.VirtualIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "virtual-ip-",
			},
			Spec: networkingv1alpha1.VirtualIPSpec{
				Type:     networkingv1alpha1.VirtualIPTypePublic,
				IPFamily: corev1.IPv4Protocol,
			},
		}
		Expect(k8sClient.Create(ctx, virtualIP)).To(Succeed())

		By("waiting for the corresponding public ip to be created")
		publicIP := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      string(virtualIP.UID),
			},
		}
		Eventually(Get(publicIP)).Should(Succeed())

		By("inspecting the created public ip")
		Expect(publicIP.Labels).To(Equal(map[string]string{
			apinetletv1alpha1.VirtualIPNamespaceLabel: virtualIP.Namespace,
			apinetletv1alpha1.VirtualIPNameLabel:      virtualIP.Name,
			apinetletv1alpha1.VirtualIPUIDLabel:       string(virtualIP.UID),
		}))
		Expect(publicIP.Spec).To(Equal(onmetalapinetv1alpha1.PublicIPSpec{
			IPFamily: corev1.IPv4Protocol,
		}))

		By("asserting the virtual ip does not get an ip address")
		Consistently(Object(virtualIP)).Should(HaveField("Status.IP", BeNil()))

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

		By("waiting for the virtual ip to reflect the allocated ip")
		Eventually(Object(virtualIP)).Should(HaveField("Status.IP", Equal(commonv1alpha1.MustParseNewIP("10.0.0.1"))))

		By("deleting the virtual ip")
		Expect(k8sClient.Delete(ctx, virtualIP)).To(Succeed())

		By("waiting for it to be gone")
		Eventually(Get(virtualIP)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding public ip is gone as well")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(publicIP), publicIP)).To(Satisfy(apierrors.IsNotFound))
	})

	It("should clean up dangling public ips", func() {
		By("creating a public ip")
		publicIP := &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "public-ip-",
				Labels: map[string]string{
					apinetletv1alpha1.VirtualIPNamespaceLabel: ns.Name,
					apinetletv1alpha1.VirtualIPNameLabel:      "some-name",
					apinetletv1alpha1.VirtualIPUIDLabel:       "some-uid",
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
