// Copyright 2023 OnMetal authors
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
	"github.com/onmetal/onmetal-api-net/apimachinery/api/net"
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
