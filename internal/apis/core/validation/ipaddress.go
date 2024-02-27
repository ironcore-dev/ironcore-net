// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"net/netip"

	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateIPAddressName(name string, prefix bool) []string {
	var errs []string
	ip, err := netip.ParseAddr(name)
	if err != nil {
		errs = append(errs, err.Error())
	} else if ip.String() != name {
		errs = append(errs, "not a valid IP address in canonical format")
	}
	return errs
}

func ValidateIPAddress(ipAddress *core.IPAddress) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(ipAddress, false, ValidateIPAddressName, field.NewPath("metadata"))...)

	return allErrs
}

func ValidateIPAddressUpdate(newIPAddress, oldIPAddress *core.IPAddress) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newIPAddress, oldIPAddress, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateIPAddress(newIPAddress)...)
	//allErrs = append(allErrs, ValidateIPAddressSpecUpdate(&newIPAddress.Spec, &oldIPAddress.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateIPAddressSpecUpdate(newSpec, oldSpec *core.IPAddressSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if newSpec.IP != oldSpec.IP {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ip"), newSpec.IP, validation.FieldImmutableErrorMsg))
	}
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.ClaimRef, oldSpec.ClaimRef, fldPath.Child("clamRef"))...)

	return allErrs
}

func ValidateIPAddressStatusUpdate(newIPAddress, oldIPAddress *core.IPAddress) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newIPAddress, oldIPAddress, field.NewPath("metadata"))...)

	return allErrs
}
