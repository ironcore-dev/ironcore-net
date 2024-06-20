// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
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

func getApiNetNetwork(ctx context.Context, c client.Client, apiNetNetworkKey client.ObjectKey) (*apinetv1alpha1.Network, error) {
	network := &apinetv1alpha1.Network{}
	if err := c.Get(ctx, apiNetNetworkKey, network); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("error getting apiNetNetwork %s: %w", apiNetNetworkKey.Name, err)
		}
		return nil, nil
	}
	return network, nil
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
