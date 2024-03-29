// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateDaemonSet(daemonSet *core.DaemonSet) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(daemonSet, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateDaemonSetSpec(&daemonSet.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateDaemonSetSpec(spec *core.DaemonSetSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, metav1validation.ValidateLabelSelector(spec.Selector, metav1validation.LabelSelectorValidationOptions{}, fldPath.Child("selector"))...)
	if sel, err := metav1.LabelSelectorAsSelector(spec.Selector); err == nil {
		if !sel.Matches(labels.Set(spec.Template.Labels)) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("template", "labels"), spec.Template.Labels, "`selector` does not match template `labels`"))
		}
	}

	return allErrs
}

func ValidateDaemonSetUpdate(newDaemonSet, oldDaemonSet *core.DaemonSet) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newDaemonSet, oldDaemonSet, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateDaemonSet(newDaemonSet)...)

	return allErrs
}

func ValidateDaemonSetStatusUpdate(newDaemonSet, oldDaemonSet *core.DaemonSet) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newDaemonSet, oldDaemonSet, field.NewPath("metadata"))...)

	return allErrs
}
