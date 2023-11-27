// Copyright 2023 IronCore authors
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
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var IPTypes = sets.New(
	core.IPTypePublic,
)

func ValidateIPType(ipType core.IPType, fldPath *field.Path) field.ErrorList {
	return ValidateEnum(IPTypes, ipType, fldPath, "must specify IP type")
}

func ValidateIP(ip *core.IP) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(ip, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateIPSpec(&ip.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateIPSpec(spec *core.IPSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, ValidateIPType(spec.Type, fldPath.Child("type"))...)
	allErrs = append(allErrs, ValidateIPFamily(spec.IPFamily, fldPath.Child("ipFamily"))...)
	if spec.IP.IsValid() {
		allErrs = append(allErrs, ValidateIPMatchesFamily(spec.IP, spec.IPFamily, fldPath.Child("ip"))...)
	}

	return allErrs
}

func ValidateIPUpdate(newIP, oldIP *core.IP) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newIP, oldIP, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateIP(newIP)...)
	allErrs = append(allErrs, ValidateIPSpecUpdate(&newIP.Spec, &oldIP.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateIPSpecUpdate(newSpec, oldSpec *core.IPSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.Type, oldSpec.Type, fldPath.Child("type"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.IPFamily, oldSpec.IPFamily, fldPath.Child("ipFamily"))...)
	if newSpec.IP != oldSpec.IP {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ip"), newSpec.IP, validation.FieldImmutableErrorMsg))
	}

	return allErrs
}

func ValidateIPStatusUpdate(newIP, oldIP *core.IP) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newIP, oldIP, field.NewPath("metadata"))...)

	return allErrs
}
