// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package loadbalancer

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

type loadBalancerIPAllocatorAccessor struct {
	core.LoadBalancer
}

func (l *loadBalancerIPAllocatorAccessor) GetRequests() []ipallocator.Request {
	return utilslices.Map(l.Spec.IPs, func(ip core.LoadBalancerIP) ipallocator.Request {
		return ipallocator.Request{
			IPFamily: ip.IPFamily,
			Addr:     ip.IP.Addr,
		}
	})
}

func (l *loadBalancerIPAllocatorAccessor) SetIP(idx int, addr netip.Addr) {
	l.Spec.IPs[idx].IP = net.IP{Addr: addr}
}

func GetLoadBalancerIPAllocatorAccessor(obj runtime.Object) (ipallocator.Accessor, error) {
	loadBalancer, ok := obj.(*core.LoadBalancer)
	if !ok {
		return nil, fmt.Errorf("object %T is not a LoadBalancer", obj)
	}

	return &loadBalancerIPAllocatorAccessor{
		*loadBalancer,
	}, nil
}

type LoadBalancerStorage struct {
	LoadBalancer *REST
	Status       *StatusREST
}

type REST struct {
	*genericregistry.Store
	allocators *ipallocator.Allocators
}

func (r *REST) beginCreate(ctx context.Context, obj runtime.Object, opts *metav1.CreateOptions) (genericregistry.FinishFunc, error) {
	loadBalancer := obj.(*core.LoadBalancer)

	if loadBalancer.Spec.Type != core.LoadBalancerTypePublic {
		return func(ctx context.Context, success bool) {}, nil
	}

	dryRun := dryrun.IsDryRun(opts.DryRun)

	txn, err := r.allocators.AllocateCreate(loadBalancer, dryRun)
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
	newLoadBalancer := obj.(*core.LoadBalancer)
	oldLoadBalancer := oldObj.(*core.LoadBalancer)

	if newLoadBalancer.Spec.Type != core.LoadBalancerTypePublic {
		return func(ctx context.Context, success bool) {}, nil
	}

	dryRun := dryrun.IsDryRun(opts.DryRun)
	txn, err := r.allocators.AllocateUpdate(newLoadBalancer, oldLoadBalancer, dryRun)
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
	loadBalancer := obj.(*core.LoadBalancer)

	if loadBalancer.Spec.Type != core.LoadBalancerTypePublic {
		return
	}

	dryRun := dryrun.IsDryRun(opts.DryRun)
	r.allocators.Release(loadBalancer, dryRun)
}

func NewStorage(
	scheme *runtime.Scheme,
	optsGetter generic.RESTOptionsGetter,
	allocatorByFamily map[corev1.IPFamily]ipallocator.Interface,
) (LoadBalancerStorage, error) {
	strategy := NewStrategy(scheme)
	statusStrategy := NewStatusStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.LoadBalancer{}
		},
		NewListFunc: func() runtime.Object {
			return &core.LoadBalancerList{}
		},
		PredicateFunc:             MatchLoadBalancer,
		DefaultQualifiedResource:  core.Resource("loadbalancers"),
		SingularQualifiedResource: core.Resource("loadbalancer"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return LoadBalancerStorage{}, err
	}

	genericStore := &REST{
		Store: store,
		allocators: ipallocator.NewAllocators(
			allocatorByFamily,
			v1alpha1.SchemeGroupVersion,
			"LoadBalancer",
			"loadbalancers",
			GetLoadBalancerIPAllocatorAccessor,
		),
	}
	store.BeginCreate = genericStore.beginCreate
	store.BeginUpdate = genericStore.beginUpdate
	store.AfterDelete = genericStore.afterDelete

	statusStore := *store
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy

	return LoadBalancerStorage{
		LoadBalancer: genericStore,
		Status:       &StatusREST{&statusStore},
	}, nil
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &core.LoadBalancer{}
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
