// Copyright 2023 OnMetal authors
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
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
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
