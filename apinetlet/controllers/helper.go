// Copyright 2022 IronCore authors
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
	"context"
	"fmt"

	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const fieldOwner = client.FieldOwner("ironcore.dev/apinetlet")

func getAPINetNetworkName(ctx context.Context, c client.Client, networkKey client.ObjectKey) (string, error) {
	network := &networkingv1alpha1.Network{}
	if err := c.Get(ctx, networkKey, network); err != nil {
		if !apierrors.IsNotFound(err) {
			return "", fmt.Errorf("error getting network %s for nat gateway: %w", networkKey.Name, err)
		}
		return "", nil
	}
	return string(network.UID), nil
}

func isPrefixAllocated(prefix *ipamv1alpha1.Prefix) bool {
	return prefix.Status.Phase == ipamv1alpha1.PrefixPhaseAllocated
}

func virtualIPClaimedPredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		vip := obj.(*networkingv1alpha1.VirtualIP)
		return vip.Spec.TargetRef != nil
	})
}

func virtualIPFreePredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		vip := obj.(*networkingv1alpha1.VirtualIP)
		return vip.Spec.TargetRef == nil
	})
}
