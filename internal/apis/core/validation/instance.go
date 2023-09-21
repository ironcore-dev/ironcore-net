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
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var InstanceTypes = sets.New(
	core.InstanceTypeLoadBalancer,
)

func ValidateInstanceType(typ core.InstanceType, fldPath *field.Path) field.ErrorList {
	return ValidateEnum(InstanceTypes, typ, fldPath, "must specify instance type")
}

func ValidateInstance(instance *core.Instance) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(instance, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateInstanceSpec(&instance.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateInstanceSpec(spec *core.InstanceSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, ValidateInstanceType(spec.Type, fldPath.Child("type"))...)

	switch spec.Type {
	case core.InstanceTypeLoadBalancer:
		allErrs = append(allErrs, ValidateLoadBalancerType(spec.LoadBalancerType, fldPath.Child("loadBalancerType"))...)
	}

	return allErrs
}

func ValidateInstanceUpdate(newInstance, oldInstance *core.Instance) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newInstance, oldInstance, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateInstance(newInstance)...)
	allErrs = append(allErrs, ValidateInstanceSpecUpdate(&newInstance.Spec, &oldInstance.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateInstanceSpecUpdate(newSpec, oldSpec *core.InstanceSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if oldSpec.NodeRef == nil {
		oldSpec.NodeRef = newSpec.NodeRef
	}

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.Type, oldSpec.Type, fldPath.Child("type"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.LoadBalancerType, oldSpec.LoadBalancerType, fldPath.Child("loadBalancerType"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.NetworkRef, oldSpec.NetworkRef, fldPath.Child("networkRef"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.NodeRef, oldSpec.NodeRef, fldPath.Child("nodeRef"))...)

	return allErrs
}

func ValidateInstanceStatusUpdate(newInstance, oldInstance *core.Instance) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newInstance, oldInstance, field.NewPath("metadata"))...)

	return allErrs
}
