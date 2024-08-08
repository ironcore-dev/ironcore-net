// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetclient "github.com/ironcore-dev/ironcore-net/internal/client"
	"github.com/ironcore-dev/ironcore-net/internal/natgateway"
	"github.com/ironcore-dev/ironcore-net/utils/maps"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	"golang.org/x/exp/slices"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type NATGatewayReconciler struct {
	client.Client
	record.EventRecorder
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=natgateways,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=natgateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=nattables,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkinterfaces,verbs=get;list;watch;patch;update

func (r *NATGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	natGateway := &v1alpha1.NATGateway{}
	if err := r.Get(ctx, req.NamespacedName, natGateway); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		natTable := &v1alpha1.NATTable{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			},
		}
		if err := r.Delete(ctx, natTable); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("error deleting NAT table: %w", err)
		}
		return ctrl.Result{}, nil
	}
	return r.reconcileExists(ctx, log, natGateway)
}

func (r *NATGatewayReconciler) reconcileExists(ctx context.Context, log logr.Logger, natGateway *v1alpha1.NATGateway) (ctrl.Result, error) {
	if !natGateway.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, natGateway)
	}
	return r.reconcile(ctx, log, natGateway)
}

func (r *NATGatewayReconciler) delete(ctx context.Context, log logr.Logger, natGateway *v1alpha1.NATGateway) (ctrl.Result, error) {
	log.V(1).Info("Delete")
	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *NATGatewayReconciler) updateNATGatewayUsedRequests(
	ctx context.Context,
	natGateway *v1alpha1.NATGateway,
	usedNATIPs int64,
	requests int64,
) error {
	base := natGateway.DeepCopy()
	natGateway.Status.UsedNATIPs = usedNATIPs
	natGateway.Status.RequestedNATIPs = requests
	if err := r.Status().Patch(ctx, natGateway, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching nat gateway status: %w", err)
	}
	return nil
}

// natIPAllocation bundles a NAT IP with a target.
type natIPAllocation struct {
	// IP is the NATed IP.
	ip net.IP
	// NATIPSection is the target of the ip.
	v1alpha1.NATIPSection
}

func (r *NATGatewayReconciler) getExistingAllocations(ctx context.Context, natGateway *v1alpha1.NATGateway, ips []net.IP) (map[types.UID]natIPAllocation, error) {
	routing := &v1alpha1.NATTable{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(natGateway), routing); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("error getting NAT gateway routing: %w", err)
		}

		return nil, nil
	}

	var (
		mgr       = natgateway.NewAllocationManager(natGateway.Spec.PortsPerNetworkInterface, ips)
		allocByID = make(map[types.UID]natIPAllocation)
	)

	for _, ip := range routing.IPs {
		if !mgr.HasIP(ip.IP) {
			// IP has been removed - short circuit iteration and continue.
			continue
		}

		for _, tgt := range ip.Sections {
			ref := tgt.TargetRef
			if ref == nil {
				// TODO: When IPs are finally unique, we don't need the TargetRef anymore.
				continue
			}
			if mgr.Use(ip.IP, tgt.Port, tgt.EndPort) {
				allocByID[ref.UID] = natIPAllocation{ip.IP, tgt}
			}
		}
	}
	return allocByID, nil
}

func (r *NATGatewayReconciler) natGatewayNetworkInterfaceSelector(natGateway *v1alpha1.NATGateway) func(*v1alpha1.NetworkInterface) bool {
	return func(nic *v1alpha1.NetworkInterface) bool {
		var found bool
		for _, ip := range nic.Spec.IPs {
			if ip.Family() == natGateway.Spec.IPFamily {
				found = true
				break
			}
		}
		if !found {
			// Network interface does not support the NAT ip family.
			return false
		}

		for _, publicIP := range nic.Spec.PublicIPs {
			if publicIP.IPFamily == natGateway.Spec.IPFamily {
				// Network interface already has a public IP.
				return false
			}
		}

		return true
	}
}

func (r *NATGatewayReconciler) manageNATTable(
	ctx context.Context,
	natGateway *v1alpha1.NATGateway,
	ips []net.IP,
	existingAllocByNicID map[types.UID]natIPAllocation,
) (used, requests int64, err error) {
	nicList := &v1alpha1.NetworkInterfaceList{}
	if err := r.List(ctx, nicList,
		client.InNamespace(natGateway.Namespace),
		client.MatchingFields{apinetclient.NetworkInterfaceSpecNetworkRefNameField: natGateway.Spec.NetworkRef.Name},
	); err != nil {
		return 0, 0, fmt.Errorf("error listing network interfaces: %w", err)
	}

	var (
		mgr            = natgateway.NewAllocationManager(natGateway.Spec.PortsPerNetworkInterface, ips)
		sel            = r.natGatewayNetworkInterfaceSelector(natGateway)
		ipToAllocation = make(map[net.IP]map[types.UID]v1alpha1.NATIPSection)
		addAlloc       = func(ip net.IP, target v1alpha1.NATIPSection) {
			ipToAllocation[ip] = maps.Append(ipToAllocation[ip], target.TargetRef.UID, target)
		}
		getNicIP = func(nic *v1alpha1.NetworkInterface) net.IP {
			for _, ip := range nic.Spec.IPs {
				if ip.Family() == natGateway.Spec.IPFamily {
					return ip
				}
			}
			return net.IP{}
		}
		processClaimed []int
		processFree    []int
		errs           []error
	)
	for i, nic := range nicList.Items {
		claimer := v1alpha1.GetNetworkInterfaceNATClaimer(&nic, natGateway.Spec.IPFamily)
		if claimer != nil {
			if claimer.UID != natGateway.UID {
				// Claimed by someone else, ignore.
				continue
			}

			if sel(&nic) {
				// We claim it and match it.
				requests++
				existing, ok := existingAllocByNicID[nic.UID]
				if ok {
					// Re-use existing allocation.
					mgr.Use(existing.ip, existing.Port, existing.EndPort)
					addAlloc(existing.ip, existing.NATIPSection)
					continue
				}

				// We claim it and match it, however there's no allocation - process to see if we can allocate it.
				processClaimed = append(processClaimed, i)
				continue
			}

			// We don't match it - release it if possible.
			if err := apinetclient.ReleaseNetworkInterfaceNAT(ctx, r.Client, &nic, natGateway.Spec.IPFamily); client.IgnoreNotFound(err) != nil {
				errs = append(errs, err)
			}
			continue
		}

		// It's not being claimed at the moment.

		if !sel(&nic) || !nic.DeletionTimestamp.IsZero() {
			// We don't want to claim it - skip it.
			continue
		}

		// Mark to be processed.
		requests++
		processFree = append(processFree, i)
	}

	var full bool
	for _, i := range processClaimed {
		nic := nicList.Items[i]

		if !full {
			ip, port, endPort, ok := mgr.UseNextFree()
			if ok {
				// Already claimed - just add the allocation and proceed.
				addAlloc(ip, v1alpha1.NATIPSection{
					IP:      getNicIP(&nic),
					Port:    port,
					EndPort: endPort,
					TargetRef: &v1alpha1.NATTableIPTargetRef{
						UID:     nic.UID,
						Name:    nic.Name,
						NodeRef: nic.Spec.NodeRef,
					},
				})
				continue
			}

			// Mark as full
			full = true
		}

		if err := apinetclient.ReleaseNetworkInterfaceNAT(ctx, r.Client, &nic, natGateway.Spec.IPFamily); client.IgnoreNotFound(err) != nil {
			errs = append(errs, err)
			continue
		}
	}

	if !full {
		// Initialize IP and ports here to re-use in case we cannot claim a network interface.
		var (
			ip                net.IP
			port, endPort     int32
			shouldUseNextFree = true
			claimRef          = v1alpha1.NetworkInterfaceNATClaimRef{
				Name: natGateway.Name,
				UID:  natGateway.UID,
			}
		)
		for _, i := range processFree {
			nic := nicList.Items[i]

			if shouldUseNextFree {
				var ok bool
				ip, port, endPort, ok = mgr.UseNextFree()
				if !ok {
					break
				}

				shouldUseNextFree = false
			}

			if err := apinetclient.ClaimNetworkInterfaceNAT(ctx, r.Client, &nic, natGateway.Spec.IPFamily, claimRef); err != nil {
				if !apierrors.IsNotFound(err) {
					// We only care about non-not-found errors - if it doesn't exist, simply don't allocate.
					errs = append(errs, err)
				}
				continue
			}

			addAlloc(ip, v1alpha1.NATIPSection{
				IP:      getNicIP(&nic),
				Port:    port,
				EndPort: endPort,
				TargetRef: &v1alpha1.NATTableIPTargetRef{
					UID:     nic.UID,
					Name:    nic.Name,
					NodeRef: nic.Spec.NodeRef,
				},
			})
			shouldUseNextFree = true // set shouldUseNextFree to true in order to issue using next IP again.
		}
	}

	if err := r.applyNATTable(ctx, natGateway, ipToAllocation); err != nil {
		errs = append(errs, err)
	}

	return mgr.Used(), requests, errors.Join(errs...)
}

func (r *NATGatewayReconciler) applyNATTable(
	ctx context.Context,
	natGateway *v1alpha1.NATGateway,
	natTableData map[net.IP]map[types.UID]v1alpha1.NATIPSection,
) error {
	natTable := &v1alpha1.NATTable{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "NATTable",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: natGateway.Namespace,
			Name:      natGateway.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(natGateway, v1alpha1.SchemeGroupVersion.WithKind("NATGateway")),
			},
		},
	}
	for ip, allocs := range natTableData {
		natIP := v1alpha1.NATIP{IP: ip}
		for _, alloc := range allocs {
			natIP.Sections = append(natIP.Sections, alloc)
		}
		slices.SortFunc(natIP.Sections, func(a, b v1alpha1.NATIPSection) int {
			return int(a.Port - b.Port)
		})
		natTable.IPs = append(natTable.IPs, natIP)
	}
	slices.SortFunc(natTable.IPs, func(a, b v1alpha1.NATIP) int {
		return a.IP.Compare(b.IP.Addr)
	})

	if err := r.Patch(ctx, natTable, client.Apply, fieldOwner, client.ForceOwnership); err != nil {
		return fmt.Errorf("error applying NAT table: %w", err)
	}
	return nil
}

func (r *NATGatewayReconciler) reconcile(ctx context.Context, log logr.Logger, natGateway *v1alpha1.NATGateway) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	ips := v1alpha1.GetNATGatewayIPs(natGateway)

	existingAllocs, err := r.getExistingAllocations(ctx, natGateway, ips)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting existing allocations: %w", err)
	}

	log.V(1).Info("Managing NAT Table")
	used, requests, err := r.manageNATTable(ctx, natGateway, ips, existingAllocs)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error managing NAT IPs: %w", err)
	}

	if used != natGateway.Status.UsedNATIPs || requests != natGateway.Status.RequestedNATIPs {
		log.V(1).Info("Updating NAT Gateway status used NAT IPs", "Used", used, "Requests", requests)
		if err := r.updateNATGatewayUsedRequests(ctx, natGateway, used, requests); err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating NAT gateway used / requested NAT IPs: %w", err)
		}
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NATGatewayReconciler) enqueueByNetworkInterfaceNAT() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		nic := obj.(*v1alpha1.NetworkInterface)

		var reqs []ctrl.Request
		for _, nicNAT := range nic.Spec.NATs {
			reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKey{
				Namespace: nic.Namespace,
				Name:      nicNAT.ClaimRef.Name,
			}})
		}
		return reqs
	})
}

func (r *NATGatewayReconciler) enqueueByNATGatewayNetworkInterfaceSelection() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		nic := obj.(*v1alpha1.NetworkInterface)
		log := ctrl.LoggerFrom(ctx)

		freeNicNATIPFamilies := utilslices.ToSetFunc(nic.Spec.IPs, net.IP.Family)
		for _, publicIP := range nic.Spec.PublicIPs {
			freeNicNATIPFamilies.Delete(publicIP.IPFamily)
			if freeNicNATIPFamilies.Len() == 0 {
				return nil
			}
		}
		for _, nicNAT := range nic.Spec.NATs {
			freeNicNATIPFamilies.Delete(nicNAT.IPFamily)
			if freeNicNATIPFamilies.Len() == 0 {
				return nil
			}
		}

		natGatewayList := &v1alpha1.NATGatewayList{}
		if err := r.List(ctx, natGatewayList,
			client.InNamespace(nic.Namespace),
		); err != nil {
			log.Error(err, "Error listing NAT gateways")
			return nil
		}

		var reqs []ctrl.Request
		for _, natGateway := range natGatewayList.Items {
			if !natGateway.DeletionTimestamp.IsZero() {
				continue
			}

			if freeNicNATIPFamilies.Has(natGateway.Spec.IPFamily) {
				reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&natGateway)})
			}
		}
		return reqs
	})
}

func (r *NATGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.NATGateway{}).
		Owns(&v1alpha1.NATTable{}).
		Watches(
			&v1alpha1.NetworkInterface{},
			r.enqueueByNetworkInterfaceNAT(),
		).
		Watches(
			&v1alpha1.NetworkInterface{},
			r.enqueueByNATGatewayNetworkInterfaceSelection(),
		).
		Complete(r)
}
