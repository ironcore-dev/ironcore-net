// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ipallocator

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	v1alpha1informers "github.com/ironcore-dev/ironcore-net/client-go/informers/externalversions/core/v1alpha1"
	v1alpha1client "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned/typed/core/v1alpha1"
	v1alpha1listers "github.com/ironcore-dev/ironcore-net/client-go/listers/core/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/generic"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrAllocated = errors.New("provided IP is already allocated")
	ErrNotFound  = errors.New("provided IP was not found")
)

type Allocator struct {
	prefix   netip.Prefix
	ipFamily corev1.IPFamily

	client   v1alpha1client.CoreV1alpha1Interface
	ipLister v1alpha1listers.IPLister
	ipSynced cache.InformerSynced
}

func New(
	prefix netip.Prefix,
	client v1alpha1client.CoreV1alpha1Interface,
	informer v1alpha1informers.IPInformer,
) (*Allocator, error) {
	var ipFamily corev1.IPFamily
	if prefix.Addr().Is6() {
		ipFamily = corev1.IPv6Protocol
	} else {
		ipFamily = corev1.IPv4Protocol
	}

	return &Allocator{
		prefix:   prefix,
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

func (a *Allocator) getIPFromLister(namespace string, addr netip.Addr) (*v1alpha1.IP, error) {
	ips, err := a.ipLister.IPs(namespace).List(labels.SelectorFromSet(labels.Set{
		v1alpha1.IPFamilyLabel: string(a.ipFamily),
		v1alpha1.IPIPLabel:     strings.ReplaceAll(addr.String(), ":", "-"),
	}))
	if err != nil {
		return nil, err
	}
	if n := len(ips); n == 0 {
		return nil, ErrNotFound
	} else if n > 1 {
		return nil, fmt.Errorf("multiple IPs found for address %s", addr)
	}
	return ips[0], nil
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
	ip, err := a.getIPFromLister(namespace, addr)
	if err != nil {
		return err
	}
	if ip.Spec.ClaimRef != nil {
		return ErrAllocated
	}

	base := ip.DeepCopy()
	ip.Spec.ClaimRef = &claimRef
	data, err := client.StrategicMergeFrom(base).Data(ip)
	if err != nil {
		return err
	}

	_, err = a.client.IPs(namespace).Patch(context.Background(), ip.Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	if err == nil {
		return nil
	}
	if apierrors.IsNotFound(err) {
		return ErrNotFound
	}
	if apierrors.IsConflict(err) {
		return ErrAllocated
	}
	return err
}

func (a *Allocator) AllocateNext(
	namespace string,
	claimRef v1alpha1.IPClaimRef,
	version, kind string,
) (netip.Addr, error) {
	return a.allocateNext(namespace, claimRef, version, kind, false)
}

func (a *Allocator) allocateNext(
	namespace string,
	claimRef v1alpha1.IPClaimRef,
	version, kind string,
	dryRun bool,
) (netip.Addr, error) {
	if !a.ipSynced() {
		return netip.Addr{}, fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return a.prefix.Masked().Addr(), nil
	}

	trace := utiltrace.New("allocate dynamic IP")
	defer trace.LogIfLong(500 * time.Millisecond)

	return a.createEphemeralIP(namespace, claimRef, version, kind)
}

func (a *Allocator) createEphemeralIP(
	namespace string,
	claimRef v1alpha1.IPClaimRef,
	version, kind string,
) (netip.Addr, error) {
	ip, err := a.client.IPs(namespace).Create(context.Background(), &v1alpha1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: claimRef.Name + "-",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         (schema.GroupVersion{Group: claimRef.Group, Version: version}).String(),
					Kind:               kind,
					Name:               claimRef.Name,
					UID:                claimRef.UID,
					Controller:         generic.Pointer(true),
					BlockOwnerDeletion: generic.Pointer(true),
				},
			},
		},
		Spec: v1alpha1.IPSpec{
			Type:     v1alpha1.IPTypePublic,
			IPFamily: a.ipFamily,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return netip.Addr{}, err
	}
	return ip.Spec.IP.Addr, nil
}

func (a *Allocator) Release(namespace string, ip netip.Addr) error {
	return a.release(namespace, ip, false)
}

func (a *Allocator) release(namespace string, addr netip.Addr, dryRun bool) error {
	if !a.ipSynced() {
		return fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return nil
	}

	ip, err := a.getIPFromClient(namespace, addr)
	if err != nil {
		klog.ErrorS(err, "error getting IP for address", "address", addr)
		return nil
	}

	// If the IP is controlled, we assume it to be ephemeral.
	// TODO: check on something more robust than just an owner reference.
	if metav1.GetControllerOf(ip) != nil {
		err := a.client.IPs(namespace).Delete(context.Background(), ip.Name, metav1.DeleteOptions{})
		if err == nil {
			return nil
		}
		klog.ErrorS(err, "error deleting IP", "ip", klog.KObj(ip))
		return nil
	}

	if err := a.releaseIP(ip); err != nil {
		klog.ErrorS(err, "error releasing IP", "ip", klog.KObj(ip))
	}
	return nil
}

func (a *Allocator) releaseIP(ip *v1alpha1.IP) error {
	base := ip.DeepCopy()
	ip.Spec.ClaimRef = nil
	data, err := client.StrategicMergeFrom(base).Data(ip)
	if err != nil {
		return err
	}
	_, err = a.client.IPs(ip.Namespace).Patch(context.Background(), ip.Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
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
	version, kind string,
) (netip.Addr, error) {
	return dry.real.allocateNext(namespace, claimRef, version, kind, true)
}

func (dry dryRunAllocator) Release(namespace string, ip netip.Addr) error {
	return dry.real.release(namespace, ip, true)
}

func (dry dryRunAllocator) DryRun() Interface {
	return dry
}

type Interface interface {
	IPFamily() corev1.IPFamily
	Allocate(namespace string, claimRef v1alpha1.IPClaimRef, ip netip.Addr) error
	AllocateNext(namespace string, claimRef v1alpha1.IPClaimRef, version, kind string) (netip.Addr, error)
	Release(namespace string, ip netip.Addr) error
	DryRun() Interface
}
