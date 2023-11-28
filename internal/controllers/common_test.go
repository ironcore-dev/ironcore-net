// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetupNetwork(namespace *corev1.Namespace) *v1alpha1.Network {
	network := &v1alpha1.Network{}

	BeforeEach(func(ctx SpecContext) {
		By("creating a network")
		*network = v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    namespace.Name,
				GenerateName: "network-",
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())
	})

	return network
}

func SetupNode() *v1alpha1.Node {
	return SetupNodeWithLabels(nil)
}

func SetupNodeWithLabels(labels map[string]string) *v1alpha1.Node {
	return SetupObjectStruct[*v1alpha1.Node](&k8sClient, func(node *v1alpha1.Node) {
		*node = v1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "node-",
				Labels:       labels,
			},
		}
	})
}
