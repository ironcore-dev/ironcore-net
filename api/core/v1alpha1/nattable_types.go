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

// NATTable is the schema for the nattables API.
type NATTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// IPs specifies how to NAT the IPs for the NAT gateway.
	IPs []NATIP `json:"ips,omitempty"`
}

type NATIP struct {
	// IP is the IP to NAT.
	IP net.IP `json:"ip"`
	// Sections are the sections of the NATIP.
	Sections []NATIPSection `json:"sections,omitempty"`
}

type NATTableIPTargetRef struct {
	// UID is the UID of the target.
	UID types.UID `json:"uid"`
	// Name is the name of the target.
	Name string `json:"name"`
	// NodeRef references the node the destination network interface is on.
	NodeRef corev1.LocalObjectReference `json:"nodeRef"`
}

type NATIPSection struct {
	// IP is the source IP.
	IP net.IP `json:"ip"`
	// Port is the start port of the section.
	Port int32 `json:"port"`
	// EndPort is the end port of the section
	EndPort int32 `json:"endPort"`
	// TargetRef references the entity having the IP.
	TargetRef *NATTableIPTargetRef `json:"targetRef,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NATTableList contains a list of NATTable.
type NATTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NATTable `json:"items"`
}
