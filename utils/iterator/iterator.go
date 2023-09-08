// Copyright 2023 OnMetal authors
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

package iterator

func OfSlice[S ~[]E, E any](s S) func(yield func(E) bool) bool {
	return func(yield func(E) bool) bool {
		for _, e := range s {
			if !yield(e) {
				return false
			}
		}
		return true
	}
}

func OfSliceRef[S ~[]E, E any](s S) func(yield func(*E) bool) bool {
	return func(yield func(*E) bool) bool {
		for i := range s {
			e := &s[i]
			if !yield(e) {
				return false
			}
		}
		return true
	}
}

func Map[I ~func(yield func(E) bool) bool, E, F any](it I, f func(E) F) func(yield func(F) bool) bool {
	return func(yield func(F) bool) bool {
		return it(func(e E) bool {
			return yield(f(e))
		})
	}
}

func Fold[I ~func(yield func(E) bool) bool, A, E any](it I, acc A, f func(A, E) A) A {
	it(func(e E) bool {
		acc = f(acc, e)
		return true
	})
	return acc
}

func Next[I ~func(yield func(E) bool) bool, E any](it I) (E, bool) {
	var (
		res E
		ok  bool
	)
	it(func(e E) bool {
		res = e
		ok = true
		return false
	})
	return res, ok
}

func Reduce[I ~func(yield func(E) bool) bool, E any](it I, f func(E, E) E) E {
	acc, ok := Next(it)
	if !ok {
		panic("iterator.Reduce: empty iterator")
	}
	it(func(e E) bool {
		acc = f(acc, e)
		return true
	})
	return acc
}

func Concat[I ~func(yield func(E) bool) bool, E any](is ...I) func(yield func(E) bool) bool {
	return func(yield func(E) bool) bool {
		for _, i := range is {
			if !i(yield) {
				return false
			}
		}
		return true
	}
}

func CollectSlice[I ~func(yield func(E) bool) bool, E any](it I) []E {
	var res []E
	it(func(e E) bool {
		res = append(res, e)
		return true
	})
	return res
}
