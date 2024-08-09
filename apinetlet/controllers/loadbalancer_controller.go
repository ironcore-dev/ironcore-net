// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	apinetlethandler "github.com/ironcore-dev/ironcore-net/apinetlet/handler"
	"github.com/ironcore-dev/ironcore-net/apinetlet/provider"
	apinetv1alpha1ac "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/core/v1alpha1"
	metav1ac "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/meta/v1"
	"github.com/ironcore-dev/ironcore-net/client-go/ironcorenet"

	"github.com/ironcore-dev/controller-utils/clientutils"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/predicates"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	loadBalancerFinalizer = "apinet.ironcore.dev/loadbalancer"
)

type LoadBalancerReconciler struct {
	client.Client
	APINetClient    client.Client
	APINetInterface ironcorenet.Interface

	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=loadbalancers,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=loadbalancers/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=loadbalancers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=loadbalancerroutings,verbs=get;list;watch
//+kubebuilder:rbac:groups=ipam.ironcore.dev,resources=prefixes,verbs=get;list;watch

//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=loadbalancers,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=loadbalancerroutings,verbs=get;list;watch;create;update;patch;delete;deletecollection

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

func (r *LoadBalancerReconciler) deleteGone(ctx context.Context, log logr.Logger, loadBalancerKey client.ObjectKey) (ctrl.Result, error) {
	log.V(1).Info("Delete gone")

	log.V(1).Info("Deleting any matching APINet load balancers")
	if err := r.APINetClient.DeleteAllOf(ctx, &apinetv1alpha1.LoadBalancer{},
		client.InNamespace(r.APINetNamespace),
		apinetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), loadBalancerKey, &networkingv1alpha1.LoadBalancer{}),
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting APINet load balancers: %w", err)
	}

	log.V(1).Info("Issued delete for any leftover APINet load balancer")
	return ctrl.Result{}, nil
}

func (r *LoadBalancerReconciler) reconcileExists(ctx context.Context, log logr.Logger, loadBalancer *networkingv1alpha1.LoadBalancer) (ctrl.Result, error) {
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

	apiNetLoadBalancer := &apinetv1alpha1.LoadBalancer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(loadBalancer.UID),
		},
	}
	if err := r.APINetClient.Delete(ctx, apiNetLoadBalancer); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting APINet load balancer: %w", err)
		}

		log.V(1).Info("APINet load balancer is gone, removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, loadBalancer, loadBalancerFinalizer); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Deleted")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Issued APINet load balancer deletion")
	return ctrl.Result{Requeue: true}, nil
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

	networkKey := client.ObjectKey{Namespace: loadBalancer.Namespace, Name: loadBalancer.Spec.NetworkRef.Name}
	apiNetNetworkName, err := getAPINetNetworkName(ctx, r.Client, networkKey)
	if err != nil {
		return ctrl.Result{}, err
	}
	if apiNetNetworkName == "" {
		log.V(1).Info("APINet network is not ready")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Applying APINet load balancer")
	apiNetLoadBalancer, err := r.applyAPINetLoadBalancer(ctx, loadBalancer, apiNetNetworkName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying APINet load balancer: %w", err)
	}

	log.V(1).Info("Manage APINet load balancer routing")
	if err := r.manageAPINetLoadBalancerRouting(ctx, loadBalancer, apiNetLoadBalancer); err != nil {
		return ctrl.Result{}, err
	}

	actualIPs := apiNetIPsToIPs(apinetv1alpha1.GetLoadBalancerIPs(apiNetLoadBalancer))
	if !slices.Equal(actualIPs, loadBalancer.Status.IPs) {
		log.V(1).Info("Updating load balancer status IPs")
		if err := r.updateLoadBalancerIPs(ctx, loadBalancer, actualIPs); err != nil {
			return ctrl.Result{}, fmt.Errorf("error patching load balancer status")
		}
	}

	log.V(1).Info("Patched load balancer status")
	return ctrl.Result{}, nil
}

func (r *LoadBalancerReconciler) manageAPINetLoadBalancerRouting(ctx context.Context, loadBalancer *networkingv1alpha1.LoadBalancer, apiNetLoadBalancer *apinetv1alpha1.LoadBalancer) error {
	loadBalancerRouting := &networkingv1alpha1.LoadBalancerRouting{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(loadBalancer), loadBalancerRouting); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("error getting load balancer routing: %w", err)
	}

	apiNetDsts := make([]apinetv1alpha1.LoadBalancerDestination, 0)
	for _, dst := range loadBalancerRouting.Destinations {
		var apiNetTargetRef *apinetv1alpha1.LoadBalancerTargetRef
		if targetRef := dst.TargetRef; targetRef != nil {
			_, name, node, uid, err := provider.ParseNetworkInterfaceID(targetRef.ProviderID)
			if err == nil {
				apiNetTargetRef = &apinetv1alpha1.LoadBalancerTargetRef{
					UID:  uid,
					Name: name,
					NodeRef: corev1.LocalObjectReference{
						Name: node,
					},
				}
			}
		}

		apiNetDsts = append(apiNetDsts, apinetv1alpha1.LoadBalancerDestination{
			IP:        ipToAPINetIP(dst.IP),
			TargetRef: apiNetTargetRef,
		})
	}

	apiNetLoadBalancerRouting := &apinetv1alpha1.LoadBalancerRouting{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apinetv1alpha1.SchemeGroupVersion.String(),
			Kind:       "LoadBalancerRouting",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      apiNetLoadBalancer.Name,
			Labels:    apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), loadBalancer),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(apiNetLoadBalancer, apinetv1alpha1.SchemeGroupVersion.WithKind("LoadBalancer")),
			},
		},
		Destinations: apiNetDsts,
	}
	if err := r.APINetClient.Patch(ctx, apiNetLoadBalancerRouting, client.Apply, fieldOwner, client.ForceOwnership); err != nil {
		return fmt.Errorf("error applying APINet load balancer routing: %w", err)
	}
	return nil
}

func (r *LoadBalancerReconciler) getPublicLoadBalancerAPINetIPs(loadBalancer *networkingv1alpha1.LoadBalancer) []*apinetv1alpha1ac.LoadBalancerIPApplyConfiguration {
	res := make([]*apinetv1alpha1ac.LoadBalancerIPApplyConfiguration, len(loadBalancer.Spec.IPFamilies))
	for i, ipFamily := range loadBalancer.Spec.IPFamilies {
		res[i] = apinetv1alpha1ac.LoadBalancerIP().
			WithName(strings.ToLower(string(ipFamily))).
			WithIPFamily(ipFamily)
	}
	return res
}

func (r *LoadBalancerReconciler) getInternalLoadBalancerAPINetIPs(ctx context.Context, loadBalancer *networkingv1alpha1.LoadBalancer) ([]*apinetv1alpha1ac.LoadBalancerIPApplyConfiguration, error) {
	var ips []*apinetv1alpha1ac.LoadBalancerIPApplyConfiguration
	for i, ip := range loadBalancer.Spec.IPs {
		switch {
		case ip.Value != nil:
			ips = append(ips,
				apinetv1alpha1ac.LoadBalancerIP().
					WithName(fmt.Sprintf("ip-%d", i)).
					WithIPFamily(ip.Value.Family()).
					WithIP(net.IP{Addr: ip.Value.Addr}),
			)
		case ip.Ephemeral != nil:
			prefix := &ipamv1alpha1.Prefix{}
			prefixName := networkingv1alpha1.LoadBalancerIPIPAMPrefixName(loadBalancer.Name, i)
			prefixKey := client.ObjectKey{Namespace: loadBalancer.Namespace, Name: prefixName}
			if err := r.Get(ctx, prefixKey, prefix); err != nil {
				if !apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("error getting prefix %s: %w", prefixName, err)
				}

				continue
			}

			if !metav1.IsControlledBy(prefix, loadBalancer) {
				// Don't use a prefix that is not controlled by the load balancer.
				continue
			}

			if !isPrefixAllocated(prefix) {
				continue
			}

			ips = append(ips,
				apinetv1alpha1ac.LoadBalancerIP().
					WithName(fmt.Sprintf("ip-%d", i)).
					WithIPFamily(prefix.Spec.IPFamily).
					WithIP(net.IP{Addr: prefix.Spec.Prefix.IP().Addr}),
			)
		}
	}
	return ips, nil
}

func (r *LoadBalancerReconciler) applyAPINetLoadBalancer(ctx context.Context, loadBalancer *networkingv1alpha1.LoadBalancer, apiNetNetworkName string) (*apinetv1alpha1.LoadBalancer, error) {
	apiNetLoadBalancerType, err := loadBalancerTypeToAPINetLoadBalancerType(loadBalancer.Spec.Type)
	if err != nil {
		return nil, err
	}

	var ips []*apinetv1alpha1ac.LoadBalancerIPApplyConfiguration
	switch loadBalancer.Spec.Type {
	case networkingv1alpha1.LoadBalancerTypeInternal:
		ips, err = r.getInternalLoadBalancerAPINetIPs(ctx, loadBalancer)
		if err != nil {
			return nil, err
		}
	case networkingv1alpha1.LoadBalancerTypePublic:
		ips = r.getPublicLoadBalancerAPINetIPs(loadBalancer)
	}

	apiNetLoadBalancerApplyCfg := apinetv1alpha1ac.LoadBalancer(string(loadBalancer.UID), r.APINetNamespace).
		WithLabels(apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), loadBalancer)).
		WithSpec(apinetv1alpha1ac.LoadBalancerSpec().
			WithType(apiNetLoadBalancerType).
			WithNetworkRef(corev1.LocalObjectReference{Name: apiNetNetworkName}).
			WithIPs(ips...).
			WithPorts(loadBalancerPortsToAPINetLoadBalancerPortConfigs(loadBalancer.Spec.Ports)...).
			WithSelector(metav1ac.LabelSelector().WithMatchLabels(apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), loadBalancer))).
			WithTemplate(
				apinetv1alpha1ac.InstanceTemplate().
					WithLabels(apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), loadBalancer)).
					WithSpec(apinetv1alpha1ac.InstanceSpec().
						WithAffinity(apinetv1alpha1ac.Affinity().
							WithInstanceAntiAffinity(apinetv1alpha1ac.InstanceAntiAffinity().WithRequiredDuringSchedulingIgnoredDuringExecution(
								apinetv1alpha1ac.InstanceAffinityTerm().
									WithTopologyKey(apinetv1alpha1.TopologyZoneLabel).
									WithLabelSelector(metav1ac.LabelSelector().WithMatchLabels(apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), loadBalancer))),
							)),
						),
					),
			),
		)
	apiNetLoadBalancer, err := r.APINetInterface.CoreV1alpha1().
		LoadBalancers(r.APINetNamespace).
		Apply(ctx, apiNetLoadBalancerApplyCfg, metav1.ApplyOptions{FieldManager: string(fieldOwner), Force: true})
	if err != nil {
		return nil, fmt.Errorf("error applying APINet load balancer: %w", err)
	}
	return apiNetLoadBalancer, nil
}

func (r *LoadBalancerReconciler) updateLoadBalancerIPs(ctx context.Context, loadBalancer *networkingv1alpha1.LoadBalancer, ips []commonv1alpha1.IP) error {
	base := loadBalancer.DeepCopy()
	loadBalancer.Status.IPs = ips
	return r.Status().Patch(ctx, loadBalancer, client.MergeFrom(base))
}

func (r *LoadBalancerReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCache cache.Cache) error {
	log := ctrl.Log.WithName("loadbalancer").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1alpha1.LoadBalancer{},
			builder.WithPredicates(
				predicates.ResourceHasFilterLabel(log, r.WatchFilterValue),
				predicates.ResourceIsNotExternallyManaged(log),
			),
		).
		WatchesRawSource(
			source.Kind(apiNetCache, &apinetv1alpha1.LoadBalancer{}),
			apinetlethandler.EnqueueRequestForSource(r.Scheme(), r.RESTMapper(), &networkingv1alpha1.LoadBalancer{}),
		).
		Owns(&ipamv1alpha1.Prefix{}).
		Watches(
			&networkingv1alpha1.LoadBalancerRouting{},
			&handler.EnqueueRequestForObject{},
		).
		Complete(r)
}
