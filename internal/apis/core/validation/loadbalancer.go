// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var LoadBalancerTypes = sets.New(
	core.LoadBalancerTypePublic,
	core.LoadBalancerTypeInternal,
)

func ValidateLoadBalancerType(typ core.LoadBalancerType, fldPath *field.Path) field.ErrorList {
	return ValidateEnum(LoadBalancerTypes, typ, fldPath, "must specify type")
}

func ValidateLoadBalancer(loadBalancer *core.LoadBalancer) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(loadBalancer, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateLoadBalancerSpec(&loadBalancer.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateLoadBalancerSpec(spec *core.LoadBalancerSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, ValidateLoadBalancerType(spec.Type, fldPath.Child("type"))...)

	for i, ip := range spec.IPs {
		fldPath := fldPath.Child("ips").Index(i)
		allErrs = append(allErrs, ValidateIPFamily(ip.IPFamily, fldPath)...)
		if ip.IP.IsValid() {
			allErrs = append(allErrs, ValidateIPMatchesFamily(ip.IP, ip.IPFamily, fldPath.Child("ip"))...)
		}
	}

	allErrs = append(allErrs, metav1validation.ValidateLabelSelector(spec.Selector, metav1validation.LabelSelectorValidationOptions{}, fldPath.Child("selector"))...)

	if sel, err := metav1.LabelSelectorAsSelector(spec.Selector); err == nil {
		if !sel.Matches(labels.Set(spec.Template.Labels)) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("template", "labels"), spec.Template.Labels, "`selector` does not match template `labels`"))
		}
	}

	return allErrs
}

func ValidateLoadBalancerUpdate(newLoadBalancer, oldLoadBalancer *core.LoadBalancer) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newLoadBalancer, oldLoadBalancer, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateLoadBalancer(newLoadBalancer)...)
	allErrs = append(allErrs, ValidateLoadBalancerSpecUpdate(&newLoadBalancer.Spec, &oldLoadBalancer.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateLoadBalancerSpecUpdate(newSpec, oldSpec *core.LoadBalancerSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.Type, oldSpec.Type, fldPath.Child("type"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.NetworkRef, oldSpec.NetworkRef, fldPath.Child("networkRef"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.Selector, oldSpec.Selector, fldPath.Child("selector"))...)

	return allErrs
}

func ValidateLoadBalancerStatusUpdate(newLoadBalancer, oldLoadBalancer *core.LoadBalancer) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newLoadBalancer, oldLoadBalancer, field.NewPath("metadata"))...)

	return allErrs
}
