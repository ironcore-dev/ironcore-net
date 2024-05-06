// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkSpec struct {
	// ID is the ID of the network.
	ID string
	// Peerings are the network peerings with this network
	Peerings []NetworkPeering
}

// NetworkPeering defines a network peering with another network.
type NetworkPeering struct {
	// Name is the semantical name of the network peering.
	Name string
	// ID is the ID of the network to peer with.
	ID string
}

type NetworkStatus struct {
	// Peerings contains the states of the network peerings for the network.
	Peerings []NetworkPeeringStatus
}

// NetworkState is the state of a network.
// +enum
type NetworkState string

// NetworkPeeringState is the state a NetworkPeering can be in
type NetworkPeeringState string

const (
	// NetworkPeeringStatePending signals that the network peering is not applied.
	NetworkPeeringStatePending NetworkPeeringState = "Pending"
	// NetworkPeeringStateReady signals that the network peering is ready.
	NetworkPeeringStateReady NetworkPeeringState = "Ready"
	// NetworkPeeringStateError signals that the network peering is in error state.
	NetworkPeeringStateError NetworkPeeringState = "Error"
)

// NetworkPeeringStatus is the status of a network peering.
type NetworkPeeringStatus struct {
	// ID is the ID of network
	ID int32 `json:"id"`
	// State represents the network peering state
	State NetworkPeeringState
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// Network is the schema for the networks API.
type Network struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   NetworkSpec
	Status NetworkStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkList contains a list of Network.
type NetworkList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Network
}
