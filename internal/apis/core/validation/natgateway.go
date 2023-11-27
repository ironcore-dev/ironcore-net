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
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateNATGateway(natGateway *core.NATGateway) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessor(natGateway, true, validation.NameIsDNSLabel, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNATGatewaySpec(&natGateway.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNATGatewaySpec(spec *core.NATGatewaySpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, ValidateIPFamily(spec.IPFamily, fldPath.Child("ipFamily"))...)

	for i, ip := range spec.IPs {
		fldPath := fldPath.Child("ips").Index(i)
		if ip.IP.IsValid() {
			allErrs = append(allErrs, ValidateIPMatchesFamily(ip.IP, spec.IPFamily, fldPath.Child("ip"))...)
		}
	}

	return allErrs
}

func ValidateNATGatewayUpdate(newNATGateway, oldNATGateway *core.NATGateway) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNATGateway, oldNATGateway, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateNATGateway(newNATGateway)...)
	allErrs = append(allErrs, ValidateNATGatewaySpecUpdate(&newNATGateway.Spec, &oldNATGateway.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateNATGatewaySpecUpdate(newSpec, oldSpec *core.NATGatewaySpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.IPFamily, oldSpec.IPFamily, fldPath.Child("ipFamily"))...)
	allErrs = append(allErrs, validation.ValidateImmutableField(newSpec.NetworkRef, oldSpec.NetworkRef, fldPath.Child("networkRef"))...)

	return allErrs
}

func ValidateNATGatewayStatusUpdate(newNATGateway, oldNATGateway *core.NATGateway) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validation.ValidateObjectMetaAccessorUpdate(newNATGateway, oldNATGateway, field.NewPath("metadata"))...)

	return allErrs
}
