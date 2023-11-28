// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	. "github.com/ironcore-dev/ironcore-net/apinetlet/provider"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Provider", func() {
	const (
		namespace = "namespace"
		node      = "node.dns.subdomain"
		name      = "name"
		uid       = types.UID("2c24a85c-b55d-44b8-bf6e-1872ecfef5db")
	)

	Context("NetworkInterfaceID", func() {
		const providerID = string("ironcore-net://" + namespace + "/" + name + "/" + node + "/" + uid)

		Describe("GetNetworkInterfaceID", func() {
			It("should produce a correct network interface ID", func() {
				Expect(GetNetworkInterfaceID(namespace, name, node, uid)).To(Equal(providerID))
			})
		})

		Describe("ParseNetworkInterfaceID", func() {
			It("should parse the given network interface ID", func() {
				actualNs, actualName, actualNode, actualUID, err := ParseNetworkInterfaceID(providerID)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualNs).To(Equal(namespace))
				Expect(actualName).To(Equal(name))
				Expect(actualNode).To(Equal(node))
				Expect(actualUID).To(Equal(uid))
			})
		})
	})

	Context("NetworkID", func() {
		const (
			id         = "foo"
			providerID = string("ironcore-net://" + namespace + "/" + name + "/" + id + "/" + uid)
		)

		Context("ParseNetworkID", func() {
			It("should parse the network id", func() {
				actualNs, actualName, actualID, actualUID, err := ParseNetworkID(providerID)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualNs).To(Equal(namespace))
				Expect(actualName).To(Equal(name))
				Expect(actualID).To(Equal(id))
				Expect(actualUID).To(Equal(uid))
			})
		})

		Context("GetNetworkID", func() {
			It("should correctly encode the network ID", func() {
				Expect(GetNetworkID(namespace, name, id, uid)).To(Equal(providerID))
			})
		})
	})
})
