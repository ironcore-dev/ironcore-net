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
