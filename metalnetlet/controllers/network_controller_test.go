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
	"github.com/onmetal/onmetal-api-net/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("NetworkController", func() {
	ns := SetupTest()

	It("should create a metalnet network for a network", func(ctx SpecContext) {
		By("creating a network")
		network := &v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
			Spec: v1alpha1.NetworkSpec{
				VNI: pointer.Int32(300),
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("patching the network to be available")
		baseNetwork := network.DeepCopy()
		network.Status.Conditions = []v1alpha1.NetworkCondition{
			{
				Type:    v1alpha1.NetworkAllocated,
				Status:  corev1.ConditionTrue,
				Reason:  "Allocated",
				Message: "The network has been allocated.",
			},
		}
		Expect(k8sClient.Status().Patch(ctx, network, client.MergeFrom(baseNetwork))).Should(Succeed())

		By("waiting for the metalnet network to be created")
		metalnetNetwork := &metalnetv1alpha1.Network{}
		metalnetNetworkKey := client.ObjectKey{Name: "network-300"}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, metalnetNetworkKey, metalnetNetwork)).To(Succeed())
			g.Expect(metalnetNetwork.Spec).To(Equal(metalnetv1alpha1.NetworkSpec{
				ID: 300,
			}))
		}).Should(Succeed())
	})
})
