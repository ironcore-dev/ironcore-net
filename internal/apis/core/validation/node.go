// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var ValidateNodeName = validation.NameIsDNSSubdomain

func ValidateNode(node *core.Node) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(node, false, ValidateNodeName, field.NewPath("metadata"))...)

	return allErrs
}

func ValidateNodeUpdate(newNode, oldNode *core.Node) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNode, oldNode, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNode(newNode)...)

	return allErrs
}

func ValidateNodeStatusUpdate(newNode, oldNode *core.Node) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNode, oldNode, field.NewPath("metadata"))...)

	return allErrs
}
