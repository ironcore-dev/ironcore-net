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

package controllers

import (
	"fmt"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetv1alpha1ac "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/core/v1alpha1"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
)

func ipToAPINetIP(ip commonv1alpha1.IP) net.IP {
	return net.IP{Addr: ip.Addr}
}

func loadBalancerTypeToAPINetLoadBalancerType(loadBalancerType networkingv1alpha1.LoadBalancerType) (apinetv1alpha1.LoadBalancerType, error) {
	switch loadBalancerType {
	case networkingv1alpha1.LoadBalancerTypePublic:
		return apinetv1alpha1.LoadBalancerTypePublic, nil
	case networkingv1alpha1.LoadBalancerTypeInternal:
		return apinetv1alpha1.LoadBalancerTypeInternal, nil
	default:
		return "", fmt.Errorf("unknown load balancer type %q", loadBalancerType)
	}
}

func loadBalancerPortToAPINetLoadBalancerPortConfig(port networkingv1alpha1.LoadBalancerPort) *apinetv1alpha1ac.LoadBalancerPortApplyConfiguration {
	res := apinetv1alpha1ac.LoadBalancerPort().
		WithPort(port.Port)
	res.Protocol = port.Protocol
	res.EndPort = port.EndPort
	return res
}

func loadBalancerPortsToAPINetLoadBalancerPortConfigs(ports []networkingv1alpha1.LoadBalancerPort) []*apinetv1alpha1ac.LoadBalancerPortApplyConfiguration {
	return utilslices.Map(ports, loadBalancerPortToAPINetLoadBalancerPortConfig)
}

func apiNetIPToIP(ip net.IP) commonv1alpha1.IP {
	return commonv1alpha1.IP{Addr: ip.Addr}
}

func apiNetIPsToIPs(ips []net.IP) []commonv1alpha1.IP {
	return utilslices.Map(ips, apiNetIPToIP)
}

func apiNetIPPrefixToIPPrefix(prefix net.IPPrefix) commonv1alpha1.IPPrefix {
	return commonv1alpha1.IPPrefix{Prefix: prefix.Prefix}
}

func apiNetIPPrefixesToIPPrefixes(ips []net.IPPrefix) []commonv1alpha1.IPPrefix {
	return utilslices.Map(ips, apiNetIPPrefixToIPPrefix)
}

func apiNetNetworkInterfaceStateToNetworkInterfaceState(state apinetv1alpha1.NetworkInterfaceState) networkingv1alpha1.NetworkInterfaceState {
	switch state {
	case apinetv1alpha1.NetworkInterfaceStatePending:
		return networkingv1alpha1.NetworkInterfaceStatePending
	case apinetv1alpha1.NetworkInterfaceStateReady:
		return networkingv1alpha1.NetworkInterfaceStateAvailable
	case apinetv1alpha1.NetworkInterfaceStateError:
		return networkingv1alpha1.NetworkInterfaceStateError
	default:
		return networkingv1alpha1.NetworkInterfaceStatePending
	}
}
