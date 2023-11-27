// Copyright 2023 IronCore authors
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

package network

import (
	"context"

	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/internal/registry/network/networkidallocator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/util/dryrun"
	"k8s.io/klog/v2"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

type NetworkStorage struct {
	Network *REST
	Status  *StatusREST
}

type REST struct {
	*genericregistry.Store
	allocator networkidallocator.Interface
}

func NewStorage(
	scheme *runtime.Scheme,
	optsGetter generic.RESTOptionsGetter,
	allocator networkidallocator.Interface,
) (NetworkStorage, error) {
	strategy := NewStrategy(scheme)
	statusStrategy := NewStatusStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.Network{}
		},
		NewListFunc: func() runtime.Object {
			return &core.NetworkList{}
		},
		PredicateFunc:             MatchNetwork,
		DefaultQualifiedResource:  core.Resource("networks"),
		SingularQualifiedResource: core.Resource("network"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return NetworkStorage{}, err
	}

	genericStore := &REST{
		Store:     store,
		allocator: allocator,
	}
	store.BeginCreate = genericStore.beginCreate
	store.AfterDelete = genericStore.afterDelete

	statusStore := *store
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy

	return NetworkStorage{
		Network: genericStore,
		Status:  &StatusREST{&statusStore},
	}, nil
}

func (r *REST) beginCreate(ctx context.Context, obj runtime.Object, options *metav1.CreateOptions) (genericregistry.FinishFunc, error) {
	network := obj.(*core.Network)

	dryRun := dryrun.IsDryRun(options.DryRun)
	allocator := r.allocator
	if dryRun {
		allocator = allocator.DryRun()
	}

	id := network.Spec.ID
	if id != "" {
		if err := allocator.AllocateNetwork(network, id); err != nil {
			return nil, err
		}
	} else {
		newID, err := allocator.AllocateNextNetwork(network)
		if err != nil {
			return nil, err
		}

		id = newID
		network.Spec.ID = id
	}
	return func(ctx context.Context, success bool) {
		if success {
			klog.InfoS("Allocated network ID", "id", id)
			return
		}

		if err := allocator.Release(id); err != nil {
			klog.InfoS("Error releasing network ID", "id", id, "err", err)
		}
	}, nil
}

func (r *REST) afterDelete(obj runtime.Object, options *metav1.DeleteOptions) {
	network := obj.(*core.Network)

	if !dryrun.IsDryRun(options.DryRun) {
		id := network.Spec.ID
		if err := r.allocator.Release(id); err != nil {
			klog.InfoS("Error releasing network ID", "id", id, "err", err)
		}
	}
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &core.Network{}
}

func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

func (r *StatusREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

func (r *StatusREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return r.store.GetResetFields()
}

func (r *StatusREST) Destroy() {}
