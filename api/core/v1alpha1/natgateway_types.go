// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NATGatewaySpec struct {
	// IPFamily is the IP family of the NAT gateway.
	IPFamily corev1.IPFamily `json:"ipFamily"`

	// NetworkRef references the network the NAT gateway is part of.
	NetworkRef corev1.LocalObjectReference `json:"networkRef"`

	// IPs specifies the IPs of the NAT gateway.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	IPs []NATGatewayIP `json:"ips,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`

	// PortsPerNetworkInterface specifies how many ports to allocate per network interface.
	PortsPerNetworkInterface int32 `json:"portsPerNetworkInterface"`
}

type NATGatewayIP struct {
	// Name is the semantic name of the NAT gateway IP.
	Name string `json:"name"`
	// IP specifies a specific IP to allocate. If empty, a random IP will be allocated if possible.
	IP net.IP `json:"ip,omitempty"`
}

type NATGatewayStatus struct {
	// UsedNATIPs is the number of NAT IPs in-use.
	UsedNATIPs int64 `json:"usedNATIPs,omitempty"`
	// RequestedNATIPs is the number of requested NAT IPs.
	RequestedNATIPs int64 `json:"requestedNATIPs,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// NATGateway is the schema for the natgateways API.
type NATGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NATGatewaySpec   `json:"spec,omitempty"`
	Status NATGatewayStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NATGatewayList contains a list of NATGateway.
type NATGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NATGateway `json:"items"`
}

func GetNATGatewayIPs(natGateway *NATGateway) []net.IP {
	res := make([]net.IP, len(natGateway.Spec.IPs))
	for i, ip := range natGateway.Spec.IPs {
		res[i] = ip.IP
	}
	return res
}
