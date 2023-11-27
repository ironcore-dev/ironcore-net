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

package natgateway

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/internal/registry/ipallocator"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/util/dryrun"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

type natGatewayIPAllocatorAccessor struct {
	core.NATGateway
}

func (acc *natGatewayIPAllocatorAccessor) GetRequests() []ipallocator.Request {
	return utilslices.Map(acc.Spec.IPs, func(ip core.NATGatewayIP) ipallocator.Request {
		return ipallocator.Request{
			IPFamily: acc.Spec.IPFamily,
			Addr:     ip.IP.Addr,
		}
	})
}

func (acc *natGatewayIPAllocatorAccessor) SetIP(idx int, addr netip.Addr) {
	acc.Spec.IPs[idx].IP = net.IP{Addr: addr}
}

func GetNATGatewayIPAllocatorAccessor(obj runtime.Object) (ipallocator.Accessor, error) {
	natGateway, ok := obj.(*core.NATGateway)
	if !ok {
		return nil, fmt.Errorf("object %T is not a NATGateway", obj)
	}

	return &natGatewayIPAllocatorAccessor{
		*natGateway,
	}, nil
}

type NATGatewayStorage struct {
	NATGateway *REST
	Status     *StatusREST
}

type REST struct {
	*genericregistry.Store
	allocators *ipallocator.Allocators
}

func (r *REST) beginCreate(ctx context.Context, obj runtime.Object, opts *metav1.CreateOptions) (genericregistry.FinishFunc, error) {
	natGateway := obj.(*core.NATGateway)

	dryRun := dryrun.IsDryRun(opts.DryRun)

	txn, err := r.allocators.AllocateCreate(natGateway, dryRun)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, success bool) {
		if success {
			txn.Commit()
		} else {
			txn.Revert()
		}
	}, nil
}

func (r *REST) beginUpdate(ctx context.Context, obj, oldObj runtime.Object, opts *metav1.UpdateOptions) (genericregistry.FinishFunc, error) {
	newNATGateway := obj.(*core.NATGateway)
	oldNATGateway := oldObj.(*core.NATGateway)

	dryRun := dryrun.IsDryRun(opts.DryRun)
	txn, err := r.allocators.AllocateUpdate(newNATGateway, oldNATGateway, dryRun)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, success bool) {
		if success {
			txn.Commit()
		} else {
			txn.Revert()
		}
	}, nil
}

func (r *REST) afterDelete(obj runtime.Object, opts *metav1.DeleteOptions) {
	natGateway := obj.(*core.NATGateway)

	dryRun := dryrun.IsDryRun(opts.DryRun)
	r.allocators.Release(natGateway, dryRun)
}

func NewStorage(
	scheme *runtime.Scheme,
	optsGetter generic.RESTOptionsGetter,
	allocByIPFamily map[corev1.IPFamily]ipallocator.Interface,
) (NATGatewayStorage, error) {
	strategy := NewStrategy(scheme)
	statusStrategy := NewStatusStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.NATGateway{}
		},
		NewListFunc: func() runtime.Object {
			return &core.NATGatewayList{}
		},
		PredicateFunc:             MatchNATGateway,
		DefaultQualifiedResource:  core.Resource("natgateways"),
		SingularQualifiedResource: core.Resource("natgateway"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return NATGatewayStorage{}, err
	}

	genericStore := &REST{
		Store: store,
		allocators: ipallocator.NewAllocators(
			allocByIPFamily,
			v1alpha1.SchemeGroupVersion,
			"NATGateway",
			"natgateways",
			GetNATGatewayIPAllocatorAccessor,
		),
	}
	store.BeginCreate = genericStore.beginCreate
	store.BeginUpdate = genericStore.beginUpdate
	store.AfterDelete = genericStore.afterDelete

	statusStore := *store
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy

	return NATGatewayStorage{
		NATGateway: genericStore,
		Status:     &StatusREST{&statusStore},
	}, nil
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &core.NATGateway{}
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
