// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkinterface

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
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

type networkInterfaceIPAllocatorAccessor struct {
	core.NetworkInterface
}

func (acc *networkInterfaceIPAllocatorAccessor) GetRequests() []ipallocator.Request {
	return utilslices.Map(acc.Spec.PublicIPs, func(ip core.NetworkInterfacePublicIP) ipallocator.Request {
		return ipallocator.Request{
			IPFamily: ip.IPFamily,
			Addr:     ip.IP.Addr,
		}
	})
}

func (acc *networkInterfaceIPAllocatorAccessor) SetIP(idx int, addr netip.Addr) {
	acc.Spec.PublicIPs[idx].IP = net.IP{Addr: addr}
}

func GetNetworkInterfaceIPAllocatorAccessor(obj runtime.Object) (ipallocator.Accessor, error) {
	networkInterface, ok := obj.(*core.NetworkInterface)
	if !ok {
		return nil, fmt.Errorf("object %T is not a NetworkInterface", obj)
	}

	return &networkInterfaceIPAllocatorAccessor{
		*networkInterface,
	}, nil
}

type NetworkInterfaceStorage struct {
	NetworkInterface *REST
	Status           *StatusREST
}

type REST struct {
	*genericregistry.Store
	allocators *ipallocator.Allocators
}

func (r *REST) beginCreate(ctx context.Context, obj runtime.Object, opts *metav1.CreateOptions) (genericregistry.FinishFunc, error) {
	networkInterface := obj.(*core.NetworkInterface)

	dryRun := dryrun.IsDryRun(opts.DryRun)

	txn, err := r.allocators.AllocateCreate(networkInterface, dryRun)
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
	newNetworkInterface := obj.(*core.NetworkInterface)
	oldNetworkInterface := oldObj.(*core.NetworkInterface)

	dryRun := dryrun.IsDryRun(opts.DryRun)
	txn, err := r.allocators.AllocateUpdate(newNetworkInterface, oldNetworkInterface, dryRun)
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
	networkInterface := obj.(*core.NetworkInterface)

	dryRun := dryrun.IsDryRun(opts.DryRun)
	r.allocators.Release(networkInterface, dryRun)
}

func NewStorage(
	scheme *runtime.Scheme,
	optsGetter generic.RESTOptionsGetter,
	allocatorByFamily map[corev1.IPFamily]ipallocator.Interface,
) (NetworkInterfaceStorage, error) {
	strategy := NewStrategy(scheme)
	statusStrategy := NewStatusStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.NetworkInterface{}
		},
		NewListFunc: func() runtime.Object {
			return &core.NetworkInterfaceList{}
		},
		PredicateFunc:             MatchNetworkInterface,
		DefaultQualifiedResource:  core.Resource("networkinterfaces"),
		SingularQualifiedResource: core.Resource("networkinterface"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return NetworkInterfaceStorage{}, err
	}

	genericStore := &REST{
		Store: store,
		allocators: ipallocator.NewAllocators(
			allocatorByFamily,
			v1alpha1.SchemeGroupVersion,
			"NetworkInterface",
			"networkinterfaces",
			GetNetworkInterfaceIPAllocatorAccessor,
		),
	}
	store.BeginCreate = genericStore.beginCreate
	store.BeginUpdate = genericStore.beginUpdate
	store.AfterDelete = genericStore.afterDelete

	statusStore := *store
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy

	return NetworkInterfaceStorage{
		NetworkInterface: genericStore,
		Status:           &StatusREST{&statusStore},
	}, nil
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &core.NetworkInterface{}
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
