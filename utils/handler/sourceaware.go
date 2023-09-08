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

package handler

import (
	"context"
	"fmt"

	utilapi "github.com/onmetal/onmetal-api-net/utils/api"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

var log = ctrl.Log.WithName("eventhandler").WithName("enqueueRequestForSource")

type SourceAwareSystem struct {
	utilapi.SourceAwareSystem
}

func NewSourceAwareSystem(system utilapi.SourceAwareSystem) *SourceAwareSystem {
	return &SourceAwareSystem{
		system,
	}
}

func (s *SourceAwareSystem) EnqueueRequestForSource(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object) handler.EventHandler {
	gvk, err := apiutil.GVKForObject(sourceObj, scheme)
	if err != nil {
		err = fmt.Errorf("error determining source kind: %w", err)
		panic(err)
	}

	return &enqueueRequestForSource{
		SourceAwareSystem: s.SourceAwareSystem,
		gvk:               gvk,
		mapper:            mapper,
	}
}

type enqueueRequestForSource struct {
	utilapi.SourceAwareSystem
	gvk    schema.GroupVersionKind
	mapper meta.RESTMapper
}

func EnqueueRequestForSource(scheme *runtime.Scheme, mapper meta.RESTMapper, sourceObj client.Object) handler.EventHandler {
	gvk, err := apiutil.GVKForObject(sourceObj, scheme)
	if err != nil {
		err = fmt.Errorf("error determining source kind: %w", err)
		panic(err)
	}

	return &enqueueRequestForSource{
		gvk:    gvk,
		mapper: mapper,
	}
}

func (e *enqueueRequestForSource) getLabels() (namespaceLabel, nameLabel string, err error) {
	mapping, err := e.mapper.RESTMapping(e.gvk.GroupKind(), e.gvk.Version)
	if err != nil {
		return "", "", err
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		namespaceLabel = e.SourceNamespaceLabel(e.gvk.Kind)
	}
	nameLabel = e.SourceNameLabel(e.gvk.Kind)
	return namespaceLabel, nameLabel, nil
}

func (e *enqueueRequestForSource) addRequests(
	obj client.Object,
	namespaceLabel, nameLabel string,
	reqs sets.Set[ctrl.Request],
) {
	var namespace string
	if namespaceLabel != "" {
		var ok bool
		namespace, ok = obj.GetLabels()[namespaceLabel]
		if !ok {
			return
		}
	}

	name, ok := obj.GetLabels()[nameLabel]
	if !ok {
		return
	}

	reqs.Insert(ctrl.Request{NamespacedName: client.ObjectKey{Namespace: namespace, Name: name}})
}

func (e *enqueueRequestForSource) enqueueRequests(reqs sets.Set[ctrl.Request], queue workqueue.RateLimitingInterface) {
	for req := range reqs {
		queue.Add(req)
	}
}

func (e *enqueueRequestForSource) Create(ctx context.Context, evt event.CreateEvent, queue workqueue.RateLimitingInterface) {
	namespaceLabel, nameLabel, err := e.getLabels()
	if err != nil {
		log.Error(err, "Error getting labels")
		return
	}

	reqs := sets.New[ctrl.Request]()
	e.addRequests(evt.Object, namespaceLabel, nameLabel, reqs)
	e.enqueueRequests(reqs, queue)
}

func (e *enqueueRequestForSource) Update(ctx context.Context, evt event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	namespaceLabel, nameLabel, err := e.getLabels()
	if err != nil {
		log.Error(err, "Error getting labels")
		return
	}

	reqs := sets.New[ctrl.Request]()
	e.addRequests(evt.ObjectOld, namespaceLabel, nameLabel, reqs)
	e.addRequests(evt.ObjectNew, namespaceLabel, nameLabel, reqs)
	e.enqueueRequests(reqs, queue)
}

func (e *enqueueRequestForSource) Delete(ctx context.Context, evt event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	namespaceLabel, nameLabel, err := e.getLabels()
	if err != nil {
		log.Error(err, "Error getting labels")
		return
	}

	reqs := sets.New[ctrl.Request]()
	e.addRequests(evt.Object, namespaceLabel, nameLabel, reqs)
	e.enqueueRequests(reqs, queue)
}

func (e *enqueueRequestForSource) Generic(ctx context.Context, evt event.GenericEvent, queue workqueue.RateLimitingInterface) {
	namespaceLabel, nameLabel, err := e.getLabels()
	if err != nil {
		log.Error(err, "Error getting labels")
		return
	}

	reqs := sets.New[ctrl.Request]()
	e.addRequests(evt.Object, namespaceLabel, nameLabel, reqs)
	e.enqueueRequests(reqs, queue)
}
