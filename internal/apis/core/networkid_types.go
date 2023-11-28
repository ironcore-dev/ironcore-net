// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type NetworkIDSpec struct {
	ClaimRef NetworkIDClaimRef
}

type NetworkIDClaimRef struct {
	Group     string
	Resource  string
	Namespace string
	Name      string
	UID       types.UID
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced

// NetworkID is the schema for the networkids API.
type NetworkID struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec NetworkIDSpec
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkIDList contains a list of NetworkID.
type NetworkIDList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []NetworkID
}

func IsNetworkIDClaimedBy(networkID *NetworkID, claimer metav1.Object) bool {
	return networkID.Spec.ClaimRef.UID == claimer.GetUID()
}
