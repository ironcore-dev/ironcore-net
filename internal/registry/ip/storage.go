// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ip

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/internal/registry/ip/ipaddressallocator"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	apisrvstorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/util/dryrun"
	"k8s.io/klog/v2"
)

type IPStorage struct {
	IP *REST
}

type REST struct {
	*genericregistry.Store
	allocatorByFamily map[corev1.IPFamily]ipaddressallocator.Interface
}

func (r *REST) beginCreate(ctx context.Context, obj runtime.Object, opts *metav1.CreateOptions) (genericregistry.FinishFunc, error) {
	ip := obj.(*core.IP)
	log := klog.FromContext(ctx).WithValues("Object", klog.KObj(ip))
	if address := ip.Spec.IP; address.IsValid() {
		log = log.WithValues("Address", address)
	}
	ctx = klog.NewContext(ctx, log)

	alloc, ok := r.allocatorByFamily[ip.Spec.IPFamily]
	if !ok {
		return nil, fmt.Errorf("cannot allocate IPs of family %s", ip.Spec.IPFamily)
	}
	if dryrun.IsDryRun(opts.DryRun) {
		alloc = alloc.DryRun()
	}

	claimRef := v1alpha1.IPAddressClaimRef{
		Group:     v1alpha1.GroupName,
		Resource:  "ips",
		Namespace: ip.Namespace,
		Name:      ip.Name,
		UID:       ip.UID,
	}

	addr := ip.Spec.IP
	if addr.IsValid() {
		if err := alloc.Allocate(ctx, claimRef, addr.Addr); err != nil {
			return nil, fmt.Errorf("error allocating IP %s: %w", addr.Addr, err)
		}
	} else {
		newAddr, err := alloc.AllocateNext(ctx, claimRef)
		if err != nil {
			if !errors.Is(err, ipaddressallocator.ErrFull) {
				return nil, fmt.Errorf("error allocating dynamic IP: %w", err)
			}

			log.V(1).Info("All IP addresses allocated")
			return nil, apierrors.NewConflict(v1alpha1.Resource("ips"), ip.Name, fmt.Errorf("all IP addresses are already allocated"))
		}

		addr = net.IP{Addr: newAddr}
		ip.Spec.IP = addr
	}
	metav1.SetMetaDataLabel(&ip.ObjectMeta, v1alpha1.IPFamilyLabel, string(alloc.IPFamily()))
	metav1.SetMetaDataLabel(&ip.ObjectMeta, v1alpha1.IPIPLabel, strings.ReplaceAll(addr.String(), ":", "-"))

	log = log.WithValues("Address", addr)

	return func(ctx context.Context, success bool) {
		if success {
			log.Info("Allocated IP")
			return
		}

		log.V(1).Info("Releasing ip after no creation success indicated")
		if err := alloc.Release(ctx, addr.Addr); err != nil {
			log.Error(err, "Error releasing IP")
		} else {
			log.V(1).Info("Released IP")
		}
	}, nil
}

func (r *REST) afterDelete(obj runtime.Object, opts *metav1.DeleteOptions) {
	ctx := context.TODO()

	ip := obj.(*core.IP)
	log := klog.FromContext(ctx).WithValues("Object", klog.KObj(ip), "Address", ip.Spec.IP)
	ctx = klog.NewContext(ctx, log)

	if !dryrun.IsDryRun(opts.DryRun) {
		alloc, ok := r.allocatorByFamily[ip.Spec.IPFamily]
		if !ok {
			return
		}

		addr := ip.Spec.IP.Addr
		if err := alloc.Release(ctx, addr); err != nil {
			log.Error(err, "Error releasing IP")
		} else {
			log.Info("Released IP")
		}
	}
}

func NewStorage(
	scheme *runtime.Scheme,
	optsGetter generic.RESTOptionsGetter,
	allocatorByFamily map[corev1.IPFamily]ipaddressallocator.Interface,
) (IPStorage, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.IP{}
		},
		NewListFunc: func() runtime.Object {
			return &core.IPList{}
		},
		PredicateFunc:             MatchIP,
		DefaultQualifiedResource:  core.Resource("ips"),
		SingularQualifiedResource: core.Resource("ip"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{
		RESTOptions: optsGetter,
		AttrFunc:    GetAttrs,
		TriggerFunc: map[string]apisrvstorage.IndexerFunc{"spec.ip": IPTriggerFunc},
		Indexers:    Indexers(),
	}
	if err := store.CompleteWithOptions(options); err != nil {
		return IPStorage{}, err
	}

	genericStore := &REST{
		Store:             store,
		allocatorByFamily: allocatorByFamily,
	}

	store.BeginCreate = genericStore.beginCreate
	store.AfterDelete = genericStore.afterDelete

	return IPStorage{
		IP: genericStore,
	}, nil
}
