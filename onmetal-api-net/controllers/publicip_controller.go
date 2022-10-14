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
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	"go4.org/netipx"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	publicIPFinalizer = "apinet.api.onmetal.de/publicip"
)

type Allocation struct {
	UID types.UID
	IPs []netip.Addr
}

type PublicIPReconciler struct {
	mu sync.RWMutex

	client.Client
	record.EventRecorder
	APIReader           client.Reader
	InitialAvailableIPs *netipx.IPSet

	ipReleased       chan struct{}
	availableIPs     *netipx.IPSet
	allocationsByKey map[client.ObjectKey]Allocation
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=publicips,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=publicips/finalizers,verbs=update
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=publicips/status,verbs=get;update;patch

func (r *PublicIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	publicIP := &onmetalapinetv1alpha1.PublicIP{}
	if err := r.Get(ctx, req.NamespacedName, publicIP); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting public ip %s: %w", req.NamespacedName, err)
		}

		r.release(ctx, req.NamespacedName)
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, publicIP)
}

func (r *PublicIPReconciler) reconcileExists(ctx context.Context, log logr.Logger, publicIP *onmetalapinetv1alpha1.PublicIP) (ctrl.Result, error) {
	if !publicIP.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, publicIP)
	}
	return r.reconcile(ctx, log, publicIP)
}

func (r *PublicIPReconciler) delete(ctx context.Context, log logr.Logger, publicIP *onmetalapinetv1alpha1.PublicIP) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	log.V(1).Info("Releasing any associated ips")
	r.release(ctx, client.ObjectKeyFromObject(publicIP))

	log.V(1).Info("Ensuring finalizer is not present anymore")
	if _, err := clientutils.PatchEnsureNoFinalizer(ctx, r.Client, publicIP, publicIPFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *PublicIPReconciler) release(ctx context.Context, key client.ObjectKey) {
	r.mu.Lock()
	defer r.mu.Unlock()

	allocation, ok := r.allocationsByKey[key]
	if !ok {
		return
	}

	var sb netipx.IPSetBuilder
	sb.AddSet(r.availableIPs)
	for _, ip := range allocation.IPs {
		sb.Add(ip)
	}

	r.availableIPs, _ = sb.IPSet()
	delete(r.allocationsByKey, key)
	select {
	case <-ctx.Done():
	case r.ipReleased <- struct{}{}:
	}
}

func (r *PublicIPReconciler) allocate(ctx context.Context, publicIP *onmetalapinetv1alpha1.PublicIP) ([]netip.Addr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := client.ObjectKeyFromObject(publicIP)
	allocation, ok := r.allocationsByKey[client.ObjectKeyFromObject(publicIP)]
	if ok {
		if allocation.UID == publicIP.UID {
			return allocation.IPs, nil
		}

		var sb netipx.IPSetBuilder
		sb.AddSet(r.availableIPs)
		for _, ip := range allocation.IPs {
			sb.Add(ip)
		}

		r.availableIPs, _ = sb.IPSet()
		select {
		case <-ctx.Done():
		case r.ipReleased <- struct{}{}:
		}
	}

	if len(publicIP.Spec.IPs) == 0 {
		var ips []netip.Addr
		set := r.availableIPs

		for _, ipFamily := range publicIP.Spec.IPFamilies {
			var prefix netip.Prefix
			prefix, set, ok = r.availableIPs.RemoveFreePrefix(IPFamilyBitLen(ipFamily))
			if !ok {
				return nil, fmt.Errorf("no ip available")
			}

			ips = append(ips, prefix.Addr())
		}

		r.availableIPs = set
		r.allocationsByKey[key] = Allocation{
			UID: publicIP.UID,
			IPs: ips,
		}
		return ips, nil
	}

	set := r.availableIPs
	for _, ip := range publicIP.Spec.IPs {
		if !set.Contains(ip.Addr) {
			return nil, fmt.Errorf("cannot allocate")
		}

		var sb netipx.IPSetBuilder
		sb.AddSet(set)
		sb.Remove(ip.Addr)
		set, _ = sb.IPSet()
	}

	r.availableIPs = set
	r.allocationsByKey[key] = Allocation{
		UID: publicIP.UID,
		IPs: CommonV1Alpha1IPsToNetIPAddrs(publicIP.Spec.IPs),
	}
	return CommonV1Alpha1IPsToNetIPAddrs(publicIP.Spec.IPs), nil
}

func (r *PublicIPReconciler) reconcile(ctx context.Context, log logr.Logger, publicIP *onmetalapinetv1alpha1.PublicIP) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, publicIP, publicIPFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Finalizer is present")

	if _, ok := publicIP.Annotations[onmetalapinetv1alpha1.ReconcileRequestAnnotation]; ok {
		log.V(1).Info("Removing reconcile annotation")
		if err := r.removeReconciliationAnnotation(ctx, publicIP); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing reconcile annotation: %w", err)
		}
		log.V(1).Info("Removed reconcile annotation, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Allocating")
	ips, err := r.allocate(ctx, publicIP)
	if err != nil {
		log.V(1).Info("Error allocating, patching public ip as pending", "Error", err)
		if err := r.patchStatusPending(ctx, publicIP); err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("Successfully marked public ip as pending")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("IPs", ips)
	log.V(1).Info("Successfully allocated ips", "IPs", ips)

	if len(publicIP.Spec.IPs) == 0 {
		log.V(1).Info("Patching allocated ips into spec")
		if err := r.patchPublicIPSpecIPs(ctx, publicIP, ips); err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("Patched allocated ips into spec, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Patching public ip as allocated")
	if err := r.patchStatusAllocated(ctx, publicIP, ips); err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info("Patched public ip status")

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *PublicIPReconciler) patchPublicIPSpecIPs(ctx context.Context, publicIP *onmetalapinetv1alpha1.PublicIP, ips []netip.Addr) error {
	base := publicIP.DeepCopy()
	publicIP.Spec.IPs = NetIPAddrsToCommonV1Alpha1IPs(ips)
	if err := r.Patch(ctx, publicIP, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching spec ips: %w", err)
	}
	return nil
}

func (r *PublicIPReconciler) patchStatusPending(ctx context.Context, publicIP *onmetalapinetv1alpha1.PublicIP) error {
	base := publicIP.DeepCopy()
	publicIP.Status.IPs = nil
	onmetalapinetv1alpha1.SetPublicIPCondition(&publicIP.Status.Conditions, onmetalapinetv1alpha1.PublicIPCondition{
		Type:    onmetalapinetv1alpha1.PublicIPAllocated,
		Reason:  "Pending",
		Status:  corev1.ConditionFalse,
		Message: "The public ip could not yet be allocated.",
	})
	if err := r.Status().Patch(ctx, publicIP, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching status: %w", err)
	}
	return nil
}

func (r *PublicIPReconciler) patchStatusAllocated(ctx context.Context, publicIP *onmetalapinetv1alpha1.PublicIP, ips []netip.Addr) error {
	base := publicIP.DeepCopy()
	publicIP.Status.IPs = NetIPAddrsToCommonV1Alpha1IPs(ips)
	onmetalapinetv1alpha1.SetPublicIPCondition(&publicIP.Status.Conditions, onmetalapinetv1alpha1.PublicIPCondition{
		Type:    onmetalapinetv1alpha1.PublicIPAllocated,
		Reason:  "Allocated",
		Status:  corev1.ConditionTrue,
		Message: "The public ip was successfully allocated.",
	})
	if err := r.Status().Patch(ctx, publicIP, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching status: %w", err)
	}
	return nil
}

func (r *PublicIPReconciler) isAllocated(publicIP *onmetalapinetv1alpha1.PublicIP) bool {
	idx := onmetalapinetv1alpha1.PublicIPConditionIndex(publicIP.Status.Conditions, onmetalapinetv1alpha1.PublicIPAllocated)
	return idx != 1 && publicIP.Status.Conditions[idx].Status == corev1.ConditionTrue
}

func (r *PublicIPReconciler) initialize(ctx context.Context) error {
	r.availableIPs = r.InitialAvailableIPs
	r.allocationsByKey = make(map[client.ObjectKey]Allocation)
	r.ipReleased = make(chan struct{}, 1024)
	publicIPList := &onmetalapinetv1alpha1.PublicIPList{}
	if err := r.APIReader.List(ctx, publicIPList); err != nil {
		return fmt.Errorf("error listing public ips: %w", err)
	}

	var sb netipx.IPSetBuilder
	sb.AddSet(r.InitialAvailableIPs)

	for _, publicIP := range publicIPList.Items {
		publicIPKey := client.ObjectKeyFromObject(&publicIP)
		if !r.isAllocated(&publicIP) {
			continue
		}

		for _, ip := range publicIP.Status.IPs {
			set, _ := sb.IPSet()
			if !set.Contains(ip.Addr) {
				return fmt.Errorf("[public ip %s] cannot allocate %s", publicIPKey, ip.Addr)
			}

			sb.Remove(ip.Addr)
		}

		r.allocationsByKey[publicIPKey] = Allocation{
			UID: publicIP.UID,
			IPs: CommonV1Alpha1IPsToNetIPAddrs(publicIP.Status.IPs),
		}
	}

	return nil
}

func ipSetCanFitPublicIP(ipSet *netipx.IPSet, publicIP *onmetalapinetv1alpha1.PublicIP) bool {
	if len(publicIP.Spec.IPs) > 0 {
		for _, ip := range publicIP.Spec.IPs {
			if !ipSet.Contains(ip.Addr) {
				return false
			}
		}
		return true
	}

	var ok bool
	for _, ipFamily := range publicIP.Spec.IPFamilies {
		if _, ipSet, ok = ipSet.RemoveFreePrefix(IPFamilyBitLen(ipFamily)); !ok {
			return false
		}
	}
	return true
}

func (r *PublicIPReconciler) determineReconciliationCandidates(publicIPs []onmetalapinetv1alpha1.PublicIP) []onmetalapinetv1alpha1.PublicIP {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []onmetalapinetv1alpha1.PublicIP
	for _, publicIP := range publicIPs {
		if !publicIP.DeletionTimestamp.IsZero() {
			continue
		}

		if r.isAllocated(&publicIP) {
			continue
		}

		if ipSetCanFitPublicIP(r.availableIPs, &publicIP) {
			candidates = append(candidates, publicIP)
		}
	}
	return candidates
}

func (r *PublicIPReconciler) requestReconciliation(ctx context.Context, publicIP *onmetalapinetv1alpha1.PublicIP) error {
	base := publicIP.DeepCopy()
	if publicIP.Annotations == nil {
		publicIP.Annotations = make(map[string]string)
	}
	publicIP.Annotations[onmetalapinetv1alpha1.ReconcileRequestAnnotation] = time.Now().Format(time.RFC3339Nano)
	if err := r.Patch(ctx, publicIP, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching reconcile annotation: %w", err)
	}
	return nil
}

func (r *PublicIPReconciler) removeReconciliationAnnotation(ctx context.Context, publicIP *onmetalapinetv1alpha1.PublicIP) error {
	base := publicIP.DeepCopy()
	delete(publicIP.Annotations, onmetalapinetv1alpha1.ReconcileRequestAnnotation)
	if err := r.Patch(ctx, publicIP, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error removing reconcile annotation: %w", err)
	}
	return nil
}

func (r *PublicIPReconciler) requeuePublicIPCandidates(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithName("publicip").WithName("requeue-candidates")
	for {
		select {
		case <-ctx.Done():
			log.V(1).Info("Shutting down candidate requeuing")
			return nil
		case <-r.ipReleased:
			if err := func() error {
				log.V(1).Info("IP released")

				log.V(1).Info("Listing public ips")
				publicIPList := &onmetalapinetv1alpha1.PublicIPList{}
				if err := r.List(ctx, publicIPList); err != nil {
					return fmt.Errorf("error listing public ips: %w", err)
				}

				log.V(1).Info("Determining candidates")
				candidates := r.determineReconciliationCandidates(publicIPList.Items)

				var errs []error
				for _, candidate := range candidates {
					log.V(1).Info("Requesting reconciliation", "CandidateKey", client.ObjectKeyFromObject(&candidate))
					if err := r.requestReconciliation(ctx, &candidate); err != nil {
						errs = append(errs, err)
					}
				}

				if len(errs) > 0 {
					return fmt.Errorf("error(s) requesting candidate reconciliation(s): %v", errs)
				}
				log.V(1).Info("Handled ip release")
				return nil
			}(); err != nil {
				log.Error(err, "Error requeuing public ip candidates")
			}
		}
	}
}

func (r *PublicIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		setupLog := ctrl.LoggerFrom(ctx).WithName("publicip").WithName("setup")
		setupLog.V(1).Info("Initializing")
		if err := r.initialize(ctx); err != nil {
			return fmt.Errorf("error initializing: %w", err)
		}
		setupLog.V(1).Info("Initialized")

		if err := mgr.Add(manager.RunnableFunc(r.requeuePublicIPCandidates)); err != nil {
			return fmt.Errorf("error adding requeue public ip candidates runnable: %w", err)
		}

		return ctrl.NewControllerManagedBy(mgr).
			For(&onmetalapinetv1alpha1.PublicIP{}).
			Complete(r)
	}))
}
