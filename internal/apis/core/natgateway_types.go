// Copyright 2022 IronCore authors
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
