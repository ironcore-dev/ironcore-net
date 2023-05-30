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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	loadBalancerFinalizer = "apinet.api.onmetal.de/loadbalancer"
)

type LoadBalancerReconciler struct {
	client.Client
	APINetClient client.Client

	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=loadbalancers,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=loadbalancers/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=loadbalancers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=publicips,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=publicips/status,verbs=get

func (r *LoadBalancerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	loadBalancer := &networkingv1alpha1.LoadBalancer{}
	if err := r.Get(ctx, req.NamespacedName, loadBalancer); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting load balancer %s: %w", req.NamespacedName, err)
		}

		return r.deleteGone(ctx, log, req.NamespacedName)
	}

	return r.reconcileExists(ctx, log, loadBalancer)
}

func (r *LoadBalancerReconciler) deleteGone(ctx context.Context, log logr.Logger, virtualIPKey client.ObjectKey) (ctrl.Result, error) {
	log.V(1).Info("Delete gone")

	log.V(1).Info("Deleting any matching apinet public ips")
	if err := r.APINetClient.DeleteAllOf(ctx, &onmetalapinetv1alpha1.PublicIP{},
		client.InNamespace(r.APINetNamespace),
		client.MatchingLabels{
			apinetletv1alpha1.LoadBalancerNamespaceLabel: virtualIPKey.Namespace,
			apinetletv1alpha1.LoadBalancerNameLabel:      virtualIPKey.Name,
		},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting apinet public ips: %w", err)
	}

	log.V(1).Info("Issued delete for any leftover apinet public ip")
	return ctrl.Result{}, nil
}

func (r *LoadBalancerReconciler) reconcileExists(ctx context.Context, log logr.Logger, loadBalancer *networkingv1alpha1.LoadBalancer) (ctrl.Result, error) {
	log = log.WithValues("UID", loadBalancer.UID)
	if !loadBalancer.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, loadBalancer)
	}
	return r.reconcile(ctx, log, loadBalancer)
}

func (r *LoadBalancerReconciler) delete(ctx context.Context, log logr.Logger, loadBalancer *networkingv1alpha1.LoadBalancer) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(loadBalancer, loadBalancerFinalizer) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	var count int
	for _, ipFamily := range loadBalancer.Spec.IPFamilies {
		if err := r.APINetClient.Delete(ctx, &onmetalapinetv1alpha1.PublicIP{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: r.APINetNamespace,
				Name:      fmt.Sprintf("%s-%s", loadBalancer.UID, strings.ToLower(string(ipFamily))),
			},
		}); err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("error deleting target public ip: %w", err)
			}
			count++

		}
	}

	if count < len(loadBalancer.Spec.IPFamilies) {
		log.V(1).Info("Target public ip is not yet gone, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Target public ip is gone, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, loadBalancer, loadBalancerFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}
	log.V(1).Info("Removed finalizer")
	return ctrl.Result{}, nil
}

func (r *LoadBalancerReconciler) reconcile(ctx context.Context, log logr.Logger, loadBalancer *networkingv1alpha1.LoadBalancer) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, loadBalancer, loadBalancerFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	ips, err := r.applyPublicIPs(ctx, log, loadBalancer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting / applying public ip: %w", err)
	}

	if err := r.patchStatus(ctx, log, loadBalancer, ips); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching load balancer status")
	}

	log.V(1).Info("Patched load balancer status")
	return ctrl.Result{}, nil
}

func (r *LoadBalancerReconciler) applyPublicIPs(ctx context.Context, log logr.Logger, loadBalancer *networkingv1alpha1.LoadBalancer) ([]netip.Addr, error) {
	var ips []netip.Addr
	for _, ipFamily := range loadBalancer.Spec.IPFamilies {
		apiNetPublicIP, err := r.applyPublicIP(ctx, log, loadBalancer, ipFamily)
		if err != nil {
			return nil, err
		}

		ips = append(ips, apiNetPublicIP)
	}
	return ips, nil
}

func (r *LoadBalancerReconciler) applyPublicIP(ctx context.Context, log logr.Logger, loadBalancer *networkingv1alpha1.LoadBalancer, ipFamily corev1.IPFamily) (netip.Addr, error) {
	apiNetPublicIP := &onmetalapinetv1alpha1.PublicIP{
		TypeMeta: metav1.TypeMeta{
			APIVersion: onmetalapinetv1alpha1.GroupVersion.String(),
			Kind:       "PublicIP",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      fmt.Sprintf("%s-%s", loadBalancer.UID, strings.ToLower(string(ipFamily))),
			Labels: map[string]string{
				apinetletv1alpha1.LoadBalancerNamespaceLabel: loadBalancer.Namespace,
				apinetletv1alpha1.LoadBalancerNameLabel:      loadBalancer.Name,
				apinetletv1alpha1.LoadBalancerUIDLabel:       string(loadBalancer.UID),
			},
		},
		Spec: onmetalapinetv1alpha1.PublicIPSpec{
			IPFamily: ipFamily,
		},
	}

	log.V(1).Info("Applying apinet public ip", "ipFamily", ipFamily)
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

func (r *LoadBalancerReconciler) patchStatus(ctx context.Context, log logr.Logger, loadBalancer *networkingv1alpha1.LoadBalancer, ips []netip.Addr) error {
	base := loadBalancer.DeepCopy()
	loadBalancer.Status.IPs = []commonv1alpha1.IP{}

	for _, ip := range ips {
		if !ip.IsValid() {
			log.V(2).Info("Public ip is not yet allocated", "ip", ip.String())
			continue
		}

		log.V(2).Info("Public ip is allocated", "ip", ip.String())
		loadBalancer.Status.IPs = append(loadBalancer.Status.IPs, commonv1alpha1.IP{
			Addr: ip,
		})
	}

	return r.Status().Patch(ctx, loadBalancer, client.MergeFrom(base))
}

var isLoadBalancerTypePublic = predicate.NewPredicateFuncs(func(obj client.Object) bool {
	loadBalancer := obj.(*networkingv1alpha1.LoadBalancer)
	return loadBalancer.Spec.Type == networkingv1alpha1.LoadBalancerTypePublic
})

func (r *LoadBalancerReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCluster cluster.Cluster) error {
	log := ctrl.Log.WithName("loadbalancer").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1alpha1.LoadBalancer{},
			builder.WithPredicates(
				predicates.ResourceHasFilterLabel(log, r.WatchFilterValue),
				predicates.ResourceIsNotExternallyManaged(log),
				isLoadBalancerTypePublic,
			),
		).
		WatchesRawSource(
			source.Kind(apiNetCluster.GetCache(), &onmetalapinetv1alpha1.PublicIP{}),
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
				apiNetPublicIP := obj.(*onmetalapinetv1alpha1.PublicIP)

				if apiNetPublicIP.Namespace != r.APINetNamespace {
					return nil
				}

				namespace, ok := apiNetPublicIP.Labels[apinetletv1alpha1.LoadBalancerNamespaceLabel]
				if !ok {
					return nil
				}

				name, ok := apiNetPublicIP.Labels[apinetletv1alpha1.LoadBalancerNameLabel]
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
