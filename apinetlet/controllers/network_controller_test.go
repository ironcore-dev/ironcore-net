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
	"strconv"

	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	apinetletv1alpha1 "github.com/onmetal/onmetal-api-net/apinetlet/api/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	"github.com/onmetal/onmetal-api/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkController", func() {
	ctx := testutils.SetupContext()
	ns := SetupTest(ctx)
	const vni int32 = 4

	It("should allocate an apinet network", func() {
		By("creating a network")
		network := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("waiting for the corresponding apinet network to be created")
		apiNetNetwork := &onmetalapinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      string(network.UID),
			},
		}
		Eventually(Get(apiNetNetwork)).Should(Succeed())

		By("inspecting the created apinet network")
		Expect(apiNetNetwork.Labels).To(Equal(map[string]string{
			apinetletv1alpha1.NetworkNamespaceLabel: network.Namespace,
			apinetletv1alpha1.NetworkNameLabel:      network.Name,
			apinetletv1alpha1.NetworkUIDLabel:       string(network.UID),
		}))
		Expect(apiNetNetwork.Spec).To(Equal(onmetalapinetv1alpha1.NetworkSpec{}))

		By("asserting the network does not report a vni")
		Consistently(Object(network)).
			Should(SatisfyAll(
				HaveField("Spec.Handle", ""),
				HaveField("Status.State", Not(Equal(networkingv1alpha1.NetworkStateAvailable))),
			))

		By("setting the apinet network spec vni")
		baseAPINetNetwork := apiNetNetwork.DeepCopy()
		apiNetNetwork.Spec.VNI = pointer.Int32(vni)
		Expect(k8sClient.Patch(ctx, apiNetNetwork, client.MergeFrom(baseAPINetNetwork))).To(Succeed())

		By("setting the apinet network status to allocated")
		baseAPINetNetwork = apiNetNetwork.DeepCopy()
		onmetalapinetv1alpha1.SetNetworkCondition(&apiNetNetwork.Status.Conditions, onmetalapinetv1alpha1.NetworkCondition{
			Type:   onmetalapinetv1alpha1.NetworkAllocated,
			Status: corev1.ConditionTrue,
		})
		Expect(k8sClient.Status().Patch(ctx, apiNetNetwork, client.MergeFrom(baseAPINetNetwork))).To(Succeed())

		By("waiting for the network to reflect the allocated vni")
		Eventually(Object(network)).
			Should(SatisfyAll(
				HaveField("Spec.Handle", Equal(strconv.FormatInt(int64(vni), 10))),
				HaveField("Status.State", Equal(networkingv1alpha1.NetworkStateAvailable)),
			))

		By("deleting the network")
		Expect(k8sClient.Delete(ctx, network)).To(Succeed())

		By("waiting for it to be gone")
		Eventually(Get(network)).Should(Satisfy(apierrors.IsNotFound))

		By("asserting the corresponding apinet network is gone as well")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(apiNetNetwork), apiNetNetwork)).To(Satisfy(apierrors.IsNotFound))
	})

	It("should clean up dangling apinet networks", func() {
		By("creating a apinet network")
		apiNetNetwork := &onmetalapinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "apinet-network-",
				Labels: map[string]string{
					apinetletv1alpha1.NetworkNamespaceLabel: ns.Name,
					apinetletv1alpha1.NetworkNameLabel:      "some-name",
					apinetletv1alpha1.NetworkUIDLabel:       "some-uid",
				},
			},
			Spec: onmetalapinetv1alpha1.NetworkSpec{},
		}
		Expect(k8sClient.Create(ctx, apiNetNetwork)).To(Succeed())

		By("waiting for the apinet network to be gone")
		Eventually(Get(apiNetNetwork)).Should(Satisfy(apierrors.IsNotFound))
	})

	It("should use the specified handle as vni if any", func() {
		By("creating a network with handle")
		vni := int32(42)
		network := &networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
			Spec: networkingv1alpha1.NetworkSpec{
				Handle: strconv.FormatInt(int64(vni), 10),
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())

		By("waiting for the corresponding apinet network to be created with correct vni")
		apiNetNetwork := &onmetalapinetv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      string(network.UID),
			},
		}

		Eventually(Object(apiNetNetwork)).Should(HaveField("Spec.VNI", Equal(&vni)))
	})
})
