// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// NetworkPolicyRule is the schema for the networkpolicyrules API.
type NetworkPolicyRule struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// NetworkRef is the network the load balancer is assigned to.
	NetworkRef LocalUIDReference
	// Targets are the targets of the network policy.
	Targets []TargetNetworkInterface
	// Priority is an optional field that specifies the order in which the policy is applied.
	Priority *int32
	// IngressRules are the ingress rules.
	IngressRules []Rule
	// EgressRules are the egress rules.
	EgressRules []Rule
}

// TargetNetworkInterface is the target of the network policy.
type TargetNetworkInterface struct {
	// IP is the IP address of the target network interface.
	IP net.IP
	// TargetRef is the target providing the destination.
	TargetRef *LocalUIDReference
}

type Rule struct {
	// CIDRBlock specifies the CIDR block from which network traffic may come or go.
	CIDRBlock []IPBlock
	// ObjectIPs are the object IPs the rule applies to.
	ObjectIPs []ObjectIP
	// NetworkPolicyPorts are the protocol type and ports.
	NetworkPolicyPorts []NetworkPolicyPort
}

type ObjectIP struct {
	// IPFamily is the IPFamily of the prefix.
	// If unset but Prefix is set, this can be inferred.
	IPFamily corev1.IPFamily
	// Prefix is the prefix of the IP.
	Prefix net.IPPrefix
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkPolicyRulesList contains a list of NetworkPolicyRule.
type NetworkPolicyRuleList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []NetworkPolicyRule
}
