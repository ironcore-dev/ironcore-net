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

package container

import "k8s.io/apimachinery/pkg/util/sets"

type MapIndex[K comparable, V any] interface {
	Update(k K, old, new V)
	Put(k K, v V)
	Delete(k K, v V)
}

type ReverseMapIndex[K, V comparable] map[V]sets.Set[K]

func (r ReverseMapIndex[K, V]) Update(k K, old, new V) {
	if old == new {
		return
	}

	r.Delete(k, old)
	r.Put(k, new)
}

func (r ReverseMapIndex[K, V]) Put(k K, v V) {
	keys := r[v]
	if keys == nil {
		keys = sets.New[K]()
		r[v] = keys
	}
	keys.Insert(k)
}

func (r ReverseMapIndex[K, V]) Delete(k K, v V) {
	keys := r[v]
	keys.Delete(k)
	if keys.Len() == 0 {
		delete(r, v)
	}
}

func (r ReverseMapIndex[K, V]) Get(v V) sets.Set[K] {
	return r[v]
}

type IndexingMap[K comparable, V any] struct {
	entries map[K]V
	indices []MapIndex[K, V]
}

func (r *IndexingMap[K, V]) forEachIndex(f func(MapIndex[K, V])) {
	if r == nil {
		return
	}

	for _, idx := range r.indices {
		f(idx)
	}
}

func (r *IndexingMap[K, V]) AddIndex(idx ...MapIndex[K, V]) {
	if r == nil {
		panic("AddIndex on nil IndexingMap")
	}

	for k, v := range r.entries {
		for _, idx := range idx {
			idx.Put(k, v)
		}
	}
	r.indices = append(r.indices, idx...)
}

func (r *IndexingMap[K, V]) Put(k K, v V) {
	if r == nil {
		panic("Put on nil IndexingMap")
	}

	if r.entries == nil {
		r.entries = make(map[K]V)
	}
	oldV, ok := r.entries[k]
	r.entries[k] = v
	if ok {
		r.forEachIndex(func(idx MapIndex[K, V]) {
			idx.Update(k, oldV, v)
		})
	} else {
		r.forEachIndex(func(idx MapIndex[K, V]) {
			idx.Put(k, v)
		})
	}
}

func (r *IndexingMap[K, V]) Delete(k K) {
	if r == nil {
		return
	}
	v, ok := r.entries[k]
	if !ok {
		return
	}
	r.forEachIndex(func(idx MapIndex[K, V]) {
		idx.Delete(k, v)
	})
}

func (r *IndexingMap[K, V]) Get(k K) (V, bool) {
	if r == nil {
		var zero V
		return zero, false
	}

	v, ok := r.entries[k]
	return v, ok
}

func (r *IndexingMap[K, V]) Range(f func(K, V) bool) {
	if r == nil {
		return
	}

	for k, v := range r.entries {
		if !f(k, v) {
			return
		}
	}
}
