// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkid

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
)

type NetworkIDStorage struct {
	NetworkID *REST
}

type REST struct {
	*genericregistry.Store
}

func NewStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (NetworkIDStorage, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.NetworkID{}
		},
		NewListFunc: func() runtime.Object {
			return &core.NetworkIDList{}
		},
		PredicateFunc:             MatchNetworkID,
		DefaultQualifiedResource:  core.Resource("networkids"),
		SingularQualifiedResource: core.Resource("networkid"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return NetworkIDStorage{}, err
	}

	return NetworkIDStorage{
		NetworkID: &REST{store},
	}, nil
}
