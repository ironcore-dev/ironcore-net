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
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	"github.com/onmetal/onmetal-api-net/networkid"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkController", func() {
	ns := SetupNamespace(&k8sClient)
	metalnetNs := SetupNamespace(&k8sClient)
	SetupTest(metalnetNs)

	It("should create a metalnet network for a network", func(ctx SpecContext) {
		By("creating a network")
		network := &v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
			Spec: v1alpha1.NetworkSpec{},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("parsing the VNI")
		vni, err := networkid.ParseVNI(network.Spec.ID)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the metalnet network to be created")
		metalnetNetwork := &metalnetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metalnetNs.Name,
				Name:      string(network.UID),
			},
		}
		Eventually(Object(metalnetNetwork)).Should(SatisfyAll(
			HaveField("Spec", metalnetv1alpha1.NetworkSpec{
				ID: vni,
			}),
		))
	})
})
