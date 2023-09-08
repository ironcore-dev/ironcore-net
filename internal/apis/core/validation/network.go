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
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNetwork(network *core.Network) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(network, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)

	return allErrs
}

func ValidateNetworkUpdate(newNetwork, oldNetwork *core.Network) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNetwork, oldNetwork, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNetwork(newNetwork)...)

	return allErrs
}

func ValidateNetworkStatusUpdate(newNetwork, oldNetwork *core.Network) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNetwork, oldNetwork, field.NewPath("metadata"))...)

	return allErrs
}
