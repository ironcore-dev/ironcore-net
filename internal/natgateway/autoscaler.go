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

package natgateway

import (
	"sync"

	"github.com/ironcore-dev/ironcore-net/utils/container"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NetworkSelector interface {
	MatchesNetwork(name string) bool
}

type networkSelector string

func (sel networkSelector) MatchesNetwork(name string) bool {
	return name == string(sel)
}

func NoVNISelector() NetworkSelector {
	return networkSelector("")
}

func SelectNetwork(name string) NetworkSelector {
	return networkSelector(name)
}

type AutoscalerSelectors struct {
	mu sync.RWMutex

	selectorByKey       *container.IndexingMap[client.ObjectKey, NetworkSelector]
	keysBySelectorIndex container.ReverseMapIndex[client.ObjectKey, NetworkSelector]
}

func NewAutoscalerSelectors() *AutoscalerSelectors {
	var (
		selectorByKey       container.IndexingMap[client.ObjectKey, NetworkSelector]
		keysBySelectorIndex = make(container.ReverseMapIndex[client.ObjectKey, NetworkSelector])
	)
	selectorByKey.AddIndex(keysBySelectorIndex)

	return &AutoscalerSelectors{
		selectorByKey:       &selectorByKey,
		keysBySelectorIndex: keysBySelectorIndex,
	}
}

func (m *AutoscalerSelectors) Put(key client.ObjectKey, sel NetworkSelector) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.selectorByKey.Put(key, sel)
}

func (m *AutoscalerSelectors) PutIfNotPresent(key client.ObjectKey, sel NetworkSelector) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.selectorByKey.Get(key); !ok {
		m.selectorByKey.Put(key, sel)
	}
}

func (m *AutoscalerSelectors) ReverseSelect(name string) sets.Set[client.ObjectKey] {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.keysBySelectorIndex.Get(SelectNetwork(name))
}

func (m *AutoscalerSelectors) Delete(key client.ObjectKey) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.selectorByKey.Delete(key)
}
