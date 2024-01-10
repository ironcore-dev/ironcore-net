// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNetwork(network *core.Network) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(network, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNetworkSpec(&network.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNetworkSpec(spec *core.NetworkSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	_ = spec
	_ = fldPath

	return allErrs
}

func ValidateNetworkUpdate(newNetwork, oldNetwork *core.Network) field.ErrorList {
	var allErrs field.ErrorList

	_ = newNetwork
	_ = oldNetwork

	return allErrs
}

func ValidateNetworkSpecUpdate(newSpec, oldSpec *core.NetworkSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	_ = newSpec
	_ = oldSpec
	_ = fldPath

	return allErrs
}

func ValidateNetworkStatusUpdate(newNetwork, oldNetwork *core.Network) field.ErrorList {
	var allErrs field.ErrorList

	_ = newNetwork
	_ = oldNetwork

	return allErrs
}
