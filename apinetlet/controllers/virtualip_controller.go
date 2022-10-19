// Copyright 2022 OnMetal authors
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

	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	apinetletclient "github.com/onmetal/onmetal-api-net/apinetlet/client"
	commonv1alpha1 "github.com/onmetal/onmetal-api/apis/common/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/apis/networking/v1alpha1"
	"github.com/onmetal/onmetal-api/util/predicates"
	brokerclient "github.com/onmetal/poollet/broker/client"
	brokerhandler "github.com/onmetal/poollet/broker/handler"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	virtualIPFinalizer = "apinet.api.onmetal.de/virtualip"
)

type VirtualIPReconciler struct {
	client.Client
	APINetClient client.Client

	ClusterName     string
	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips/status,verbs=get;update;patch

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

	log.V(1).Info("Listing if any public ips are present for gone virtual ip")
	publicIPList := &onmetalapinetv1alpha1.PublicIPList{}
	if err := r.APINetClient.List(ctx, publicIPList,
		client.InNamespace(r.APINetNamespace),
		client.MatchingFields{apinetletclient.PublicIPVirtualIPController: virtualIPKey.String()},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing public ips: %w", err)
	}

	var errs []error
	for _, publicIP := range publicIPList.Items {
		log.V(1).Info("Deleting public ip", "PublicIPKey", client.ObjectKeyFromObject(&publicIP))
		if err := r.Delete(ctx, &publicIP); client.IgnoreNotFound(err) != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return ctrl.Result{}, fmt.Errorf("error deleting related public ip(s): %v", errs)
	}

	log.V(1).Info("Issued delete for any leftover public ip")
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

	log.V(1).Info("Deleting target public ip if any")
	if err := r.APINetClient.Delete(ctx, &onmetalapinetv1alpha1.PublicIP{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(virtualIP.UID),
		},
	}); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting target public ip: %w", err)
		}

		log.V(1).Info("Target public ip is gone, removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, virtualIP, virtualIPFinalizer); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Removed finalizer")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Target public ip is not yet gone, requeueing")
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

	ip, err := r.applyPublicIP(ctx, log, virtualIP)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting / applying public ip: %w", err)
	}
	if ip == nil {
		log.V(1).Info("Public ip is not yet allocated, patching status")
		if err := r.patchStatusUnallocated(ctx, virtualIP); err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("Patched virtual ip status")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("IP", *ip)
	log.V(1).Info("Public ip is allocated")

	log.V(1).Info("Patching virtual ip status ip allocated")
	if err := r.patchStatusAllocated(ctx, virtualIP, *ip); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching virtual ip status")
	}
	log.V(1).Info("Patched virtual ip status ip allocated")
	return ctrl.Result{}, nil
}

func (r *VirtualIPReconciler) applyPublicIP(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (*commonv1alpha1.IP, error) {
	publicIP := &onmetalapinetv1alpha1.PublicIP{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(virtualIP.UID),
		},
	}
	log.V(1).Info("Applying public ip")
	if _, err := brokerclient.BrokerControlledCreateOrPatch(ctx, r.APINetClient, r.ClusterName, virtualIP, publicIP, func() error {
		publicIP.Spec.IPFamilies = []corev1.IPFamily{virtualIP.Spec.IPFamily}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error applying public ip: %w", err)
	}
	log.V(1).Info("Applied public ip")

	if ips := publicIP.Status.IPs; len(ips) > 0 {
		return &ips[0], nil
	}
	return nil, nil
}

func (r *VirtualIPReconciler) patchStatusAllocated(ctx context.Context, virtualIP *networkingv1alpha1.VirtualIP, ip commonv1alpha1.IP) error {
	base := virtualIP.DeepCopy()
	virtualIP.Status.IP = &ip
	if err := r.Status().Patch(ctx, virtualIP, client.MergeFrom(base)); err != nil {
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

func (r *VirtualIPReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCluster cluster.Cluster) error {
	log := ctrl.Log.WithName("virtualip").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.VirtualIP{}).
		WithEventFilter(predicates.ResourceHasFilterLabel(log, r.WatchFilterValue)).
		Watches(
			source.NewKindWithCache(&onmetalapinetv1alpha1.PublicIP{}, apiNetCluster.GetCache()),
			&brokerhandler.EnqueueRequestForBrokerOwner{
				ClusterName:  r.ClusterName,
				OwnerType:    &networkingv1alpha1.VirtualIP{},
				IsController: true,
			},
		).
		Complete(r)
}
