// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNetworkPolicyRule(networkPolicyRule *core.NetworkPolicyRule) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(networkPolicyRule, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)

	return allErrs
}

func ValidateNetworkPolicyRuleUpdate(newNetworkPolicyRule, oldNetworkPolicyRule *core.NetworkPolicyRule) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNetworkPolicyRule, oldNetworkPolicyRule, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNetworkPolicyRule(newNetworkPolicyRule)...)

	return allErrs
}
