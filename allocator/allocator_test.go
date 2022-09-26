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

package allocator_test

import (
	. "github.com/onmetal/onmetal-api-net/allocator"
	commonv1alpha1 "github.com/onmetal/onmetal-api/apis/common/v1alpha1"
	"github.com/onmetal/onmetal-api/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Allocator", func() {
	ctx := testutils.SetupContext()
	_, allocator := SetupTest(ctx)

	assertAllocations := func(expected ...Allocation) {
		EventuallyWithOffset(1, func() ([]Allocation, error) {
			return allocator.List(ctx)
		}).Should(ConsistOf(expected))
	}

	It("should allow allocating ips", func() {
		By("allocating an ip")
		ip1, err := allocator.Allocate(ctx, "foo", corev1.IPv4Protocol)
		Expect(err).NotTo(HaveOccurred())

		By("inspecting the ip")
		Expect(ipv4Prefix.Contains(ip1.Addr)).To(BeTrue(), "ip not contained in prefix: %s - %s", ip1, ipv4Prefix)

		By("asserting the allocator state")
		assertAllocations(Allocation{ID: "foo", IP: ip1})

		By("allocating another ip")
		var ip2 commonv1alpha1.IP
		Eventually(func() error {
			ip2, err = allocator.Allocate(ctx, "bar", corev1.IPv4Protocol)
			return err
		}).Should(Succeed())

		By("inspecting the ip")
		Expect(ipv4Prefix.Contains(ip2.Addr)).To(BeTrue(), "ip not contained in prefix: %s - %s", ip2, ipv4Prefix)

		By("asserting the allocator state")
		assertAllocations(Allocation{ID: "foo", IP: ip1}, Allocation{ID: "bar", IP: ip2})

		By("releasing the first ip")
		Eventually(func() error {
			return allocator.Release(ctx, "foo")
		}).Should(Succeed())

		By("asserting the allocator state")
		assertAllocations(Allocation{ID: "bar", IP: ip2})
	})
})
