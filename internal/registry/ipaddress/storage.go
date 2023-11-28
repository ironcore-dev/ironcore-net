// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ipaddress

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
)

type IPAddressStorage struct {
	IPAddress *REST
}

type REST struct {
	*genericregistry.Store
}

func NewStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (IPAddressStorage, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.IPAddress{}
		},
		NewListFunc: func() runtime.Object {
			return &core.IPAddressList{}
		},
		PredicateFunc:             MatchIPAddress,
		DefaultQualifiedResource:  core.Resource("ipaddresses"),
		SingularQualifiedResource: core.Resource("ipaddress"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return IPAddressStorage{}, err
	}

	return IPAddressStorage{
		IPAddress: &REST{store},
	}, nil
}
