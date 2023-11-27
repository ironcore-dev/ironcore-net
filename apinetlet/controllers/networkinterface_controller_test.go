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
	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	"github.com/ironcore-dev/ironcore-net/apinetlet/provider"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("NetworkInterfaceController", func() {
	ns := SetupNamespace(&k8sClient)
	apiNetNs := SetupNamespace(&k8sClient)
	SetupTest(apiNetNs)
	network, apiNetNetwork := SetupNetwork(ns, apiNetNs)

	It("should claim and reconcile the network interface", func(ctx SpecContext) {
		By("creating a virtual IP")
		vip := &networkingv1alpha1.VirtualIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "vip-",
			},
			Spec: networkingv1alpha1.VirtualIPSpec{
				Type:     networkingv1alpha1.VirtualIPTypePublic,
				IPFamily: corev1.IPv4Protocol,
			},
		}
		Expect(k8sClient.Create(ctx, vip)).To(Succeed())

		By("waiting for the virtual IP report an IP")
		Eventually(Object(vip)).Should(HaveField("Status.IP", Not(BeNil())))
		publicIP := *vip.Status.IP

		By("creating an apinet network interface")
		apiNetNic := &apinetv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    apiNetNs.Name,
				GenerateName: "apinet-nic-",
			},
			Spec: apinetv1alpha1.NetworkInterfaceSpec{
				NetworkRef: corev1.LocalObjectReference{Name: apiNetNetwork.Name},
				IPs:        []net.IP{net.MustParseIP("192.168.178.1")},
				NodeRef:    corev1.LocalObjectReference{Name: "my-node"},
			},
		}
		Expect(k8sClient.Create(ctx, apiNetNic)).To(Succeed())

		By("creating a network interface")
		nic := &networkingv1alpha1.NetworkInterface{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "nic-",
			},
			Spec: networkingv1alpha1.NetworkInterfaceSpec{
				ProviderID: provider.GetNetworkInterfaceID(apiNetNs.Name, apiNetNic.Name, "node", apiNetNic.UID),
				NetworkRef: corev1.LocalObjectReference{Name: network.Name},
				IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol},
				IPs: []networkingv1alpha1.IPSource{
					{Value: commonv1alpha1.MustParseNewIP("192.168.178.1")},
				},
				VirtualIP: &networkingv1alpha1.VirtualIPSource{
					VirtualIPRef: &corev1.LocalObjectReference{Name: vip.Name},
				},
			},
		}
		Expect(k8sClient.Create(ctx, nic)).To(Succeed())

		By("waiting for the APINet network interface to be claimed")
		Eventually(Object(apiNetNic)).Should(SatisfyAll(
			HaveField("Spec.PublicIPs", ConsistOf(HaveField("IP", net.IP{Addr: publicIP.Addr}))),
			WithTransform(func(apiNetNic *apinetv1alpha1.NetworkInterface) *apinetletclient.SourceObjectData {
				return apinetletclient.SourceObjectDataFromObject(
					k8sClient.Scheme(),
					k8sClient.RESTMapper(),
					nic,
					apiNetNic,
				)
			}, Equal(&apinetletclient.SourceObjectData{
				Namespace: nic.Namespace,
				Name:      nic.Name,
				UID:       nic.UID,
			}))),
		)
	})
})
