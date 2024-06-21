// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"fmt"
	"net/netip"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore/utils/generic"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func ipPrefixToMetalnetPrefix(p net.IPPrefix) metalnetv1alpha1.IPPrefix {
	return metalnetv1alpha1.IPPrefix{Prefix: p.Prefix}
}

func ipPrefixesToMetalnetPrefixes(ps []net.IPPrefix) []metalnetv1alpha1.IPPrefix {
	return utilslices.Map(ps, ipPrefixToMetalnetPrefix)
}

func ipToMetalnetIP(ip net.IP) metalnetv1alpha1.IP {
	return metalnetv1alpha1.IP{Addr: ip.Addr}
}

func ipsToMetalnetIPs(ips []net.IP) []metalnetv1alpha1.IP {
	return utilslices.Map(ips, ipToMetalnetIP)
}

func ipToMetalnetIPPrefix(ip net.IP) metalnetv1alpha1.IPPrefix {
	return metalnetv1alpha1.IPPrefix{Prefix: netip.PrefixFrom(ip.Addr, ip.BitLen())}
}

func ipsToMetalnetIPPrefixes(ips []net.IP) []metalnetv1alpha1.IPPrefix {
	return utilslices.Map(ips, ipToMetalnetIPPrefix)
}

func ipsIPFamilies(ips []net.IP) []corev1.IPFamily {
	return utilslices.Map(ips, net.IP.Family)
}

func metalnetNetworkInterfaceStateToNetworkInterfaceStatus(mStatus metalnetv1alpha1.NetworkInterfaceState) apinetv1alpha1.NetworkInterfaceState {
	switch mStatus {
	case metalnetv1alpha1.NetworkInterfaceStatePending:
		return apinetv1alpha1.NetworkInterfaceStatePending
	case metalnetv1alpha1.NetworkInterfaceStateReady:
		return apinetv1alpha1.NetworkInterfaceStateReady
	case metalnetv1alpha1.NetworkInterfaceStateError:
		return apinetv1alpha1.NetworkInterfaceStateError
	default:
		return apinetv1alpha1.NetworkInterfaceStatePending
	}
}

func metalnetIPToIP(ip metalnetv1alpha1.IP) net.IP {
	return net.IP{Addr: ip.Addr}
}

func metalnetIPsToIPs(ips []metalnetv1alpha1.IP) []net.IP {
	return utilslices.Map(ips, metalnetIPToIP)
}

func metalnetIPPrefixToIPPrefix(prefix metalnetv1alpha1.IPPrefix) net.IPPrefix {
	return net.IPPrefix{Prefix: prefix.Prefix}
}

func metalnetIPPrefixesToIPPrefixes(prefixes []metalnetv1alpha1.IPPrefix) []net.IPPrefix {
	return utilslices.Map(prefixes, metalnetIPPrefixToIPPrefix)
}

func loadBalancerTypeToMetalnetLoadBalancerType(loadBalancerType apinetv1alpha1.LoadBalancerType) (metalnetv1alpha1.LoadBalancerType, error) {
	switch loadBalancerType {
	case apinetv1alpha1.LoadBalancerTypePublic:
		return metalnetv1alpha1.LoadBalancerTypePublic, nil
	case apinetv1alpha1.LoadBalancerTypeInternal:
		return metalnetv1alpha1.LoadBalancerTypeInternal, nil
	default:
		return "", fmt.Errorf("unknown load balancer type %q", loadBalancerType)
	}
}

func loadBalancerPortToMetalnetLoadBalancerPort(port apinetv1alpha1.LoadBalancerPort) metalnetv1alpha1.LBPort {
	protocol := generic.Deref(port.Protocol, corev1.ProtocolTCP)

	return metalnetv1alpha1.LBPort{
		Protocol: string(protocol),
		Port:     port.Port,
	}
}

func loadBalancerPortsToMetalnetLoadBalancerPorts(ports []apinetv1alpha1.LoadBalancerPort) []metalnetv1alpha1.LBPort {
	return utilslices.Map(ports, loadBalancerPortToMetalnetLoadBalancerPort)
}

// workaroundMetalnetNoIPv6VirtualIPSupportIPsToIP works around the missing public IPv6 support in metalnet
// by propagating only IPv4 addresses to metalnet.
// TODO: Remove this as soon as https://github.com/ironcore-dev/metalnet/issues/53 is resolved.
func workaroundMetalnetNoIPv6VirtualIPSupportIPsToIP(metalnetVirtualIPs []metalnetv1alpha1.IP) *metalnetv1alpha1.IP {
	for _, metalnetVirtualIP := range metalnetVirtualIPs {
		if metalnetVirtualIP.Is4() {
			ip := metalnetVirtualIP
			return &ip
		}
	}
	return nil
}

// workaroundMetalnetNoIPv6IPToIPs works around the missing public IPv6 support in metalnet by
// making a slice of the single virtual IP.
func workaroundMetalnetNoIPv6IPToIPs(metalnetVirtualIP *metalnetv1alpha1.IP) []metalnetv1alpha1.IP {
	if metalnetVirtualIP.IsZero() {
		return nil
	}
	return []metalnetv1alpha1.IP{*metalnetVirtualIP}
}

// workaroundMetalnetNoIPv6NATIPToIPs works around the missing public IPv6 support in metalnet by
// making a slice of the single virtual IP.
func workaroundMetalnetNoIPv6NATIPToIPs(natDetails *metalnetv1alpha1.NATDetails) []metalnetv1alpha1.IP {
	if natDetails == nil {
		return nil
	}
	return []metalnetv1alpha1.IP{*natDetails.IP}
}

// workaroundMetalnetNoIPv6NATDetailsToNATDetailsPointer works around the missing NAT IPv6 support in metalnet by
// returning only the IPv4 NAT details.
func workaroundMetalnetNoIPv6NATDetailsToNATDetailsPointer(natDetails []metalnetv1alpha1.NATDetails) *metalnetv1alpha1.NATDetails {
	for _, natDetails := range natDetails {
		if natDetails.IP.Is4() {
			details := natDetails
			return &details
		}
	}
	return nil
}

func metalnetNetworkPeeringsStatusToNetworkPeeringsStatus(peerings []metalnetv1alpha1.NetworkPeeringStatus) []apinetv1alpha1.NetworkPeeringStatus {
	return utilslices.Map(peerings, metalnetNetworkPeeringStatusToNetworkPeeringStatus)
}

func metalnetNetworkPeeringStatusToNetworkPeeringStatus(peering metalnetv1alpha1.NetworkPeeringStatus) apinetv1alpha1.NetworkPeeringStatus {
	return apinetv1alpha1.NetworkPeeringStatus{
		ID:    peering.ID,
		State: apinetv1alpha1.NetworkPeeringState(peering.State),
	}
}
