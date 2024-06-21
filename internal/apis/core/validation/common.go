// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"sort"

	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore-net/apimachinery/equality"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/validation"
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

var supportedProtocols = sets.New(
	corev1.ProtocolTCP,
	corev1.ProtocolUDP,
	corev1.ProtocolSCTP,
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

func ValidateImmutableField(newVal, oldVal interface{}, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if !equality.Semantic.DeepEqual(oldVal, newVal) {
		allErrs = append(allErrs, field.Forbidden(fldPath, validation.FieldImmutableErrorMsg))
	}
	return allErrs
}

func ValidateProtocol(protocol corev1.Protocol, fldPath *field.Path) field.ErrorList {
	return ValidateEnum(supportedProtocols, protocol, fldPath, "must specify protocol")
}
