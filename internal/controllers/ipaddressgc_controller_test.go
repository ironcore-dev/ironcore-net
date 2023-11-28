// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/google/uuid"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("IPAddressGarbageCollectorController", func() {
	ns := SetupNamespace(&k8sClient)

	It("should not release IP addresses that have a claimer", func(ctx SpecContext) {
		By("creating an IP")
		ip := &v1alpha1.IP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "ip-",
			},
			Spec: v1alpha1.IPSpec{
				Type:     v1alpha1.IPTypePublic,
				IPFamily: corev1.IPv4Protocol,
			},
		}
		Expect(k8sClient.Create(ctx, ip)).To(Succeed())

		By("asserting the IP address stays present")
		ipAddress := &v1alpha1.IPAddress{
			ObjectMeta: metav1.ObjectMeta{
				Name: ip.Spec.IP.String(),
			},
		}
		Consistently(Get(ipAddress)).Should(Succeed())
	})

	It("should release a IP address with a non-existent claimer", func(ctx SpecContext) {
		By("creating a IP address")
		var addr *v1alpha1.IPAddress
		// We have to create the IP address in a loop since we can *not* know the free IP addresses.
		// This is 'ugly', however, it's the best we can do with the current implementation.
		for ip := PrefixV4().Masked().Addr(); PrefixV4().Contains(ip); ip = ip.Next() {
			addr = &v1alpha1.IPAddress{
				ObjectMeta: metav1.ObjectMeta{
					Name: ip.String(),
				},
				Spec: v1alpha1.IPAddressSpec{
					ClaimRef: v1alpha1.IPAddressClaimRef{
						Group:    v1alpha1.GroupName,
						Resource: "natgateways",
						UID:      types.UID(uuid.NewString()),
						Name:     "should-not-exist",
					},
				},
			}
			err := k8sClient.Create(ctx, addr)
			if err == nil {
				break
			}
			Expect(err).To(Satisfy(apierrors.IsAlreadyExists))
		}
		Expect(addr).NotTo(BeNil(), "no free IP address could be found / created")

		By("waiting for the IP address to be deleted")
		Eventually(Get(addr)).Should(Satisfy(apierrors.IsNotFound))
	})
})
