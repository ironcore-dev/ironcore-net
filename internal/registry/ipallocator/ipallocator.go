// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ipallocator

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	v1alpha1informers "github.com/ironcore-dev/ironcore-net/client-go/informers/externalversions/core/v1alpha1"
	v1alpha1client "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned/typed/core/v1alpha1"
	v1alpha1listers "github.com/ironcore-dev/ironcore-net/client-go/listers/core/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	utiltrace "k8s.io/utils/trace"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrAllocated = errors.New("provided IP is already allocated")
	ErrNotFound  = errors.New("provided IP was not found")
)

type Allocator struct {
	prefixes []netip.Prefix
	ipFamily corev1.IPFamily

	client   v1alpha1client.CoreV1alpha1Interface
	ipLister v1alpha1listers.IPLister
	ipSynced cache.InformerSynced
}

func New(
	prefixes []netip.Prefix,
	client v1alpha1client.CoreV1alpha1Interface,
	informer v1alpha1informers.IPInformer,
) (*Allocator, error) {
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("at least one prefix must be provided")
	}

	var ipFamily corev1.IPFamily
	if prefixes[0].Addr().Is6() {
		ipFamily = corev1.IPv6Protocol
	} else {
		ipFamily = corev1.IPv4Protocol
	}

	for _, prefix := range prefixes {
		if prefix.Addr().Is6() != (ipFamily == corev1.IPv6Protocol) {
			return nil, fmt.Errorf("all prefixes must be of the same IP family")
		}
	}

	return &Allocator{
		prefixes: prefixes,
		ipFamily: ipFamily,
		client:   client,
		ipLister: informer.Lister(),
		ipSynced: informer.Informer().HasSynced,
	}, nil
}

func (a *Allocator) IPFamily() corev1.IPFamily {
	return a.ipFamily
}

func (a *Allocator) Allocate(namespace string, claimRef v1alpha1.IPClaimRef, ip netip.Addr) error {
	return a.allocate(namespace, claimRef, ip, false)
}

func (a *Allocator) allocate(namespace string, claimRef v1alpha1.IPClaimRef, ip netip.Addr, dryRun bool) error {
	if !a.ipSynced() {
		return fmt.Errorf("allocator not ready")
	}
	if !ip.IsValid() {
		return fmt.Errorf("invalid IP")
	}

	if dryRun {
		return nil
	}

	return a.claimIP(namespace, claimRef, ip)
}

func (a *Allocator) getIPFromClient(namespace string, addr netip.Addr) (*v1alpha1.IP, error) {
	ipList, err := a.client.IPs(namespace).List(context.Background(), metav1.ListOptions{
		FieldSelector: (fields.Set{"spec.ip": addr.String()}).String(),
	})
	if err != nil {
		return nil, err
	}
	if n := len(ipList.Items); n == 0 {
		return nil, ErrNotFound
	} else if n > 1 {
		return nil, fmt.Errorf("multiple IPs found for address %s", addr)
	}
	ip := ipList.Items[0]
	return &ip, nil
}

func (a *Allocator) claimIP(namespace string, claimRef v1alpha1.IPClaimRef, addr netip.Addr) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ip, err := a.getIPFromClient(namespace, addr)
		if err != nil {
			return err
		}
		if ip.Spec.ClaimRef != nil {
			return ErrAllocated
		}

		base := ip.DeepCopy()
		ip.Spec.ClaimRef = &claimRef
		patch := client.MergeFromWithOptions(base, &client.MergeFromWithOptimisticLock{})
		data, err := patch.Data(ip)
		if err != nil {
			return err
		}

		_, err = a.client.IPs(namespace).Patch(context.Background(), ip.Name, patch.Type(), data, metav1.PatchOptions{})
		if err == nil {
			return nil
		}
		if apierrors.IsNotFound(err) {
			return ErrNotFound
		}
		return err
	})
}

func (a *Allocator) AllocateNext(
	namespace string,
	claimRef v1alpha1.IPClaimRef,
) (netip.Addr, error) {
	return a.allocateNext(namespace, claimRef, false)
}

func (a *Allocator) allocateNext(
	namespace string,
	claimRef v1alpha1.IPClaimRef,
	dryRun bool,
) (netip.Addr, error) {
	if !a.ipSynced() {
		return netip.Addr{}, fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return a.prefixes[0].Masked().Addr(), nil
	}

	trace := utiltrace.New("allocate dynamic IP")
	defer trace.LogIfLong(500 * time.Millisecond)

	// Try each prefix in order
	for range a.prefixes {
		ip, err := a.createEphemeralIP(namespace, claimRef)
		if err == nil {
			return ip, nil
		}
		if err != ErrAllocated {
			return netip.Addr{}, err
		}
	}

	return netip.Addr{}, ErrAllocated
}

func (a *Allocator) createEphemeralIP(
	namespace string,
	claimRef v1alpha1.IPClaimRef,
) (netip.Addr, error) {
	ip, err := a.client.IPs(namespace).Create(context.Background(), &v1alpha1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: claimRef.Name + "-",
			Labels: map[string]string{
				v1alpha1.IPEphemeralLabel: "true",
			},
		},
		Spec: v1alpha1.IPSpec{
			Type:     v1alpha1.IPTypePublic,
			IPFamily: a.ipFamily,
			ClaimRef: &claimRef,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return netip.Addr{}, err
	}
	return ip.Spec.IP.Addr, nil
}

func (a *Allocator) Release(namespace string, ip netip.Addr, claimRef v1alpha1.IPClaimRef) error {
	return a.release(namespace, ip, claimRef, false)
}

func (a *Allocator) release(namespace string, addr netip.Addr, claimRef v1alpha1.IPClaimRef, dryRun bool) error {
	if !a.ipSynced() {
		return fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ip, err := a.getIPFromClient(namespace, addr)
		if err != nil {
			// IP is already gone — nothing to release.
			if err == ErrNotFound {
				return nil
			}
			return err
		}

		// If the claim no longer belongs to the expected owner, the IP was
		// reclaimed by another resource — nothing to release.
		if ip.Spec.ClaimRef == nil || ip.Spec.ClaimRef.UID != claimRef.UID {
			return nil
		}

		// If the IP is labeled as ephemeral, delete it.
		if ip.Labels[v1alpha1.IPEphemeralLabel] == "true" {
			return a.client.IPs(namespace).Delete(context.Background(), ip.Name, metav1.DeleteOptions{
				Preconditions: &metav1.Preconditions{ResourceVersion: &ip.ResourceVersion},
			})
		}

		return a.releaseIP(ip)
	})
}

func (a *Allocator) releaseIP(ip *v1alpha1.IP) error {
	base := ip.DeepCopy()
	ip.Spec.ClaimRef = nil
	patch := client.MergeFromWithOptions(base, &client.MergeFromWithOptimisticLock{})
	data, err := patch.Data(ip)
	if err != nil {
		return err
	}
	_, err = a.client.IPs(ip.Namespace).Patch(context.Background(), ip.Name, patch.Type(), data, metav1.PatchOptions{})
	return err
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

func (dry dryRunAllocator) Allocate(namespace string, claimRef v1alpha1.IPClaimRef, ip netip.Addr) error {
	return dry.real.allocate(namespace, claimRef, ip, true)
}

func (dry dryRunAllocator) AllocateNext(
	namespace string,
	claimRef v1alpha1.IPClaimRef,
) (netip.Addr, error) {
	return dry.real.allocateNext(namespace, claimRef, true)
}

func (dry dryRunAllocator) Release(namespace string, ip netip.Addr, claimRef v1alpha1.IPClaimRef) error {
	return dry.real.release(namespace, ip, claimRef, true)
}

func (dry dryRunAllocator) DryRun() Interface {
	return dry
}

type Interface interface {
	IPFamily() corev1.IPFamily
	Allocate(namespace string, claimRef v1alpha1.IPClaimRef, ip netip.Addr) error
	AllocateNext(namespace string, claimRef v1alpha1.IPClaimRef) (netip.Addr, error)
	Release(namespace string, ip netip.Addr, claimRef v1alpha1.IPClaimRef) error
	DryRun() Interface
}
