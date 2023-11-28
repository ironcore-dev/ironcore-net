// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/networkid"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNetworkIDName(name string, prefix bool) []string {
	var errs []string
	vni, err := networkid.ParseVNI(name)
	if err != nil {
		errs = append(errs, err.Error())
	} else if networkid.EncodeVNI(vni) != name {
		errs = append(errs, "not a valid VNI in canonical format")
	}
	return errs
}

func ValidateNetworkID(networkID *core.NetworkID) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(networkID, false, ValidateNetworkIDName, field.NewPath("metadata"))...)

	return allErrs
}

func ValidateNetworkIDUpdate(newNetworkID, oldNetworkID *core.NetworkID) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNetworkID, oldNetworkID, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNetworkID(newNetworkID)...)
	allErrs = append(allErrs, ValidateNetworkIDSpecUpdate(&newNetworkID.Spec, &oldNetworkID.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNetworkIDSpecUpdate(newSpec, oldSpec *core.NetworkIDSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.ClaimRef, oldSpec.ClaimRef, fldPath.Child("claimRef"))...)

	return allErrs
}
