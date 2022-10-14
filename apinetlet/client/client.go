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

package client

import (
	"context"

	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/apis/networking/v1alpha1"
	mcmeta "github.com/onmetal/poollet/multicluster/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const PublicIPSpecVirtualIPAllocatorField = "public-ip-spec-virtual-ip-allocator"

func IndexPublicIPSpecVirtualIPAllocatorField(ctx context.Context, clusterName string, indexer client.FieldIndexer) error {
	return indexer.IndexField(ctx, &onmetalapinetv1alpha1.PublicIP{}, PublicIPSpecVirtualIPAllocatorField, func(object client.Object) []string {
		publicIP := object.(*onmetalapinetv1alpha1.PublicIP)
		allocatorRef := publicIP.Spec.AllocatorRef
		if allocatorRef.ClusterName != clusterName {
			return nil
		}

		if allocatorRef.Group != networkingv1alpha1.SchemeGroupVersion.Group ||
			allocatorRef.Resource != "virtualips" {
			return nil
		}

		allocatorKey := client.ObjectKey{Namespace: allocatorRef.Namespace, Name: allocatorRef.Name}
		return []string{allocatorKey.String()}
	})
}

const VirtualIPRootAncestorUIDField = "virtual-ip-root-ancestor-uid"

func IndexVirtualIPRootAncestorField(ctx context.Context, indexer client.FieldIndexer) error {
	return indexer.IndexField(ctx, &networkingv1alpha1.VirtualIP{}, VirtualIPRootAncestorUIDField, func(object client.Object) []string {
		virtualIP := object.(*networkingv1alpha1.VirtualIP)
		ancestors := mcmeta.GetAncestors(virtualIP)
		if len(ancestors) == 0 {
			return nil
		}

		return []string{string(ancestors[0].UID)}
	})
}
