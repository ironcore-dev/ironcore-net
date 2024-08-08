// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/ironcore-dev/controller-utils/clientutils"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	metalnetletclient "github.com/ironcore-dev/ironcore-net/metalnetlet/client"
	utilhandler "github.com/ironcore-dev/ironcore-net/metalnetlet/handler"
	netiputils "github.com/ironcore-dev/ironcore-net/utils/netip"
	"github.com/ironcore-dev/ironcore/utils/generic"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"

	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type NetworkInterfaceReconciler struct {
	client.Client
	MetalnetClient client.Client

	PartitionName string

	MetalnetNamespace string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkinterfaces,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkinterfaces/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkinterfaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=loadbalancerroutings,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=loadbalancers,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=nattables,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=natgateways,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkpolicies,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkpolicyrules,verbs=get;list;watch

//+cluster=metalnet:kubebuilder:rbac:groups=networking.metalnet.ironcore.dev,resources=networkinterfaces,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+cluster=metalnet:kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

func (r *NetworkInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	nic := &v1alpha1.NetworkInterface{}
	if err := r.Get(ctx, req.NamespacedName, nic); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		log.V(1).Info("Deleting all matching metalnet network interfaces")
		exists, err := metalnetletclient.DeleteAllOfAndAnyExists(ctx, r.MetalnetClient, &metalnetv1alpha1.NetworkInterface{},
			client.InNamespace(r.MetalnetNamespace),
			metalnetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), req.NamespacedName, &v1alpha1.NetworkInterface{}),
		)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error deleting matching metalnet network interfaces: %w", err)
		}
		if exists {
			log.V(1).Info("Matching metalnet network interfaces are still present, requeueing")
			return ctrl.Result{Requeue: true}, nil
		}
		log.V(1).Info("All matching metalnet network interfaces are gone")
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, nic)
}

func (r *NetworkInterfaceReconciler) reconcileExists(ctx context.Context, log logr.Logger, nic *v1alpha1.NetworkInterface) (ctrl.Result, error) {
	if !nic.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, nic)
	}
	return r.reconcile(ctx, log, nic)
}

func (r *NetworkInterfaceReconciler) delete(ctx context.Context, log logr.Logger, nic *v1alpha1.NetworkInterface) (ctrl.Result, error) {
	log.V(1).Info("Delete")
	if !controllerutil.ContainsFinalizer(nic, PartitionFinalizer(r.PartitionName)) {
		log.V(1).Info("No partition finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Partition finalizer present, doing cleanup")

	log.V(1).Info("Deleting any matching metalnet network interface")
	metalnetNic := &metalnetv1alpha1.NetworkInterface{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.MetalnetNamespace,
			Name:      string(nic.UID),
		},
	}
	if err := r.MetalnetClient.Delete(ctx, metalnetNic); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting metalnet network interface %s: %w", metalnetNic.Name, err)
		}
		log.V(1).Info("Any matching metalnet network interface deleted")

		log.V(1).Info("Removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, nic, PartitionFinalizer(r.PartitionName)); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Finalizer removed")
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Issued metalnet network interface deletion, requeueing")
	return ctrl.Result{Requeue: true}, nil
}

func (r *NetworkInterfaceReconciler) getMetalnetNetworkNameForNetworkInterface(ctx context.Context, nic *v1alpha1.NetworkInterface) (string, error) {
	network := &v1alpha1.Network{}
	networkKey := client.ObjectKey{Namespace: nic.Namespace, Name: nic.Spec.NetworkRef.Name}
	if err := r.Get(ctx, networkKey, network); err != nil {
		if !apierrors.IsNotFound(err) {
			return "", fmt.Errorf("error getting network %s: %w", networkKey.Name, err)
		}
		return "", nil
	}

	return string(network.UID), nil
}

func (r *NetworkInterfaceReconciler) getLoadBalancerTargetsForNetworkInterface(ctx context.Context, nic *v1alpha1.NetworkInterface) ([]net.IP, error) {
	lbRoutingList := &v1alpha1.LoadBalancerRoutingList{}
	if err := r.List(ctx, lbRoutingList,
		client.InNamespace(nic.Namespace),
	); err != nil {
		return nil, fmt.Errorf("error listing load balancer routings: %w", err)
	}

	ipSet := sets.New[net.IP]()
	for _, lbRouting := range lbRoutingList.Items {
		hasDst := slices.ContainsFunc(lbRouting.Destinations,
			func(dst v1alpha1.LoadBalancerDestination) bool {
				return slices.Contains(nic.Spec.IPs, dst.IP)
			},
		)
		if hasDst {
			loadBalancer := &v1alpha1.LoadBalancer{}
			loadBalancerKey := client.ObjectKeyFromObject(&lbRouting)
			if err := r.Get(ctx, loadBalancerKey, loadBalancer); client.IgnoreNotFound(err) != nil {
				return nil, err
			}

			ipSet.Insert(v1alpha1.GetLoadBalancerIPs(loadBalancer)...)
		}
	}

	ips := ipSet.UnsortedList()
	slices.SortFunc(ips, func(ip1, ip2 net.IP) int { return ip1.Compare(ip2.Addr) })
	return ips, nil
}

func (r *NetworkInterfaceReconciler) getNetworkPolicyRulesForNetworkInterface(ctx context.Context, nic *v1alpha1.NetworkInterface) ([]metalnetv1alpha1.FirewallRule, error) {
	var firewallRules []metalnetv1alpha1.FirewallRule

	npRuleList := &v1alpha1.NetworkPolicyRuleList{}
	if err := r.List(ctx, npRuleList,
		client.InNamespace(nic.Namespace),
	); err != nil {
		return nil, fmt.Errorf("error listing network policy rules: %w", err)
	}

	for _, npRule := range npRuleList.Items {
		hasDst := slices.ContainsFunc(npRule.Targets,
			func(target v1alpha1.TargetNetworkInterface) bool {
				return slices.Contains(nic.Spec.IPs, target.IP)
			},
		)
		if hasDst {
			rules := getFirewallRulesFromNetworkPolicyRule(&npRule)
			firewallRules = append(firewallRules, rules...)
		}
	}

	return firewallRules, nil
}

func getFirewallRulesFromNetworkPolicyRule(npRule *v1alpha1.NetworkPolicyRule) []metalnetv1alpha1.FirewallRule {
	var firewallRules []metalnetv1alpha1.FirewallRule
	priority := npRule.Priority

	for _, ingressRule := range npRule.IngressRules {
		rules := extractFirewallRulesFromRule(ingressRule, metalnetv1alpha1.FirewallRuleDirectionIngress, priority)
		firewallRules = append(firewallRules, rules...)
	}

	for _, egressRule := range npRule.EgressRules {
		rules := extractFirewallRulesFromRule(egressRule, metalnetv1alpha1.FirewallRuleDirectionEgress, priority)
		firewallRules = append(firewallRules, rules...)
	}

	return firewallRules
}

func extractFirewallRulesFromRule(rule v1alpha1.Rule, direction metalnetv1alpha1.FirewallRuleDirection, priority *int32) []metalnetv1alpha1.FirewallRule {
	var firewallRules []metalnetv1alpha1.FirewallRule

	for _, port := range rule.NetworkPolicyPorts {
		baseFirewallRule := metalnetv1alpha1.FirewallRule{
			Direction:     direction,
			Action:        metalnetv1alpha1.FirewallRuleActionAccept,
			Priority:      priority,
			ProtocolMatch: &metalnetv1alpha1.ProtocolMatch{},
		}

		switch *port.Protocol {
		case corev1.ProtocolTCP:
			baseFirewallRule.ProtocolMatch.ProtocolType = generic.Pointer(metalnetv1alpha1.FirewallRuleProtocolTypeTCP)
		case corev1.ProtocolUDP:
			baseFirewallRule.ProtocolMatch.ProtocolType = generic.Pointer(metalnetv1alpha1.FirewallRuleProtocolTypeUDP)
			//TODO: no support for SCTP protocol in metalnetlet and metalnetlet FirewallRuleProtocolTypeICMP is not defined in ironcore
		}

		if port.Port != 0 {
			if direction == metalnetv1alpha1.FirewallRuleDirectionIngress {
				baseFirewallRule.ProtocolMatch.PortRange = &metalnetv1alpha1.PortMatch{SrcPort: &port.Port}
			} else {
				baseFirewallRule.ProtocolMatch.PortRange = &metalnetv1alpha1.PortMatch{DstPort: &port.Port}
			}
			if port.EndPort != nil {
				if direction == metalnetv1alpha1.FirewallRuleDirectionIngress {
					baseFirewallRule.ProtocolMatch.PortRange.EndSrcPort = *port.EndPort
				} else {
					baseFirewallRule.ProtocolMatch.PortRange.EndDstPort = *port.EndPort
				}
			}
		}

		for _, cidrBlock := range rule.CIDRBlock {
			firewallRule := baseFirewallRule
			firewallRule.FirewallRuleID = types.UID(uuid.New().String())
			firewallRule.IpFamily = netiputils.GetIPFamilyFromPrefix(cidrBlock.CIDR)

			if direction == metalnetv1alpha1.FirewallRuleDirectionIngress {
				firewallRule.SourcePrefix = &metalnetv1alpha1.IPPrefix{Prefix: cidrBlock.CIDR.Prefix}
			} else {
				firewallRule.DestinationPrefix = &metalnetv1alpha1.IPPrefix{Prefix: cidrBlock.CIDR.Prefix}
			}

			firewallRules = append(firewallRules, firewallRule)

			if len(cidrBlock.Except) > 0 {
				for _, exceptCIDR := range cidrBlock.Except {
					exceptFirewallRule := firewallRule
					exceptFirewallRule.FirewallRuleID = types.UID(uuid.New().String())
					exceptFirewallRule.Action = metalnetv1alpha1.FirewallRuleActionDeny

					if direction == metalnetv1alpha1.FirewallRuleDirectionIngress {
						exceptFirewallRule.SourcePrefix = &metalnetv1alpha1.IPPrefix{Prefix: exceptCIDR.Prefix}
					} else {
						exceptFirewallRule.DestinationPrefix = &metalnetv1alpha1.IPPrefix{Prefix: exceptCIDR.Prefix}
					}

					firewallRules = append(firewallRules, exceptFirewallRule)
				}
			}
		}

		for _, objectIP := range rule.ObjectIPs {
			firewallRule := baseFirewallRule
			firewallRule.FirewallRuleID = types.UID(uuid.New().String())
			firewallRule.IpFamily = netiputils.GetIPFamilyFromPrefix(objectIP.Prefix)

			if direction == metalnetv1alpha1.FirewallRuleDirectionIngress {
				firewallRule.SourcePrefix = &metalnetv1alpha1.IPPrefix{Prefix: objectIP.Prefix.Prefix}
			} else {
				firewallRule.DestinationPrefix = &metalnetv1alpha1.IPPrefix{Prefix: objectIP.Prefix.Prefix}
			}

			firewallRules = append(firewallRules, firewallRule)
		}
	}

	return firewallRules
}

func (r *NetworkInterfaceReconciler) getNATDetailsForNetworkInterface(
	ctx context.Context,
	nic *v1alpha1.NetworkInterface,
) ([]metalnetv1alpha1.NATDetails, error) {
	var res []metalnetv1alpha1.NATDetails
	for _, nat := range nic.Spec.NATs {
		natDetails, err := r.getNATIPsForNetworkInterfaceNAT(ctx, nic, &nat)
		if err != nil {
			return nil, err
		}
		if natDetails == nil {
			continue
		}

		res = append(res, *natDetails)
	}
	return res, nil
}

func (r *NetworkInterfaceReconciler) getNATIPsForNetworkInterfaceNAT(
	ctx context.Context,
	nic *v1alpha1.NetworkInterface,
	nat *v1alpha1.NetworkInterfaceNAT,
) (*metalnetv1alpha1.NATDetails, error) {
	natGateway := &v1alpha1.NATGateway{}
	natGatewayKey := client.ObjectKey{Namespace: nic.Namespace, Name: nat.ClaimRef.Name}
	if err := r.Get(ctx, natGatewayKey, natGateway); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, nil
	}
	if natGateway.UID != nat.ClaimRef.UID {
		return nil, nil
	}

	natTable := &v1alpha1.NATTable{}
	if err := r.Get(ctx, natGatewayKey, natTable); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, nil
	}

	for _, natIP := range natTable.IPs {
		for _, target := range natIP.Sections {
			// TODO: Do matching based on IP in the future.
			ref := target.TargetRef
			if ref == nil || ref.UID != nic.UID {
				continue
			}

			return &metalnetv1alpha1.NATDetails{
				IP:      &metalnetv1alpha1.IP{Addr: natIP.IP.Addr},
				Port:    target.Port,
				EndPort: target.EndPort,
			}, nil
		}
	}
	return nil, nil
}

func (r *NetworkInterfaceReconciler) updateStatus(
	ctx context.Context,
	nic *v1alpha1.NetworkInterface,
	metalnetNic *metalnetv1alpha1.NetworkInterface,
) error {
	base := nic.DeepCopy()
	nic.Status.State = metalnetNetworkInterfaceStateToNetworkInterfaceStatus(metalnetNic.Status.State)
	if pciAddr := metalnetNic.Status.PCIAddress; pciAddr != nil {
		nic.Status.PCIAddress = &v1alpha1.PCIAddress{
			Domain:   pciAddr.Domain,
			Bus:      pciAddr.Bus,
			Slot:     pciAddr.Slot,
			Function: pciAddr.Function,
		}
	} else {
		nic.Status.PCIAddress = nil
	}
	nic.Status.PublicIPs = metalnetIPsToIPs(workaroundMetalnetNoIPv6IPToIPs(metalnetNic.Status.VirtualIP))
	nic.Status.NATIPs = metalnetIPsToIPs(workaroundMetalnetNoIPv6NATIPToIPs(metalnetNic.Status.NatIP))
	nic.Status.Prefixes = metalnetIPPrefixesToIPPrefixes(metalnetNic.Spec.Prefixes)
	if err := r.Status().Patch(ctx, nic, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching network interface status: %w", err)
	}
	return nil
}

func (r *NetworkInterfaceReconciler) deleteMatchingMetalnetNetworkInterface(ctx context.Context, nic *v1alpha1.NetworkInterface) (existed bool, err error) {
	metalnetNic := &metalnetv1alpha1.NetworkInterface{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.MetalnetNamespace,
			Name:      string(nic.UID),
		},
	}
	if err := r.MetalnetClient.Delete(ctx, metalnetNic); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("error deleting metalnet network interface %s: %w", metalnetNic.Name, err)
		}

		return false, nil
	}
	return true, nil
}

func (r *NetworkInterfaceReconciler) reconcile(ctx context.Context, log logr.Logger, nic *v1alpha1.NetworkInterface) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	metalnetNode, err := GetMetalnetNode(ctx, r.PartitionName, r.MetalnetClient, nic.Spec.NodeRef.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if metalnetNode == nil || !metalnetNode.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(nic, PartitionFinalizer(r.PartitionName)) {
			log.V(1).Info("Finalizer not present and metalnet node not found / deleting, nothing to do")
			return ctrl.Result{}, nil
		}

		log.V(1).Info("Finalizer present but metalnet node not found / deleting, cleaning up")
		existed, err := r.deleteMatchingMetalnetNetworkInterface(ctx, nic)
		if err != nil {
			return ctrl.Result{}, err
		}
		if existed {
			log.V(1).Info("Issued metalnet network interface deletion, requeueing")
			return ctrl.Result{Requeue: true}, nil
		}

		log.V(1).Info("Metalnet network interface is gone, removing partition finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, nic, PartitionFinalizer(r.PartitionName)); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Removed partition finalizer")
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Metalnet node present and not deleting, ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, nic, PartitionFinalizer(r.PartitionName))
	if err != nil {
		return ctrl.Result{}, err
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Finalizer is present")

	log.V(1).Info("Managing metalnet network interface")
	metalnetNic, ready, err := r.applyMetalnetNic(ctx, log, nic, metalnetNode.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error managing metalnet network interface: %w", err)
	}
	if !ready {
		log.V(1).Info("Metalnet network interface is not yet ready")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Updating status")
	if err := r.updateStatus(ctx, nic, metalnetNic); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating status: %w", err)
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkInterfaceReconciler) applyMetalnetNic(ctx context.Context, log logr.Logger, nic *v1alpha1.NetworkInterface, metalnetNodeName string) (*metalnetv1alpha1.NetworkInterface, bool, error) {
	log.V(1).Info("Getting network vni")
	metalnetNetworkName, err := r.getMetalnetNetworkNameForNetworkInterface(ctx, nic)
	if err != nil {
		return nil, false, err
	}
	if metalnetNetworkName == "" {
		log.V(1).Info("Network is not yet ready")
		return nil, false, nil
	}

	publicIPs := v1alpha1.GetNetworkInterfacePublicIPs(nic)

	log.V(1).Info("Getting load balancer targets")
	targets, err := r.getLoadBalancerTargetsForNetworkInterface(ctx, nic)
	if err != nil {
		return nil, false, fmt.Errorf("error getting load balancer targets: %w", err)
	}

	log.V(1).Info("Getting network policy rules")
	npRules, err := r.getNetworkPolicyRulesForNetworkInterface(ctx, nic)
	if err != nil {
		return nil, false, fmt.Errorf("error getting network policy rules: %w", err)
	}

	log.V(1).Info("Getting NAT IPs")
	natIPs, err := r.getNATDetailsForNetworkInterface(ctx, nic)
	if err != nil {
		return nil, false, fmt.Errorf("error getting NAT IPs: %w", err)
	}

	metalnetNic := &metalnetv1alpha1.NetworkInterface{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metalnetv1alpha1.GroupVersion.String(),
			Kind:       "NetworkInterface",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.MetalnetNamespace,
			Name:      string(nic.UID),
			Labels:    metalnetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), nic),
		},
		Spec: metalnetv1alpha1.NetworkInterfaceSpec{
			NetworkRef:          corev1.LocalObjectReference{Name: metalnetNetworkName},
			IPFamilies:          ipsIPFamilies(nic.Spec.IPs),
			IPs:                 ipsToMetalnetIPs(nic.Spec.IPs),
			VirtualIP:           workaroundMetalnetNoIPv6VirtualIPSupportIPsToIP(ipsToMetalnetIPs(publicIPs)),
			Prefixes:            ipPrefixesToMetalnetPrefixes(nic.Spec.Prefixes),
			LoadBalancerTargets: ipsToMetalnetIPPrefixes(targets),
			NAT:                 workaroundMetalnetNoIPv6NATDetailsToNATDetailsPointer(natIPs),
			NodeName:            &metalnetNodeName,
			FirewallRules:       npRules,
		},
	}
	log.V(1).Info("Applying metalnet network interface")
	if err := r.MetalnetClient.Patch(ctx, metalnetNic, client.Apply, MetalnetFieldOwner, client.ForceOwnership); err != nil {
		return nil, false, fmt.Errorf("error applying metalnet network interface: %w", err)
	}
	return metalnetNic, true, nil
}

func (r *NetworkInterfaceReconciler) isPartitionNetworkInterface() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		nic := obj.(*v1alpha1.NetworkInterface)
		_, err := ParseNodeName(r.PartitionName, nic.Spec.NodeRef.Name)
		return err == nil
	})
}

func (r *NetworkInterfaceReconciler) enqueueByNATTable() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		natTable := obj.(*v1alpha1.NATTable)

		var reqs []ctrl.Request
		for _, natIP := range natTable.IPs {
			for _, s := range natIP.Sections {
				ref := s.TargetRef
				if ref == nil {
					continue
				}
				if _, err := ParseNodeName(r.PartitionName, ref.NodeRef.Name); err != nil {
					continue
				}

				reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKey{
					Namespace: natTable.Namespace,
					Name:      ref.Name,
				}})
			}
		}
		return reqs
	})
}

func (r *NetworkInterfaceReconciler) reconcileRequestsByLoadBalancerRouting(
	ctx context.Context,
	log logr.Logger,
	loadBalancerRouting *v1alpha1.LoadBalancerRouting,
) []ctrl.Request {
	nicList := &v1alpha1.NetworkInterfaceList{}
	if err := r.List(ctx, nicList,
		client.InNamespace(loadBalancerRouting.Namespace),
	); err != nil {
		log.Error(err, "Error listing network interfaces")
		return nil
	}

	metalnetNodeList := &corev1.NodeList{}
	if err := r.MetalnetClient.List(ctx, metalnetNodeList); err != nil {
		log.Error(err, "Error listing metalnet nodes")
		return nil
	}

	dstIPs := utilslices.ToSetFunc(loadBalancerRouting.Destinations,
		func(dst v1alpha1.LoadBalancerDestination) net.IP { return dst.IP },
	)

	var reqs []ctrl.Request
	for _, nic := range nicList.Items {
		if _, err := ParseNodeName(r.PartitionName, nic.Spec.NodeRef.Name); err != nil {
			continue
		}

		if dstIPs.HasAny(nic.Spec.IPs...) {
			reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&nic)})
		}
	}
	return reqs
}

func (r *NetworkInterfaceReconciler) enqueueByLoadBalancerRouting() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		loadBalancerRouting := obj.(*v1alpha1.LoadBalancerRouting)
		log := ctrl.LoggerFrom(ctx)

		return r.reconcileRequestsByLoadBalancerRouting(ctx, log, loadBalancerRouting)
	})
}

func (r *NetworkInterfaceReconciler) enqueueByLoadBalancer() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		loadBalancer := obj.(*v1alpha1.LoadBalancer)
		log := ctrl.LoggerFrom(ctx)

		loadBalancerKey := client.ObjectKeyFromObject(loadBalancer)
		loadBalancerRouting := &v1alpha1.LoadBalancerRouting{}
		if err := r.Get(ctx, loadBalancerKey, loadBalancerRouting); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "Error getting load balancer routing")
			}
			return nil
		}

		return r.reconcileRequestsByLoadBalancerRouting(ctx, log, loadBalancerRouting)
	})
}

func (r *NetworkInterfaceReconciler) reconcileRequestsByNetworkPolicyRule(
	ctx context.Context,
	log logr.Logger,
	networkPolicyRule *v1alpha1.NetworkPolicyRule,
) []ctrl.Request {
	nicList := &v1alpha1.NetworkInterfaceList{}
	if err := r.List(ctx, nicList,
		client.InNamespace(networkPolicyRule.Namespace),
	); err != nil {
		log.Error(err, "Error listing network interfaces")
		return nil
	}

	metalnetNodeList := &corev1.NodeList{}
	if err := r.MetalnetClient.List(ctx, metalnetNodeList); err != nil {
		log.Error(err, "Error listing metalnet nodes")
		return nil
	}

	targetIPs := utilslices.ToSetFunc(networkPolicyRule.Targets,
		func(target v1alpha1.TargetNetworkInterface) net.IP { return target.IP },
	)

	var reqs []ctrl.Request
	for _, nic := range nicList.Items {
		if _, err := ParseNodeName(r.PartitionName, nic.Spec.NodeRef.Name); err != nil {
			continue
		}

		if targetIPs.HasAny(nic.Spec.IPs...) {
			reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&nic)})
		}
	}
	return reqs
}

func (r *NetworkInterfaceReconciler) enqueueByNetworkPolicyRule() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		networkPolicyRule := obj.(*v1alpha1.NetworkPolicyRule)
		log := ctrl.LoggerFrom(ctx)

		return r.reconcileRequestsByNetworkPolicyRule(ctx, log, networkPolicyRule)
	})
}

func (r *NetworkInterfaceReconciler) enqueueByNetworkPolicy() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		networkPolicy := obj.(*v1alpha1.NetworkPolicy)
		log := ctrl.LoggerFrom(ctx)

		networkPolicyKey := client.ObjectKeyFromObject(networkPolicy)
		networkPolicyRule := &v1alpha1.NetworkPolicyRule{}
		if err := r.Get(ctx, networkPolicyKey, networkPolicyRule); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "Error getting network policy rule")
			}
			return nil
		}

		return r.reconcileRequestsByNetworkPolicyRule(ctx, log, networkPolicyRule)
	})
}

func (r *NetworkInterfaceReconciler) enqueueByMetalnetNode() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		metalnetNode := obj.(*corev1.Node)
		log := ctrl.LoggerFrom(ctx)

		nicList := &v1alpha1.NetworkInterfaceList{}
		if err := r.List(ctx, nicList); err != nil {
			log.Error(err, "Error listing network interfaces")
			return nil
		}

		var (
			nodeName = PartitionNodeName(r.PartitionName, metalnetNode.Name)
			reqs     []ctrl.Request
		)
		for _, nic := range nicList.Items {
			if nic.Spec.NodeRef.Name != nodeName {
				continue
			}

			reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&nic)})
		}
		return reqs
	})
}

func (r *NetworkInterfaceReconciler) SetupWithManager(mgr ctrl.Manager, metalnetCache cache.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1alpha1.NetworkInterface{},
			builder.WithPredicates(r.isPartitionNetworkInterface()),
		).
		Watches(
			&v1alpha1.NATTable{},
			r.enqueueByNATTable(),
		).
		Watches(
			&v1alpha1.LoadBalancer{},
			r.enqueueByLoadBalancer(),
		).
		Watches(
			&v1alpha1.LoadBalancerRouting{},
			r.enqueueByLoadBalancerRouting(),
		).
		Watches(
			&v1alpha1.NetworkPolicy{},
			r.enqueueByNetworkPolicy(),
		).
		Watches(
			&v1alpha1.NetworkPolicyRule{},
			r.enqueueByNetworkPolicyRule(),
		).
		WatchesRawSource(
			source.Kind(metalnetCache, &metalnetv1alpha1.NetworkInterface{}),
			utilhandler.EnqueueRequestForSource(r.Scheme(), r.RESTMapper(), &v1alpha1.NetworkInterface{}),
		).
		WatchesRawSource(
			source.Kind(metalnetCache, &corev1.Node{}),
			r.enqueueByMetalnetNode(),
		).
		Complete(r)
}
