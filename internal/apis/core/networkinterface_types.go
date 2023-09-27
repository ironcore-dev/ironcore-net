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

package core

import (
	"github.com/onmetal/onmetal-api-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type NetworkInterfaceSpec struct {
	// NodeRef is the node the network interface is hosted on.
	NodeRef corev1.LocalObjectReference

	// NetworkRef references the network that the network interface is in.
	NetworkRef corev1.LocalObjectReference

	// IPs are the internal IPs of the network interface.
	IPs []net.IP

	// Prefixes are additional prefixes to route to the network interface.
	Prefixes []net.IPPrefix

	// NATs specify the NAT of the network interface IP family.
	// Can only be set if there is no matching IP family in PublicIPs.
	NATs []NetworkInterfaceNAT

	// PublicIPs are the public IPs the network interface should have.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	PublicIPs []NetworkInterfacePublicIP
}

type NetworkInterfaceNAT struct {
	// IPFamily is the IP family of the handling NAT gateway.
	IPFamily corev1.IPFamily
	// ClaimRef references the NAT claim handling the network interface's NAT.
	ClaimRef NetworkInterfaceNATClaimRef
}

type NetworkInterfaceNATClaimRef struct {
	// Name is the name of the claiming NAT gateway.
	Name string
	// UID is the uid of the claiming NAT gateway.
	UID types.UID
}

type NetworkInterfacePublicIP struct {
	// Name is the semantic name of the network interface public IP.
	Name string
	// IPFamily is the IP family of the IP. Has to match IP if specified. If unspecified and IP is specified,
	// will be defaulted by using the IP family of IP.
	// If only IPFamily is specified, a random IP of that family will be allocated if possible.
	IPFamily corev1.IPFamily
	// IP specifies a specific IP to allocate. If empty, a random ephemeral IP will be allocated.
	IP net.IP
}

type NetworkInterfaceState string

const (
	// NetworkInterfaceStateReady is used for any NetworkInterface that is ready.
	NetworkInterfaceStateReady NetworkInterfaceState = "Ready"
	// NetworkInterfaceStatePending is used for any NetworkInterface that is in an intermediate state.
	NetworkInterfaceStatePending NetworkInterfaceState = "Pending"
	// NetworkInterfaceStateError is used for any NetworkInterface that is some error occurred.
	NetworkInterfaceStateError NetworkInterfaceState = "Error"
)

// PCIAddress is a PCI address.
type PCIAddress struct {
	Domain   string
	Bus      string
	Slot     string
	Function string
}

type NetworkInterfaceStatus struct {
	State      NetworkInterfaceState
	PCIAddress *PCIAddress
	Prefixes   []net.IPPrefix
	PublicIPs  []net.IP
	NATIPs     []net.IP
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// NetworkInterface is the schema for the networkinterfaces API.
type NetworkInterface struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   NetworkInterfaceSpec
	Status NetworkInterfaceStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkInterfaceList contains a list of NetworkInterface.
type NetworkInterfaceList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []NetworkInterface
}

func GetNetworkInterfaceNATClaimer(nic *NetworkInterface, ipFamily corev1.IPFamily) *NetworkInterfaceNATClaimRef {
	for _, nicNAT := range nic.Spec.NATs {
		if nicNAT.IPFamily == ipFamily {
			claimRef := nicNAT.ClaimRef
			return &claimRef
		}
	}
	return nil
}

func IsNetworkInterfaceNATClaimedBy(nic *NetworkInterface, claimer *NATGateway) bool {
	for _, nat := range nic.Spec.NATs {
		if nat.ClaimRef.UID == claimer.UID {
			return true
		}
	}
	return false
}

func GetNetworkInterfacePublicIPs(nic *NetworkInterface) []net.IP {
	res := make([]net.IP, len(nic.Spec.PublicIPs))
	for i, publicIP := range nic.Spec.PublicIPs {
		res[i] = publicIP.IP
	}
	return res
}
