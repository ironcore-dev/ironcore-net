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

package networking_test

import (
	"github.com/onmetal/onmetal-api-net/allocator"

	networkingv1alpha1 "github.com/onmetal/onmetal-api/apis/networking/v1alpha1"
	"github.com/onmetal/onmetal-api/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("VirtualIPController", func() {
	ctx := testutils.SetupContext()
	ns, alloc := SetupTest(ctx)

	DescribeTable("allocate and release virtual ips",
		func(ipFamily corev1.IPFamily) {
			By("creating a virtual ip")
			virtualIP := &networkingv1alpha1.VirtualIP{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "virtualip-",
				},
				Spec: networkingv1alpha1.VirtualIPSpec{
					Type:     networkingv1alpha1.VirtualIPTypePublic,
					IPFamily: ipFamily,
				},
			}
			Expect(k8sClient.Create(ctx, virtualIP)).To(Succeed())

			By("waiting for the virtual ip to be allocated")
			Eventually(Object(virtualIP)).Should(HaveField("Status.IP", Not(BeNil())))
			ip := virtualIP.Status.IP

			By("deleting the virtual ip")
			Expect(k8sClient.Delete(ctx, virtualIP)).Should(Succeed())

			By("checking if the allocator does not have the ip anymore")
			Eventually(func() ([]allocator.Allocation, error) {
				return alloc.List(ctx)
			}).ShouldNot(ContainElement(HaveField("IP", ip)))
		},
		Entry(nil, corev1.IPv4Protocol),
		Entry(nil, corev1.IPv6Protocol),
	)
})
