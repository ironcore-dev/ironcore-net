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

package ipallocator

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/utils/core"
	"github.com/ironcore-dev/ironcore-net/utils/iterator"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
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

type Accessor interface {
	GetNamespace() string
	GetName() string
	GetUID() types.UID
	GetRequests() []Request
	SetIP(idx int, addr netip.Addr)
}

type Allocators struct {
	allocByFamily map[corev1.IPFamily]Interface

	gv       schema.GroupVersion
	kind     string
	resource string

	accessorFor func(obj runtime.Object) (Accessor, error)
}

func NewAllocators(
	allocByIPFamily map[corev1.IPFamily]Interface,
	gv schema.GroupVersion,
	kind, resource string,
	accessorFor func(obj runtime.Object) (Accessor, error),
) *Allocators {
	return &Allocators{
		allocByFamily: allocByIPFamily,
		gv:            gv,
		kind:          kind,
		resource:      resource,
		accessorFor:   accessorFor,
	}
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

func (a *Allocators) releaseIPs(allocByIPFamily map[corev1.IPFamily]Interface, namespace string, ips []netip.Addr) ([]netip.Addr, error) {
	var (
		released []netip.Addr
		errs     []error
	)
	for _, ip := range ips {
		alloc := allocByIPFamily[core.IPFamilyForAddr(ip)]
		if err := alloc.Release(namespace, ip); err != nil {
			errs = append(errs, err)
			continue
		}

		released = append(released, ip)
	}
	return released, errors.Join(errs...)
}

func (a *Allocators) allocateIPs(allocByFamily map[corev1.IPFamily]Interface, acc Accessor, reqs []Request) ([]netip.Addr, error) {
	var allocated []netip.Addr
	for _, req := range reqs {
		alloc := allocByFamily[req.IPFamily]

		addr := req.Addr
		claimRef := v1alpha1.IPClaimRef{
			Group:    a.gv.Group,
			Resource: a.resource,
			Name:     acc.GetName(),
			UID:      acc.GetUID(),
		}
		if addr.IsValid() {
			if err := alloc.Allocate(acc.GetNamespace(), claimRef, addr); err != nil {
				return allocated, err
			}
		} else {
			newAddr, err := alloc.AllocateNext(acc.GetNamespace(), claimRef, a.gv.Version, a.kind)
			if err != nil {
				return allocated, err
			}

			addr = newAddr
		}

		allocated = append(allocated, addr)
	}
	return allocated, nil
}

func (a *Allocators) AllocateCreate(obj runtime.Object, dryRun bool) (Transaction, error) {
	acc, err := a.accessorFor(obj)
	if err != nil {
		return nil, err
	}

	reqs := acc.GetRequests()
	allocs, err := a.allocatorsForRequestIterator(iterator.OfSlice(reqs), dryRun)
	if err != nil {
		return nil, err
	}

	allocated, err := a.allocateIPs(allocs, acc, reqs)
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
				klog.InfoS("Allocated IPs", "ips", allocated)
			}
		},
		RevertFunc: func() {
			if dryRun {
				return
			}

			actuallyReleased, err := a.releaseIPs(allocs, acc.GetNamespace(), allocated)
			if err != nil {
				klog.ErrorS(err, "Error releasing IPs",
					"shouldRelease", allocated,
					"released", actuallyReleased,
				)
			}
		},
	}, nil
}

func (a *Allocators) AllocateUpdate(obj, oldObj runtime.Object, dryRun bool) (Transaction, error) {
	acc, err := a.accessorFor(obj)
	if err != nil {
		return nil, err
	}

	oldAcc, err := a.accessorFor(oldObj)
	if err != nil {
		return nil, err
	}

	newReqs := acc.GetRequests()
	oldReqs := oldAcc.GetRequests()

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

	allocated, err := a.allocateIPs(allocs, acc, toAllocate)
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
				klog.InfoS("allocated IPs", "ips", allocated)
			}

			toRelease := toReleaseSet.UnsortedList()
			if actuallyReleased, err := a.releaseIPs(allocs, acc.GetNamespace(), toRelease); err != nil {
				klog.ErrorS(err, "Error releasing IPs",
					"shouldRelease", toRelease,
					"released", actuallyReleased,
				)
			}
		},
		RevertFunc: func() {
			if dryRun {
				return
			}

			if actuallyReleased, err := a.releaseIPs(allocs, acc.GetNamespace(), toReleaseSet.UnsortedList()); err != nil {
				klog.ErrorS(err, "Error releasing IPs",
					"shouldRelease", allocated,
					"released", actuallyReleased,
				)
			}
		},
	}, nil
}

func (a *Allocators) Release(obj runtime.Object, dryRun bool) {
	acc, err := a.accessorFor(obj)
	if err != nil {
		klog.ErrorS(err, "Error getting accessor for object", "object", obj)
		return
	}

	if dryRun {
		return
	}

	reqs := acc.GetRequests()
	allocs, err := a.allocatorsForRequestIterator(iterator.OfSlice(reqs), false)
	if err != nil {
		klog.ErrorS(err, "Error getting allocators")
		return
	}

	allocated := utilslices.Map(reqs, func(r Request) netip.Addr { return r.Addr })
	actuallyReleased, err := a.releaseIPs(allocs, acc.GetNamespace(), allocated)
	if err != nil {
		klog.ErrorS(err, "Error releasing IPs",
			"shouldRelease", allocated,
			"released", actuallyReleased,
		)
	}
}
