// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ReconcileRequestAnnotation = "reconcile.apinet.ironcore.dev/requestedAt"

	// APINetletsGroup is the system rbac group all apinetlets are in.
	APINetletsGroup = "apinet.ironcore.dev:system:apinetlets"

	// APINetletUserNamePrefix is the prefix all apinetlet users should have.
	APINetletUserNamePrefix = "apinet.ironcore.dev:system:apinetlet:"

	// MetalnetletsGroup is the system rbac group all metalnetlets are in.
	MetalnetletsGroup = "apinet.ironcore.dev:system:metalnetlets"

	// MetalnetletUserNamePrefix is the prefix all metalnetlet users should have.
	MetalnetletUserNamePrefix = "apinet.ironcore.dev:system:metalnetlet:"
)

// APINetletCommonName constructs the common name for a certificate of an apinetlet user.
func APINetletCommonName(name string) string {
	return APINetletUserNamePrefix + name
}

// MetalnetletCommonName constructs the common name for a certificate of a metalnetlet user.
func MetalnetletCommonName(name string) string {
	return MetalnetletUserNamePrefix + name
}

// ObjectSelector specifies how to select objects of a certain kind.
type ObjectSelector struct {
	// Kind is the kind of object to select.
	Kind string `json:"kind"`
	// LabelSelector is the label selector to select objects of the specified Kind by.
	metav1.LabelSelector `json:",inline"`
}

// LocalUIDReference is a reference to another entity including its UID
// +structType=atomic
type LocalUIDReference struct {
	// Name is the name of the referenced entity.
	Name string `json:"name"`
	// UID is the UID of the referenced entity.
	UID types.UID `json:"uid"`
}
