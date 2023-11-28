// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type IPType string

const (
	IPTypePublic IPType = "Public"
)

type IPSpec struct {
	Type     IPType
	IPFamily corev1.IPFamily
	IP       net.IP
	ClaimRef *IPClaimRef
}

type IPClaimRef struct {
	Group    string
	Resource string
	Name     string
	UID      types.UID
}

type IPStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// IP is the schema for the ips API.
type IP struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   IPSpec
	Status IPStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IPList contains a list of IP.
type IPList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []IP
}

func IsIPClaimedBy(ip *IP, claimer metav1.Object) bool {
	claimRef := ip.Spec.ClaimRef
	if claimRef == nil {
		return false
	}

	return claimRef.UID == claimer.GetUID()
}
