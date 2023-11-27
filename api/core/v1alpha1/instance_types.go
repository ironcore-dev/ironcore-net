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
)

type InstanceType string

const (
	InstanceTypeLoadBalancer InstanceType = "LoadBalancer"
)

type InstanceSpec struct {
	// Type specifies the InstanceType to deploy.
	Type InstanceType `json:"type"`

	// LoadBalancerType is the load balancer type this instance is for.
	LoadBalancerType LoadBalancerType `json:"loadBalancerType"`

	// NetworkRef references the network the instance is on.
	NetworkRef corev1.LocalObjectReference `json:"networkRef"`

	// IPs are the IPs of the instance.
	IPs []net.IP `json:"ips,omitempty"`

	// LoadBalancerPorts are the load balancer ports of this instance.
	LoadBalancerPorts []LoadBalancerPort `json:"loadBalancerPorts,omitempty"`

	// Affinity are affinity constraints.
	Affinity *Affinity `json:"affinity,omitempty"`

	// TopologySpreadConstraints describes how a group of instances ought to spread across topology
	// domains. Scheduler will schedule instances in a way which abides by the constraints.
	// All topologySpreadConstraints are ANDed.
	TopologySpreadConstraints []TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// NodeRef references the node hosting the load balancer instance.
	// Will be set by the scheduler if empty.
	NodeRef *corev1.LocalObjectReference `json:"nodeRef,omitempty"`
}

type Affinity struct {
	NodeAffinity         *NodeAffinity         `json:"nodeAffinity,omitempty"`
	InstanceAntiAffinity *InstanceAntiAffinity `json:"instanceAntiAffinity,omitempty"`
}

type InstanceAntiAffinity struct {
	// RequiredDuringSchedulingIgnoredDuringExecution specifies anti-affinity requirements at
	// scheduling time, that, if not met, will cause the instance not be scheduled onto the node.
	// When there are multiple elements, the lists of nodes corresponding to each
	// instanceAffinityTerm are intersected, i.e. all terms must be satisfied.
	RequiredDuringSchedulingIgnoredDuringExecution []InstanceAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// InstanceAffinityTerm defines a set of instances (namely those matching the labelSelector that this instance should be
// co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose
// value of the label with key <topologyKey> matches that of any node on which a instance of the set of instances is running.
type InstanceAffinityTerm struct {
	// LabelSelector over a set of resources, in this case instances.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	// TopologyKey indicates that this instance should be co-located (affinity) or not co-located (anti-affinity)
	// with the instances matching the labelSelector, where co-located is defined as running on a
	// node whose value of the label with key topologyKey matches that of any node on which any of the
	// selected instances is running.
	// Empty topologyKey is not allowed.
	TopologyKey string `json:"topologyKey"`
}

type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution *NodeSelector `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeSelector represents the union of the results of one or more queries
// over a set of nodes; that is, it represents the OR of the selectors represented
// by the node selector terms.
type NodeSelector struct {
	// Required. A list of node selector terms. The terms are ORed.
	NodeSelectorTerms []NodeSelectorTerm `json:"nodeSelectorTerms"`
}

// NodeSelectorTerm matches no objects if it's empty. The requirements of the selector are ANDed.
type NodeSelectorTerm struct {
	// MatchExpressions matches nodes by the label selector requirements.
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty"`
	// MatchFields matches the nodes by their fields.
	MatchFields []NodeSelectorRequirement `json:"matchFields,omitempty"`
}

// NodeSelectorRequirement is a requirement for a selector. It's a combination of the key to match, the operator
// to match with, and zero to n values, depending on the operator.
type NodeSelectorRequirement struct {
	// Key is the key the selector applies to.
	Key string `json:"key"`
	// Operator represents the key's relationship to the values.
	// Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
	Operator NodeSelectorOperator `json:"operator"`
	// Values are the values to relate the key to via the operator.
	Values []string `json:"values,omitempty"`
}

// NodeSelectorOperator is the set of operators that can be used in
// a node selector requirement.
type NodeSelectorOperator string

const (
	NodeSelectorOpIn           NodeSelectorOperator = "In"
	NodeSelectorOpNotIn        NodeSelectorOperator = "NotIn"
	NodeSelectorOpExists       NodeSelectorOperator = "Exists"
	NodeSelectorOpDoesNotExist NodeSelectorOperator = "DoesNotExist"
	NodeSelectorOpGt           NodeSelectorOperator = "Gt"
	NodeSelectorOpLt           NodeSelectorOperator = "Lt"
)

type UnsatisfiableConstraintAction string

const (
	// DoNotSchedule instructs the scheduler not to schedule the instance
	// when constraints are not satisfied.
	DoNotSchedule UnsatisfiableConstraintAction = "DoNotSchedule"
)

// TopologySpreadConstraint specifies how to spread matching instances among the given topology.
type TopologySpreadConstraint struct {
	// MaxSkew describes the degree to which instances may be unevenly distributed.
	// When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference
	// between the number of matching instances in the target topology and the global minimum.
	// The global minimum is the minimum number of matching instances in an eligible domain
	// or zero if the number of eligible domains is less than MinDomains.
	MaxSkew int32 `json:"maxSkew"`
	// TopologyKey is the key of node labels. Nodes that have a label with this key
	// and identical values are considered to be in the same topology.
	// We consider each <key, value> as a "bucket", and try to put balanced number
	// of instances into each bucket.
	// We define a domain as a particular instance of a topology.
	// Also, we define an eligible domain as a domain whose nodes meet the requirements of
	// nodeAffinityPolicy and nodeTaintsPolicy.
	TopologyKey string `json:"topologyKey"`
	// WhenUnsatisfiable indicates how to deal with a instance if it doesn't satisfy
	// the spread constraint.
	// - DoNotSchedule (default) tells the scheduler not to schedule it.
	// - ScheduleAnyway tells the scheduler to schedule the instance in any location,
	//   but giving higher precedence to topologies that would help reduce the
	//   skew.
	WhenUnsatisfiable UnsatisfiableConstraintAction `json:"whenUnsatisfiable"`
	// LabelSelector is used to find matching instances.
	// Instances that match this label selector are counted to determine the number of instances
	// in their corresponding topology domain.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

type InstanceStatus struct {
	IPs            []net.IP `json:"ips,omitempty"`
	CollisionCount *int32   `json:"collisionCount,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// Instance is the schema for the instances API.
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceSpec   `json:"spec,omitempty"`
	Status InstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstanceList contains a list of Instance.
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

type InstanceTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstanceSpec `json:"spec,omitempty"`
}
