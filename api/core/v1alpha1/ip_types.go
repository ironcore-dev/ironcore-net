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

package v1alpha1

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
	Type     IPType          `json:"type"`
	IPFamily corev1.IPFamily `json:"ipFamily,omitempty"`
	IP       net.IP          `json:"ip,omitempty"`
	ClaimRef *IPClaimRef     `json:"claimRef,omitempty"`
}

type IPClaimRef struct {
	Group    string    `json:"group,omitempty"`
	Resource string    `json:"resource,omitempty"`
	Name     string    `json:"name,omitempty"`
	UID      types.UID `json:"uid,omitempty"`
}

type IPStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// IP is the schema for the ips API.
type IP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPSpec   `json:"spec,omitempty"`
	Status IPStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IPList contains a list of IP.
type IPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IP `json:"items"`
}

func IsIPClaimedBy(ip *IP, claimer metav1.Object) bool {
	claimRef := ip.Spec.ClaimRef
	if claimRef == nil {
		return false
	}

	return claimRef.UID == claimer.GetUID()
}
