// Copyright 2023 OnMetal authors
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

package ipaddress

import (
	"github.com/onmetal/onmetal-api-net/internal/apis/core"
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
