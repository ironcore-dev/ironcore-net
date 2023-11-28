// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DaemonSetSpec struct {
	// Selector selects all Instance that are managed by this daemon set.
	Selector *metav1.LabelSelector

	// Template is the instance template.
	Template InstanceTemplate
}

type DaemonSetStatus struct {
	CollisionCount *int32
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// DaemonSet is the schema for the daemonsets API.
type DaemonSet struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   DaemonSetSpec
	Status DaemonSetStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DaemonSetList contains a list of DaemonSet.
type DaemonSetList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []DaemonSet
}
