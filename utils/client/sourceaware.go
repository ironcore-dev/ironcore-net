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

package client

import (
	"github.com/onmetal/onmetal-api-net/utils/api"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type SourceAwareSystem struct {
	api.SourceAwareSystem
}

func NewSourceAwareSystemE(system api.SourceAwareSystem) (*SourceAwareSystem, error) {
	return &SourceAwareSystem{system}, nil
}

func NewSourceAwareSystem(system api.SourceAwareSystem) *SourceAwareSystem {
	s, err := NewSourceAwareSystemE(system)
	if err != nil {
		panic(err)
	}
	return s
}

func (s *SourceAwareSystem) SourceLabelKeysE(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object) ([]string, error) {
	sourceGVK, err := apiutil.GVKForObject(sourceObj, scheme)
	if err != nil {
		return nil, err
	}

	sourceNamespaced, err := apiutil.IsGVKNamespaced(sourceGVK, mapper)
	if err != nil {
		return nil, err
	}

	keys := []string{
		s.SourceUIDLabel(sourceGVK.Kind),
		s.SourceNameLabel(sourceGVK.Kind),
	}
	if sourceNamespaced {
		keys = append(keys, s.SourceNamespaceLabel(sourceGVK.Kind))
	}

	return keys, nil
}

func (s *SourceAwareSystem) SourceLabelKeys(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object) []string {
	keys, err := s.SourceLabelKeysE(scheme, mapper, sourceObj)
	if err != nil {
		panic(err)
	}
	return keys
}

func (s *SourceAwareSystem) SourceLabelsE(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object) (map[string]string, error) {
	sourceGVK, err := apiutil.GVKForObject(sourceObj, scheme)
	if err != nil {
		return nil, err
	}

	sourceNamespaced, err := apiutil.IsGVKNamespaced(sourceGVK, mapper)
	if err != nil {
		return nil, err
	}

	lbls := map[string]string{
		s.SourceUIDLabel(sourceGVK.Kind):  string(sourceObj.GetUID()),
		s.SourceNameLabel(sourceGVK.Kind): sourceObj.GetName(),
	}
	if sourceNamespaced {
		lbls[s.SourceNamespaceLabel(sourceGVK.Kind)] = sourceObj.GetNamespace()
	}

	return lbls, nil
}

func (s *SourceAwareSystem) SourceLabels(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object) map[string]string {
	lbls, err := s.SourceLabelsE(scheme, mapper, sourceObj)
	if err != nil {
		panic(err)
	}
	return lbls
}

func (s *SourceAwareSystem) MatchingSourceLabelsE(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object) (client.MatchingLabels, error) {
	lbls, err := s.SourceLabelsE(scheme, mapper, sourceObj)
	if err != nil {
		return nil, err
	}

	return lbls, nil
}

func (s *SourceAwareSystem) MatchingSourceLabels(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object) client.MatchingLabels {
	matchingLbls, err := s.MatchingSourceLabelsE(scheme, mapper, sourceObj)
	if err != nil {
		panic(err)
	}

	return matchingLbls
}

func (s *SourceAwareSystem) HasSourceLabelsE(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj, obj client.Object) (bool, error) {
	sourceLabels, err := s.SourceLabelsE(scheme, mapper, sourceObj)
	if err != nil {
		return false, err
	}

	sel := labels.SelectorFromSet(sourceLabels)
	return sel.Matches(labels.Set(obj.GetLabels())), nil
}

func (s *SourceAwareSystem) HasSourceLabels(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj, obj client.Object) bool {
	ok, err := s.HasSourceLabelsE(scheme, mapper, sourceObj, obj)
	if err != nil {
		panic(err)
	}
	return ok
}

func (s *SourceAwareSystem) SourceKeyLabelsE(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceKey client.ObjectKey, sourceObj client.Object) (map[string]string, error) {
	sourceGVK, err := apiutil.GVKForObject(sourceObj, scheme)
	if err != nil {
		return nil, err
	}

	sourceNamespaced, err := apiutil.IsGVKNamespaced(sourceGVK, mapper)
	if err != nil {
		return nil, err
	}

	lbls := map[string]string{
		s.SourceNameLabel(sourceGVK.Kind): sourceKey.Name,
	}
	if sourceNamespaced {
		lbls[s.SourceNamespaceLabel(sourceGVK.Kind)] = sourceKey.Namespace
	}

	return lbls, nil
}

func (s *SourceAwareSystem) SourceKeyLabels(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceKey client.ObjectKey, sourceObj client.Object) map[string]string {
	lbls, err := s.SourceKeyLabelsE(scheme, mapper, sourceKey, sourceObj)
	if err != nil {
		panic(err)
	}

	return lbls
}

func (s *SourceAwareSystem) MatchingSourceKeyLabelsE(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceKey client.ObjectKey, sourceObj client.Object) (client.MatchingLabels, error) {
	lbls, err := s.SourceKeyLabelsE(scheme, mapper, sourceKey, sourceObj)
	if err != nil {
		return nil, err
	}

	return lbls, nil
}

func (s *SourceAwareSystem) MatchingSourceKeyLabels(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceKey client.ObjectKey, sourceObj client.Object) client.MatchingLabels {
	lbls, err := s.MatchingSourceKeyLabelsE(scheme, mapper, sourceKey, sourceObj)
	if err != nil {
		panic(err)
	}
	return lbls
}

func (s *SourceAwareSystem) SourceObjectKeyFromObjectE(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object, obj client.Object) (*client.ObjectKey, error) {
	sourceGVK, err := apiutil.GVKForObject(sourceObj, scheme)
	if err != nil {
		return nil, err
	}

	sourceNamespaced, err := apiutil.IsGVKNamespaced(sourceGVK, mapper)
	if err != nil {
		return nil, err
	}

	lbls := obj.GetLabels()

	var namespace string
	if sourceNamespaced {
		var ok bool
		namespace, ok = lbls[s.SourceNamespaceLabel(sourceGVK.Kind)]
		if !ok {
			return nil, nil
		}
	}

	name, ok := lbls[s.SourceNameLabel(sourceGVK.Kind)]
	if !ok {
		return nil, nil
	}

	return &client.ObjectKey{Namespace: namespace, Name: name}, nil
}

func (s *SourceAwareSystem) SourceObjectKeyFromObject(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object, obj client.Object) *client.ObjectKey {
	key, err := s.SourceObjectKeyFromObjectE(scheme, mapper, sourceObj, obj)
	if err != nil {
		panic(err)
	}
	return key
}

type SourceObjectData struct {
	Namespace string
	Name      string
	UID       types.UID
}

func (s *SourceAwareSystem) SourceObjectDataFromObjectE(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object, obj client.Object) (*SourceObjectData, error) {
	sourceGVK, err := apiutil.GVKForObject(sourceObj, scheme)
	if err != nil {
		return nil, err
	}

	sourceNamespaced, err := apiutil.IsGVKNamespaced(sourceGVK, mapper)
	if err != nil {
		return nil, err
	}

	lbls := obj.GetLabels()

	var namespace string
	if sourceNamespaced {
		var ok bool
		namespace, ok = lbls[s.SourceNamespaceLabel(sourceGVK.Kind)]
		if !ok {
			return nil, nil
		}
	}

	name, ok := lbls[s.SourceNameLabel(sourceGVK.Kind)]
	if !ok {
		return nil, nil
	}

	uid, ok := lbls[s.SourceUIDLabel(sourceGVK.Kind)]
	if !ok {
		return nil, nil
	}

	return &SourceObjectData{
		Namespace: namespace,
		Name:      name,
		UID:       types.UID(uid),
	}, nil
}

func (s *SourceAwareSystem) SourceObjectDataFromObject(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object, obj client.Object) *SourceObjectData {
	data, err := s.SourceObjectDataFromObjectE(scheme, mapper, sourceObj, obj)
	if err != nil {
		panic(err)
	}
	return data
}
