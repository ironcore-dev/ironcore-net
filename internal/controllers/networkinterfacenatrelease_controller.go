// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/lru"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type NetworkInterfaceNATReleaseReconciler struct {
	client.Client
	APIReader client.Reader

	AbsenceCache *lru.Cache
}

func (r *NetworkInterfaceNATReleaseReconciler) networkInterfaceNATExists(
	ctx context.Context,
	nic *v1alpha1.NetworkInterface,
	nat *v1alpha1.NetworkInterfaceNAT,
) (bool, error) {
	claimRef := nat.ClaimRef
	if _, ok := r.AbsenceCache.Get(claimRef.UID); ok {
		return false, nil
	}

	natGateway := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "NATGateway",
		},
	}
	natGatewayKey := client.ObjectKey{Namespace: nic.Namespace, Name: claimRef.Name}
	if err := r.APIReader.Get(ctx, natGatewayKey, natGateway); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("error getting NAT gateway: %w", err)
		}

		r.AbsenceCache.Add(claimRef.UID, nil)
		return false, nil
	}
	if claimRef.UID != natGateway.UID {
		r.AbsenceCache.Add(claimRef.UID, nil)
		return false, nil
	}
	return true, nil
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkinterfaces,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=natgateways,verbs=get;list;watch

func (r *NetworkInterfaceNATReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	nic := &v1alpha1.NetworkInterface{}
	if err := r.Get(ctx, req.NamespacedName, nic); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !nic.DeletionTimestamp.IsZero() {
		log.V(1).Info("Network interface is already deleting")
		return ctrl.Result{}, nil
	}

	var filtered []v1alpha1.NetworkInterfaceNAT
	for _, nat := range nic.Spec.NATs {
		ok, err := r.networkInterfaceNATExists(ctx, nic, &nat)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error checking whether NAT %s exists: %w", nat.IPFamily, err)
		}
		if !ok {
			continue
		}

		filtered = append(filtered, nat)
	}
	if len(filtered) == len(nic.Spec.NATs) {
		log.V(1).Info("All NATs are present, nothing to do")
		return ctrl.Result{}, nil
	}

	base := nic.DeepCopy()
	nic.Spec.NATs = filtered
	if err := r.Patch(ctx, nic, client.StrategicMergeFrom(base)); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching network interface: %w", err)
	}

	log.V(1).Info("Filtered NATs", "Filtered", filtered, "Original", nic.Spec.NATs)
	return ctrl.Result{}, nil
}

func (r *NetworkInterfaceNATReleaseReconciler) enqueueByNATGateway() handler.EventHandler {
	mapAndEnqueue := func(ctx context.Context, natGateway *v1alpha1.NATGateway, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
		log := ctrl.LoggerFrom(ctx)

		nicList := &v1alpha1.NetworkInterfaceList{}
		if err := r.List(ctx, nicList,
			client.InNamespace(natGateway.GetNamespace()),
		); err != nil {
			log.Error(err, "Error listing network interfaces")
			return
		}

		for _, nic := range nicList.Items {
			if v1alpha1.IsNetworkInterfaceNATClaimedBy(&nic, natGateway) {
				queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&nic)})
			}
		}
	}

	return &handler.Funcs{
		DeleteFunc: func(ctx context.Context, event event.DeleteEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			natGateway := event.Object.(*v1alpha1.NATGateway)
			mapAndEnqueue(ctx, natGateway, queue)
		},
		GenericFunc: func(ctx context.Context, event event.GenericEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			natGateway := event.Object.(*v1alpha1.NATGateway)
			if !natGateway.GetDeletionTimestamp().IsZero() {
				mapAndEnqueue(ctx, natGateway, queue)
			}
		},
	}
}

func (r *NetworkInterfaceNATReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.NetworkInterface{}).
		Watches(
			&v1alpha1.NATGateway{},
			r.enqueueByNATGateway(),
		).
		Complete(r)
}
