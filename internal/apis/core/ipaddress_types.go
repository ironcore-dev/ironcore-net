// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type IPAddressSpec struct {
	IP       net.IP
	ClaimRef IPAddressClaimRef
}

type IPAddressClaimRef struct {
	Group     string
	Resource  string
	Namespace string
	Name      string
	UID       types.UID
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced

// IPAddress is the schema for the ipaddresses API.
type IPAddress struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec IPAddressSpec
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IPAddressList contains a list of IPAddress.
type IPAddressList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []IPAddress
}

func IsIPAddressClaimedBy(addr *IPAddress, claimer metav1.Object) bool {
	return addr.Spec.ClaimRef.UID == claimer.GetUID()
}
