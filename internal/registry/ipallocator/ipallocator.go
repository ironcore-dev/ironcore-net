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

	"github.com/go-logr/logr"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	ErrAllocated = errors.New("provided IP is already allocated")
	ErrNotFound  = errors.New("provided IP was not found")
)

type Allocator struct {
	scheme *runtime.Scheme

	prefixes []netip.Prefix
	ipFamily corev1.IPFamily

	client   v1alpha1client.CoreV1alpha1Interface
	ipLister v1alpha1listers.IPLister
	ipSynced cache.InformerSynced
}

func New(
	scheme *runtime.Scheme,
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
		scheme:   scheme,
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

func (a *Allocator) Allocate(ctx context.Context, claimer *Claimer, ip netip.Addr) error {
	return a.allocate(ctx, claimer, ip, false)
}

func (a *Allocator) allocate(ctx context.Context, claimer *Claimer, ip netip.Addr, dryRun bool) error {
	log := klog.FromContext(ctx).
		WithValues(
			"resource", claimer.Resource,
			"claimer", klog.KObj(claimer.Object),
			"dryRun", dryRun,
		)

	if !a.ipSynced() {
		return fmt.Errorf("allocator not ready")
	}
	if !ip.IsValid() {
		return fmt.Errorf("invalid IP")
	}

	if dryRun {
		return nil
	}

	return a.claimIP(ctx, log, claimer, ip)
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

func (a *Allocator) getIPFromClient(ctx context.Context, namespace string, addr netip.Addr) (*v1alpha1.IP, error) {
	ipList, err := a.client.IPs(namespace).List(ctx, metav1.ListOptions{
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

func newClaimRef(group string, claimer *Claimer) *v1alpha1.IPClaimRef {
	return &v1alpha1.IPClaimRef{
		Group:    group,
		Resource: claimer.Resource,
		Name:     claimer.Object.GetName(),
		UID:      claimer.Object.GetUID(),
	}
}

func (a *Allocator) claimRefFor(claimer *Claimer) (*v1alpha1.IPClaimRef, error) {
	gvk, err := apiutil.GVKForObject(claimer.Object, a.scheme)
	if err != nil {
		return nil, err
	}

	return newClaimRef(gvk.Group, claimer), nil
}

func referSameClaimer(refA, refB v1alpha1.IPClaimRef) bool {
	return refA.UID == refB.UID
}

func (a *Allocator) claimIP(ctx context.Context, log logr.Logger, claimer *Claimer, addr netip.Addr) error {
	claimRef, err := a.claimRefFor(claimer)
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ip, err := a.getIPFromLister(claimer.Object.GetNamespace(), addr)
		if err != nil {
			return err
		}
		if ip.Spec.ClaimRef != nil && !referSameClaimer(*ip.Spec.ClaimRef, *claimRef) {
			return ErrAllocated
		}

		base := ip.DeepCopy()
		ip.Spec.ClaimRef = claimRef
		data, err := client.StrategicMergeFrom(base, client.MergeFromWithOptimisticLock{}).Data(ip)
		if err != nil {
			return err
		}

		_, err = a.client.IPs(claimer.Object.GetNamespace()).Patch(ctx, ip.Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return ErrNotFound
			}
			return err
		}

		log.V(1).Info("Claimed IP", "ip", klog.KObj(ip))
		return nil
	})
	if err != nil {
		if apierrors.IsConflict(err) {
			return ErrAllocated
		}
		return err
	}
	return nil
}

func (a *Allocator) AllocateNext(
	ctx context.Context,
	claimer *Claimer,
) (netip.Addr, error) {
	return a.allocateNext(ctx, claimer, false)
}

func (a *Allocator) allocateNext(
	ctx context.Context,
	claimer *Claimer,
	dryRun bool,
) (netip.Addr, error) {
	log := klog.FromContext(ctx).
		WithValues(
			"resource", claimer.Resource,
			"claimer", klog.KObj(claimer.Object),
			"dryRun", dryRun,
		)

	if !a.ipSynced() {
		return netip.Addr{}, fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return a.prefixes[0].Masked().Addr(), nil
	}

	trace := utiltrace.New("allocate dynamic IP")
	defer trace.LogIfLong(500 * time.Millisecond)

	ip, err := a.createEphemeralIP(ctx, claimer)
	if err != nil {
		return netip.Addr{}, err
	}

	log.V(1).Info("Allocated dynamic IP", "ip", ip)
	return ip, err
}

func (a *Allocator) createEphemeralIP(
	ctx context.Context,
	claimer *Claimer,
) (netip.Addr, error) {
	gvk, err := apiutil.GVKForObject(claimer.Object, a.scheme)
	if err != nil {
		return netip.Addr{}, err
	}

	ip, err := a.client.IPs(claimer.Object.GetNamespace()).Create(ctx, &v1alpha1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    claimer.Object.GetNamespace(),
			GenerateName: claimer.Object.GetName() + "-",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         schema.GroupVersion{Group: gvk.Group, Version: claimer.ExternalVersion}.String(),
					Kind:               gvk.Kind,
					Name:               claimer.Object.GetName(),
					UID:                claimer.Object.GetUID(),
					Controller:         generic.Pointer(true),
					BlockOwnerDeletion: generic.Pointer(true),
				},
			},
		},
		Spec: v1alpha1.IPSpec{
			Type:     v1alpha1.IPTypePublic,
			IPFamily: a.ipFamily,
			ClaimRef: newClaimRef(gvk.Group, claimer),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return netip.Addr{}, err
	}
	return ip.Spec.IP.Addr, nil
}

func (a *Allocator) Release(ctx context.Context, claimer *Claimer, ip netip.Addr) error {
	return a.release(ctx, claimer, ip, false)
}

func isIPClaimedBy(ip *v1alpha1.IP, obj metav1.Object) bool {
	claimRef := ip.Spec.ClaimRef
	if claimRef == nil {
		return false
	}

	return claimRef.UID == obj.GetUID()
}

func (a *Allocator) release(ctx context.Context, claimer *Claimer, addr netip.Addr, dryRun bool) error {
	log := klog.FromContext(ctx).
		WithValues(
			"Address", addr,
			"Resource", claimer.Resource,
			"Claimer", klog.KObj(claimer.Object),
			"DryRun", dryRun,
		)

	if !a.ipSynced() {
		return fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return nil
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ip, err := a.getIPFromClient(ctx, claimer.Object.GetNamespace(), addr)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "Error getting IP for address")
			}

			log.V(1).Info("IP already gone")
			return nil
		}

		log := log.WithValues("IP", klog.KObj(ip))

		switch {
		case !isIPClaimedBy(ip, claimer.Object) && !metav1.IsControlledBy(ip, claimer.Object):
			log.V(1).Info("IP neither claimed nor controlled by claimer, no need to release")
			return nil
		case !isIPClaimedBy(ip, claimer.Object) && metav1.IsControlledBy(ip, claimer.Object):
			log.Info("IP is not claimed but controlled by claimer")
			return nil
		case isIPClaimedBy(ip, claimer.Object) && metav1.IsControlledBy(ip, claimer.Object):
			log.V(1).Info("IP is claimed and controlled by claimer, deleting to release")
			if err := a.client.IPs(claimer.Object.GetNamespace()).Delete(ctx, ip.Name, metav1.DeleteOptions{}); err != nil {
				if !apierrors.IsNotFound(err) {
					return err
				}
			}
			return nil
		default:
			if err := a.patchReleaseIP(ctx, ip); err != nil {
				log.Error(err, "Error releasing IP")
			}
			return nil
		}
	})
	if err != nil {
		return err
	}

	log.V(1).Info("Released IP")
	return nil
}

func (a *Allocator) patchReleaseIP(ctx context.Context, ip *v1alpha1.IP) error {
	base := ip.DeepCopy()
	ip.Spec.ClaimRef = nil
	data, err := client.StrategicMergeFrom(base, client.MergeFromWithOptimisticLock{}).Data(ip)
	if err != nil {
		return err
	}
	_, err = a.client.IPs(ip.Namespace).Patch(ctx, ip.Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
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

func (dry dryRunAllocator) Allocate(ctx context.Context, claimer *Claimer, ip netip.Addr) error {
	return dry.real.allocate(ctx, claimer, ip, true)
}

func (dry dryRunAllocator) AllocateNext(
	ctx context.Context,
	claimer *Claimer,
) (netip.Addr, error) {
	return dry.real.allocateNext(ctx, claimer, true)
}

func (dry dryRunAllocator) Release(ctx context.Context, claimer *Claimer, ip netip.Addr) error {
	return dry.real.release(ctx, claimer, ip, true)
}

func (dry dryRunAllocator) DryRun() Interface {
	return dry
}

// Claimer represents a claimer of an IP.
type Claimer struct {
	// Object is the claiming object.
	Object client.Object
	// Resource is the API resource of the claiming object.
	Resource string
	// ExternalVersion is the external API version to use for references.
	ExternalVersion string
}

type Interface interface {
	IPFamily() corev1.IPFamily
	Allocate(ctx context.Context, claimer *Claimer, ip netip.Addr) error
	AllocateNext(ctx context.Context, claimer *Claimer) (netip.Addr, error)
	Release(ctx context.Context, claimer *Claimer, ip netip.Addr) error
	DryRun() Interface
}
