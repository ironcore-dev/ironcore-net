// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
