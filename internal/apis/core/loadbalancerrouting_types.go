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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// LoadBalancerRouting is the schema for the loadbalancerroutings API.
type LoadBalancerRouting struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// IPs are the IPs the routing is for.
	IPs []net.IP

	Destinations []LoadBalancerDestination
}

// LoadBalancerDestination is the destination of the load balancer.
type LoadBalancerDestination struct {
	// IP is the target IP.
	IP net.IP
	// TargetRef is the target providing the destination.
	TargetRef *LoadBalancerTargetRef
}

// LoadBalancerTargetRef is a load balancer target.
type LoadBalancerTargetRef struct {
	// UID is the UID of the target.
	UID types.UID
	// Name is the name of the target.
	Name string
	// NodeRef references the node the destination network interface is on.
	NodeRef corev1.LocalObjectReference
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadBalancerRoutingList contains a list of LoadBalancerRouting.
type LoadBalancerRoutingList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []LoadBalancerRouting
}
