// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/go-logr/logr"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	"github.com/ironcore-dev/ironcore-net/apinetlet/handler"
	apinetv1alpha1ac "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/core/v1alpha1"
	ironcorenet "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned"

	"github.com/ironcore-dev/controller-utils/clientutils"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/predicates"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	virtualIPFinalizer = "apinet.ironcore.dev/virtualip"
)

type VirtualIPReconciler struct {
	client.Client
	APINetClient    client.Client
	APINetInterface ironcorenet.Interface

	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=virtualips,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=virtualips/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=virtualips/status,verbs=get;update;patch

//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=ips,verbs=get;list;watch;create;update;patch;delete;deletecollection

func (r *VirtualIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	virtualIP := &networkingv1alpha1.VirtualIP{}
	if err := r.Get(ctx, req.NamespacedName, virtualIP); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting virtual ip %s: %w", req.NamespacedName, err)
		}

		return r.deleteGone(ctx, log, req.NamespacedName)
	}

	return r.reconcileExists(ctx, log, virtualIP)
}

func (r *VirtualIPReconciler) deleteGone(ctx context.Context, log logr.Logger, virtualIPKey client.ObjectKey) (ctrl.Result, error) {
	log.V(1).Info("Delete gone")

	log.V(1).Info("Deleting any matching APINet ips")
	if err := r.APINetClient.DeleteAllOf(ctx, &apinetv1alpha1.IP{},
		client.InNamespace(r.APINetNamespace),
		apinetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), virtualIPKey, &networkingv1alpha1.VirtualIP{}),
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting APINet ips: %w", err)
	}

	log.V(1).Info("Issued delete for any leftover APINet ips")
	return ctrl.Result{}, nil
}

func (r *VirtualIPReconciler) reconcileExists(
	ctx context.Context,
	log logr.Logger,
	virtualIP *networkingv1alpha1.VirtualIP,
) (ctrl.Result, error) {
	log = log.WithValues("UID", virtualIP.UID)
	if !virtualIP.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, virtualIP)
	}
	return r.reconcile(ctx, log, virtualIP)
}

func (r *VirtualIPReconciler) delete(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(virtualIP, virtualIPFinalizer) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Deleting target APINet IP if any")
	apiNetIP := &apinetv1alpha1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(virtualIP.UID),
		},
	}
	if err := r.APINetClient.Delete(ctx, apiNetIP); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting target APINet ip: %w", err)
		}

		log.V(1).Info("Target APINet ip is gone, removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, virtualIP, virtualIPFinalizer); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Removed finalizer")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Target APINet ip is not yet gone, requeueing")
	return ctrl.Result{Requeue: true}, nil
}

func (r *VirtualIPReconciler) reconcile(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, virtualIP, virtualIPFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	ip, err := r.applyIP(ctx, log, virtualIP)
	if err != nil {
		if virtualIP.Status.IP == nil {
			if err := r.patchStatusUnallocated(ctx, virtualIP); err != nil {
				log.Error(err, "Error patching virtual IP status")
			}
		}
		return ctrl.Result{}, fmt.Errorf("error applying APINet ip: %w", err)
	}

	if err := r.patchStatusAllocated(ctx, virtualIP, ip); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching virtual ip status")
	}
	log.V(1).Info("Patched virtual ip status ip allocated")
	return ctrl.Result{}, nil
}

func (r *VirtualIPReconciler) applyIP(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (netip.Addr, error) {
	apiNetIPApplyCfg := apinetv1alpha1ac.IP(string(virtualIP.UID), r.APINetNamespace).
		WithLabels(apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), virtualIP)).
		WithSpec(apinetv1alpha1ac.IPSpec().
			WithType(apinetv1alpha1.IPTypePublic).
			WithIPFamily(virtualIP.Spec.IPFamily),
		)

	apiNetIP, err := r.APINetInterface.CoreV1alpha1().
		IPs(r.APINetNamespace).
		Apply(ctx, apiNetIPApplyCfg, metav1.ApplyOptions{FieldManager: string(fieldOwner), Force: true})
	if err != nil {
		return netip.Addr{}, fmt.Errorf("error applying APINet ip: %w", err)
	}

	log.V(1).Info("Applied APINet ip")
	ip := apiNetIP.Spec.IP
	return ip.Addr, nil
}

func (r *VirtualIPReconciler) patchStatusAllocated(ctx context.Context, virtualIP *networkingv1alpha1.VirtualIP, addr netip.Addr) error {
	base := virtualIP.DeepCopy()
	virtualIP.Status.IP = &commonv1alpha1.IP{Addr: addr}
	if err := r.Status().Patch(ctx, virtualIP, client.StrategicMergeFrom(base)); err != nil {
		return fmt.Errorf("error patching virtual ip status: %w", err)
	}
	return nil
}

func (r *VirtualIPReconciler) patchStatusUnallocated(ctx context.Context, virtualIP *networkingv1alpha1.VirtualIP) error {
	base := virtualIP.DeepCopy()
	virtualIP.Status.IP = nil
	if err := r.Status().Patch(ctx, virtualIP, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching virtual ip status: %w", err)
	}
	return nil
}

func (r *VirtualIPReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCache cache.Cache) error {
	log := ctrl.Log.WithName("virtualip").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1alpha1.VirtualIP{},
			builder.WithPredicates(
				predicates.ResourceHasFilterLabel(log, r.WatchFilterValue),
				predicates.ResourceIsNotExternallyManaged(log),
			),
		).
		WatchesRawSource(
			source.Kind[client.Object](
				apiNetCache,
				&apinetv1alpha1.IP{},
				handler.EnqueueRequestForSource(r.Scheme(), mgr.GetRESTMapper(), &networkingv1alpha1.VirtualIP{}),
			),
		).
		Complete(r)
}
