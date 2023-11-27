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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DaemonSetSpec struct {
	// Selector selects all Instance that are managed by this daemon set.
	Selector *metav1.LabelSelector

	// Template is the instance template.
	Template InstanceTemplate
}

type DaemonSetStatus struct {
	CollisionCount *int32
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// DaemonSet is the schema for the daemonsets API.
type DaemonSet struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   DaemonSetSpec
	Status DaemonSetStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DaemonSetList contains a list of DaemonSet.
type DaemonSetList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []DaemonSet
}
