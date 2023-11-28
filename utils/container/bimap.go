// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package container

type BiMap[K, V comparable] struct {
	forward map[K]V
	inverse map[V]K
}

type BiMapOptions[K, V comparable] struct {
	Len *int
}

func (o *BiMapOptions[K, V]) ApplyToBiMap(o2 *BiMapOptions[K, V]) {
	if o.Len != nil {
		o2.Len = o.Len
	}
}

func (o *BiMapOptions[K, V]) ApplyOptions(opts []BiMapOption[K, V]) *BiMapOptions[K, V] {
	for _, opt := range opts {
		opt.ApplyToBiMap(o)
	}
	return o
}

type BiMapOption[K, V comparable] interface {
	ApplyToBiMap(o *BiMapOptions[K, V])
}

type WithLen[K, V comparable] int

func (w WithLen[K, V]) ApplyToBiMap(o *BiMapOptions[K, V]) {
	o.Len = (*int)(&w)
}

func NewBiMap[K, V comparable](opts ...BiMapOption[K, V]) *BiMap[K, V] {
	o := (&BiMapOptions[K, V]{}).ApplyOptions(opts)

	var (
		forward map[K]V
		inverse map[V]K
	)
	if o.Len != nil {
		forward = make(map[K]V, *o.Len)
		inverse = make(map[V]K, *o.Len)
	} else {
		forward = make(map[K]V)
		inverse = make(map[V]K)
	}

	return &BiMap[K, V]{
		forward: forward,
		inverse: inverse,
	}
}

func (b *BiMap[K, V]) Insert(k K, v V) {
	b.forward[k] = v
	b.inverse[v] = k
}

func (b *BiMap[K, V]) Delete(k K) {
	v, ok := b.forward[k]
	if !ok {
		return
	}
	delete(b.forward, k)
	delete(b.inverse, v)
}

func (b *BiMap[K, V]) Has(k K) bool {
	_, ok := b.forward[k]
	return ok
}

func (b *BiMap[K, V]) Get(k K) (V, bool) {
	v, ok := b.forward[k]
	return v, ok
}

func (b *BiMap[K, V]) GetValue(k K) V {
	return b.forward[k]
}

func (b *BiMap[K, V]) Range(f func(K, V) bool) {
	for k, v := range b.forward {
		if !f(k, v) {
			return
		}
	}
}

func (b *BiMap[K, V]) Inverse() *BiMap[V, K] {
	return &BiMap[V, K]{
		forward: b.inverse,
		inverse: b.forward,
	}
}
