// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// NetworkPeeringApplyConfiguration represents an declarative configuration of the NetworkPeering type for use
// with apply.
type NetworkPeeringApplyConfiguration struct {
	Name *string `json:"name,omitempty"`
	ID   *string `json:"id,omitempty"`
}

// NetworkPeeringApplyConfiguration constructs an declarative configuration of the NetworkPeering type for use with
// apply.
func NetworkPeering() *NetworkPeeringApplyConfiguration {
	return &NetworkPeeringApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *NetworkPeeringApplyConfiguration) WithName(value string) *NetworkPeeringApplyConfiguration {
	b.Name = &value
	return b
}

// WithID sets the ID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ID field is set to the value of the last call.
func (b *NetworkPeeringApplyConfiguration) WithID(value string) *NetworkPeeringApplyConfiguration {
	b.ID = &value
	return b
}