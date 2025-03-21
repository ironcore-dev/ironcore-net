// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	types "k8s.io/apimachinery/pkg/types"
)

// NATTableIPTargetRefApplyConfiguration represents a declarative configuration of the NATTableIPTargetRef type for use
// with apply.
type NATTableIPTargetRefApplyConfiguration struct {
	UID     *types.UID               `json:"uid,omitempty"`
	Name    *string                  `json:"name,omitempty"`
	NodeRef *v1.LocalObjectReference `json:"nodeRef,omitempty"`
}

// NATTableIPTargetRefApplyConfiguration constructs a declarative configuration of the NATTableIPTargetRef type for use with
// apply.
func NATTableIPTargetRef() *NATTableIPTargetRefApplyConfiguration {
	return &NATTableIPTargetRefApplyConfiguration{}
}

// WithUID sets the UID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the UID field is set to the value of the last call.
func (b *NATTableIPTargetRefApplyConfiguration) WithUID(value types.UID) *NATTableIPTargetRefApplyConfiguration {
	b.UID = &value
	return b
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *NATTableIPTargetRefApplyConfiguration) WithName(value string) *NATTableIPTargetRefApplyConfiguration {
	b.Name = &value
	return b
}

// WithNodeRef sets the NodeRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NodeRef field is set to the value of the last call.
func (b *NATTableIPTargetRefApplyConfiguration) WithNodeRef(value v1.LocalObjectReference) *NATTableIPTargetRefApplyConfiguration {
	b.NodeRef = &value
	return b
}
