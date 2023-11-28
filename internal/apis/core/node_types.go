// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeSpec struct {
}

type NodeStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced

// Node is the schema for the nodes API.
type Node struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   NodeSpec
	Status NodeStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeList contains a list of Node.
type NodeList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Node
}
