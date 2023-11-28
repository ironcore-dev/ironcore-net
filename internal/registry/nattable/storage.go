// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package nattable

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
)

type NATTableStorage struct {
	NATTable *REST
}

type REST struct {
	*genericregistry.Store
}

func NewStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (NATTableStorage, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.NATTable{}
		},
		NewListFunc: func() runtime.Object {
			return &core.NATTableList{}
		},
		PredicateFunc:             MatchNATTable,
		DefaultQualifiedResource:  core.Resource("nattables"),
		SingularQualifiedResource: core.Resource("nattable"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return NATTableStorage{}, err
	}

	return NATTableStorage{
		NATTable: &REST{store},
	}, nil
}
