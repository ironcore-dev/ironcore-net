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

type PublicIPSpec struct {
	IPFamilies []corev1.IPFamily `json:"ipFamilies,omitempty"`
	IPs        []IP              `json:"ips,omitempty"`
}

type PublicIPConditionType string

const (
	PublicIPAllocated PublicIPConditionType = "Allocated"
)

type PublicIPCondition struct {
	Type    PublicIPConditionType  `json:"type"`
	Status  corev1.ConditionStatus `json:"status"`
	Reason  string                 `json:"reason,omitempty"`
	Message string                 `json:"message,omitempty"`
}

func PublicIPConditionIndex(conditions []PublicIPCondition, conditionType PublicIPConditionType) int {
	for i, condition := range conditions {
		if condition.Type == conditionType {
			return i
		}
	}
	return -1
}

func SetPublicIPCondition(conditions *[]PublicIPCondition, condition PublicIPCondition) {
	if idx := PublicIPConditionIndex(*conditions, condition.Type); idx != -1 {
		(*conditions)[idx] = condition
	} else {
		*conditions = append(*conditions, condition)
	}
}

type PublicIPStatus struct {
	// Conditions are the conditions of a PublicIP.
	Conditions []PublicIPCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="IPFamilies",type=string,JSONPath=`.status.ipFamilies`
// +kubebuilder:printcolumn:name="IPs",type=string,JSONPath=`.status.ips`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.conditions[?(@.type == "Allocated")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// PublicIP is the schema for the publicips API.
type PublicIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PublicIPSpec   `json:"spec,omitempty"`
	Status PublicIPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PublicIPList contains a list of PublicIP.
type PublicIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PublicIP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PublicIP{}, &PublicIPList{})
}
