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
	"fmt"
	"slices"

	"github.com/onmetal/onmetal-api-net/internal/apis/core"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
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

	seenInternalIPFamilies := sets.New[corev1.IPFamily]()
	for i, ip := range spec.IPs {
		fldPath := field.NewPath("ips").Index(i)
		if !ip.IsValid() {
			allErrs = append(allErrs, field.Invalid(fldPath, ip, "must specify valid IP"))
		} else if seenInternalIPFamilies.Has(ip.Family()) {
			allErrs = append(allErrs, field.Invalid(fldPath, ip, fmt.Sprintf("cannot have multiple internal IPs of family %s", ip.Family())))
		} else {
			seenInternalIPFamilies.Insert(ip.Family())
		}
	}

	seenExternalIPFamilies := sets.New[corev1.IPFamily]()
	for i, ip := range spec.PublicIPs {
		fldPath := fldPath.Child("publicIPs").Index(i)
		allErrs = append(allErrs, ValidateIPFamily(ip.IPFamily, fldPath.Child("ipFamily"))...)
		if seenExternalIPFamilies.Has(ip.IPFamily) {
			allErrs = append(allErrs, field.Forbidden(fldPath, fmt.Sprintf("cannot have multiple external IPs of family %s", ip.IPFamily)))
		} else {
			seenExternalIPFamilies.Insert(ip.IPFamily)
		}
	}

	for i, nat := range spec.NATs {
		fldPath := fldPath.Child("nats").Index(i)
		allErrs = append(allErrs, ValidateIPFamily(nat.IPFamily, fldPath.Child("ipFamily"))...)
		if seenExternalIPFamilies.Has(nat.IPFamily) {
			allErrs = append(allErrs, field.Forbidden(fldPath, fmt.Sprintf("cannot have nat for already present external IP family %s", nat.IPFamily)))
		} else {
			seenExternalIPFamilies.Insert(nat.IPFamily)
		}
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
	if !slices.Equal(newSpec.IPs, oldSpec.IPs) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ips"), newSpec.IPs, validation.FieldImmutableErrorMsg))
	}

	return allErrs
}

func ValidateNetworkInterfaceStatusUpdate(newNetworkInterface, oldNetworkInterface *core.NetworkInterface) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNetworkInterface, oldNetworkInterface, field.NewPath("metadata"))...)

	return allErrs
}
