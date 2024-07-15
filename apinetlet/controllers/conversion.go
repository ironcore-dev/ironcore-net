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
	apinetmetav1ac "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/meta/v1"

	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore/api/core/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func iPPrefixToAPINetIPPrefix(prefix commonv1alpha1.IPPrefix) *net.IPPrefix {
	return &net.IPPrefix{Prefix: prefix.Prefix}
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
			prefixStatus := []networkingv1alpha1.PeeringPrefixStatus{}
			if peering.State == apinetv1alpha1.NetworkPeeringStateReady {
				for _, peeringPrefix := range specPeerings[idx].Prefixes {
					prefixStatus = append(prefixStatus, networkingv1alpha1.PeeringPrefixStatus{
						Name:   peeringPrefix.Name,
						Prefix: (*commonv1alpha1.IPPrefix)(peeringPrefix.Prefix),
					})
				}
			}
			networkPeeringsStatus = append(networkPeeringsStatus, networkingv1alpha1.NetworkPeeringStatus{
				Name:     specPeerings[idx].Name,
				State:    networkingv1alpha1.NetworkPeeringState(peering.State),
				Prefixes: prefixStatus,
			})
		}
	}
	return networkPeeringsStatus
}

func networkPolicyTypesToAPINetNetworkPolicyTypes(policyTypes []networkingv1alpha1.PolicyType) ([]apinetv1alpha1.PolicyType, error) {
	var apiNetPolicyTypes []apinetv1alpha1.PolicyType
	for _, policyType := range policyTypes {
		switch policyType {
		case networkingv1alpha1.PolicyTypeIngress:
			apiNetPolicyTypes = append(apiNetPolicyTypes, apinetv1alpha1.PolicyTypeIngress)
		case networkingv1alpha1.PolicyTypeEgress:
			apiNetPolicyTypes = append(apiNetPolicyTypes, apinetv1alpha1.PolicyTypeEgress)
		default:
			return nil, fmt.Errorf("invalid policy type: %s", policyType)
		}
	}
	return apiNetPolicyTypes, nil
}

func translatePeers(peers []networkingv1alpha1.NetworkPolicyPeer) []apinetv1alpha1ac.NetworkPolicyPeerApplyConfiguration {
	var apiNetPeers []apinetv1alpha1ac.NetworkPolicyPeerApplyConfiguration
	for _, peer := range peers {
		apiNetPeer := apinetv1alpha1ac.NetworkPolicyPeer().
			WithObjectSelector(translateObjectSelector(peer.ObjectSelector)).
			WithIPBlock(translateIPBlock(peer.IPBlock))
		apiNetPeers = append(apiNetPeers, *apiNetPeer)
	}
	return apiNetPeers
}

func translateIPBlock(ipBlock *networkingv1alpha1.IPBlock) *apinetv1alpha1ac.IPBlockApplyConfiguration {
	if ipBlock == nil {
		return nil
	}

	var except []net.IPPrefix
	for _, prefix := range ipBlock.Except {
		except = append(except, net.IPPrefix(prefix))
	}

	return apinetv1alpha1ac.IPBlock().
		WithCIDR(net.IPPrefix(ipBlock.CIDR)).
		WithExcept(except...)
}

func translateObjectSelector(objSel *corev1alpha1.ObjectSelector) *apinetv1alpha1ac.ObjectSelectorApplyConfiguration {
	if objSel == nil {
		return nil
	}

	return apinetv1alpha1ac.ObjectSelector().
		WithKind(objSel.Kind).
		WithMatchLabels(objSel.MatchLabels).
		WithMatchExpressions(translateLabelSelectorRequirements(objSel.MatchExpressions)...)
}

func translateLabelSelectorRequirements(reqs []metav1.LabelSelectorRequirement) []*apinetmetav1ac.LabelSelectorRequirementApplyConfiguration {
	var translated []*apinetmetav1ac.LabelSelectorRequirementApplyConfiguration
	for _, req := range reqs {
		translated = append(translated, apinetmetav1ac.LabelSelectorRequirement().
			WithKey(req.Key).
			WithOperator(req.Operator).
			WithValues(req.Values...))
	}
	return translated
}

func translateLabelSelector(labelSelector metav1.LabelSelector) *apinetmetav1ac.LabelSelectorApplyConfiguration {
	return apinetmetav1ac.LabelSelector().
		WithMatchLabels(labelSelector.MatchLabels).
		WithMatchExpressions(translateLabelSelectorRequirements(labelSelector.MatchExpressions)...)
}

func translatePorts(ports []networkingv1alpha1.NetworkPolicyPort) []apinetv1alpha1ac.NetworkPolicyPortApplyConfiguration {
	var apiNetPorts []apinetv1alpha1ac.NetworkPolicyPortApplyConfiguration
	for _, port := range ports {
		apiNetPort := apinetv1alpha1ac.NetworkPolicyPortApplyConfiguration{
			Protocol: port.Protocol,
			Port:     &port.Port,
			EndPort:  port.EndPort,
		}
		apiNetPorts = append(apiNetPorts, apiNetPort)
	}
	return apiNetPorts
}
