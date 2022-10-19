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
	mcclient "github.com/onmetal/poollet/multicluster/client"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const PublicIPVirtualIPController = "public-ip-virtual-ip-controller"

func IndexPublicIPVirtualIPControllerField(
	ctx context.Context,
	indexer client.FieldIndexer,
	clusterName string,
	scheme *runtime.Scheme,
) error {
	return mcclient.IndexClusterTypeOwnerReferencesField(
		ctx,
		indexer,
		clusterName,
		&networkingv1alpha1.VirtualIP{},
		&onmetalapinetv1alpha1.PublicIP{},
		PublicIPVirtualIPController,
		scheme,
		true,
	)
}

const NetworkNetworkController = "network-network-controller"

func IndexNetworkNetworkControllerField(
	ctx context.Context,
	indexer client.FieldIndexer,
	clusterName string,
	scheme *runtime.Scheme,
) error {
	return mcclient.IndexClusterTypeOwnerReferencesField(
		ctx,
		indexer,
		clusterName,
		&networkingv1alpha1.Network{},
		&onmetalapinetv1alpha1.Network{},
		NetworkNetworkController,
		scheme,
		true,
	)
}
