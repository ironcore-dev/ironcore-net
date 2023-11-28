// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NATGatewaySpec struct {
	// IPFamily is the IP family of the NAT gateway.
	IPFamily corev1.IPFamily

	// NetworkRef references the network the NAT gateway is part of.
	NetworkRef corev1.LocalObjectReference

	// IPs specifies the IPs of the NAT gateway.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	IPs []NATGatewayIP

	// PortsPerNetworkInterface specifies how many ports to allocate per network interface.
	PortsPerNetworkInterface int32
}

type NATGatewayIP struct {
	// Name is the semantic name of the NAT gateway IP.
	Name string
	// IP specifies a specific IP to allocate. If empty, a random IP will be allocated if possible.
	IP net.IP
}

type NATGatewayStatus struct {
	// UsedNATIPs is the number of NAT IPs in-use.
	UsedNATIPs int64
	// RequestedNATIPs is the number of requested NAT IPs.
	RequestedNATIPs int64
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// NATGateway is the schema for the natgateways API.
type NATGateway struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   NATGatewaySpec
	Status NATGatewayStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NATGatewayList contains a list of NATGateway.
type NATGatewayList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []NATGateway
}

func GetNATGatewayIPs(natGateway *NATGateway) []net.IP {
	res := make([]net.IP, len(natGateway.Spec.IPs))
	for i, ip := range natGateway.Spec.IPs {
		res[i] = ip.IP
	}
	return res
}
