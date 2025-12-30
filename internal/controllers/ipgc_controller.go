// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ironcore-dev/controller-utils/clientutils"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/lru"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// IPOwnerVerificationFinalizer is the finalizer used to prevent premature deletion of IP objects
	// until the owner is visible to Kubernetes GC.
	IPOwnerVerificationFinalizer = "apinet.ironcore.dev/ip-owner-verification"
)

var errOwnerNotReady = errors.New("owner not found yet; will retry")

type IPGCReconciler struct {
	client.Client
	APIReader  client.Reader
	RESTMapper meta.RESTMapper

	AbsenceCache *lru.Cache
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=ips,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=ips/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=loadbalancers,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=natgateways,verbs=get;list;watch

func (r *IPGCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	ip := &v1alpha1.IP{}
	if err := r.Get(ctx, req.NamespacedName, ip); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Only process IPs with our finalizer
	if !controllerutil.ContainsFinalizer(ip, IPOwnerVerificationFinalizer) {
		return ctrl.Result{}, nil
	}

	ownerRef := metav1.GetControllerOf(ip)
	if ownerRef == nil {
		log.V(1).Info("IP has no owner reference, removing finalizer")
		return r.removeFinalizer(ctx, ip)
	}

	ownerExists, ownerAge, ownerDeleting, err := r.ownerExists(ctx, ip, ownerRef)
	if err != nil {
		if errors.Is(err, errOwnerNotReady) {
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		return ctrl.Result{}, fmt.Errorf("error checking owner existence: %w", err)
	}

	if !ownerExists {
		log.V(1).Info("Owner does not exist, removing finalizer")
		return r.removeFinalizer(ctx, ip)
	}
	// Owner exists - check if it's been in etcd long enough for GC to see it
	const minOwnerAge = 5 * time.Second
	if ownerAge < minOwnerAge {
		requeueAfter := minOwnerAge - ownerAge
		if requeueAfter > 2*time.Second {
			requeueAfter = 2 * time.Second // Cap at 2 seconds
		}
		log.V(1).Info("Owner exists but is too new, keeping finalizer and requeueing", "age", ownerAge, "requeueAfter", requeueAfter, "deletionTimestamp", !ip.DeletionTimestamp.IsZero())
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	if !ip.DeletionTimestamp.IsZero() {
		if ownerDeleting {
			log.V(1).Info("Owner is being deleted (cascading deletion), removing finalizer", "age", ownerAge)
			return r.removeFinalizer(ctx, ip)
		}

		return ctrl.Result{}, nil
	}

	// Owner exists, is old enough, and IP is not being deleted
	// Safe to remove finalizer after GC propagation
	log.V(1).Info("Owner exists and is old enough, removing finalizer", "age", ownerAge)
	return r.removeFinalizer(ctx, ip)
}

func (r *IPGCReconciler) ownerExists(ctx context.Context, ip *v1alpha1.IP, ownerRef *metav1.OwnerReference) (bool, time.Duration, bool, error) {
	const ownerNotFoundGrace = 10 * time.Second
	ipAge := time.Since(ip.CreationTimestamp.Time)

	if _, ok := r.AbsenceCache.Get(ownerRef.UID); ok {
		return false, 0, false, nil
	}

	gv, err := schema.ParseGroupVersion(ownerRef.APIVersion)
	if err != nil {
		return false, 0, false, fmt.Errorf("error parsing owner APIVersion %s: %w", ownerRef.APIVersion, err)
	}

	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    ownerRef.Kind,
	}

	mapping, err := r.RESTMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, 0, false, fmt.Errorf("error getting REST mapping for %s: %w", gvk, err)
	}

	owner := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ownerRef.APIVersion,
			Kind:       ownerRef.Kind,
		},
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ownerKey := client.ObjectKey{Name: ownerRef.Name, Namespace: ip.Namespace}
		if err := r.APIReader.Get(ctx, ownerKey, owner); err == nil {
			if ownerRef.UID == owner.UID {
				age := time.Since(owner.CreationTimestamp.Time)
				ownerDeleting := !owner.DeletionTimestamp.IsZero()
				return true, age, ownerDeleting, nil
			}
		} else if !apierrors.IsNotFound(err) {
			return false, 0, false, fmt.Errorf("error getting owner %s %s in namespace %s: %w", gvk, ownerRef.Name, ip.Namespace, err)
		} else if ipAge < ownerNotFoundGrace {
			return false, 0, false, errOwnerNotReady
		}

		ownerKey = client.ObjectKey{Name: ownerRef.Name, Namespace: "default"}
		if err := r.APIReader.Get(ctx, ownerKey, owner); err == nil {
			if ownerRef.UID == owner.UID {
				age := time.Since(owner.CreationTimestamp.Time)
				ownerDeleting := !owner.DeletionTimestamp.IsZero()
				return true, age, ownerDeleting, nil
			}
		} else if !apierrors.IsNotFound(err) {
			return false, 0, false, fmt.Errorf("error getting owner %s %s in namespace default: %w", gvk, ownerRef.Name, err)
		} else if ipAge < ownerNotFoundGrace {
			return false, 0, false, errOwnerNotReady
		}

		r.AbsenceCache.Add(ownerRef.UID, nil)
		return false, 0, false, nil
	}

	ownerKey := client.ObjectKey{Name: ownerRef.Name}

	if err := r.APIReader.Get(ctx, ownerKey, owner); err != nil {
		if apierrors.IsNotFound(err) {
			if ipAge < ownerNotFoundGrace {
				return false, 0, false, errOwnerNotReady
			}
			r.AbsenceCache.Add(ownerRef.UID, nil)
			return false, 0, false, nil
		}
		return false, 0, false, fmt.Errorf("error getting owner %s %s: %w", gvk, ownerRef.Name, err)
	}

	if ownerRef.UID != owner.UID {
		r.AbsenceCache.Add(ownerRef.UID, nil)
		return false, 0, false, nil
	}

	age := time.Since(owner.CreationTimestamp.Time)
	ownerDeleting := !owner.DeletionTimestamp.IsZero()
	return true, age, ownerDeleting, nil
}

func (r *IPGCReconciler) removeFinalizer(ctx context.Context, ip *v1alpha1.IP) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(ip, IPOwnerVerificationFinalizer) {
		return ctrl.Result{}, nil
	}

	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, ip, IPOwnerVerificationFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *IPGCReconciler) enqueueByOwner() handler.EventHandler {
	mapAndEnqueue := func(ctx context.Context, owner client.Object, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
		log := ctrl.LoggerFrom(ctx)

		ipList := &v1alpha1.IPList{}
		if err := r.List(ctx, ipList); err != nil {
			log.Error(err, "Error listing IPs")
			return
		}

		for _, ip := range ipList.Items {
			ownerRef := metav1.GetControllerOf(&ip)
			if ownerRef != nil && ownerRef.UID == owner.GetUID() {
				if controllerutil.ContainsFinalizer(&ip, IPOwnerVerificationFinalizer) {
					queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&ip)})
				}
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

func (r *IPGCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("ipgc").
		For(&v1alpha1.IP{}).
		Watches(
			&v1alpha1.LoadBalancer{},
			r.enqueueByOwner(),
		).
		Watches(
			&v1alpha1.NATGateway{},
			r.enqueueByOwner(),
		).
		Complete(r)
}
