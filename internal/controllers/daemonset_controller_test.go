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
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	"github.com/onmetal/onmetal-api-net/apimachinery/api/net"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("DaemonSetController", func() {
	ns := SetupNamespace(&k8sClient)
	network := SetupNetwork(ns)
	BeforeEach(func() {
		Eventually(New(mgrClient).ObjectList(&v1alpha1.NodeList{})).
			Should(HaveField("Items", BeEmpty()))
	})
	node1, node2 := SetupNode(), SetupNode()

	It("should correctly manage the daemon set instances", func(ctx SpecContext) {
		By("creating a daemon set")
		ds := &v1alpha1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "ds-",
			},
			Spec: v1alpha1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"foo": "bar"},
				},
				Template: v1alpha1.InstanceTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"foo": "bar"},
					},
					Spec: v1alpha1.InstanceSpec{
						Type:             v1alpha1.InstanceTypeLoadBalancer,
						LoadBalancerType: v1alpha1.LoadBalancerTypePublic,
						NetworkRef:       corev1.LocalObjectReference{Name: network.Name},
						IPs:              []net.IP{net.MustParseIP("10.0.0.1")},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, ds)).To(Succeed())

		By("waiting for instances to be present with affinity for each node")
		nodeAffinityFor := func(nodeName string) *v1alpha1.Affinity {
			return &v1alpha1.Affinity{
				NodeAffinity: &v1alpha1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &v1alpha1.NodeSelector{
						NodeSelectorTerms: []v1alpha1.NodeSelectorTerm{
							{
								MatchFields: []v1alpha1.NodeSelectorRequirement{
									{
										Key:      "metadata.name",
										Operator: v1alpha1.NodeSelectorOpIn,
										Values:   []string{nodeName},
									},
								},
							},
						},
					},
				},
			}
		}
		Eventually(ObjectList(&v1alpha1.InstanceList{},
			client.InNamespace(ns.Name),
		)).Should(HaveField("Items", SatisfyAll(
			HaveEach(HaveField("Spec.IPs", []net.IP{net.MustParseIP("10.0.0.1")})),
			ConsistOf(
				HaveField("Spec.Affinity", nodeAffinityFor(node1.Name)),
				HaveField("Spec.Affinity", nodeAffinityFor(node2.Name)),
			)),
		))

		By("updating the daemon set template IPs")
		Eventually(Update(ds, func() {
			ds.Spec.Template.Spec.IPs = []net.IP{net.MustParseIP("192.168.178.1")}
		})).Should(Succeed())

		By("waiting until the instance IPs are updated")
		Eventually(ObjectList(&v1alpha1.InstanceList{},
			client.InNamespace(ns.Name),
		)).Should(HaveField("Items", HaveEach(HaveField("Spec.IPs", []net.IP{net.MustParseIP("192.168.178.1")}))))

		By("deleting the instances")
		Expect(k8sClient.DeleteAllOf(ctx, &v1alpha1.Instance{}, client.InNamespace(ns.Name))).To(Succeed())

		By("waiting for new instances to be created again")
		Eventually(ObjectList(&v1alpha1.InstanceList{},
			client.InNamespace(ns.Name),
		)).Should(HaveField("Items", SatisfyAll(
			ContainElements(
				HaveField("DeletionTimestamp", BeNil()),
				HaveField("DeletionTimestamp", BeNil()),
			)),
		))
	})
})
