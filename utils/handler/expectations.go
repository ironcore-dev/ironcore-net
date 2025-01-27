// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/ironcore-net/utils/expectations"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var observeExpectationsForControllerLog = ctrl.Log.WithName("eventhandler").WithName("observeExpectationsForController")

type observeExpectationsForController struct {
	controllerType client.Object
	expectations   *expectations.Expectations

	groupKind schema.GroupKind

	mapper meta.RESTMapper
}

func ObserveExpectationsForController(
	scheme *runtime.Scheme,
	mapper meta.RESTMapper,
	controllerType client.Object,
	expectations *expectations.Expectations,
) handler.EventHandler {
	o := &observeExpectationsForController{
		controllerType: controllerType,
		expectations:   expectations,
		mapper:         mapper,
	}
	if err := o.parseControllerTypeGroupKind(scheme); err != nil {
		panic(err)
	}
	return o
}

func (o *observeExpectationsForController) parseControllerTypeGroupKind(scheme *runtime.Scheme) error {
	gvk, err := apiutil.GVKForObject(o.controllerType, scheme)
	if err != nil {
		return err
	}

	o.groupKind = gvk.GroupKind()
	return nil
}

func (o *observeExpectationsForController) getControllerKey(object metav1.Object) (*client.ObjectKey, error) {
	ctrl := metav1.GetControllerOf(object)
	if ctrl == nil {
		return nil, nil
	}

	ctrlGV, err := schema.ParseGroupVersion(ctrl.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("error parsing controller APIVersion: %w", err)
	}

	if ctrl.Kind != o.groupKind.Kind || ctrlGV.Group != o.groupKind.Group {
		return nil, nil
	}

	key := client.ObjectKey{Name: ctrl.Name}

	mapping, err := o.mapper.RESTMapping(o.groupKind, ctrlGV.Version)
	if err != nil {
		return nil, fmt.Errorf("error retrieving rest mapping: %w", err)
	}
	if mapping.Scope.Name() != meta.RESTScopeNameRoot {
		key.Namespace = object.GetNamespace()
	}

	return &key, nil
}

func (o *observeExpectationsForController) delete(obj client.Object) {
	ctrlKey, err := o.getControllerKey(obj)
	if err != nil {
		observeExpectationsForControllerLog.Error(err, "Error getting controller key")
		return
	}
	if ctrlKey == nil {
		return
	}

	observeExpectationsForControllerLog.V(4).Info("Deletion observed")
	o.expectations.DeletionObserved(*ctrlKey, client.ObjectKeyFromObject(obj))
}

func (o *observeExpectationsForController) add(obj client.Object) {
	if !obj.GetDeletionTimestamp().IsZero() {
		o.delete(obj)
		return
	}

	ctrlKey, err := o.getControllerKey(obj)
	if err != nil {
		observeExpectationsForControllerLog.Error(err, "Error getting controller key")
		return
	}
	if ctrlKey == nil {
		return
	}
	observeExpectationsForControllerLog.V(4).Info("Creation observed")
	o.expectations.CreationObserved(*ctrlKey, client.ObjectKeyFromObject(obj))
}

func (o *observeExpectationsForController) Create(_ context.Context, evt event.CreateEvent, _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	o.add(evt.Object)
}

func (o *observeExpectationsForController) Update(_ context.Context, _ event.UpdateEvent, _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
}

func (o *observeExpectationsForController) Delete(_ context.Context, evt event.DeleteEvent, _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	o.delete(evt.Object)
}

func (o *observeExpectationsForController) Generic(_ context.Context, evt event.GenericEvent, _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	o.add(evt.Object)
}
