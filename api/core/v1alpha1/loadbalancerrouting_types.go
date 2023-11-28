// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// LoadBalancerRouting is the schema for the loadbalancerroutings API.
type LoadBalancerRouting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Destinations []LoadBalancerDestination `json:"destinations,omitempty"`
}

// LoadBalancerDestination is the destination of the load balancer.
type LoadBalancerDestination struct {
	// IP is the target IP.
	IP net.IP `json:"ip"`
	// TargetRef is the target providing the destination.
	TargetRef *LoadBalancerTargetRef `json:"targetRef,omitempty"`
}

// LoadBalancerTargetRef is a load balancer target.
type LoadBalancerTargetRef struct {
	// UID is the UID of the target.
	UID types.UID `json:"uid"`
	// Name is the name of the target.
	Name string `json:"name"`
	// NodeRef references the node the destination network interface is on.
	NodeRef corev1.LocalObjectReference `json:"nodeRef"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadBalancerRoutingList contains a list of LoadBalancerRouting.
type LoadBalancerRoutingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadBalancerRouting `json:"items"`
}
