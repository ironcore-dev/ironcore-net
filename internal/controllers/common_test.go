// Copyright 2023 IronCore authors
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
