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
	"net/netip"
	"strings"

	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	apinetletv1alpha1 "github.com/onmetal/onmetal-api-net/apinetlet/api/v1alpha1"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	"github.com/onmetal/onmetal-api/utils/predicates"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	natGatewayFinalizer = "apinet.api.onmetal.de/natgateway"
)

type NATGatewayReconciler struct {
	client.Client
	APINetClient client.Client

	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=natgateways,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=natgateways/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=natgateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=publicips,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=publicips/status,verbs=get

func (r *NATGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	natGateway := &networkingv1alpha1.NATGateway{}
	if err := r.Get(ctx, req.NamespacedName, natGateway); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting nat gateway %s: %w", req.NamespacedName, err)
		}

		return r.deleteGone(ctx, log, req.NamespacedName)
	}

	return r.reconcileExists(ctx, log, natGateway)
}

func (r *NATGatewayReconciler) deleteGone(ctx context.Context, log logr.Logger, natGatewayKey client.ObjectKey) (ctrl.Result, error) {
	log.V(1).Info("Delete gone")

	log.V(1).Info("Deleting any matching apinet public ips")
	if err := r.APINetClient.DeleteAllOf(ctx, &onmetalapinetv1alpha1.PublicIP{},
		client.InNamespace(r.APINetNamespace),
		client.MatchingLabels{
			apinetletv1alpha1.NATGatewayNamespaceLabel: natGatewayKey.Namespace,
			apinetletv1alpha1.NATGatewayNameLabel:      natGatewayKey.Name,
		},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting apinet public ips: %w", err)
	}

	log.V(1).Info("Issued delete for any leftover apinet public ip")
	return ctrl.Result{}, nil
}

func (r *NATGatewayReconciler) reconcileExists(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway) (ctrl.Result, error) {
	log = log.WithValues("UID", natGateway.UID)
	if !natGateway.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, natGateway)
	}
	return r.reconcile(ctx, log, natGateway)
}

func (r *NATGatewayReconciler) delete(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(natGateway, natGatewayFinalizer) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	var count int
	for _, ipFamily := range natGateway.Spec.IPFamilies {
		for _, ip := range natGateway.Spec.IPs {
			if err := r.APINetClient.Delete(ctx, &onmetalapinetv1alpha1.PublicIP{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: r.APINetNamespace,
					Name:      fmt.Sprintf("%s-%s-%s", natGateway.UID, ip.Name, strings.ToLower(string(ipFamily))),
				},
			}); err != nil {
				if !apierrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("error deleting target public ip: %w", err)
				}
				count++

			}

		}
	}

	if count < len(natGateway.Spec.IPs)*len(natGateway.Spec.IPFamilies) {
		log.V(1).Info("Target public ip is not yet gone, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Target public ip is gone, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, natGateway, natGatewayFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}
	log.V(1).Info("Removed finalizer")
	return ctrl.Result{}, nil
}

func (r *NATGatewayReconciler) reconcile(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, natGateway, natGatewayFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	ips, err := r.applyPublicIPs(ctx, log, natGateway)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting / applying public ip: %w", err)
	}

	if err := r.patchStatus(ctx, log, natGateway, ips); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching nat gateway status")
	}

	log.V(1).Info("Patched nat gateway status")
	return ctrl.Result{}, nil
}

func (r *NATGatewayReconciler) applyPublicIP(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway, ipName string, ipFamily corev1.IPFamily) (netip.Addr, error) {
	apiNetPublicIP := &onmetalapinetv1alpha1.PublicIP{
		TypeMeta: metav1.TypeMeta{
			APIVersion: onmetalapinetv1alpha1.GroupVersion.String(),
			Kind:       "PublicIP",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      fmt.Sprintf("%s-%s-%s", natGateway.UID, ipName, strings.ToLower(string(ipFamily))),
			Labels: map[string]string{
				apinetletv1alpha1.NATGatewayNamespaceLabel: natGateway.Namespace,
				apinetletv1alpha1.NATGatewayNameLabel:      natGateway.Name,
				apinetletv1alpha1.NATGatewayUIDLabel:       string(natGateway.UID),
			},
		},
		Spec: onmetalapinetv1alpha1.PublicIPSpec{
			IPFamily: ipFamily,
		},
	}

	log.V(1).Info("Applying apinet public ip", "ipName", ipName, "ipFamily", ipFamily)
	if err := r.APINetClient.Patch(ctx, apiNetPublicIP, client.Apply,
		client.FieldOwner(apinetletv1alpha1.FieldOwner),
		client.ForceOwnership,
	); err != nil {
		return netip.Addr{}, fmt.Errorf("error applying apinet public ip: %w", err)
	}
	log.V(1).Info("Applied apinet public ip")

	if !apiNetPublicIP.IsAllocated() {
		return netip.Addr{}, nil
	}
	ip := apiNetPublicIP.Spec.IP
	return ip.Addr, nil
}

func (r *NATGatewayReconciler) applyPublicIPs(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway) (map[string]netip.Addr, error) {
	ips := map[string]netip.Addr{}
	for _, ipFamily := range natGateway.Spec.IPFamilies {
		for _, ip := range natGateway.Spec.IPs {
			apiNetPublicIP, err := r.applyPublicIP(ctx, log, natGateway, ip.Name, ipFamily)
			if err != nil {
				return nil, err
			}
			ips[ip.Name] = apiNetPublicIP
		}
	}
	return ips, nil
}

func (r *NATGatewayReconciler) patchStatus(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway, ips map[string]netip.Addr) error {
	base := natGateway.DeepCopy()
	natGateway.Status.IPs = []networkingv1alpha1.NATGatewayIPStatus{}

	for ipName, ip := range ips {
		if !ip.IsValid() {
			log.V(2).Info("Public ip is not yet allocated", "ipName", ipName)
			continue
		}

		log.V(2).Info("Public ip is allocated", "ipName", ipName)
		natGateway.Status.IPs = append(natGateway.Status.IPs, networkingv1alpha1.NATGatewayIPStatus{
			Name: ipName,
			IP: commonv1alpha1.IP{
				Addr: ip,
			},
		})
	}

	return r.Status().Patch(ctx, natGateway, client.MergeFrom(base))
}

func (r *NATGatewayReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCluster cluster.Cluster) error {
	log := ctrl.Log.WithName("natgateway").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1alpha1.NATGateway{},
			builder.WithPredicates(
				predicates.ResourceHasFilterLabel(log, r.WatchFilterValue),
				predicates.ResourceIsNotExternallyManaged(log),
			),
		).
		WatchesRawSource(
			source.Kind(apiNetCluster.GetCache(), &onmetalapinetv1alpha1.PublicIP{}),
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
				apiNetPublicIP := obj.(*onmetalapinetv1alpha1.PublicIP)

				if apiNetPublicIP.Namespace != r.APINetNamespace {
					return nil
				}

				namespace, ok := apiNetPublicIP.Labels[apinetletv1alpha1.NATGatewayNamespaceLabel]
				if !ok {
					return nil
				}

				name, ok := apiNetPublicIP.Labels[apinetletv1alpha1.NATGatewayNameLabel]
				if !ok {
					return nil
				}

				return []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: namespace, Name: name}}}
			}),
			builder.WithPredicates(
				getApiNetPublicIPAllocationChangedPredicate(),
			),
		).
		Complete(r)
}
