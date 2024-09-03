// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/go-logr/logr"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	apinetlethandler "github.com/ironcore-dev/ironcore-net/apinetlet/handler"
	apinetv1alpha1ac "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/client-go/ironcorenet"
	apinetclient "github.com/ironcore-dev/ironcore-net/internal/client"

	"github.com/ironcore-dev/controller-utils/clientutils"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/predicates"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/workqueue"
	klog "k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	networkPolicyFinalizer = "apinet.ironcore.dev/networkpolicy"
)

var networkPolicyFieldOwner = client.FieldOwner(networkingv1alpha1.Resource("networkpolicies").String())

type NetworkPolicyReconciler struct {
	client.Client
	APINetClient    client.Client
	APINetInterface ironcorenet.Interface

	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=networkpolicies,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=networkpolicies/finalizers,verbs=update;patch

//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkpolicyrules,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networks,verbs=get;list;watch
//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkinterfaces,verbs=get;list;watch

func (r *NetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	networkPolicy := &networkingv1alpha1.NetworkPolicy{}
	if err := r.Get(ctx, req.NamespacedName, networkPolicy); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting network policy %s: %w", req.NamespacedName, err)
		}

		return r.deleteGone(ctx, log, req.NamespacedName)
	}

	return r.reconcileExists(ctx, log, networkPolicy)
}

func (r *NetworkPolicyReconciler) deleteGone(ctx context.Context, log logr.Logger, networkPolicyKey client.ObjectKey) (ctrl.Result, error) {
	log.V(1).Info("Delete gone")

	log.V(1).Info("Deleting any matching APINet network policies")
	if err := r.APINetClient.DeleteAllOf(ctx, &apinetv1alpha1.NetworkPolicy{},
		client.InNamespace(r.APINetNamespace),
		apinetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), networkPolicyKey, &networkingv1alpha1.NetworkPolicy{}),
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting APINet network policies: %w", err)
	}

	log.V(1).Info("Deleted any leftover APINet network policy")
	return ctrl.Result{}, nil
}

func (r *NetworkPolicyReconciler) reconcileExists(ctx context.Context, log logr.Logger, networkPolicy *networkingv1alpha1.NetworkPolicy) (ctrl.Result, error) {
	log = log.WithValues("UID", networkPolicy.UID)
	if !networkPolicy.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, networkPolicy)
	}
	return r.reconcile(ctx, log, networkPolicy)
}

func (r *NetworkPolicyReconciler) delete(ctx context.Context, log logr.Logger, networkPolicy *networkingv1alpha1.NetworkPolicy) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(networkPolicy, networkPolicyFinalizer) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	apiNetNetworkPolicy := &apinetv1alpha1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(networkPolicy.UID),
		},
	}
	if err := r.APINetClient.Delete(ctx, apiNetNetworkPolicy); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting APINet network policy: %w", err)
		}
	}

	log.V(1).Info("APINet network policy is gone, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, networkPolicy, networkPolicyFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}

	log.V(1).Info("Deleted")

	return ctrl.Result{}, nil
}

func (r *NetworkPolicyReconciler) reconcile(ctx context.Context, log logr.Logger, networkPolicy *networkingv1alpha1.NetworkPolicy) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, networkPolicy, networkPolicyFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	networkKey := client.ObjectKey{Namespace: networkPolicy.Namespace, Name: networkPolicy.Spec.NetworkRef.Name}
	apiNetNetworkName, err := getAPINetNetworkName(ctx, r.Client, networkKey)
	if err != nil {
		return ctrl.Result{}, err
	}
	if apiNetNetworkName == "" {
		log.V(1).Info("APINet network is not ready")
		return ctrl.Result{}, nil
	}

	apiNetNetworkKey := client.ObjectKey{Namespace: r.APINetNamespace, Name: apiNetNetworkName}
	apiNetNetwork, err := getApiNetNetwork(ctx, r.APINetClient, apiNetNetworkKey)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("Applying APINet network policy")
	apiNetNetworkPolicy, err := r.applyAPINetNetworkPolicy(ctx, networkPolicy, apiNetNetworkName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying APINet network policy: %w", err)
	}

	log.V(1).Info("Finding APINet network interface targets")
	targets, err := r.findTargets(ctx, apiNetNetworkPolicy)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error finding targets: %w", err)
	}

	log.V(1).Info("Parsing ingress rules")
	ingressRules, err := r.parseIngressRules(ctx, apiNetNetworkPolicy)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error parsing ingress rules: %w", err)
	}

	log.V(1).Info("Parsing egress rules")
	egressRules, err := r.parseEgressRules(ctx, apiNetNetworkPolicy)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error parsing egress rules: %w", err)
	}

	log.V(1).Info("Applying APINet network policy rule", "targets", targets, "Network", klog.KObj(apiNetNetwork))
	if err := r.applyNetworkPolicyRule(ctx, networkPolicy, apiNetNetworkPolicy, targets, apiNetNetwork, ingressRules, egressRules); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying APINet network policy rule: %w", err)
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkPolicyReconciler) findTargets(ctx context.Context, apiNetNetworkPolicy *apinetv1alpha1.NetworkPolicy) ([]apinetv1alpha1.TargetNetworkInterface, error) {
	sel, err := metav1.LabelSelectorAsSelector(&apiNetNetworkPolicy.Spec.NetworkInterfaceSelector)
	if err != nil {
		return nil, err
	}

	apiNetNicList := &apinetv1alpha1.NetworkInterfaceList{}
	if err := r.APINetClient.List(ctx, apiNetNicList,
		client.InNamespace(r.APINetNamespace),
		client.MatchingLabelsSelector{Selector: sel},
		client.MatchingFields{apinetclient.NetworkInterfaceSpecNetworkRefNameField: apiNetNetworkPolicy.Spec.NetworkRef.Name},
	); err != nil {
		return nil, fmt.Errorf("error listing APINet network interfaces: %w", err)
	}

	// Make slice non-nil so omitempty does not file.
	targets := make([]apinetv1alpha1.TargetNetworkInterface, 0)
	for _, apiNetNic := range apiNetNicList.Items {
		if apiNetNic.Status.State != apinetv1alpha1.NetworkInterfaceStateReady {
			continue
		}

		for _, ip := range apiNetNic.Spec.IPs {
			targets = append(targets, apinetv1alpha1.TargetNetworkInterface{
				IP: ip,
				TargetRef: &apinetv1alpha1.LocalUIDReference{
					UID:  apiNetNic.UID,
					Name: apiNetNic.Name,
				},
			})
		}
	}

	return targets, nil
}

func (r *NetworkPolicyReconciler) parseIngressRules(ctx context.Context, np *apinetv1alpha1.NetworkPolicy) ([]apinetv1alpha1.Rule, error) {
	var rules []apinetv1alpha1.Rule

	for _, ingress := range np.Spec.Ingress {
		rule := apinetv1alpha1.Rule{}
		for _, port := range ingress.Ports {
			rule.NetworkPolicyPorts = append(rule.NetworkPolicyPorts, apinetv1alpha1.NetworkPolicyPort{
				Protocol: port.Protocol,
				Port:     port.Port,
				EndPort:  port.EndPort,
			})
		}

		for _, from := range ingress.From {
			if from.IPBlock != nil {
				rule.CIDRBlock = append(rule.CIDRBlock, *from.IPBlock)
			}

			if from.ObjectSelector != nil {
				ips, err := r.processObjectSelector(ctx, np, from.ObjectSelector)
				if err != nil {
					return nil, err
				}
				rule.ObjectIPs = append(rule.ObjectIPs, ips...)
			}
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (r *NetworkPolicyReconciler) parseEgressRules(ctx context.Context, np *apinetv1alpha1.NetworkPolicy) ([]apinetv1alpha1.Rule, error) {
	var rules []apinetv1alpha1.Rule

	for _, egress := range np.Spec.Egress {
		rule := apinetv1alpha1.Rule{}
		for _, port := range egress.Ports {
			rule.NetworkPolicyPorts = append(rule.NetworkPolicyPorts, apinetv1alpha1.NetworkPolicyPort{
				Protocol: port.Protocol,
				Port:     port.Port,
				EndPort:  port.EndPort,
			})
		}

		for _, to := range egress.To {
			if to.IPBlock != nil {
				rule.CIDRBlock = append(rule.CIDRBlock, *to.IPBlock)
			}

			if to.ObjectSelector != nil {
				ips, err := r.processObjectSelector(ctx, np, to.ObjectSelector)
				if err != nil {
					return nil, err
				}
				rule.ObjectIPs = append(rule.ObjectIPs, ips...)
			}
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (r *NetworkPolicyReconciler) processObjectSelector(ctx context.Context, np *apinetv1alpha1.NetworkPolicy, objectSelector *apinetv1alpha1.ObjectSelector) ([]apinetv1alpha1.ObjectIP, error) {
	switch objectSelector.Kind {
	case "NetworkInterface":
		return r.fetchIPsFromNetworkInterfaces(ctx, np, objectSelector)
	case "LoadBalancer":
		return r.fetchIPsFromLoadBalancers(ctx, np, objectSelector)
	// TODO: add more objects selector support if needed
	default:
		return nil, fmt.Errorf("unsupported object kind: %s", objectSelector.Kind)
	}
}

func (r *NetworkPolicyReconciler) fetchIPsFromNetworkInterfaces(ctx context.Context, np *apinetv1alpha1.NetworkPolicy, objectSelector *apinetv1alpha1.ObjectSelector) ([]apinetv1alpha1.ObjectIP, error) {
	sel, err := metav1.LabelSelectorAsSelector(&objectSelector.LabelSelector)
	if err != nil {
		return nil, err
	}

	nicList := &apinetv1alpha1.NetworkInterfaceList{}
	if err := r.APINetClient.List(ctx, nicList,
		client.InNamespace(np.Namespace),
		client.MatchingLabelsSelector{Selector: sel},
		client.MatchingFields{apinetclient.NetworkInterfaceSpecNetworkRefNameField: np.Spec.NetworkRef.Name},
	); err != nil {
		return nil, fmt.Errorf("error listing APINet network interfaces: %w", err)
	}

	var ips []apinetv1alpha1.ObjectIP

	for _, nic := range nicList.Items {
		if nic.Status.State != apinetv1alpha1.NetworkInterfaceStateReady {
			continue
		}

		for _, ip := range nic.Spec.IPs {
			ipFamily := corev1.IPv4Protocol
			if ip.Addr.Is6() {
				ipFamily = corev1.IPv6Protocol
			}
			ips = append(ips, apinetv1alpha1.ObjectIP{
				Prefix:   net.IPPrefix{Prefix: netip.PrefixFrom(ip.Addr, ip.Addr.BitLen())},
				IPFamily: ipFamily,
			})
		}
	}

	return ips, nil
}

func (r *NetworkPolicyReconciler) fetchIPsFromLoadBalancers(ctx context.Context, np *apinetv1alpha1.NetworkPolicy, objectSelector *apinetv1alpha1.ObjectSelector) ([]apinetv1alpha1.ObjectIP, error) {
	sel, err := metav1.LabelSelectorAsSelector(&objectSelector.LabelSelector)
	if err != nil {
		return nil, err
	}

	// TODO: apinet load balancer need to inherit labels from ironcore load balancer
	lbList := &apinetv1alpha1.LoadBalancerList{}
	if err := r.APINetClient.List(ctx, lbList,
		client.InNamespace(np.Namespace),
		client.MatchingLabelsSelector{Selector: sel},
	); err != nil {
		return nil, fmt.Errorf("error listing apinet load balancers: %w", err)
	}

	var ips []apinetv1alpha1.ObjectIP

	for _, lb := range lbList.Items {
		// TODO: handle loadbalancer ports
		for _, ip := range lb.Spec.IPs {
			// TODO: handle LoadBalancerIP when only IPFamily is specified to allocate a random IP.
			ips = append(ips, apinetv1alpha1.ObjectIP{
				Prefix:   net.IPPrefix{Prefix: netip.PrefixFrom(ip.IP.Addr, ip.IP.Addr.BitLen())},
				IPFamily: ip.IPFamily,
			})
		}
	}

	return ips, nil
}

func (r *NetworkPolicyReconciler) applyNetworkPolicyRule(ctx context.Context, networkPolicy *networkingv1alpha1.NetworkPolicy, apiNetNetworkPolicy *apinetv1alpha1.NetworkPolicy, targets []apinetv1alpha1.TargetNetworkInterface, network *apinetv1alpha1.Network, ingressRules, egressRules []apinetv1alpha1.Rule) error {
	networkPolicyRule := &apinetv1alpha1.NetworkPolicyRule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicyRule",
			APIVersion: apinetv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: apiNetNetworkPolicy.Namespace,
			Name:      apiNetNetworkPolicy.Name,
			Labels:    apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), networkPolicy),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(apiNetNetworkPolicy, apinetv1alpha1.SchemeGroupVersion.WithKind("NetworkPolicy")),
			},
		},
		NetworkRef: apinetv1alpha1.LocalUIDReference{
			Name: network.Name,
			UID:  network.UID,
		},
		Priority:     apiNetNetworkPolicy.Spec.Priority,
		Targets:      targets,
		IngressRules: ingressRules,
		EgressRules:  egressRules,
	}
	err := ctrl.SetControllerReference(apiNetNetworkPolicy, networkPolicyRule, r.Scheme())
	if err != nil {
		return fmt.Errorf("error setting controller reference: %w", err)
	}

	if err := r.APINetClient.Patch(ctx, networkPolicyRule, client.Apply, networkPolicyFieldOwner, client.ForceOwnership); err != nil {
		return fmt.Errorf("error applying network policy rule: %w", err)
	}
	return nil
}

func (r *NetworkPolicyReconciler) applyAPINetNetworkPolicy(ctx context.Context, networkPolicy *networkingv1alpha1.NetworkPolicy, apiNetNetworkName string) (*apinetv1alpha1.NetworkPolicy, error) {
	var apiNetIngressRules []*apinetv1alpha1ac.NetworkPolicyIngressRuleApplyConfiguration
	for _, ingressRule := range networkPolicy.Spec.Ingress {
		apiNetIngressRule := &apinetv1alpha1ac.NetworkPolicyIngressRuleApplyConfiguration{
			From:  translatePeers(ingressRule.From),
			Ports: translatePorts(ingressRule.Ports),
		}
		apiNetIngressRules = append(apiNetIngressRules, apiNetIngressRule)
	}

	var apiNetEgressRules []*apinetv1alpha1ac.NetworkPolicyEgressRuleApplyConfiguration
	for _, egressRule := range networkPolicy.Spec.Egress {
		apiNetEgressRule := &apinetv1alpha1ac.NetworkPolicyEgressRuleApplyConfiguration{
			To:    translatePeers(egressRule.To),
			Ports: translatePorts(egressRule.Ports),
		}
		apiNetEgressRules = append(apiNetEgressRules, apiNetEgressRule)
	}

	apiNetNetworkPolicyTypes, err := networkPolicyTypesToAPINetNetworkPolicyTypes(networkPolicy.Spec.PolicyTypes)
	if err != nil {
		return nil, err
	}

	nicSelector := translateLabelSelector(networkPolicy.Spec.NetworkInterfaceSelector)

	apiNetNetworkPolicyApplyCfg := apinetv1alpha1ac.NetworkPolicy(string(networkPolicy.UID), r.APINetNamespace).
		WithLabels(apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), networkPolicy)).
		WithSpec(apinetv1alpha1ac.NetworkPolicySpec().
			WithNetworkRef(corev1.LocalObjectReference{Name: apiNetNetworkName}).
			WithNetworkInterfaceSelector(nicSelector).
			WithPriority(1000). // set default value since networkingv1alpha1.NetworkPolicy does not have this field
			WithIngress(apiNetIngressRules...).
			WithEgress(apiNetEgressRules...).
			WithPolicyTypes(apiNetNetworkPolicyTypes...),
		)
	apiNetNetworkPolicy, err := r.APINetInterface.CoreV1alpha1().
		NetworkPolicies(r.APINetNamespace).
		Apply(ctx, apiNetNetworkPolicyApplyCfg, metav1.ApplyOptions{FieldManager: string(fieldOwner), Force: true})
	if err != nil {
		return nil, fmt.Errorf("error applying APINet network policy: %w", err)
	}
	return apiNetNetworkPolicy, nil
}

func (r *NetworkPolicyReconciler) enqueueByNetwork() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		log := ctrl.LoggerFrom(ctx)
		apiNetNetwork := obj.(*apinetv1alpha1.Network)

		networkPolicyList := &apinetv1alpha1.NetworkPolicyList{}
		if err := r.APINetClient.List(ctx, networkPolicyList,
			client.InNamespace(apiNetNetwork.Namespace),
			client.MatchingFields{apinetletclient.NetworkPolicyNetworkNameField: apiNetNetwork.Name},
		); err != nil {
			log.Error(err, "Error listing network policies for network")
			return nil
		}

		return apinetletclient.ReconcileRequestsFromObjectStructSlice[*apinetv1alpha1.NetworkPolicy](networkPolicyList.Items)
	})
}

func (r *NetworkPolicyReconciler) enqueueByNetworkInterface() handler.EventHandler {
	getEnqueueFunc := func(ctx context.Context, nic *apinetv1alpha1.NetworkInterface) func(nics []*apinetv1alpha1.NetworkInterface, queue workqueue.RateLimitingInterface) {
		log := ctrl.LoggerFrom(ctx)
		networkPolicyList := &apinetv1alpha1.NetworkPolicyList{}
		if err := r.APINetClient.List(ctx, networkPolicyList,
			client.InNamespace(nic.Namespace),
			client.MatchingFields{apinetletclient.NetworkPolicyNetworkNameField: nic.Spec.NetworkRef.Name},
		); err != nil {
			log.Error(err, "Error listing APINet network policies for nic")
			return nil
		}

		return func(nics []*apinetv1alpha1.NetworkInterface, queue workqueue.RateLimitingInterface) {
			for _, networkPolicy := range networkPolicyList.Items {
				networkPolicyKey := client.ObjectKeyFromObject(&networkPolicy)
				log := log.WithValues("networkPolicyKey", networkPolicyKey)
				nicSelector := networkPolicy.Spec.NetworkInterfaceSelector
				if len(nicSelector.MatchLabels) == 0 && len(nicSelector.MatchExpressions) == 0 {
					return
				}

				sel, err := metav1.LabelSelectorAsSelector(&nicSelector)
				if err != nil {
					log.Error(err, "Invalid network interface selector")
					continue
				}

				for _, nic := range nics {
					if sel.Matches(labels.Set(nic.Labels)) {
						queue.Add(ctrl.Request{NamespacedName: networkPolicyKey})
						break
					}
				}
			}
		}
	}

	return handler.Funcs{
		CreateFunc: func(ctx context.Context, evt event.CreateEvent, queue workqueue.RateLimitingInterface) {
			nic := evt.Object.(*apinetv1alpha1.NetworkInterface)
			enqueueFunc := getEnqueueFunc(ctx, nic)
			if enqueueFunc != nil {
				enqueueFunc([]*apinetv1alpha1.NetworkInterface{nic}, queue)
			}
		},
		UpdateFunc: func(ctx context.Context, evt event.UpdateEvent, queue workqueue.RateLimitingInterface) {
			newNic := evt.ObjectNew.(*apinetv1alpha1.NetworkInterface)
			oldNic := evt.ObjectOld.(*apinetv1alpha1.NetworkInterface)
			enqueueFunc := getEnqueueFunc(ctx, newNic)
			if enqueueFunc != nil {
				enqueueFunc([]*apinetv1alpha1.NetworkInterface{newNic, oldNic}, queue)
			}
		},
		DeleteFunc: func(ctx context.Context, evt event.DeleteEvent, queue workqueue.RateLimitingInterface) {
			nic := evt.Object.(*apinetv1alpha1.NetworkInterface)
			enqueueFunc := getEnqueueFunc(ctx, nic)
			if enqueueFunc != nil {
				enqueueFunc([]*apinetv1alpha1.NetworkInterface{nic}, queue)
			}
		},
		GenericFunc: func(ctx context.Context, evt event.GenericEvent, queue workqueue.RateLimitingInterface) {
			nic := evt.Object.(*apinetv1alpha1.NetworkInterface)
			enqueueFunc := getEnqueueFunc(ctx, nic)
			if enqueueFunc != nil {
				enqueueFunc([]*apinetv1alpha1.NetworkInterface{nic}, queue)
			}
		},
	}
}

func (r *NetworkPolicyReconciler) networkInterfaceReadyPredicate() predicate.Predicate {
	isNetworkInterfaceReady := func(nic *apinetv1alpha1.NetworkInterface) bool {
		return nic.Status.State == apinetv1alpha1.NetworkInterfaceStateReady
	}
	return predicate.Funcs{
		CreateFunc: func(evt event.CreateEvent) bool {
			nic := evt.Object.(*apinetv1alpha1.NetworkInterface)
			return isNetworkInterfaceReady(nic)
		},
		UpdateFunc: func(evt event.UpdateEvent) bool {
			oldNic := evt.ObjectOld.(*apinetv1alpha1.NetworkInterface)
			newNic := evt.ObjectNew.(*apinetv1alpha1.NetworkInterface)
			return isNetworkInterfaceReady(oldNic) || isNetworkInterfaceReady(newNic)
		},
		DeleteFunc: func(evt event.DeleteEvent) bool {
			nic := evt.Object.(*apinetv1alpha1.NetworkInterface)
			return isNetworkInterfaceReady(nic)
		},
		GenericFunc: func(evt event.GenericEvent) bool {
			nic := evt.Object.(*apinetv1alpha1.NetworkInterface)
			return isNetworkInterfaceReady(nic)
		},
	}
}

func (r *NetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCache cache.Cache) error {
	log := ctrl.Log.WithName("networkpolicy").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1alpha1.NetworkPolicy{},
			builder.WithPredicates(
				predicates.ResourceHasFilterLabel(log, r.WatchFilterValue),
				predicates.ResourceIsNotExternallyManaged(log),
			),
		).
		WatchesRawSource(
			source.Kind(apiNetCache, &apinetv1alpha1.NetworkPolicy{}),
			apinetlethandler.EnqueueRequestForSource(r.Scheme(), r.RESTMapper(), &networkingv1alpha1.NetworkPolicy{}),
		).
		WatchesRawSource(
			source.Kind(apiNetCache, &apinetv1alpha1.Network{}),
			r.enqueueByNetwork(),
		).
		WatchesRawSource(
			source.Kind(apiNetCache, &apinetv1alpha1.NetworkInterface{}),
			r.enqueueByNetworkInterface(),
			builder.WithPredicates(r.networkInterfaceReadyPredicate()),
		).
		Complete(r)
}
