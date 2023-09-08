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
	"github.com/onmetal/onmetal-api-net/networkid"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNetworkIDName(name string, prefix bool) []string {
	var errs []string
	vni, err := networkid.ParseVNI(name)
	if err != nil {
		errs = append(errs, err.Error())
	} else if networkid.EncodeVNI(vni) != name {
		errs = append(errs, "not a valid VNI in canonical format")
	}
	return errs
}

func ValidateNetworkID(networkID *core.NetworkID) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(networkID, false, ValidateNetworkIDName, field.NewPath("metadata"))...)

	return allErrs
}

func ValidateNetworkIDUpdate(newNetworkID, oldNetworkID *core.NetworkID) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNetworkID, oldNetworkID, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNetworkID(newNetworkID)...)

	return allErrs
}
