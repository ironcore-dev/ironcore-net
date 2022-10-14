// Copyright 2022 OnMetal authors
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

package controllers

import (
	"net/netip"

	commonv1alpha1 "github.com/onmetal/onmetal-api/apis/common/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func IPFamilyBitLen(ipFamily corev1.IPFamily) uint8 {
	switch ipFamily {
	case corev1.IPv4Protocol:
		return 32
	case corev1.IPv6Protocol:
		return 128
	default:
		return 0
	}
}

func CommonV1Alpha1IPsToNetIPAddrs(ips []commonv1alpha1.IP) []netip.Addr {
	res := make([]netip.Addr, len(ips))
	for i, ip := range ips {
		res[i] = ip.Addr
	}
	return res
}

func NetIPAddrsToCommonV1Alpha1IPs(addrs []netip.Addr) []commonv1alpha1.IP {
	res := make([]commonv1alpha1.IP, len(addrs))
	for i, addr := range addrs {
		res[i] = commonv1alpha1.IP{Addr: addr}
	}
	return res
}
