// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkidallocator

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	v1alpha1informers "github.com/ironcore-dev/ironcore-net/client-go/informers/externalversions/core/v1alpha1"
	v1alpha1client "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned/typed/core/v1alpha1"
	v1alpha1listers "github.com/ironcore-dev/ironcore-net/client-go/listers/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/networkid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"
)

var (
	ErrFull       = errors.New("all IDs are allocated")
	ErrAllocated  = errors.New("provided ID is already allocated")
	ErrNotInRange = errors.New("the provided ID is not in range")
)

type Allocator struct {
	minVNI int32
	maxVNI int32

	client          v1alpha1client.CoreV1alpha1Interface
	networkIDLister v1alpha1listers.NetworkIDLister
	networkIDSynced cache.InformerSynced
}

func NewNetworkIDAllocator(
	client v1alpha1client.CoreV1alpha1Interface,
	networkIDInformer v1alpha1informers.NetworkIDInformer,
	minVNI, maxVNI int32,
) (*Allocator, error) {
	if minVNI < 0 || maxVNI < 0 || minVNI > maxVNI || maxVNI == 0 || minVNI == maxVNI {
		return nil, fmt.Errorf("invalid min / max vnis %d/%d", minVNI, maxVNI)
	}

	return &Allocator{
		minVNI:          minVNI,
		maxVNI:          maxVNI,
		client:          client,
		networkIDLister: networkIDInformer.Lister(),
		networkIDSynced: networkIDInformer.Informer().HasSynced,
	}, nil
}

func (a *Allocator) AllocateNetwork(network *core.Network, id string) error {
	return a.allocateNetwork(network, id, false)
}

func (a *Allocator) allocateNetwork(network *core.Network, id string, dryRun bool) error {
	if !a.networkIDSynced() {
		return fmt.Errorf("allocator not ready")
	}

	vni, err := networkid.ParseVNI(id)
	if err != nil {
		return err
	}

	if vni < a.minVNI || vni > a.maxVNI {
		return ErrNotInRange
	}
	if dryRun {
		return nil
	}
	return a.createNetworkID(networkid.EncodeVNI(vni), network)
}

func (a *Allocator) AllocateNextNetwork(network *core.Network) (string, error) {
	return a.allocateNextNetwork(network, false)
}

func (a *Allocator) allocateNextNetwork(network *core.Network, dryRun bool) (string, error) {
	if !a.networkIDSynced() {
		return "", fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return networkid.EncodeVNI(a.minVNI), nil
	}

	trace := utiltrace.New("allocate dynamic NetworkID ID")
	defer trace.LogIfLong(500 * time.Millisecond)

	start := randomVNIIteratorStart(a.minVNI, a.maxVNI)
	iterator := vniIterator(a.minVNI, a.maxVNI, start)
	return a.allocateFromRange(iterator, network)
}

func (a *Allocator) allocateFromRange(iterator func() (int32, bool), network *core.Network) (string, error) {
	for {
		vni, ok := iterator()
		if !ok {
			return "", ErrFull
		}

		name := networkid.EncodeVNI(vni)
		_, err := a.networkIDLister.Get(name)
		if err == nil {
			continue
		}
		if !apierrors.IsNotFound(err) {
			klog.InfoS("Unexpected error", "err", err)
			continue
		}

		err = a.createNetworkID(name, network)
		if err != nil {
			klog.InfoS("Cannot create network ID", "name", name, "err", err)
			continue
		}

		return networkid.EncodeVNI(vni), nil
	}
}

func (a *Allocator) Release(id string) error {
	return a.release(id, false)
}

func (a *Allocator) release(id string, dryRun bool) error {
	if !a.networkIDSynced() {
		return fmt.Errorf("allocator not ready")
	}
	if dryRun {
		return nil
	}

	name := id
	err := a.client.NetworkIDs().Delete(context.Background(), name, metav1.DeleteOptions{})
	if err == nil {
		return nil
	}
	klog.InfoS("error releasing ID", "name", name, "err", err)
	return nil
}

func (a *Allocator) DryRun() Interface {
	return &dryRunAllocator{real: a}
}

func networkToRef(network *core.Network) v1alpha1.NetworkIDClaimRef {
	return v1alpha1.NetworkIDClaimRef{
		Group:     core.GroupName,
		Resource:  "networks",
		Namespace: network.Namespace,
		Name:      network.Name,
		UID:       network.UID,
	}
}

func randomVNIIteratorStart(minVNI, maxVNI int32) int32 {
	diff := maxVNI - minVNI
	n := rand.Int31n(diff)
	return minVNI + n
}

func vniIterator(minVNI, maxVNI, start int32) func() (int32, bool) {
	var (
		next = func(vni int32) int32 {
			if vni == maxVNI {
				return minVNI
			}
			return vni + 1
		}
		seen bool
		vni  = start
	)

	return func() (int32, bool) {
		value := vni
		if value == start {
			if seen {
				return 0, false
			}
			seen = true
		}
		vni = next(vni)
		return value, true
	}
}

func (a *Allocator) createNetworkID(name string, network *core.Network) error {
	networkID := v1alpha1.NetworkID{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.NetworkIDSpec{
			ClaimRef: networkToRef(network),
		},
	}
	_, err := a.client.NetworkIDs().Create(context.Background(), &networkID, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return ErrAllocated
		}
		return err
	}
	return nil
}

type dryRunAllocator struct {
	real *Allocator
}

func (dry dryRunAllocator) AllocateNetwork(network *core.Network, id string) error {
	return dry.real.allocateNetwork(network, id, true)
}

func (dry dryRunAllocator) AllocateNextNetwork(network *core.Network) (string, error) {
	return dry.real.allocateNextNetwork(network, true)
}

func (dry dryRunAllocator) Release(id string) error {
	return dry.real.release(id, true)
}

func (dry dryRunAllocator) DryRun() Interface {
	return dry
}

type Interface interface {
	AllocateNetwork(network *core.Network, id string) error
	AllocateNextNetwork(network *core.Network) (string, error)
	Release(id string) error
	DryRun() Interface
}
