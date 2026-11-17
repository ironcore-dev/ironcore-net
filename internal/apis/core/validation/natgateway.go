// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/internal/natgateway"
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

	allErrs = append(allErrs, ValidatePortsPerNetworkInterface(spec.PortsPerNetworkInterface, fldPath.Child("portsPerNetworkInterface"))...)

	return allErrs
}

func ValidatePortsPerNetworkInterface(portsPerNetworkInterface int32, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if !natgateway.IsValidPortsPerNetworkInterface(portsPerNetworkInterface) {
		allErrs = append(allErrs, field.Invalid(
			fldPath,
			portsPerNetworkInterface,
			fmt.Sprintf("must be between %d and %d", natgateway.MinPortsPerNetworkInterface, natgateway.MaxPortsPerNetworkInterface),
		))
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
