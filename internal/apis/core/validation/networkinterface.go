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

package validation

import (
	"github.com/onmetal/onmetal-api-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNetworkInterface(networkInterface *core.NetworkInterface) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(networkInterface, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNetworkInterfaceSpec(&networkInterface.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNetworkInterfaceSpec(spec *core.NetworkInterfaceSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if spec.NodeRef.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("nodeRef", "name"), "must specify target node"))
	}

	return allErrs
}

func ValidateNetworkInterfaceUpdate(newNetworkInterface, oldNetworkInterface *core.NetworkInterface) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNetworkInterface, oldNetworkInterface, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNetworkInterface(newNetworkInterface)...)
	allErrs = append(allErrs, ValidateNetworkInterfaceSpecUpdate(&newNetworkInterface.Spec, &oldNetworkInterface.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNetworkInterfaceSpecUpdate(newSpec, oldSpec *core.NetworkInterfaceSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.NetworkRef, oldSpec.NetworkRef, fldPath.Child("networkRef"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.NodeRef, oldSpec.NodeRef, fldPath.Child("nodeRef"))...)

	return allErrs
}

func ValidateNetworkInterfaceStatusUpdate(newNetworkInterface, oldNetworkInterface *core.NetworkInterface) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNetworkInterface, oldNetworkInterface, field.NewPath("metadata"))...)

	return allErrs
}
