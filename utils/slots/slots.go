// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package slots

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/bits-and-blooms/bitset"
)

type Slots struct {
	mu sync.RWMutex

	count    int
	set      *bitset.BitSet
	strategy NextClearStrategy
}

type randomNextClearStrategyWithOffset struct {
	offset int
}

func (s randomNextClearStrategyWithOffset) NextClear(set *bitset.BitSet) (int, bool) {
	subLen := int(set.Len()) - s.offset
	start := rand.Intn(subLen)
	for i := 0; i < subLen; i++ {
		idx := s.offset + ((start + i) % subLen)
		if !set.Test(uint(idx)) {
			return idx, true
		}
	}

	start = rand.Intn(s.offset)
	for i := 0; i < s.offset; i++ {
		idx := s.offset + ((start + i) % s.offset)
		if !set.Test(uint(idx)) {
			return idx, true
		}
	}
	return 0, false
}

type NextClearStrategy interface {
	NextClear(set *bitset.BitSet) (int, bool)
}

type Options struct {
	Offset int
}

func (o *Options) ApplyOptions(opts []Option) *Options {
	for _, opt := range opts {
		opt.ApplyTo(o)
	}
	return o
}

type Option interface {
	ApplyTo(opts *Options)
}

type WithOffset int

func (o WithOffset) ApplyTo(opts *Options) {
	opts.Offset = int(o)
}

func New(len int, opts ...Option) *Slots {
	if len < 0 {
		panic("slots.New: cannot provide negative length")
	}

	o := (&Options{}).ApplyOptions(opts)

	return &Slots{
		count:    0,
		set:      bitset.New(uint(len)),
		strategy: randomNextClearStrategyWithOffset{offset: o.Offset},
	}
}

func (s *Slots) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count = 0
	s.set.ClearAll()
}

func (s *Slots) Allocate(index int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= int(s.set.Len()) {
		return false, fmt.Errorf("index %d out of range [0,%d]", index, s.set.Len())
	}

	if s.set.Test(uint(index)) {
		return false, nil
	}

	s.set.Set(uint(index))
	s.count++
	return true, nil
}

func (s *Slots) AllocateNext() (int, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.count >= int(s.set.Len()) {
		return 0, false, nil
	}

	idx, ok := s.strategy.NextClear(s.set)
	if !ok {
		return 0, false, nil
	}

	s.set.Set(uint(idx))
	s.count++
	return idx, true, nil
}

// Release releases an index. Releasing an unallocated or out-of-range index is considered a no-op.
func (s *Slots) Release(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 {
		return nil
	}

	if !s.set.Test(uint(index)) {
		return nil
	}

	s.set.Clear(uint(index))
	s.count--
	return nil
}

func (s *Slots) Has(index int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 {
		return false
	}
	return s.set.Test(uint(index))
}

func (s *Slots) Iterate(f func(int) bool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := uint(0); i < s.set.Len(); i++ {
		if s.set.Test(i) {
			if !f(int(i)) {
				return false
			}
		}
	}
	return true
}

func (s *Slots) Free() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return int(s.set.Len()) - s.count
}
