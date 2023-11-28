// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNATGateway(natGateway *core.NATGateway) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(natGateway, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNATGatewaySpec(&natGateway.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNATGatewaySpec(spec *core.NATGatewaySpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, ValidateIPFamily(spec.IPFamily, fldPath.Child("ipFamily"))...)

	for i, ip := range spec.IPs {
		fldPath := fldPath.Child("ips").Index(i)
		if ip.IP.IsValid() {
			allErrs = append(allErrs, ValidateIPMatchesFamily(ip.IP, spec.IPFamily, fldPath.Child("ip"))...)
		}
	}

	return allErrs
}

func ValidateNATGatewayUpdate(newNATGateway, oldNATGateway *core.NATGateway) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNATGateway, oldNATGateway, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNATGateway(newNATGateway)...)
	allErrs = append(allErrs, ValidateNATGatewaySpecUpdate(&newNATGateway.Spec, &oldNATGateway.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNATGatewaySpecUpdate(newSpec, oldSpec *core.NATGatewaySpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.IPFamily, oldSpec.IPFamily, fldPath.Child("ipFamily"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.NetworkRef, oldSpec.NetworkRef, fldPath.Child("networkRef"))...)

	return allErrs
}

func ValidateNATGatewayStatusUpdate(newNATGateway, oldNATGateway *core.NATGateway) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNATGateway, oldNATGateway, field.NewPath("metadata"))...)

	return allErrs
}
