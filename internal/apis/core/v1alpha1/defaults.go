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

package v1alpha1

import (
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
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
