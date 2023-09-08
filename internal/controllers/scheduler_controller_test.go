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
	cclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onmetal/onmetal-api-net/internal/controllers/scheduler"
	. "github.com/onmetal/onmetal-api/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("Scheduler", func() {
	ns := SetupNamespace(&k8sClient)

	BeforeEach(func() {
		By("waiting for the scheduler cache to report no nodes")
		snapshot := schedulerCache.Snapshot()
		Eventually(func() []*scheduler.ContainerInfo {
			snapshot.Update()
			return snapshot.ListNodes()
		}).Should(BeEmpty())
	})

	Context("when a node is present", func() {
		node := SetupNode()

		It("should set the instance's node ref to the available node", func(ctx SpecContext) {
			By("creating a load balancer instance")
			loadBalancerInstance := &v1alpha1.Instance{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "lb-inst-",
				},
				Spec: v1alpha1.InstanceSpec{
					Type:             v1alpha1.InstanceTypeLoadBalancer,
					LoadBalancerType: v1alpha1.LoadBalancerTypePublic,
					IPs:              []net.IP{net.MustParseIP("10.0.0.1")},
				},
			}
			Expect(k8sClient.Create(ctx, loadBalancerInstance)).To(Succeed())

			By("waiting for the load balancer instance to be scheduled")
			Eventually(Object(loadBalancerInstance)).Should(HaveField("Spec.NodeRef", &corev1.LocalObjectReference{
				Name: node.Name,
			}))
		})
	})

	Context("when nodes with multiple topologies are present", func() {
		const (
			zoneKey = "apinet.api.onmetal.de/zone"
			zoneA   = "zone-a"
			zoneB   = "zone-b"
		)
		var (
			zoneANode1 = SetupNodeWithLabels(map[string]string{
				zoneKey: zoneA,
			})
			zoneBNode1 = SetupNodeWithLabels(map[string]string{
				zoneKey: zoneB,
			})
		)

		It("should not schedule an instance if the instance anti affinity forbids it", func(ctx SpecContext) {
			By("creating three instances to spread over the zone topology")
			for i := 0; i < 3; i++ {
				inst := &v1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:    ns.Name,
						GenerateName: "inst-",
						Labels: map[string]string{
							"topology-test": "",
						},
					},
					Spec: v1alpha1.InstanceSpec{
						Type:             v1alpha1.InstanceTypeLoadBalancer,
						LoadBalancerType: v1alpha1.LoadBalancerTypePublic,
						IPs:              []net.IP{net.MustParseIP("10.0.0.1")},
						Affinity: &v1alpha1.Affinity{
							InstanceAntiAffinity: &v1alpha1.InstanceAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []v1alpha1.InstanceAffinityTerm{
									{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{"topology-test": ""},
										},
										TopologyKey: zoneKey,
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, inst)).To(Succeed())
			}

			By("waiting for two to be scheduled while one being unscheduled")
			haveAllInstancesScheduledExceptOne := HaveField("Items", ConsistOf(
				HaveField("Spec.NodeRef", &corev1.LocalObjectReference{Name: zoneANode1.Name}),
				HaveField("Spec.NodeRef", &corev1.LocalObjectReference{Name: zoneBNode1.Name}),
				HaveField("Spec.NodeRef", BeNil()),
			))
			Eventually(ObjectList(&v1alpha1.InstanceList{}, cclient.InNamespace(ns.Name))).Should(haveAllInstancesScheduledExceptOne)

			By("asserting it stays that way")
			Consistently(ObjectList(&v1alpha1.InstanceList{}, cclient.InNamespace(ns.Name))).Should(haveAllInstancesScheduledExceptOne)
		})
	})

	Context("when no node is present", func() {
		It("leave the instance's node ref empty", func(ctx SpecContext) {
			By("creating a load balancer instance")
			loadBalancerInstance := &v1alpha1.Instance{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "lb-inst-",
				},
				Spec: v1alpha1.InstanceSpec{
					Type:             v1alpha1.InstanceTypeLoadBalancer,
					LoadBalancerType: v1alpha1.LoadBalancerTypePublic,
					IPs:              []net.IP{net.MustParseIP("10.0.0.1")},
				},
			}
			Expect(k8sClient.Create(ctx, loadBalancerInstance)).To(Succeed())

			By("waiting for the load balancer instance to be scheduled")
			Consistently(Object(loadBalancerInstance)).Should(HaveField("Spec.NodeRef", BeNil()))
		})
	})
})
