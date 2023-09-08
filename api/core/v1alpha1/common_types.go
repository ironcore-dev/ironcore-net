// Copyright 2022 OnMetal authors
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

package v1alpha1

const (
	ReconcileRequestAnnotation = "reconcile.apinet.api.onmetal.de/requestedAt"

	// APINetletsGroup is the system rbac group all apinetlets are in.
	APINetletsGroup = "apinet.api.onmetal.de:system:apinetlets"

	// APINetletUserNamePrefix is the prefix all apinetlet users should have.
	APINetletUserNamePrefix = "apinet.api.onmetal.de:system:apinetlet:"

	// MetalnetletsGroup is the system rbac group all metalnetlets are in.
	MetalnetletsGroup = "apinet.api.onmetal.de:system:metalnetlets"

	// MetalnetletUserNamePrefix is the prefix all metalnetlet users should have.
	MetalnetletUserNamePrefix = "apinet.api.onmetal.de:system:metalnetlet:"
)

// APINetletCommonName constructs the common name for a certificate of an apinetlet user.
func APINetletCommonName(name string) string {
	return APINetletUserNamePrefix + name
}

// MetalnetletCommonName constructs the common name for a certificate of a metalnetlet user.
func MetalnetletCommonName(name string) string {
	return MetalnetletUserNamePrefix + name
}
