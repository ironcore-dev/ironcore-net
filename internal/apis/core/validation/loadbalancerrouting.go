// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateLoadBalancerRouting(loadBalancerRouting *core.LoadBalancerRouting) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(loadBalancerRouting, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)

	return allErrs
}

func ValidateLoadBalancerRoutingUpdate(newLoadBalancerRouting, oldLoadBalancerRouting *core.LoadBalancerRouting) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newLoadBalancerRouting, oldLoadBalancerRouting, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateLoadBalancerRouting(newLoadBalancerRouting)...)

	return allErrs
}
