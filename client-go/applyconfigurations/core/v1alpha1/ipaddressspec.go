// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	net "github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
)

// IPAddressSpecApplyConfiguration represents a declarative configuration of the IPAddressSpec type for use
// with apply.
type IPAddressSpecApplyConfiguration struct {
	IP       *net.IP                              `json:"ip,omitempty"`
	ClaimRef *IPAddressClaimRefApplyConfiguration `json:"claimRef,omitempty"`
}

// IPAddressSpecApplyConfiguration constructs a declarative configuration of the IPAddressSpec type for use with
// apply.
func IPAddressSpec() *IPAddressSpecApplyConfiguration {
	return &IPAddressSpecApplyConfiguration{}
}

// WithIP sets the IP field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IP field is set to the value of the last call.
func (b *IPAddressSpecApplyConfiguration) WithIP(value net.IP) *IPAddressSpecApplyConfiguration {
	b.IP = &value
	return b
}

// WithClaimRef sets the ClaimRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClaimRef field is set to the value of the last call.
func (b *IPAddressSpecApplyConfiguration) WithClaimRef(value *IPAddressClaimRefApplyConfiguration) *IPAddressSpecApplyConfiguration {
	b.ClaimRef = value
	return b
}
