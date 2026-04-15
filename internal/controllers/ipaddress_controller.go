// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/controller-utils/clientutils"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/internal/ipaddress"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/lru"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type IPAddressReconciler struct {
	client.Client
	APIReader    client.Reader
	AbsenceCache *lru.Cache
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=ipaddresses/finalizers,verbs=update
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=ipaddresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=ips,verbs=get;list;watch

func (r *IPAddressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	ipAddress := &v1alpha1.IPAddress{}
	if err := r.Get(ctx, req.NamespacedName, ipAddress); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcileExists(ctx, log, ipAddress)
}

func (r *IPAddressReconciler) reconcileExists(
	ctx context.Context,
	log logr.Logger,
	ipAddress *v1alpha1.IPAddress,
) (ctrl.Result, error) {
	if !ipAddress.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, ipAddress)
	}
	return r.reconcile(ctx, log, ipAddress)
}

func (r *IPAddressReconciler) delete(ctx context.Context, log logr.Logger, ipAddress *v1alpha1.IPAddress) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(ipAddress, ipaddress.ProtectionFinalizer) {
		log.Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	claimRef := ipAddress.Spec.ClaimRef
	log = log.WithValues("ClaimRef", claimRef)

	claimer, err := NewPartialObjectMetadata(r.RESTMapper(), schema.GroupVersionResource{
		Group:    claimRef.Group,
		Resource: claimRef.Resource,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error constructing claimer object: %w", err)
	}

	claimerKey := client.ObjectKey{
		Namespace: claimRef.Namespace,
		Name:      claimRef.Name,
	}
	err = GetWithAbsenceCache(ctx, r.APIReader, r.AbsenceCache, claimerKey, claimer, claimRef.UID)
	switch {
	case err != nil && !apierrors.IsNotFound(err):
		return ctrl.Result{}, fmt.Errorf("error getting claimer %s: %w", claimerKey, err)
	case err == nil:
		log.V(4).Info("Claimer not yet gone, requeue")
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	log.V(1).Info("Claimer gone, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, ipAddress, ipaddress.ProtectionFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *IPAddressReconciler) reconcile(ctx context.Context, log logr.Logger, ipAddress *v1alpha1.IPAddress) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")
	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *IPAddressReconciler) enqueueByIP() handler.EventHandler {
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
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if !event.ObjectNew.GetDeletionTimestamp().IsZero() {
				mapAndEnqueue(ctx, event.ObjectNew, queue)
			}
		},
		GenericFunc: func(ctx context.Context, event event.GenericEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if !event.Object.GetDeletionTimestamp().IsZero() {
				mapAndEnqueue(ctx, event.Object, queue)
			}
		},
	}
}

func (r *IPAddressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IPAddress{}).
		Watches(
			&v1alpha1.IP{},
			r.enqueueByIP(),
		).
		Complete(r)
}
