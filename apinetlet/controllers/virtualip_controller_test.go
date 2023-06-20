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
	apinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	apinetletclient "github.com/onmetal/onmetal-api-net/apinetlet/client"
	. "github.com/onmetal/onmetal-api-net/utils/testing"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("VirtualIPController", func() {
	ns := SetupNamespace(&k8sClient)
	apiNetNs := SetupNamespace(&k8sClient)
	SetupTest(apiNetNs)

	It("should allocate a virtual ip", func(ctx SpecContext) {
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

		By("waiting for the corresponding ip to be created")
		ip := &apinetv1alpha1.IP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(virtualIP.UID),
			},
		}
		Eventually(Object(ip)).Should(SatisfyAll(
			HaveField("Labels", HaveKeysWithValues(apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), virtualIP))),
			HaveField("Spec", MatchFields(IgnoreExtras, Fields{
				"Type":     Equal(apinetv1alpha1.IPTypePublic),
				"IPFamily": Equal(corev1.IPv4Protocol),
			})),
		))

		By("waiting for the virtual ip to reflect the allocated ip")
		Eventually(Object(virtualIP)).Should(HaveField("Status.IP", &commonv1alpha1.IP{Addr: ip.Spec.IP.Addr}))

		By("deleting the virtual ip")
		Expect(k8sClient.Delete(ctx, virtualIP)).To(Succeed())

		By("waiting for it to be gone")
		Eventually(Get(virtualIP)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding ip is gone as well")
		ipKey := client.ObjectKeyFromObject(ip)
		Expect(k8sClient.Get(ctx, ipKey, ip)).To(Satisfy(apierrors.IsNotFound))
	})
})
