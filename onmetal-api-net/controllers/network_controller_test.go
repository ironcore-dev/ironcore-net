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
	"github.com/onmetal/onmetal-api/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkController", func() {
	ctx := testutils.SetupContext()
	ns := SetupTest(ctx)

	It("should allocate a vni", func() {
		By("creating a network")
		network := &onmetalapinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
			Spec: onmetalapinetv1alpha1.NetworkSpec{},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("waiting for the network to be allocated")
		Eventually(Object(network)).Should(BeAllocatedNetwork())
	})

	It("should mark networks as pending if they can't be allocated and allocate them as soon as there's space", func() {
		By("creating networks until we run out of vnis")
		networkKeys := make([]client.ObjectKey, NoOfVNIs)
		for i := 0; i < NoOfVNIs; i++ {
			network := &onmetalapinetv1alpha1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "block-network-",
				},
				Spec: onmetalapinetv1alpha1.NetworkSpec{},
			}
			Expect(k8sClient.Create(ctx, network)).To(Succeed())
			networkKeys[i] = client.ObjectKeyFromObject(network)

			By("waiting for the network to be allocated")
			Eventually(Object(network)).Should(BeAllocatedNetwork())
		}

		By("creating another network")
		network := &onmetalapinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
			Spec: onmetalapinetv1alpha1.NetworkSpec{},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("waiting for the network to be marked as non-allocated")
		Eventually(Object(network)).Should(BeNonAllocatedNetwork())

		By("asserting it stays that way")
		Consistently(Object(network)).Should(BeNonAllocatedNetwork())

		By("deleting one of the original networks")
		Expect(k8sClient.Delete(ctx, &onmetalapinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: networkKeys[0].Namespace,
				Name:      networkKeys[0].Name,
			},
		})).To(Succeed())

		By("waiting for the network to be allocated")
		Eventually(Object(network)).Should(BeAllocatedNetwork())
	})
})

func BeNonAllocatedNetwork() types.GomegaMatcher {
	return HaveField("Status", SatisfyAll(
		HaveField("VNI", BeZero()),
		HaveField("Conditions", ConsistOf(
			SatisfyAll(
				HaveField("Type", onmetalapinetv1alpha1.NetworkAllocated),
				HaveField("Status", corev1.ConditionFalse),
			)),
		),
	))
}

func BeAllocatedNetwork() types.GomegaMatcher {
	return HaveField("Status", SatisfyAll(
		HaveField("Conditions", ConsistOf(
			SatisfyAll(
				HaveField("Type", onmetalapinetv1alpha1.NetworkAllocated),
				HaveField("Status", corev1.ConditionTrue),
			)),
		),
		HaveField("VNI", SatisfyAll(
			BeNumerically(">=", MinVNI),
			BeNumerically("<=", MaxVNI),
		)),
	))
}
