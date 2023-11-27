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

package expectations

import (
	"time"

	"github.com/ironcore-dev/ironcore/broker/common/sync"
	utilclient "github.com/ironcore-dev/ironcore/utils/client"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type expectation struct {
	timestamp time.Time
	delete    sets.Set[client.ObjectKey]
	create    sets.Set[client.ObjectKey]
}

type Expectations struct {
	timeout time.Duration

	entriesMu *sync.MutexMap[client.ObjectKey]
	entries   map[client.ObjectKey]*expectation
}

type NewOptions struct {
	Timeout time.Duration
}

func setNewOptionsDefaults(o *NewOptions) {
	if o.Timeout <= 0 {
		o.Timeout = 5 * time.Minute
	}
}

func (o *NewOptions) ApplyOptions(opts []NewOption) {
	for _, opt := range opts {
		opt.ApplyToNew(o)
	}
}

func (o *NewOptions) ApplyToNew(o2 *NewOptions) {
	if o.Timeout > 0 {
		o2.Timeout = o.Timeout
	}
}

type NewOption interface {
	ApplyToNew(o *NewOptions)
}

type WithTimeout time.Duration

func (w WithTimeout) ApplyToNew(o *NewOptions) {
	o.Timeout = time.Duration(w)
}

func New(opts ...NewOption) *Expectations {
	o := &NewOptions{}
	o.ApplyOptions(opts)
	setNewOptionsDefaults(o)

	return &Expectations{
		timeout:   o.Timeout,
		entriesMu: sync.NewMutexMap[client.ObjectKey](),
		entries:   make(map[client.ObjectKey]*expectation),
	}
}

func (e *Expectations) Delete(ctrlKey client.ObjectKey) {
	e.entriesMu.Lock(ctrlKey)
	defer e.entriesMu.Unlock(ctrlKey)

	delete(e.entries, ctrlKey)
}

func (e *Expectations) ExpectDeletions(ctrlKey client.ObjectKey, deletedKeys []client.ObjectKey) {
	e.entriesMu.Lock(ctrlKey)
	defer e.entriesMu.Unlock(ctrlKey)

	e.entries[ctrlKey] = &expectation{
		timestamp: time.Now(),
		delete:    sets.New(deletedKeys...),
	}
}

func (e *Expectations) ExpectCreations(ctrlKey client.ObjectKey, createdKeys []client.ObjectKey) {
	e.entriesMu.Lock(ctrlKey)
	defer e.entriesMu.Unlock(ctrlKey)

	e.entries[ctrlKey] = &expectation{
		timestamp: time.Now(),
		create:    sets.New(createdKeys...),
	}
}

func (e *Expectations) ExpectCreationsAndDeletions(ctrlKey client.ObjectKey, createdKeys, deletedKeys []client.ObjectKey) {
	e.entriesMu.Lock(ctrlKey)
	defer e.entriesMu.Unlock(ctrlKey)

	e.entries[ctrlKey] = &expectation{
		timestamp: time.Now(),
		create:    sets.New(createdKeys...),
		delete:    sets.New(deletedKeys...),
	}
}

func (e *Expectations) CreationObserved(ctrlKey, createdKey client.ObjectKey) {
	e.entriesMu.Lock(ctrlKey)
	defer e.entriesMu.Unlock(ctrlKey)

	exp, ok := e.entries[ctrlKey]
	if !ok {
		return
	}

	exp.create.Delete(createdKey)
}

func (e *Expectations) DeletionObserved(ctrlKey, deletedKey client.ObjectKey) {
	e.entriesMu.Lock(ctrlKey)
	defer e.entriesMu.Unlock(ctrlKey)

	exp, ok := e.entries[ctrlKey]
	if !ok {
		return
	}

	exp.delete.Delete(deletedKey)
}

func (e *Expectations) Satisfied(ctrlKey client.ObjectKey) bool {
	e.entriesMu.Lock(ctrlKey)
	defer e.entriesMu.Unlock(ctrlKey)

	exp, ok := e.entries[ctrlKey]
	if !ok {
		// We didn't record any expectation and are good to go.
		return true
	}
	if time.Since(exp.timestamp) > e.timeout {
		// Expectations timed out, release.
		return true
	}
	if exp.create.Len() == 0 && exp.delete.Len() == 0 {
		// All expectations satisfied
		return true
	}

	// There are still some pending expectations.
	return false
}

// TODO: Make all these constants configurable via dynamic options in GenerateCreateNames.
const (
	maxObjectNameLength               = validation.DNS1035LabelMaxLength
	noOfObjectGenerateNameRandomChars = 10
	maxGenerateNamePrefixLength       = maxObjectNameLength - noOfObjectGenerateNameRandomChars - 1 // -1 for the '-'
)

func GenerateCreateNames(name string, ct int) []string {
	prefix := name
	if len(prefix) > maxGenerateNamePrefixLength {
		prefix = prefix[:maxGenerateNamePrefixLength]
	}
	prefix = prefix + "-"

	names := sets.New[string]()
	for names.Len() < ct {
		name := prefix + utilrand.String(noOfObjectGenerateNameRandomChars)
		names.Insert(name)
	}
	return names.UnsortedList()
}

func ObjectKeysFromNames(namespace string, names []string) []client.ObjectKey {
	keys := make([]client.ObjectKey, len(names))
	for i, name := range names {
		keys[i] = client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
	}
	return keys
}

func ObjectKeysFromObjectStructSlice[O utilclient.Object[OStruct], S ~[]OStruct, OStruct any](objs S) []client.ObjectKey {
	keys := make([]client.ObjectKey, len(objs))
	for i, obj := range objs {
		keys[i] = client.ObjectKeyFromObject(O(&obj))
	}
	return keys
}
