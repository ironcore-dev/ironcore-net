// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

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
