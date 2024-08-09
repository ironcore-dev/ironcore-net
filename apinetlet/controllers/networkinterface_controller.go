// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	apinetlethandler "github.com/ironcore-dev/ironcore-net/apinetlet/handler"
	"github.com/ironcore-dev/ironcore-net/apinetlet/provider"
	utilgeneric "github.com/ironcore-dev/ironcore-net/utils/generic"

	"github.com/ironcore-dev/controller-utils/clientutils"
	"github.com/ironcore-dev/controller-utils/metautils"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/claimmanager"
	"github.com/ironcore-dev/ironcore/utils/generic"
	"github.com/ironcore-dev/ironcore/utils/predicates"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	networkInterfaceFinalizer = "apinet.ironcore.dev/networkinterface"
)

type NetworkInterfaceReconciler struct {
	client.Client
	APINetClient    client.Client
	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=networkinterfaces,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=networkinterfaces/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=networkinterfaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=virtualips,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=ipam.ironcore.dev,resources=prefixes,verbs=get;list;watch

//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkinterfaces,verbs=get;list;watch;update;patch

func (r *NetworkInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	nic := &networkingv1alpha1.NetworkInterface{}
	if err := r.Get(ctx, req.NamespacedName, nic); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		if err := r.releaseNetworkInterfaceKeyAPINetInterfaces(ctx, req.NamespacedName); err != nil {
			return ctrl.Result{}, fmt.Errorf("error releasing APINet network interfaces by key: %w", err)
		}
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, nic)
}

func (r *NetworkInterfaceReconciler) reconcileExists(ctx context.Context, log logr.Logger, nic *networkingv1alpha1.NetworkInterface) (ctrl.Result, error) {
	if !nic.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, nic)
	}
	return r.reconcile(ctx, log, nic)
}

func (r *NetworkInterfaceReconciler) releaseNetworkInterfaceKeyAPINetInterfaces(ctx context.Context, nicKey client.ObjectKey) error {
	apiNetNicList := &apinetv1alpha1.NetworkInterfaceList{}
	if err := r.APINetClient.List(ctx, apiNetNicList,
		client.InNamespace(r.APINetNamespace),
		apinetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), nicKey, &networkingv1alpha1.NetworkInterface{}),
	); err != nil {
		return fmt.Errorf("error listing APINet network interfaces: %w", err)
	}

	var errs []error
	for _, apiNetNic := range apiNetNicList.Items {
		if err := r.releaseAPINetNetworkInterface(ctx, &apiNetNic); client.IgnoreNotFound(err) != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *NetworkInterfaceReconciler) releaseAPINetNetworkInterface(ctx context.Context, apiNetNic *apinetv1alpha1.NetworkInterface) error {
	keys := apinetletclient.SourceLabelKeys(r.Scheme(), r.RESTMapper(), &networkingv1alpha1.NetworkInterface{})
	base := apiNetNic.DeepCopy()
	metautils.DeleteLabels(apiNetNic, keys)
	return r.APINetClient.Patch(ctx, apiNetNic, client.StrategicMergeFrom(base))
}

func (r *NetworkInterfaceReconciler) releaseNetworkInterfaceAPINetNetworkInterfaces(ctx context.Context, nic *networkingv1alpha1.NetworkInterface) error {
	apiNetNicList := &apinetv1alpha1.NetworkInterfaceList{}
	if err := r.APINetClient.List(ctx, apiNetNicList,
		client.InNamespace(r.APINetNamespace),
		apinetletclient.MatchingSourceLabels(r.Scheme(), r.RESTMapper(), nic),
	); err != nil {
		return fmt.Errorf("error listing APINet network interfaces: %w", err)
	}

	// create a shallow copy of the network interface with the
	// deletion timestamp removed in order to properly release the api net network interfaces.
	nonDeletingNic := *nic
	nonDeletingNic.DeletionTimestamp = nil

	var (
		strat    = &apiNetNetworkInterfaceClaimStrategy{r.APINetClient}
		claimMgr = claimmanager.New(&nonDeletingNic, claimmanager.NothingSelector(), strat)
		errs     []error
	)
	for _, apiNetNic := range apiNetNicList.Items {
		_, err := claimMgr.Claim(ctx, &apiNetNic)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *NetworkInterfaceReconciler) releaseVirtualIPsForNetworkInterface(ctx context.Context, nic *networkingv1alpha1.NetworkInterface) error {
	vipList := &networkingv1alpha1.VirtualIPList{}
	if err := r.List(ctx, vipList,
		client.InNamespace(nic.Namespace),
	); err != nil {
		return fmt.Errorf("error listing virtual IPs: %w", err)
	}

	// create a shallow copy of the network interface with the
	// deletion timestamp removed in order to properly release the virtual IP
	nonDeletingNic := *nic
	nonDeletingNic.DeletionTimestamp = nil

	var (
		strat    = &virtualIPClaimStrategy{r.Client}
		claimMgr = claimmanager.New(&nonDeletingNic, claimmanager.NothingSelector(), strat)
		errs     []error
	)

	for _, vip := range vipList.Items {
		_, err := claimMgr.Claim(ctx, &vip)
		if err != nil {
			errs = append(errs, err)
			continue
		}

	}
	return errors.Join(errs...)
}

func (r *NetworkInterfaceReconciler) delete(ctx context.Context, log logr.Logger, nic *networkingv1alpha1.NetworkInterface) (ctrl.Result, error) {
	log.V(1).Info("Delete")
	if !controllerutil.ContainsFinalizer(nic, networkInterfaceFinalizer) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	if err := r.releaseNetworkInterfaceAPINetNetworkInterfaces(ctx, nic); err != nil {
		return ctrl.Result{}, fmt.Errorf("error releasing APINet network interfaces: %w", err)
	}
	log.V(1).Info("Released APINet network interfaces")

	if err := r.releaseVirtualIPsForNetworkInterface(ctx, nic); err != nil {
		return ctrl.Result{}, fmt.Errorf("error releasing virtual IPs: %w", err)
	}
	log.V(1).Info("Released virtual IPs")

	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, nic, networkInterfaceFinalizer); err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info("Removed finalizer")

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *NetworkInterfaceReconciler) networkInterfaceVirtualIPSelector(nic *networkingv1alpha1.NetworkInterface) claimmanager.Selector {
	vipNames := sets.New[string]()
	if vipSrc := nic.Spec.VirtualIP; vipSrc != nil {
		name := networkingv1alpha1.NetworkInterfaceVirtualIPName(nic.Name, *vipSrc)
		vipNames.Insert(name)
	}
	return claimmanager.SelectorFunc(func(obj client.Object) bool {
		vip := obj.(*networkingv1alpha1.VirtualIP)
		return vipNames.Has(vip.Name)
	})
}

type apiNetNetworkInterfaceClaimStrategy struct {
	client.Client
}

func (s *apiNetNetworkInterfaceClaimStrategy) ClaimState(claimer client.Object, obj client.Object) claimmanager.ClaimState {
	apiNetNic := obj.(*apinetv1alpha1.NetworkInterface)
	if data := apinetletclient.SourceObjectDataFromObject(s.Scheme(), s.RESTMapper(), claimer, apiNetNic); data != nil {
		if data.UID == claimer.GetUID() {
			return claimmanager.ClaimStateClaimed
		}
		return claimmanager.ClaimStateTaken
	}
	return claimmanager.ClaimStateFree
}

func (s *apiNetNetworkInterfaceClaimStrategy) Adopt(ctx context.Context, claimer client.Object, obj client.Object) error {
	apiNetNic := obj.(*apinetv1alpha1.NetworkInterface)
	base := apiNetNic.DeepCopy()
	combinedLabels := make(map[string]string)
	if claimerLabels := claimer.GetLabels(); claimerLabels != nil {
		for key, value := range claimerLabels {
			combinedLabels[key] = value
		}
	}
	if sourceLabels := apinetletclient.SourceLabels(s.Scheme(), s.RESTMapper(), claimer); sourceLabels != nil {
		for key, value := range sourceLabels {
			combinedLabels[key] = value
		}
	}
	apiNetNic.SetLabels(combinedLabels)
	return s.Patch(ctx, apiNetNic, client.StrategicMergeFrom(base))
}

func (s *apiNetNetworkInterfaceClaimStrategy) Release(ctx context.Context, claimer client.Object, obj client.Object) error {
	apiNetNic := obj.(*apinetv1alpha1.NetworkInterface)
	base := apiNetNic.DeepCopy()
	keys := apinetletclient.SourceLabelKeys(s.Scheme(), s.RESTMapper(), claimer)
	metautils.DeleteLabels(apiNetNic, keys)
	apiNetNic.Spec.PublicIPs = nil
	apiNetNic.Spec.Prefixes = nil
	return s.Patch(ctx, apiNetNic, client.StrategicMergeFrom(base))
}

type virtualIPClaimStrategy struct {
	client.Client
}

func (s *virtualIPClaimStrategy) ClaimState(claimer client.Object, obj client.Object) claimmanager.ClaimState {
	vip := obj.(*networkingv1alpha1.VirtualIP)
	if targetRef := vip.Spec.TargetRef; targetRef != nil {
		if targetRef.UID == claimer.GetUID() {
			return claimmanager.ClaimStateClaimed
		}
		return claimmanager.ClaimStateTaken
	}
	return claimmanager.ClaimStateFree
}

func (s *virtualIPClaimStrategy) Adopt(ctx context.Context, claimer client.Object, obj client.Object) error {
	vip := obj.(*networkingv1alpha1.VirtualIP)
	base := vip.DeepCopy()
	vip.Spec.TargetRef = &commonv1alpha1.LocalUIDReference{
		Name: claimer.GetName(),
		UID:  claimer.GetUID(),
	}
	return s.Patch(ctx, vip, client.StrategicMergeFrom(base))
}

func (s *virtualIPClaimStrategy) Release(ctx context.Context, claimer client.Object, obj client.Object) error {
	vip := obj.(*networkingv1alpha1.VirtualIP)
	base := vip.DeepCopy()
	vip.Spec.TargetRef = nil
	return s.Patch(ctx, vip, client.StrategicMergeFrom(base))
}

func (r *NetworkInterfaceReconciler) getVirtualIPsForNetworkInterface(ctx context.Context, nic *networkingv1alpha1.NetworkInterface) ([]networkingv1alpha1.VirtualIP, error) {
	vipList := &networkingv1alpha1.VirtualIPList{}
	if err := r.List(ctx, vipList,
		client.InNamespace(nic.Namespace),
	); err != nil {
		return nil, fmt.Errorf("error listing virtual IPs: %w", err)
	}

	var (
		sel      = r.networkInterfaceVirtualIPSelector(nic)
		strategy = &virtualIPClaimStrategy{r.Client}
		claimMgr = claimmanager.New(nic, sel, strategy)
		vips     []networkingv1alpha1.VirtualIP
		errs     []error
	)
	for _, vip := range vipList.Items {
		ok, err := claimMgr.Claim(ctx, &vip)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !ok {
			continue
		}

		vips = append(vips, vip)
	}
	return vips, errors.Join(errs...)
}

func (r *NetworkInterfaceReconciler) networkInterfaceAPINetNetworkInterfaceSelector(log logr.Logger, nic *networkingv1alpha1.NetworkInterface) claimmanager.Selector {
	namespace, name, node, uid, err := provider.ParseNetworkInterfaceID(nic.Spec.ProviderID)
	if err != nil {
		log.V(1).Info("Error parsing network interface id", "error", err)
		return claimmanager.NothingSelector()
	}

	log.V(1).Info("Parsed network interface ID",
		"apiNetNetworkInterfaceNamespace", namespace,
		"apiNetNetworkInterfaceName", name,
		"apiNetNetworkInterfaceNode", node,
		"apiNetNetworkInterfaceUID", uid,
	)
	return claimmanager.SelectorFunc(func(obj client.Object) bool {
		apiNetNic := obj.(*apinetv1alpha1.NetworkInterface)
		return apiNetNic.UID == uid
	})
}

func (r *NetworkInterfaceReconciler) getAPINetNetworkInterfaceForNetworkInterface(ctx context.Context, log logr.Logger, nic *networkingv1alpha1.NetworkInterface) (*apinetv1alpha1.NetworkInterface, error) {
	apiNetNicList := &apinetv1alpha1.NetworkInterfaceList{}
	if err := r.APINetClient.List(ctx, apiNetNicList,
		client.InNamespace(r.APINetNamespace),
	); err != nil {
		return nil, fmt.Errorf("error listing APINet network interfaces: %w", err)
	}

	var (
		sel            = r.networkInterfaceAPINetNetworkInterfaceSelector(log, nic)
		claimMgr       = claimmanager.New(nic, sel, &apiNetNetworkInterfaceClaimStrategy{r.APINetClient})
		foundAPINetNic *apinetv1alpha1.NetworkInterface
		errs           []error
	)
	for _, apiNetNic := range apiNetNicList.Items {
		ok, err := claimMgr.Claim(ctx, &apiNetNic)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !ok {
			continue
		}

		foundAPINetNic = generic.Pointer(apiNetNic)
	}
	return foundAPINetNic, errors.Join(errs...)
}

func (r *NetworkInterfaceReconciler) getPrefixesForNetworkInterface(ctx context.Context, nic *networkingv1alpha1.NetworkInterface) ([]net.IPPrefix, error) {
	var res []net.IPPrefix
	for idx, prefixSrc := range nic.Spec.Prefixes {
		switch {
		case prefixSrc.Value != nil:
			res = append(res, net.IPPrefix{Prefix: prefixSrc.Value.Prefix})
		case prefixSrc.Ephemeral != nil:
			ipamPrefix := &ipamv1alpha1.Prefix{}
			ipamPrefixKey := client.ObjectKey{Namespace: nic.Namespace, Name: networkingv1alpha1.NetworkInterfacePrefixIPAMPrefixName(nic.Name, idx)}
			if err := r.Get(ctx, ipamPrefixKey, ipamPrefix); err != nil {
				if !apierrors.IsNotFound(err) {
					return nil, err
				}
				continue
			}

			if ipamPrefix.Status.Phase != ipamv1alpha1.PrefixPhaseAllocated {
				continue
			}

			res = append(res, net.IPPrefix{Prefix: ipamPrefix.Spec.Prefix.Prefix})
		}
	}
	return res, nil
}

func (r *NetworkInterfaceReconciler) manageAPINetNetworkInterface(ctx context.Context, nic *networkingv1alpha1.NetworkInterface, apiNetNic *apinetv1alpha1.NetworkInterface, vips []networkingv1alpha1.VirtualIP, prefixes []net.IPPrefix) error {
	_ = nic

	var (
		publicIPs        []apinetv1alpha1.NetworkInterfacePublicIP
		publicIPFamilies = sets.New[corev1.IPFamily]()
	)
	for _, vip := range vips {
		if ip := vip.Status.IP; ip.IsValid() {
			publicIPFamilies.Insert(ip.Family())
			publicIPs = append(publicIPs, apinetv1alpha1.NetworkInterfacePublicIP{
				Name:     ip.String(),
				IPFamily: ip.Family(),
				IP:       net.IP{Addr: ip.Addr},
			})
		}
	}
	filteredNATs := utilslices.FilterFunc(apiNetNic.Spec.NATs,
		func(nat apinetv1alpha1.NetworkInterfaceNAT) bool {
			return !publicIPFamilies.Has(nat.IPFamily)
		},
	)

	if slices.Equal(apiNetNic.Spec.PublicIPs, publicIPs) &&
		slices.Equal(apiNetNic.Spec.Prefixes, prefixes) &&
		slices.Equal(apiNetNic.Spec.NATs, filteredNATs) {
		return nil
	}

	base := apiNetNic.DeepCopy()
	apiNetNic.Spec.PublicIPs = publicIPs
	apiNetNic.Spec.NATs = filteredNATs
	apiNetNic.Spec.Prefixes = prefixes
	return r.APINetClient.Patch(ctx, apiNetNic, client.StrategicMergeFrom(base))
}

func (r *NetworkInterfaceReconciler) setNetworkInterfacePending(ctx context.Context, nic *networkingv1alpha1.NetworkInterface) error {
	now := metav1.Now()

	base := nic.DeepCopy()
	nic.Status.VirtualIP = nil
	nic.Status.IPs = nil
	nic.Status.Prefixes = nil
	if nic.Status.State != networkingv1alpha1.NetworkInterfaceStatePending {
		nic.Status.LastStateTransitionTime = &now
	}
	nic.Status.State = networkingv1alpha1.NetworkInterfaceStatePending

	return r.Status().Patch(ctx, nic, client.StrategicMergeFrom(base))
}

func (r *NetworkInterfaceReconciler) reconcile(ctx context.Context, log logr.Logger, nic *networkingv1alpha1.NetworkInterface) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, nic, networkInterfaceFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Finalizer is present")

	var errs []error

	apiNetNic, err := r.getAPINetNetworkInterfaceForNetworkInterface(ctx, log, nic)
	if err != nil {
		errs = append(errs, err)
	}

	vips, err := r.getVirtualIPsForNetworkInterface(ctx, nic)
	if err != nil {
		errs = append(errs, err)
	}

	prefixes, err := r.getPrefixesForNetworkInterface(ctx, nic)
	if err != nil {
		errs = append(errs, err)
	}

	if err := errors.Join(errs...); err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting APINet network interface for network interface: %w", err)
	}

	if apiNetNic == nil {
		if err := r.setNetworkInterfacePending(ctx, nic); err != nil {
			return ctrl.Result{}, fmt.Errorf("error setting network interface to pending: %w", err)
		}
		log.V(1).Info("Set network interface to pending")
		return ctrl.Result{}, nil
	}

	if err := r.manageAPINetNetworkInterface(ctx, nic, apiNetNic, vips, prefixes); err != nil {
		return ctrl.Result{}, fmt.Errorf("error managing APINet network interface: %w", err)
	}

	var (
		expectedState     = apiNetNetworkInterfaceStateToNetworkInterfaceState(apiNetNic.Status.State)
		expectedIPs       = apiNetIPsToIPs(apiNetNic.Spec.IPs)
		expectedPrefixes  = apiNetIPPrefixesToIPPrefixes(apiNetNic.Spec.Prefixes)
		expectedVirtualIP = WorkaroundOnlyV4VirtualIPs(apiNetIPsToIPs(apiNetNic.Status.PublicIPs))
	)
	if !NetworkInterfaceStatusUpToDate(nic, expectedState, expectedIPs, expectedPrefixes, expectedVirtualIP) {
		if err := r.updateNetworkInterfaceStatus(ctx, nic, expectedState, expectedIPs, expectedPrefixes, expectedVirtualIP); err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating network interface status: %w", err)
		}
		log.V(1).Info("Updated network interface status")
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkInterfaceReconciler) updateNetworkInterfaceStatus(ctx context.Context, nic *networkingv1alpha1.NetworkInterface, state networkingv1alpha1.NetworkInterfaceState, ips []commonv1alpha1.IP, prefixes []commonv1alpha1.IPPrefix, virtualIP *commonv1alpha1.IP) error {
	now := metav1.Now()
	base := nic.DeepCopy()

	if nic.Status.State != state {
		nic.Status.LastStateTransitionTime = &now
	}
	nic.Status.State = state
	nic.Status.IPs = ips
	nic.Status.Prefixes = prefixes
	nic.Status.VirtualIP = virtualIP

	if err := r.Status().Patch(ctx, nic, client.StrategicMergeFrom(base)); err != nil {
		return fmt.Errorf("error patching status: %w", err)
	}
	return nil
}

func NetworkInterfaceStatusUpToDate(nic *networkingv1alpha1.NetworkInterface, expectedState networkingv1alpha1.NetworkInterfaceState, expectedIPs []commonv1alpha1.IP, expectedIPPrefixes []commonv1alpha1.IPPrefix, expectedVirtualIP *commonv1alpha1.IP) bool {
	return nic.Status.State == expectedState &&
		slices.Equal(nic.Status.IPs, expectedIPs) &&
		slices.Equal(nic.Status.Prefixes, expectedIPPrefixes) &&
		utilgeneric.EqualPointers(nic.Status.VirtualIP, expectedVirtualIP)
}

func WorkaroundOnlyV4VirtualIPs(ips []commonv1alpha1.IP) *commonv1alpha1.IP {
	for i := range ips {
		ip := &ips[i]
		if ip.Family() == corev1.IPv4Protocol {
			return ip
		}
	}
	return nil
}

func (r *NetworkInterfaceReconciler) enqueueByVirtualIP() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		vip := obj.(*networkingv1alpha1.VirtualIP)
		targetRef := vip.Spec.TargetRef
		if targetRef == nil {
			return nil
		}
		return []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: vip.Namespace, Name: targetRef.Name}}}
	})
}

func (r *NetworkInterfaceReconciler) enqueueByNetworkInterfaceVirtualIPSelection() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		vip := obj.(*networkingv1alpha1.VirtualIP)
		log := ctrl.LoggerFrom(ctx)

		nicList := &networkingv1alpha1.NetworkInterfaceList{}
		if err := r.List(ctx, nicList,
			client.InNamespace(vip.Namespace),
		); err != nil {
			log.Error(err, "Error listing network interfaces")
			return nil
		}

		var reqs []ctrl.Request
		for _, nic := range nicList.Items {
			sel := r.networkInterfaceVirtualIPSelector(&nic)
			if sel.Match(vip) {
				reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&nic)})
			}
		}

		return reqs
	})
}

func (r *NetworkInterfaceReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCache cache.Cache) error {
	log := ctrl.Log.WithName("networkinterface").WithName("setup")
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1alpha1.NetworkInterface{},
			builder.WithPredicates(
				predicates.ResourceHasFilterLabel(log, r.WatchFilterValue),
				predicates.ResourceIsNotExternallyManaged(log),
			),
		).
		WatchesRawSource(
			source.Kind(apiNetCache, &apinetv1alpha1.NetworkInterface{}),
			apinetlethandler.EnqueueRequestForSource(r.Scheme(), r.RESTMapper(), &networkingv1alpha1.NetworkInterface{}),
		).
		Owns(&ipamv1alpha1.Prefix{}).
		Watches(
			&networkingv1alpha1.VirtualIP{},
			r.enqueueByVirtualIP(),
			builder.WithPredicates(virtualIPClaimedPredicate()),
		).
		Watches(
			&networkingv1alpha1.VirtualIP{},
			r.enqueueByNetworkInterfaceVirtualIPSelection(),
			builder.WithPredicates(virtualIPFreePredicate()),
		).
		Complete(r)
}
