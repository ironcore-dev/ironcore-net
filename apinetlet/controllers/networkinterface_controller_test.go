// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	"github.com/ironcore-dev/ironcore-net/apinetlet/provider"
	. "github.com/ironcore-dev/ironcore-net/utils/testing"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
				Labels: map[string]string{
					"app": "test",
				},
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
			HaveField("Labels", HaveKeyWithValue("app", "test")),
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

	It("should release IP Object when claimer NetworkInterface is deleted", func(ctx SpecContext) {
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

		By("waiting for the corresponding IP object in ironcore-net")
		ip := &apinetv1alpha1.IP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: apiNetNs.Name,
				Name:      string(vip.UID),
			},
		}
		Eventually(Object(ip)).Should(SatisfyAll(
			HaveField("Labels", HaveKeysWithValues(apinetletclient.SourceLabels(k8sClient.Scheme(), k8sClient.RESTMapper(), vip))),
		))

		By("creating an ironcore-net network interface")
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

		By("waiting for the IP to be claimed")
		Eventually(Object(vip)).Should(HaveField("Spec.TargetRef.UID", Equal(nic.UID)))
		Eventually(Object(ip)).Should(HaveField("Spec.ClaimRef.UID", Equal(apiNetNic.UID)))

		By("Deleting the network interface")
		Expect(k8sClient.Delete(ctx, nic)).To(Succeed())
		ni := &networkingv1alpha1.NetworkInterface{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: nic.Name}, ni)
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())

		By("waiting for the IP to be released")
		Eventually(Object(vip)).Should(HaveField("Spec.TargetRef", BeNil()))
		Eventually(Object(ip)).Should(HaveField("Spec.ClaimRef", BeNil()))
	})
})
