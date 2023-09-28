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

package provider_test

import (
	. "github.com/onmetal/onmetal-api-net/apinetlet/provider"
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
		const providerID = string("onmetal-api-net://" + namespace + "/" + name + "/" + node + "/" + uid)

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
			providerID = string("onmetal-api-net://" + namespace + "/" + name + "/" + id + "/" + uid)
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
