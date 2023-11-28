// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type NetworkIDSpec struct {
	ClaimRef NetworkIDClaimRef `json:"claimRef"`
}

type NetworkIDClaimRef struct {
	Group     string    `json:"group,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Namespace string    `json:"namespace,omitempty"`
	Name      string    `json:"name,omitempty"`
	UID       types.UID `json:"uid,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced

// NetworkID is the schema for the networkids API.
type NetworkID struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkIDSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkIDList contains a list of NetworkID.
type NetworkIDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkID `json:"items"`
}

func IsNetworkIDClaimedBy(networkID *NetworkID, claimer metav1.Object) bool {
	return networkID.Spec.ClaimRef.UID == claimer.GetUID()
}
