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
	"github.com/onmetal/onmetal-api-net/apimachinery/api/net"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkInterfaceController", func() {
	ns := SetupNamespace(&k8sClient)
	metalnetNs := SetupNamespace(&k8sClient)
	SetupTest(metalnetNs)

	metalnetNode := SetupMetalnetNode()
	network := SetupNetwork(ns)

	It("should create a metalnet network for a network", func(ctx SpecContext) {
		By("creating a network")

		By("creating a network interface")
		nic := &v1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nic-",
			},
			Spec: v1alpha1.NetworkInterfaceSpec{
				NodeRef: corev1.LocalObjectReference{
					Name: PartitionNodeName(partitionName, metalnetNode.Name),
				},
				NetworkRef: corev1.LocalObjectReference{
					Name: network.Name,
				},
				IPs: []net.IP{
					net.MustParseIP("10.0.0.1"),
				},
			},
		}
		Expect(k8sClient.Create(ctx, nic)).To(Succeed())

		By("waiting for the network interface to have a finalizer")
		Eventually(Object(nic)).Should(HaveField("Finalizers", []string{PartitionFinalizer(partitionName)}))

		By("waiting for the metalnet network interface to be present with the expected values")
		metalnetNic := &metalnetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metalnetNs.Name,
				Name:      string(nic.UID),
			},
		}
		Eventually(Object(metalnetNic)).Should(HaveField("Spec", metalnetv1alpha1.NetworkInterfaceSpec{
			NetworkRef: corev1.LocalObjectReference{Name: string(network.UID)},
			IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
			IPs:        []metalnetv1alpha1.IP{metalnetv1alpha1.MustParseIP("10.0.0.1")},
			NodeName:   &metalnetNode.Name,
		}))

		By("deleting the network interface")
		Expect(k8sClient.Delete(ctx, nic)).To(Succeed())

		By("waiting for the metalnet network interface to be gone")
		Eventually(Get(metalnetNic)).Should(Satisfy(apierrors.IsNotFound))
	})
})
