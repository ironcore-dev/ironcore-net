// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"github.com/onmetal/onmetal-api-net/internal/apis/core"
	"github.com/onmetal/onmetal-api/utils/generic"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNATGatewayAutoscaler(natGatewayAutoscaler *core.NATGatewayAutoscaler) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(natGatewayAutoscaler, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNATGatewayAutoscalerSpec(&natGatewayAutoscaler.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNATGatewayAutoscalerSpec(spec *core.NATGatewayAutoscalerSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	minPublicIPs := generic.DerefZero(spec.MinPublicIPs)
	allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(minPublicIPs), fldPath.Child("minPublicIPs"))...)

	maxPublicIPs := generic.DerefZero(spec.MaxPublicIPs)
	allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(maxPublicIPs), fldPath.Child("maxPublicIPs"))...)

	if minPublicIPs > maxPublicIPs {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("maxPublicIPs"), maxPublicIPs, "must >= minPublicIPs"))
	}

	return allErrs
}

func ValidateNATGatewayAutoscalerUpdate(newNATGatewayAutoscaler, oldNATGatewayAutoscaler *core.NATGatewayAutoscaler) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNATGatewayAutoscaler, oldNATGatewayAutoscaler, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNATGatewayAutoscaler(newNATGatewayAutoscaler)...)

	return allErrs
}

func ValidateNATGatewayAutoscalerSpecUpdate(newSpec, oldSpec *core.NATGatewayAutoscalerSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.NATGatewayRef, oldSpec.NATGatewayRef, fldPath.Child("natGatewayRef"))...)

	return allErrs
}

func ValidateNATGatewayAutoscalerStatusUpdate(newNATGatewayAutoscaler, oldNATGatewayAutoscaler *core.NATGatewayAutoscaler) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNATGatewayAutoscaler, oldNATGatewayAutoscaler, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNATGatewayAutoscalerSpecUpdate(&newNATGatewayAutoscaler.Spec, &oldNATGatewayAutoscaler.Spec, field.NewPath("spec"))...)

	return allErrs
}
