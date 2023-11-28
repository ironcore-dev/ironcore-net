// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

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
	metav1.TypeMeta
	metav1.ObjectMeta

	// IPs specifies how to NAT the IPs for the NAT gateway.
	IPs []NATIP
}

type NATIP struct {
	// IP is the IP to NAT.
	IP net.IP
	// Sections are the sections of the NATIP.
	Sections []NATIPSection
}

type NATTableIPTargetRef struct {
	// UID is the UID of the target.
	UID types.UID
	// Name is the name of the target.
	Name string
	// NodeRef references the node the destination network interface is on.
	NodeRef corev1.LocalObjectReference
}

type NATIPSection struct {
	// IP is the source IP.
	IP net.IP
	// Port is the start port of the section.
	Port int32
	// EndPort is the end port of the section
	EndPort int32
	// TargetRef references the entity having the IP.
	TargetRef *NATTableIPTargetRef
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NATTableList contains a list of NATTable.
type NATTableList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []NATTable
}
