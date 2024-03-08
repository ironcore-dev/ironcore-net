// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkSpec struct {
	// ID is the ID of the network.
	ID string
	// PeeredIDs are the IDs of networks to peer with.
	PeeredIDs []string `json:"peeredIDs,omitempty"`
}

type NetworkStatus struct {
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
