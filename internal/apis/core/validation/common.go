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
	"fmt"
	"sort"

	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateEnum[E comparable](allowed sets.Set[E], value E, fldPath *field.Path, requiredDetail string) field.ErrorList {
	var allErrs field.ErrorList
	var zero E
	if value == zero && !allowed.Has(zero) {
		allErrs = append(allErrs, field.Required(fldPath, requiredDetail))
	} else if !allowed.Has(value) {
		validValues := make([]string, 0, allowed.Len())
		for item := range allowed {
			validValues = append(validValues, fmt.Sprint(item))
		}
		sort.Strings(validValues)

		allErrs = append(allErrs, field.NotSupported(fldPath, value, validValues))
	}
	return allErrs
}

var IPFamilies = sets.New(
	corev1.IPv4Protocol,
	corev1.IPv6Protocol,
)

func ValidateIPFamily(ipFamily corev1.IPFamily, fldPath *field.Path) field.ErrorList {
	return ValidateEnum(IPFamilies, ipFamily, fldPath, "must specify IP family")
}

func ValidateIPMatchesFamily(ip net.IP, ipFamily corev1.IPFamily, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if ip.Family() != ipFamily {
		allErrs = append(allErrs, field.Invalid(fldPath, ip, fmt.Sprintf("IP should have family %s", ipFamily)))
	}
	return allErrs
}
