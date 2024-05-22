// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"fmt"
	"slices"
	"strconv"

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

func apiNetNetworkPeeringsStatusToNetworkPeeringsStatus(peerings []apinetv1alpha1.NetworkPeeringStatus, specPeerings []apinetv1alpha1.NetworkPeering) []networkingv1alpha1.NetworkPeeringStatus {
	networkPeeringsStatus := []networkingv1alpha1.NetworkPeeringStatus{}
	for _, peering := range peerings {
		idx := slices.IndexFunc(specPeerings, func(specPeering apinetv1alpha1.NetworkPeering) bool {
			return specPeering.ID == strconv.Itoa(int(peering.ID))
		})
		if idx != -1 {
			networkPeeringsStatus = append(networkPeeringsStatus, networkingv1alpha1.NetworkPeeringStatus{
				Name:  specPeerings[idx].Name,
				State: networkingv1alpha1.NetworkPeeringState(peering.State),
			})
		}
	}
	return networkPeeringsStatus
}
