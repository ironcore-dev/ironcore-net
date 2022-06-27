// Copyright 2022 OnMetal authors
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

package allocator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	commonv1alpha1 "github.com/onmetal/onmetal-api/apis/common/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/apis/networking/v1alpha1"
	"inet.af/netaddr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(networkingv1alpha1.AddToScheme(scheme))
}

// Allocator allows allocating IPs of a given family
type Allocator interface {
	// Allocate allocates an ip of the target family for the given id.
	Allocate(ctx context.Context, id string, ipFamily corev1.IPFamily) (commonv1alpha1.IP, error)
	// List lists all currently active allocations.
	List(ctx context.Context) ([]Allocation, error)
	// Release releases all allocations for the corresponding id.
	Release(ctx context.Context, id string) error
}

// Allocation is an ip allocation.
type Allocation struct {
	// ID is the id of the allocation.
	ID string
	// IP is the allocated ip.
	IP commonv1alpha1.IP
}

// SecretAllocator is an Allocator that manages its allocations in a central secret.
type SecretAllocator struct {
	client client.Client

	ipv4Set *netaddr.IPSet
	ipv6Set *netaddr.IPSet

	secretKey client.ObjectKey
}

// Options are options for creating an Allocator.
type Options struct {
	// SecretKey is the key of the secret to manage allocations in.
	SecretKey client.ObjectKey
	// IPv4Set is the root set of ipv4 addresses to allocate from. If nil, no ipv4 addresses can be allocated.
	IPv4Set *netaddr.IPSet
	// IPv6Set is the root set of ipv6 addresses to allocate from. If nil, no ipv6 addresses can be allocated.
	IPv6Set *netaddr.IPSet
}

// NewSecretAllocator creates a new Allocator.
func NewSecretAllocator(cfg *rest.Config, opts Options) (*SecretAllocator, error) {
	if opts.SecretKey == (client.ObjectKey{}) {
		return nil, fmt.Errorf("must specify secret key")
	}
	if opts.IPv4Set == nil && opts.IPv6Set == nil {
		return nil, fmt.Errorf("must specify at least one ip set (v4 / v6)")
	}

	c, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	var ipV4Set *netaddr.IPSet
	if opts.IPv4Set != nil {
		set := *opts.IPv4Set
		ipV4Set = &set
	}

	var ipV6Set *netaddr.IPSet
	if opts.IPv6Set != nil {
		set := *opts.IPv4Set
		ipV6Set = &set
	}

	return &SecretAllocator{
		client:    c,
		ipv4Set:   ipV4Set,
		ipv6Set:   ipV6Set,
		secretKey: opts.SecretKey,
	}, nil
}

func decodeAllocatorSecret(secret *corev1.Secret) (*secretState, error) {
	stateData, ok := secret.Data[stateField]
	if !ok {
		return &secretState{
			Allocations: make(map[string]commonv1alpha1.IP),
		}, nil
	}

	state := &secretState{}
	if err := json.Unmarshal(stateData, state); err != nil {
		return nil, err
	}
	if state.Allocations == nil {
		state.Allocations = make(map[string]commonv1alpha1.IP)
	}
	return state, nil
}

func encodeAllocatorSecret(secret *corev1.Secret, state *secretState) error {
	stateData, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[stateField] = stateData

	return nil
}

type secretState struct {
	Allocations map[string]commonv1alpha1.IP `json:"allocations"`
}

const (
	fieldOwner = client.FieldOwner("net.networking.api.onmetal.de/secret-allocator")

	stateField = "state"
)

func (s *SecretAllocator) load(ctx context.Context) (*corev1.Secret, *secretState, error) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.secretKey.Namespace,
			Name:      s.secretKey.Name,
		},
	}
	if err := s.client.Patch(ctx, secret, client.Apply, fieldOwner); err != nil {
		return nil, nil, err
	}

	state, err := decodeAllocatorSecret(secret)
	if err != nil {
		return nil, nil, err
	}

	return secret, state, err
}

func (s *SecretAllocator) write(ctx context.Context, secret *corev1.Secret, state *secretState) error {
	base := secret.DeepCopy()

	if err := encodeAllocatorSecret(secret, state); err != nil {
		return err
	}

	if err := s.client.Patch(ctx, secret, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{})); err != nil {
		return err
	}
	return nil
}

func (s *SecretAllocator) ipSetFor(ipFamily corev1.IPFamily) (*netaddr.IPSet, bool) {
	switch ipFamily {
	case corev1.IPv4Protocol:
		return s.ipv4Set, s.ipv4Set != nil
	case corev1.IPv6Protocol:
		return s.ipv6Set, s.ipv6Set != nil
	default:
		return nil, false
	}
}

var (
	// ErrCannotHandleIPFamily indicates that the ip family cannot be handled.
	ErrCannotHandleIPFamily = errors.New("cannot handle ip family")
	// ErrNoSpaceLeft indicates that currently there is no space left for allocating an IP.
	ErrNoSpaceLeft = errors.New("no space left")
)

func ipFamilyBitLength(ipFamily corev1.IPFamily) uint8 {
	switch ipFamily {
	case corev1.IPv4Protocol:
		return 32
	case corev1.IPv6Protocol:
		return 128
	default:
		panic(fmt.Sprintf("invalid ip family %q", ipFamily))
	}
}

// Allocate implements allocator.
func (s *SecretAllocator) Allocate(ctx context.Context, id string, ipFamily corev1.IPFamily) (commonv1alpha1.IP, error) {
	ipSet, ok := s.ipSetFor(ipFamily)
	if !ok {
		return commonv1alpha1.IP{}, ErrCannotHandleIPFamily
	}

	secret, state, err := s.load(ctx)
	if err != nil {
		return commonv1alpha1.IP{}, err
	}

	allocation, ok := state.Allocations[id]
	if ok && allocation.Family() == ipFamily {
		return allocation, nil
	}

	var availableBldr netaddr.IPSetBuilder
	availableBldr.AddSet(ipSet)
	for _, allocation := range state.Allocations {
		availableBldr.Remove(allocation.IP)
	}

	available, _ := availableBldr.IPSet()
	allocatedPrefix, _, ok := available.RemoveFreePrefix(ipFamilyBitLength(ipFamily))
	if !ok {
		return commonv1alpha1.IP{}, ErrNoSpaceLeft
	}

	allocated := commonv1alpha1.IP{IP: allocatedPrefix.IP()}
	state.Allocations[id] = allocated
	if err := s.write(ctx, secret, state); err != nil {
		return commonv1alpha1.IP{}, err
	}

	return allocated, nil
}

// List implements Allocator.
func (s *SecretAllocator) List(ctx context.Context) ([]Allocation, error) {
	_, state, err := s.load(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]Allocation, 0, len(state.Allocations))
	for id, ip := range state.Allocations {
		res = append(res, Allocation{
			ID: id,
			IP: ip,
		})
	}
	return res, nil
}

// Release implements Allocator.
func (s *SecretAllocator) Release(ctx context.Context, id string) error {
	secret, state, err := s.load(ctx)
	if err != nil {
		return err
	}

	delete(state.Allocations, id)
	return s.write(ctx, secret, state)
}
