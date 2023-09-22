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
)

type LoadBalancerType string

const (
	LoadBalancerTypePublic   LoadBalancerType = "Public"
	LoadBalancerTypeInternal LoadBalancerType = "Internal"
)

type LoadBalancerSpec struct {
	// Type specifies the type of load balancer.
	Type LoadBalancerType

	// NetworkRef references the network the load balancer is part of.
	NetworkRef corev1.LocalObjectReference

	// IPs specifies the IPs of the load balancer.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	IPs []LoadBalancerIP

	// Ports are the ports the load balancer should allow.
	// If empty, the load balancer allows all ports.
	Ports []LoadBalancerPort

	// Selector selects all Instance that are managed by this daemon set.
	Selector *metav1.LabelSelector

	// Template is the instance template.
	Template InstanceTemplate
}

type LoadBalancerIP struct {
	// Name is the name of the load balancer IP.
	Name string
	// IPFamily is the IP family of the IP. Has to match IP if specified. If unspecified and IP is specified,
	// will be defaulted by using the IP family of IP.
	// If only IPFamily is specified, a random IP of that family will be allocated if possible.
	IPFamily corev1.IPFamily
	// IP specifies a specific IP to allocate. If empty, a random IP will be allocated if possible.
	IP net.IP
}

type LoadBalancerPort struct {
	// Protocol is the protocol the load balancer should allow.
	// If not specified, defaults to TCP.
	Protocol *corev1.Protocol
	// Port is the port to allow.
	Port int32
	// EndPort marks the end of the port range to allow.
	// If unspecified, only a single port, Port, will be allowed.
	EndPort *int32
}

type LoadBalancerStatus struct {
	// CollisionCount is used to construct names for IP addresses for the load balancer.
	CollisionCount *int32
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// LoadBalancer is the schema for the loadbalancers API.
type LoadBalancer struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   LoadBalancerSpec
	Status LoadBalancerStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadBalancerList contains a list of LoadBalancer.
type LoadBalancerList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []LoadBalancer
}

func LoadBalancerDaemonSetName(lbName string) string {
	return "lb-" + lbName
}

func GetLoadBalancerIPs(loadBalancer *LoadBalancer) []net.IP {
	res := make([]net.IP, len(loadBalancer.Spec.IPs))
	for i, ip := range loadBalancer.Spec.IPs {
		res[i] = ip.IP
	}
	return res
}
