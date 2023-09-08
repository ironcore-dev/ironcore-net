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

type NATGatewayAutoscalerSpec struct {
	// NATGatewayRef points to the target NATGateway to scale.
	NATGatewayRef corev1.LocalObjectReference `json:"natGatewayRef"`

	// MinPublicIPs is the minimum number of public IPs to allocate for a NAT Gateway.
	MinPublicIPs *int32 `json:"minPublicIPs,omitempty"`
	// MaxPublicIPs is the maximum number of public IPs to allocate for a NAT Gateway.
	MaxPublicIPs *int32 `json:"maxPublicIPs,omitempty"`
}

type NATGatewayAutoscalerStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// NATGatewayAutoscaler is the schema for the natgatewayautoscalers API.
type NATGatewayAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NATGatewayAutoscalerSpec   `json:"spec,omitempty"`
	Status NATGatewayAutoscalerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NATGatewayAutoscalerList contains a list of NATGatewayAutoscaler.
type NATGatewayAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NATGatewayAutoscaler `json:"items"`
}
