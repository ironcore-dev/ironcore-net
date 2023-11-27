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

package controller

import (
	"context"
	"sync"

	"github.com/ironcore-dev/ironcore/utils/generic"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func MatchLabelSelectorFunc[O client.Object](sel labels.Selector) func(O) bool {
	return func(obj O) bool {
		return sel.Matches(labels.Set(obj.GetLabels()))
	}
}

type RefManager[O client.Object] struct {
	client     client.Client
	controller client.Object
	match      func(O) bool

	gvkOnce sync.Once
	gvk     schema.GroupVersionKind
	gvkErr  error
}

func NewRefManager[O client.Object](c client.Client, controller client.Object, match func(O) bool) *RefManager[O] {
	return &RefManager[O]{
		client:     c,
		controller: controller,
		match:      match,
	}
}

func (r *RefManager[O]) getGVK() (schema.GroupVersionKind, error) {
	r.gvkOnce.Do(func() {
		r.gvk, r.gvkErr = apiutil.GVKForObject(r.controller, r.client.Scheme())
	})
	return r.gvk, r.gvkErr
}

func (r *RefManager[O]) adopt(ctx context.Context, obj O) error {
	gvk, err := r.getGVK()
	if err != nil {
		return err
	}

	base := obj.DeepCopyObject().(client.Object)
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               r.controller.GetName(),
		UID:                r.controller.GetUID(),
		Controller:         generic.Pointer(true),
		BlockOwnerDeletion: generic.Pointer(true),
	}))
	return r.client.Patch(ctx, obj, client.StrategicMergeFrom(base))
}

func (r *RefManager[O]) release(ctx context.Context, obj O) error {
	var (
		ownerRefs = obj.GetOwnerReferences()
		filtered  []metav1.OwnerReference
	)
	for _, ownerRef := range ownerRefs {
		if ownerRef.UID == r.controller.GetUID() {
			continue
		}

		filtered = append(filtered, ownerRef)
	}

	base := obj.DeepCopyObject().(client.Object)
	obj.SetOwnerReferences(filtered)
	return r.client.Patch(ctx, obj, client.StrategicMergeFrom(base))
}

func (r *RefManager[O]) ClaimObject(
	ctx context.Context,
	obj O,
) (bool, error) {
	controllerRef := metav1.GetControllerOf(obj)
	if controllerRef != nil {
		if controllerRef.UID != r.controller.GetUID() {
			// Owned by someone else. Ignore
			return false, nil
		}

		if r.match(obj) {
			// We own it and match. All OK.
			return true, nil
		}

		if !r.controller.GetDeletionTimestamp().IsZero() {
			// Don't try to own if we're deleting.
			return false, nil
		}

		if err := r.release(ctx, obj); err != nil {
			if !apierrors.IsNotFound(err) {
				return false, err
			}
			// If it's already gone, don't care about it.
			return false, nil
		}
		return false, nil
	}

	if !r.controller.GetDeletionTimestamp().IsZero() || !r.match(obj) {
		// Ignore if we're being deleted or don't match.
		return false, nil
	}
	if !obj.GetDeletionTimestamp().IsZero() {
		// Ignore if the object is being deleted.
		return false, nil
	}

	if err := r.adopt(ctx, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		// If it's already gone, ignore it.
		return false, nil
	}
	// Successfully adopted.
	return true, nil
}
