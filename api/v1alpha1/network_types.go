// Copyright 2022 OnMetal authors
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkSpec struct {
	// +kubebuilder:validation:Maximum=16777215
	// +kubebuilder:validation:Minimum=0
	// VNI is the requested network vni.
	VNI int32 `json:"vni,omitempty"`
}

type NetworkConditionType string

const (
	NetworkAllocated NetworkConditionType = "Allocated"
)

type NetworkCondition struct {
	Type    NetworkConditionType   `json:"type"`
	Status  corev1.ConditionStatus `json:"status"`
	Reason  string                 `json:"reason,omitempty"`
	Message string                 `json:"message,omitempty"`
}

func NetworkConditionIndex(conditions []NetworkCondition, conditionType NetworkConditionType) int {
	for i, condition := range conditions {
		if condition.Type == conditionType {
			return i
		}
	}
	return -1
}

func SetNetworkCondition(conditions *[]NetworkCondition, condition NetworkCondition) {
	if idx := NetworkConditionIndex(*conditions, condition.Type); idx != -1 {
		(*conditions)[idx] = condition
	} else {
		*conditions = append(*conditions, condition)
	}
}

type NetworkStatus struct {
	// +kubebuilder:validation:Maximum=16777215
	// +kubebuilder:validation:Minimum=0
	// VNI is the allocated network vni.
	VNI int32 `json:"vni,omitempty"`

	Conditions []NetworkCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Request",type=string,JSONPath=`.spec.vni`
// +kubebuilder:printcolumn:name="VNI",type=string,JSONPath=`.status.vni`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[?(@.type == "Allocated")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// Network is the schema for the publicips API.
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkSpec   `json:"spec,omitempty"`
	Status NetworkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkList contains a list of Network.
type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Network `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Network{}, &NetworkList{})
}
