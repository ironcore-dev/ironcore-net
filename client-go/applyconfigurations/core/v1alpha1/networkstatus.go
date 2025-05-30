// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	corev1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
)

// NetworkStatusApplyConfiguration represents a declarative configuration of the NetworkStatus type for use
// with apply.
type NetworkStatusApplyConfiguration struct {
	Peerings map[string][]corev1alpha1.NetworkPeeringStatus `json:"peerings,omitempty"`
}

// NetworkStatusApplyConfiguration constructs a declarative configuration of the NetworkStatus type for use with
// apply.
func NetworkStatus() *NetworkStatusApplyConfiguration {
	return &NetworkStatusApplyConfiguration{}
}

// WithPeerings puts the entries into the Peerings field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Peerings field,
// overwriting an existing map entries in Peerings field with the same key.
func (b *NetworkStatusApplyConfiguration) WithPeerings(entries map[string][]corev1alpha1.NetworkPeeringStatus) *NetworkStatusApplyConfiguration {
	if b.Peerings == nil && len(entries) > 0 {
		b.Peerings = make(map[string][]corev1alpha1.NetworkPeeringStatus, len(entries))
	}
	for k, v := range entries {
		b.Peerings[k] = v
	}
	return b
}
