// Copyright 2023 IronCore authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
