// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func SetDefaults_IPSpec(spec *v1alpha1.IPSpec) {
	if spec.IPFamily == "" && spec.IP.IsValid() {
		spec.IPFamily = spec.IP.Family()
	}
}

func SetDefaults_NetworkInterfacePublicIP(ip *v1alpha1.NetworkInterfacePublicIP) {
	if ip.IPFamily == "" && ip.IP.IsValid() {
		ip.IPFamily = ip.IP.Family()
	}
}

func SetDefaults_LoadBalancerIP(ip *v1alpha1.LoadBalancerIP) {
	if ip.IPFamily == "" && ip.IP.IsValid() {
		ip.IPFamily = ip.IP.Family()
	}
}
