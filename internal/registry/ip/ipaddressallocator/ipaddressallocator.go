// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ipaddressallocator

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/netip"
	"time"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	v1alpha1informers "github.com/ironcore-dev/ironcore-net/client-go/informers/externalversions/core/v1alpha1"
	v1alpha1client "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned/typed/core/v1alpha1"
	v1alpha1listers "github.com/ironcore-dev/ironcore-net/client-go/listers/core/v1alpha1"
	netiputils "github.com/ironcore-dev/ironcore-net/utils/netip"
	"go4.org/netipx"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"
)

var (
	ErrFull       = errors.New("all IPs are allocated")
	ErrAllocated  = errors.New("provided IP is already allocated")
	ErrNotInRange = errors.New("the provided IP is not in range")
)

type prefixMetaInformation struct {
	prefix  netip.Prefix
	firstIP netip.Addr
	lastIP  netip.Addr
	size    int64
}

type Allocator struct {
	family corev1.IPFamily

	prefixMetaInformation []prefixMetaInformation

	client          v1alpha1client.CoreV1alpha1Interface
	ipAddressLister v1alpha1listers.IPAddressLister
	ipAddressSynced cache.InformerSynced
}

func New(
	prefixes []netip.Prefix,
	client v1alpha1client.CoreV1alpha1Interface,
	informer v1alpha1informers.IPAddressInformer,
) (*Allocator, error) {
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("at least one prefix must be provided")
	}

	var family corev1.IPFamily
	if prefixes[0].Addr().Is6() {
		family = corev1.IPv6Protocol
	} else {
		family = corev1.IPv4Protocol
	}

	prefixMetaInfo := make([]prefixMetaInformation, len(prefixes))
	for i, prefix := range prefixes {
		if prefix.Addr().Is6() != (family == corev1.IPv6Protocol) {
			return nil, fmt.Errorf("all prefixes must be of the same IP family")
		}
		prefixMetaInfo[i] = prefixMetaInformation{
			prefix:  prefix,
			firstIP: prefix.Masked().Addr(),
			lastIP:  netipx.PrefixLastIP(prefix),
			size:    netiputils.PrefixSize(prefix),
		}
	}

	return &Allocator{
		family:                family,
		prefixMetaInformation: prefixMetaInfo,
		client:                client,
		ipAddressLister:       informer.Lister(),
		ipAddressSynced:       informer.Informer().HasSynced,
	}, nil
}

func (a *Allocator) IPFamily() corev1.IPFamily {
	return a.family
}

func (a *Allocator) Allocate(claimRef v1alpha1.IPAddressClaimRef, ip netip.Addr) error {
	return a.allocate(claimRef, ip, false)
}

func (a *Allocator) allocate(claimRef v1alpha1.IPAddressClaimRef, ip netip.Addr, dryRun bool) error {
	if !a.ipAddressSynced() {
		return fmt.Errorf("allocator not ready")
	}
	if !ip.IsValid() {
		return fmt.Errorf("invalid IP")
	}

	// Check if IP is in any of the prefixes
	for _, meta := range a.prefixMetaInformation {
		if !ip.Less(meta.firstIP) && !meta.lastIP.Less(ip) {
			if dryRun {
				return nil
			}
			return a.createIPAddress(ip.String(), claimRef)
		}
	}

	return ErrNotInRange
}

func (a *Allocator) createIPAddress(name string, claimRef v1alpha1.IPAddressClaimRef) error {
	ipAddress := &v1alpha1.IPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.IPAddressSpec{
			ClaimRef: claimRef,
		},
	}
	_, err := a.client.IPAddresses().Create(context.Background(), ipAddress, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return ErrAllocated
		}
		return err
	}
	return nil
}

func (a *Allocator) AllocateNext(claimRef v1alpha1.IPAddressClaimRef) (netip.Addr, error) {
	return a.allocateNext(claimRef, false)
}

func (a *Allocator) allocateNext(claimRef v1alpha1.IPAddressClaimRef, dryRun bool) (netip.Addr, error) {
	if !a.ipAddressSynced() {
		return netip.Addr{}, fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return a.prefixMetaInformation[0].prefix.Addr(), nil
	}

	trace := utiltrace.New("allocate dynamic IPAddress")
	defer trace.LogIfLong(500 * time.Millisecond)

	// Try each prefix in order
	for _, meta := range a.prefixMetaInformation {
		offset := rand.Int63n(meta.size)
		iterator := ipIterator(meta.firstIP, meta.lastIP, uint64(offset))
		addr, err := a.allocateFromIterator(claimRef, iterator)
		if err == nil {
			return addr, nil
		}
		if err != ErrFull {
			return netip.Addr{}, err
		}
	}

	return netip.Addr{}, ErrFull
}

func (a *Allocator) allocateFromIterator(claimRef v1alpha1.IPAddressClaimRef, it func() netip.Addr) (netip.Addr, error) {
	for {
		addr := it()
		if !addr.IsValid() {
			return netip.Addr{}, ErrFull
		}

		name := addr.String()
		_, err := a.client.IPAddresses().Get(context.Background(), name, metav1.GetOptions{})
		if err == nil {
			continue
		}

		if !apierrors.IsNotFound(err) {
			klog.InfoS("unexpected error", "err", err)
			continue
		}

		err = a.createIPAddress(name, claimRef)
		if err != nil {
			klog.InfoS("can not create IP address", "name", name, "err", err)
			continue
		}

		return addr, nil
	}
}

func (a *Allocator) Release(ip netip.Addr) error {
	return a.release(ip, false)
}

func (a *Allocator) release(ip netip.Addr, dryRun bool) error {
	if !a.ipAddressSynced() {
		return fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return nil
	}

	name := ip.String()
	err := a.client.IPAddresses().Delete(context.Background(), name, metav1.DeleteOptions{})
	if err == nil {
		return nil
	}
	klog.InfoS("error releasing IP", "ip", ip, "err", err)
	return nil
}

func (a *Allocator) DryRun() Interface {
	return dryRunAllocator{real: a}
}

type dryRunAllocator struct {
	real *Allocator
}

func (dry dryRunAllocator) IPFamily() corev1.IPFamily {
	return dry.real.IPFamily()
}

func (dry dryRunAllocator) Allocate(claimRef v1alpha1.IPAddressClaimRef, ip netip.Addr) error {
	return dry.real.allocate(claimRef, ip, true)
}

func (dry dryRunAllocator) AllocateNext(claimRef v1alpha1.IPAddressClaimRef) (netip.Addr, error) {
	return dry.real.allocateNext(claimRef, true)
}

func (dry dryRunAllocator) Release(ip netip.Addr) error {
	return dry.real.release(ip, true)
}

func (dry dryRunAllocator) DryRun() Interface {
	return dry
}

func ipIterator(first netip.Addr, last netip.Addr, offset uint64) func() netip.Addr {
	// There are no modulo operations for IP addresses
	modulo := func(addr netip.Addr) netip.Addr {
		if addr.Compare(last) == 1 {
			return first
		}
		return addr
	}
	next := func(addr netip.Addr) netip.Addr {
		return modulo(addr.Next())
	}
	start, err := netiputils.AddOffsetAddress(first, offset)
	if err != nil {
		return func() netip.Addr { return netip.Addr{} }
	}
	start = modulo(start)
	ip := start
	seen := false
	return func() netip.Addr {
		value := ip
		// is the last or the first iteration
		if value == start {
			if seen {
				return netip.Addr{}
			}
			seen = true
		}
		ip = next(ip)
		return value
	}

}

type Interface interface {
	IPFamily() corev1.IPFamily
	Allocate(claimRef v1alpha1.IPAddressClaimRef, ip netip.Addr) error
	AllocateNext(claimRef v1alpha1.IPAddressClaimRef) (netip.Addr, error)
	Release(ip netip.Addr) error
	DryRun() Interface
}
