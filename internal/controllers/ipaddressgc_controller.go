// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/lru"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type IPAddressGCReconciler struct {
	client.Client
	APIReader client.Reader

	AbsenceCache *lru.Cache
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=ipaddresses,verbs=get;list;watch;delete
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=ips,verbs=get;list;watch

func (r *IPAddressGCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	addr := &v1alpha1.IPAddress{}
	if err := r.Get(ctx, req.NamespacedName, addr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !addr.DeletionTimestamp.IsZero() {
		// Don't try to GC addresses that are already deleting.
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Checking whether IP address claimer exists")
	ok, err := r.ipAddressClaimerExists(ctx, addr)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error checking whether IP address claimer exists: %w", err)
	}
	if ok {
		log.V(1).Info("IP address claimer is still present")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("IP address claimer does not exist, releasing IP address")
	if err := r.Delete(ctx, addr); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting IP address: %w", err)
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *IPAddressGCReconciler) ipAddressClaimerExists(ctx context.Context, addr *v1alpha1.IPAddress) (bool, error) {
	claimRef := addr.Spec.ClaimRef
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
			return false, fmt.Errorf("error getting claiming %s %s: %w", gvk, claimRef.Name, err)
		}

		r.AbsenceCache.Add(claimRef.UID, nil)
		return false, nil
	}
	if claimRef.UID != claimer.UID {
		r.AbsenceCache.Add(claimRef.UID, nil)
		return false, nil
	}
	return true, nil
}

func (r *IPAddressGCReconciler) enqueueByClaimer() handler.EventHandler {
	mapAndEnqueue := func(ctx context.Context, claimer client.Object, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
		log := ctrl.LoggerFrom(ctx)

		addrList := &v1alpha1.IPAddressList{}
		if err := r.List(ctx, addrList,
			client.InNamespace(claimer.GetNamespace()),
		); err != nil {
			log.Error(err, "Error listing IP addresses")
			return
		}

		for _, addr := range addrList.Items {
			if v1alpha1.IsIPAddressClaimedBy(&addr, claimer) {
				queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&addr)})
			}
		}
	}

	return &handler.Funcs{
		DeleteFunc: func(ctx context.Context, event event.DeleteEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			mapAndEnqueue(ctx, event.Object, queue)
		},
		GenericFunc: func(ctx context.Context, event event.GenericEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if !event.Object.GetDeletionTimestamp().IsZero() {
				mapAndEnqueue(ctx, event.Object, queue)
			}
		},
	}
}

func (r *IPAddressGCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("ipaddressgc").
		For(&v1alpha1.IPAddress{}).
		Watches(
			&v1alpha1.IP{},
			r.enqueueByClaimer(),
		).
		Complete(r)
}
