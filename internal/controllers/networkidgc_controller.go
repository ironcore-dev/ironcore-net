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

package controllers

import (
	"context"
	"fmt"

	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/utils/lru"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type NetworkIDGCReconciler struct {
	client.Client
	APIReader client.Reader

	AbsenceCache *lru.Cache
}

//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=networkids,verbs=get;list;watch;delete
//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=networks,verbs=get;list;watch

func (r *NetworkIDGCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	networkID := &v1alpha1.NetworkID{}
	if err := r.Get(ctx, req.NamespacedName, networkID); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !networkID.DeletionTimestamp.IsZero() {
		// Don't try to GC addresses that are already deleting.
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Checking whether network ID claimer exists")
	ok, err := r.networkIDClaimerExists(ctx, networkID)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error checking whether network ID claimer exists: %w", err)
	}
	if ok {
		log.V(1).Info("Network ID claimer is still present")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Network ID claimer does not exist, releasing network ID")
	if err := r.Delete(ctx, networkID); err != nil {
		return ctrl.Result{}, fmt.Errorf("error releasing network ID: %w", err)
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkIDGCReconciler) networkIDClaimerExists(ctx context.Context, networkID *v1alpha1.NetworkID) (bool, error) {
	claimRef := networkID.Spec.ClaimRef
	if _, ok := r.AbsenceCache.Get(claimRef.UID); ok {
		return false, nil
	}

	gvr := schema.GroupVersionResource{
		Resource: claimRef.Resource,
		Group:    claimRef.Group,
	}
	resList, err := r.RESTMapper().KindsFor(gvr)
	if err != nil {
		return false, fmt.Errorf("error getting kinds for %s: %w", gvr.GroupResource(), err)
	}
	if len(resList) == 0 {
		return false, fmt.Errorf("no kind for %s", gvr.GroupResource())
	}

	gvk := resList[0]

	mapping, err := r.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, fmt.Errorf("error getting REST mapping for %s: %w", gvk, err)
	}

	claimer := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
	}
	claimerKey := client.ObjectKey{Name: claimRef.Name}
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		claimerKey.Namespace = claimRef.Namespace
	}

	if err := r.APIReader.Get(ctx, claimerKey, claimer); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("error getting claiming %s %s: %w", gvk, klog.KRef(claimerKey.Namespace, claimerKey.Name), err)
		}

		r.AbsenceCache.Add(claimRef.UID, nil)
		return false, nil
	}
	return true, nil
}

func (r *NetworkIDGCReconciler) enqueueByClaimer() handler.EventHandler {
	mapAndEnqueue := func(ctx context.Context, claimer client.Object, queue workqueue.RateLimitingInterface) {
		log := ctrl.LoggerFrom(ctx)

		networkIDList := &v1alpha1.NetworkIDList{}
		if err := r.List(ctx, networkIDList); err != nil {
			log.Error(err, "Error listing Network IDs")
			return
		}

		for _, networkID := range networkIDList.Items {
			if v1alpha1.IsNetworkIDClaimedBy(&networkID, claimer) {
				queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&networkID)})
			}
		}
	}

	return &handler.Funcs{
		DeleteFunc: func(ctx context.Context, event event.DeleteEvent, queue workqueue.RateLimitingInterface) {
			mapAndEnqueue(ctx, event.Object, queue)
		},
		GenericFunc: func(ctx context.Context, event event.GenericEvent, queue workqueue.RateLimitingInterface) {
			if !event.Object.GetDeletionTimestamp().IsZero() {
				mapAndEnqueue(ctx, event.Object, queue)
			}
		},
	}
}

func (r *NetworkIDGCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("networkidgc").
		For(&v1alpha1.NetworkID{}).
		Watches(
			&v1alpha1.Network{},
			r.enqueueByClaimer(),
		).
		Complete(r)
}
