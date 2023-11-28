// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DaemonSetSpec struct {
	// Selector selects all Instance that are managed by this daemon set.
	Selector *metav1.LabelSelector `json:"nodeSelector,omitempty"`

	// Template is the instance template.
	Template InstanceTemplate `json:"template"`
}

type DaemonSetStatus struct {
	CollisionCount *int32 `json:"collisionCount,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// DaemonSet is the schema for the daemonsets API.
type DaemonSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DaemonSetSpec   `json:"spec,omitempty"`
	Status DaemonSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DaemonSetList contains a list of DaemonSet.
type DaemonSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DaemonSet `json:"items"`
}
