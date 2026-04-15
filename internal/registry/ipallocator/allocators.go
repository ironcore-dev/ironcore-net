// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ipallocator

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/ironcore-dev/ironcore-net/utils/core"
	"github.com/ironcore-dev/ironcore-net/utils/iterator"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Transaction interface {
	Commit()
	Revert()
}

type Request struct {
	IPFamily corev1.IPFamily
	Addr     netip.Addr
}

type transactionFuncs struct {
	CommitFunc func()
	RevertFunc func()
}

func (f transactionFuncs) Commit() {
	if f.CommitFunc != nil {
		f.CommitFunc()
	}
}

func (f transactionFuncs) Revert() {
	if f.RevertFunc != nil {
		f.RevertFunc()
	}
}

type Requester interface {
	GetRequests() []Request
	SetIP(idx int, addr netip.Addr)
}

type Allocators struct {
	allocByFamily map[corev1.IPFamily]Interface

	gv       schema.GroupVersion
	kind     string
	resource string

	requesterFor func(obj runtime.Object) (Requester, error)
}

func NewAllocators(
	allocByIPFamily map[corev1.IPFamily]Interface,
	gv schema.GroupVersion,
	kind, resource string,
	requesterFor func(obj runtime.Object) (Requester, error),
) *Allocators {
	return &Allocators{
		allocByFamily: allocByIPFamily,
		gv:            gv,
		kind:          kind,
		resource:      resource,
		requesterFor:  requesterFor,
	}
}

func (a *Allocators) requesterAndClaimerForObject(obj client.Object) (Requester, *Claimer, error) {
	requester, err := a.requesterFor(obj)
	if err != nil {
		return nil, nil, err
	}

	claimer := &Claimer{
		Object:          obj,
		Resource:        a.resource,
		ExternalVersion: a.gv.Version,
	}

	return requester, claimer, nil
}

func (a *Allocators) allocatorsForRequestIterator(it func(yield func(Request) bool) bool, dryRun bool) (map[corev1.IPFamily]Interface, error) {
	var (
		allocs = make(map[corev1.IPFamily]Interface)
		err    error
	)
	it(func(req Request) bool {
		alloc, ok := a.allocByFamily[req.IPFamily]
		if !ok {
			err = fmt.Errorf("no allocator for IPs of family %s", req.IPFamily)
			return false
		}
		if dryRun {
			alloc = alloc.DryRun()
		}

		allocs[req.IPFamily] = alloc
		return len(allocs) != 2
	})
	return allocs, err
}

func (a *Allocators) releaseIPs(
	ctx context.Context,
	allocByIPFamily map[corev1.IPFamily]Interface,
	claimedBy *Claimer,
	ips []netip.Addr,
) ([]netip.Addr, error) {
	var (
		released []netip.Addr
		errs     []error
	)
	for _, ip := range ips {
		alloc := allocByIPFamily[core.IPFamilyForAddr(ip)]
		if err := alloc.Release(ctx, claimedBy, ip); err != nil {
			errs = append(errs, err)
			continue
		}

		released = append(released, ip)
	}
	return released, errors.Join(errs...)
}

func (a *Allocators) allocateIPs(
	ctx context.Context,
	allocByFamily map[corev1.IPFamily]Interface,
	claimedBy *Claimer,
	reqs []Request,
) ([]netip.Addr, error) {
	var allocated []netip.Addr
	for _, req := range reqs {
		alloc := allocByFamily[req.IPFamily]

		addr := req.Addr
		if addr.IsValid() {
			if err := alloc.Allocate(ctx, claimedBy, addr); err != nil {
				return allocated, err
			}
		} else {
			newAddr, err := alloc.AllocateNext(ctx, claimedBy)
			if err != nil {
				return allocated, err
			}

			addr = newAddr
		}

		allocated = append(allocated, addr)
	}
	return allocated, nil
}

func (a *Allocators) AllocateCreate(ctx context.Context, obj client.Object, dryRun bool) (Transaction, error) {
	acc, claimedBy, err := a.requesterAndClaimerForObject(obj)
	if err != nil {
		return nil, err
	}

	reqs := acc.GetRequests()

	log := klog.FromContext(ctx).WithValues(
		"Requester", klog.KObj(claimedBy.Object),
		"Requests", reqs,
	)

	allocs, err := a.allocatorsForRequestIterator(iterator.OfSlice(reqs), dryRun)
	if err != nil {
		return nil, err
	}

	allocated, err := a.allocateIPs(ctx, allocs, claimedBy, reqs)
	if err != nil {
		return nil, err
	}

	for i, ip := range allocated {
		acc.SetIP(i, ip)
	}

	return transactionFuncs{
		CommitFunc: func() {
			if dryRun {
				return
			}
			if len(allocated) > 0 {
				log.Info("Allocated IPs", "Allocated", allocated)
			}
		},
		RevertFunc: func() {
			if dryRun {
				return
			}

			actuallyReleased, err := a.releaseIPs(ctx, allocs, claimedBy, allocated)
			if err != nil {
				log.Error(err, "Error releasing IPs",
					"shouldRelease", allocated,
					"released", actuallyReleased,
				)
			} else {
				log.Info("Released IPs", "Released", actuallyReleased)
			}
		},
	}, nil
}

func (a *Allocators) AllocateUpdate(ctx context.Context, obj, oldObj client.Object, dryRun bool) (Transaction, error) {
	acc, claimedBy, err := a.requesterAndClaimerForObject(obj)
	if err != nil {
		return nil, err
	}

	oldAcc, err := a.requesterFor(oldObj)
	if err != nil {
		return nil, err
	}

	newReqs := acc.GetRequests()
	oldReqs := oldAcc.GetRequests()

	log := klog.FromContext(ctx).WithValues(
		"Requester", klog.KObj(claimedBy.Object),
		"OldRequests", oldReqs,
		"NewRequests", newReqs,
	)

	reqsIt := iterator.Concat(iterator.OfSlice(oldReqs), iterator.OfSlice(newReqs))
	allocs, err := a.allocatorsForRequestIterator(reqsIt, dryRun)
	if err != nil {
		return nil, err
	}

	var (
		toReleaseSet         = utilslices.ToSetFunc(oldReqs, func(r Request) netip.Addr { return r.Addr })
		toAllocate           []Request
		indexToAllocateIndex = make(map[int]int)
	)
	for i, newReq := range newReqs {
		if toReleaseSet.Has(newReq.Addr) {
			toReleaseSet.Delete(newReq.Addr)
		} else {
			indexToAllocateIndex[i] = len(toAllocate)
			toAllocate = append(toAllocate, newReq)
		}
	}

	allocated, err := a.allocateIPs(ctx, allocs, claimedBy, toAllocate)
	if err != nil {
		return nil, err
	}

	for idx, allocateIdx := range indexToAllocateIndex {
		acc.SetIP(idx, allocated[allocateIdx])
	}

	return transactionFuncs{
		CommitFunc: func() {
			if dryRun {
				return
			}

			if len(allocated) > 0 {
				log.Info("Allocated IPs", "Allocated", allocated)
			}

			toRelease := toReleaseSet.UnsortedList()
			if len(toRelease) == 0 {
				return
			}

			if actuallyReleased, err := a.releaseIPs(ctx, allocs, claimedBy, toRelease); err != nil {
				log.Error(err, "Error releasing IPs",
					"ShouldRelease", toRelease,
					"Released", actuallyReleased,
				)
			} else {
				log.Info("Released IPs", "Released", actuallyReleased)
			}
		},
		RevertFunc: func() {
			if dryRun {
				return
			}

			if actuallyReleased, err := a.releaseIPs(ctx, allocs, claimedBy, toReleaseSet.UnsortedList()); err != nil {
				log.Error(err, "Error releasing IPs",
					"ShouldRelease", allocated,
					"Released", actuallyReleased,
				)
			} else {
				log.Info("Released IPs", "Released", actuallyReleased)
			}
		},
	}, nil
}

func (a *Allocators) Release(ctx context.Context, obj client.Object, dryRun bool) {
	log := klog.FromContext(ctx)

	acc, claimedBy, err := a.requesterAndClaimerForObject(obj)
	if err != nil {
		log.Error(err, "Error getting requester / claimed by for object", "Object", obj)
		return
	}

	reqs := acc.GetRequests()
	allocated := utilslices.Map(reqs, func(r Request) netip.Addr { return r.Addr })

	log = log.WithValues(
		"Requester", klog.KObj(claimedBy.Object),
		"Requests", reqs,
		"Allocated", allocated,
	)

	allocs, err := a.allocatorsForRequestIterator(iterator.OfSlice(reqs), dryRun)
	if err != nil {
		log.Error(err, "Error getting allocators")
		return
	}

	actuallyReleased, err := a.releaseIPs(ctx, allocs, claimedBy, allocated)
	if err != nil {
		log.Error(err, "Error releasing IPs",
			"ShouldRelease", allocated,
			"Released", actuallyReleased,
		)
	} else {
		log.Info("Released IPs", "Released", actuallyReleased)
	}
}
