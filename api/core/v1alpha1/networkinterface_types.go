// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type NetworkInterfaceSpec struct {
	// NodeRef is the node the network interface is hosted on.
	NodeRef corev1.LocalObjectReference `json:"nodeRef"`

	// NetworkRef references the network that the network interface is in.
	NetworkRef corev1.LocalObjectReference `json:"networkRef"`

	// IPs are the internal IPs of the network interface.
	IPs []net.IP `json:"ips,omitempty"`

	// Prefixes are additional prefixes to route to the network interface.
	Prefixes []net.IPPrefix `json:"prefixes,omitempty"`

	// NATs specify the NAT of the network interface IP family.
	// Can only be set if there is no matching IP family in PublicIPs.
	NATs []NetworkInterfaceNAT `json:"natGateways,omitempty"`

	// PublicIPs are the public IPs the network interface should have.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	PublicIPs []NetworkInterfacePublicIP `json:"publicIPs,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

type NetworkInterfaceNAT struct {
	// IPFamily is the IP family of the handling NAT gateway.
	IPFamily corev1.IPFamily `json:"ipFamily"`
	// ClaimRef references the NAT claim handling the network interface's NAT.
	ClaimRef NetworkInterfaceNATClaimRef `json:"claimRef"`
}

type NetworkInterfaceNATClaimRef struct {
	// Name is the name of the claiming NAT gateway.
	Name string `json:"name"`
	// UID is the uid of the claiming NAT gateway.
	UID types.UID `json:"uid"`
}

type NetworkInterfacePublicIP struct {
	// Name is the semantic name of the network interface public IP.
	Name string `json:"name"`
	// IPFamily is the IP family of the IP. Has to match IP if specified. If unspecified and IP is specified,
	// will be defaulted by using the IP family of IP.
	// If only IPFamily is specified, a random IP of that family will be allocated if possible.
	IPFamily corev1.IPFamily `json:"ipFamily,omitempty"`
	// IP specifies a specific IP to allocate. If empty, a random ephemeral IP will be allocated.
	IP net.IP `json:"ip,omitempty"`
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
	Domain   string `json:"domain,omitempty"`
	Bus      string `json:"bus,omitempty"`
	Slot     string `json:"slot,omitempty"`
	Function string `json:"function,omitempty"`
}

// TAPDevice is a TAP device.
type TAPDevice struct {
	// Name is the name of the TAP device.
	Name string `json:"name,omitempty"`
}

// NetworkInterfaceStatus defines the observed state of NetworkInterface.
type NetworkInterfaceStatus struct {
	// State is the state of the network interface.
	State NetworkInterfaceState `json:"state,omitempty"`
	// PCIAddress is the PCI address of the network interface.
	PCIAddress *PCIAddress `json:"pciAddress,omitempty"`
	// TAPDevice is the TAP device of the network interface.
	TAPDevice *TAPDevice `json:"tapDevice,omitempty"`
	// Prefixes are the prefixes of the network interface.
	Prefixes []net.IPPrefix `json:"prefixes,omitempty"`
	// PublicIPs are the public IPs of the network interface.
	PublicIPs []net.IP `json:"publicIPs,omitempty"`
	// NATIPs are the NAT IPs of the network interface.
	NATIPs []net.IP `json:"natIPs,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// NetworkInterface is the schema for the networkinterfaces API.
type NetworkInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkInterfaceSpec   `json:"spec,omitempty"`
	Status NetworkInterfaceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkInterfaceList contains a list of NetworkInterface.
type NetworkInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkInterface `json:"items"`
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
