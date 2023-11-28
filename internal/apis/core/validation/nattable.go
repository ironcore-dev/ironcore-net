// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNATTable(natTable *core.NATTable) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(natTable, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)

	return allErrs
}

func ValidateNATTableUpdate(newNATTable, oldNATTable *core.NATTable) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNATTable, oldNATTable, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNATTable(newNATTable)...)

	return allErrs
}
