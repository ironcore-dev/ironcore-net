// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"

	"github.com/ironcore-dev/ironcore-net/utils/origin"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func EnqueueRequestByOrigin(origin *origin.Origin) handler.EventHandler {
	return &enqueueRequestByOrigin{
		origin: origin,
	}
}

type enqueueRequestByOrigin struct {
	origin *origin.Origin
}

func (e *enqueueRequestByOrigin) addRequests(
	obj client.Object,
	reqs sets.Set[ctrl.Request],
) {
	data := e.origin.DataOf(obj)
	if data == nil {
		return
	}

	reqs.Insert(ctrl.Request{NamespacedName: client.ObjectKey{Namespace: data.Namespace, Name: data.Name}})
}

func (e *enqueueRequestByOrigin) enqueueRequests(reqs sets.Set[ctrl.Request], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	for req := range reqs {
		queue.Add(req)
	}
}

func (e *enqueueRequestByOrigin) Create(ctx context.Context, evt event.CreateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	reqs := sets.New[ctrl.Request]()
	e.addRequests(evt.Object, reqs)
	e.enqueueRequests(reqs, queue)
}

func (e *enqueueRequestByOrigin) Update(ctx context.Context, evt event.UpdateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	reqs := sets.New[ctrl.Request]()
	e.addRequests(evt.ObjectNew, reqs)
	e.enqueueRequests(reqs, queue)
}

func (e *enqueueRequestByOrigin) Delete(ctx context.Context, evt event.DeleteEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	reqs := sets.New[ctrl.Request]()
	e.addRequests(evt.Object, reqs)
	e.enqueueRequests(reqs, queue)
}

func (e *enqueueRequestByOrigin) Generic(ctx context.Context, evt event.GenericEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	reqs := sets.New[ctrl.Request]()
	e.addRequests(evt.Object, reqs)
	e.enqueueRequests(reqs, queue)
}
