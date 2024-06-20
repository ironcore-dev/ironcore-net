// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	NetworkInterfaceProviderIDField = ".status.providerID"
	NetworkPolicyNetworkNameField   = "networkpolicy-network-name"
)

func SetupNetworkPolicyNetworkNameFieldIndexer(ctx context.Context, indexer client.FieldIndexer) error {
	return indexer.IndexField(ctx, &apinetv1alpha1.NetworkPolicy{}, NetworkPolicyNetworkNameField, func(obj client.Object) []string {
		networkPolicy := obj.(*apinetv1alpha1.NetworkPolicy)
		return []string{networkPolicy.Spec.NetworkRef.Name}
	})
}

type Object[O any] interface {
	client.Object
	*O
}

func ReconcileRequestsFromObjectStructSlice[O Object[OStruct], S ~[]OStruct, OStruct any](objs S) []reconcile.Request {
	res := make([]reconcile.Request, len(objs))
	for i := range objs {
		obj := O(&objs[i])
		res[i] = reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(obj),
		}
	}
	return res
}
