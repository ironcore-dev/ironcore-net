// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

func SetupNetwork(ns, apiNetNS *corev1.Namespace) (*networkingv1alpha1.Network, *v1alpha1.Network) {
	var (
		network       = &networkingv1alpha1.Network{}
		apiNetNetwork = &v1alpha1.Network{}
	)

	ginkgo.BeforeEach(func(ctx ginkgo.SpecContext) {
		ginkgo.By("creating a network")
		*network = networkingv1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "network-",
			},
		}
		Expect(k8sClient.Create(ctx, network))

		ginkgo.By("waiting for the APINet network to be present")
		*apiNetNetwork = v1alpha1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNS.Name,
				Name:      string(network.UID),
			},
		}
		Eventually(komega.Get(apiNetNetwork)).Should(Succeed())
	})

	return network, apiNetNetwork
}
