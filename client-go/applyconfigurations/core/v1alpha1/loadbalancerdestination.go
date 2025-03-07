// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	net "github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
)

// LoadBalancerDestinationApplyConfiguration represents a declarative configuration of the LoadBalancerDestination type for use
// with apply.
type LoadBalancerDestinationApplyConfiguration struct {
	IP        *net.IP                                  `json:"ip,omitempty"`
	TargetRef *LoadBalancerTargetRefApplyConfiguration `json:"targetRef,omitempty"`
}

// LoadBalancerDestinationApplyConfiguration constructs a declarative configuration of the LoadBalancerDestination type for use with
// apply.
func LoadBalancerDestination() *LoadBalancerDestinationApplyConfiguration {
	return &LoadBalancerDestinationApplyConfiguration{}
}

// WithIP sets the IP field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IP field is set to the value of the last call.
func (b *LoadBalancerDestinationApplyConfiguration) WithIP(value net.IP) *LoadBalancerDestinationApplyConfiguration {
	b.IP = &value
	return b
}

// WithTargetRef sets the TargetRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TargetRef field is set to the value of the last call.
func (b *LoadBalancerDestinationApplyConfiguration) WithTargetRef(value *LoadBalancerTargetRefApplyConfiguration) *LoadBalancerDestinationApplyConfiguration {
	b.TargetRef = value
	return b
}
