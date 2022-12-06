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

package controllers

import (
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func getApiNetPublicIPAllocationChangedPredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(event event.UpdateEvent) bool {
			oldAPINetPublicIP, newAPINetPublicIP := event.ObjectOld.(*onmetalapinetv1alpha1.PublicIP), event.ObjectNew.(*onmetalapinetv1alpha1.PublicIP)
			return oldAPINetPublicIP.IsAllocated() != newAPINetPublicIP.IsAllocated()
		},
	}
}
