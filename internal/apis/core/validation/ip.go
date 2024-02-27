// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var IPTypes = sets.New(
	core.IPTypePublic,
)

func ValidateIPType(ipType core.IPType, fldPath *field.Path) field.ErrorList {
	return ValidateEnum(IPTypes, ipType, fldPath, "must specify IP type")
}

func ValidateIP(ip *core.IP) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(ip, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateIPSpec(&ip.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateIPSpec(spec *core.IPSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, ValidateIPType(spec.Type, fldPath.Child("type"))...)
	allErrs = append(allErrs, ValidateIPFamily(spec.IPFamily, fldPath.Child("ipFamily"))...)
	if spec.IP.IsValid() {
		allErrs = append(allErrs, ValidateIPMatchesFamily(spec.IP, spec.IPFamily, fldPath.Child("ip"))...)
	}

	return allErrs
}

func ValidateIPUpdate(newIP, oldIP *core.IP) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newIP, oldIP, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateIP(newIP)...)
	allErrs = append(allErrs, ValidateIPSpecUpdate(&newIP.Spec, &oldIP.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateIPSpecUpdate(newSpec, oldSpec *core.IPSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.Type, oldSpec.Type, fldPath.Child("type"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.IPFamily, oldSpec.IPFamily, fldPath.Child("ipFamily"))...)
	//if newSpec.IP != oldSpec.IP {
	//	allErrs = append(allErrs, field.Invalid(fldPath.Child("ip"), newSpec.IP, validation.FieldImmutableErrorMsg))
	//}

	return allErrs
}

func ValidateIPStatusUpdate(newIP, oldIP *core.IP) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newIP, oldIP, field.NewPath("metadata"))...)

	return allErrs
}
