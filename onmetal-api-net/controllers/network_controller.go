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
	"sync"

	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	"github.com/onmetal/controller-utils/set"
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	DefaultMinVNI int32 = 200
	DefaultMaxVNI int32 = (1 << 24) - 1

	networkFinalizer = "apinet.api.onmetal.de/network"
)

type networkAllocation struct {
	UID types.UID
	VNI int32
}

type NetworkReconciler struct {
	mu sync.RWMutex

	record.EventRecorder
	client.Client
	APIReader client.Reader

	MinVNI int32
	MaxVNI int32

	allocationsByKey map[client.ObjectKey]networkAllocation
	taken            set.Set[int32]
	released         chan struct{}
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=networks,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=networks/finalizers,verbs=update
//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=networks/status,verbs=get;update;patch

func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	network := &onmetalapinetv1alpha1.Network{}
	if err := r.Get(ctx, req.NamespacedName, network); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting network %s: %w", req.NamespacedName, err)
		}

		r.release(ctx, req.NamespacedName)
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, network)
}

func (r *NetworkReconciler) release(ctx context.Context, key client.ObjectKey) {
	r.mu.Lock()
	defer r.mu.Unlock()

	allocation, ok := r.allocationsByKey[key]
	if !ok {
		return
	}

	r.taken.Delete(allocation.VNI)
	delete(r.allocationsByKey, key)

	select {
	case <-ctx.Done():
	case r.released <- struct{}{}:
	}
}

func (r *NetworkReconciler) allocate(ctx context.Context, network *onmetalapinetv1alpha1.Network) (int32, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := client.ObjectKeyFromObject(network)
	allocation, ok := r.allocationsByKey[key]
	if ok {
		if allocation.UID == network.UID {
			return allocation.VNI, nil
		}

		r.taken.Delete(allocation.VNI)
		delete(r.allocationsByKey, key)

		select {
		case <-ctx.Done():
		case r.released <- struct{}{}:
		}
	}

	if vni := network.Spec.VNI; vni != nil {
		if r.taken.Has(*vni) {
			return 0, fmt.Errorf("vni %d is already taken", *vni)
		}

		r.taken.Insert(*vni)
		r.allocationsByKey[key] = networkAllocation{
			UID: network.UID,
			VNI: *vni,
		}
		return *vni, nil
	}

	if allVNIsTaken(r.taken, r.MinVNI, r.MaxVNI) {
		return 0, fmt.Errorf("no vni available")
	}

	vni := r.MinVNI
	for ; r.taken.Has(vni); vni++ {
	}

	r.taken.Insert(vni)
	r.allocationsByKey[key] = networkAllocation{
		UID: network.UID,
		VNI: vni,
	}
	return vni, nil
}

func (r *NetworkReconciler) reconcileExists(ctx context.Context, log logr.Logger, network *onmetalapinetv1alpha1.Network) (ctrl.Result, error) {
	if !network.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, network)
	}
	return r.reconcile(ctx, log, network)
}

func (r *NetworkReconciler) delete(ctx context.Context, log logr.Logger, network *onmetalapinetv1alpha1.Network) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	log.V(1).Info("Releasing any associated vnis")
	r.release(ctx, client.ObjectKeyFromObject(network))

	log.V(1).Info("Ensuring finalizer is not present anymore")
	if _, err := clientutils.PatchEnsureNoFinalizer(ctx, r.Client, network, networkFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) reconcile(ctx context.Context, log logr.Logger, network *onmetalapinetv1alpha1.Network) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, network, networkFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Finalizer is present")

	if _, ok := network.Annotations[onmetalapinetv1alpha1.ReconcileRequestAnnotation]; ok {
		log.V(1).Info("Removing reconcile annotation")
		if err := PatchRemoveReconcileAnnotation(ctx, r.Client, network); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing reconcile annotation: %w", err)
		}
		log.V(1).Info("Removed reconcile annotation, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Allocating")
	vni, err := r.allocate(ctx, network)
	if err != nil {
		log.V(1).Info("Error allocating, patching as pending", "Error", err)
		r.Eventf(network, corev1.EventTypeNormal, FailedAllocatingNetwork, "Failed allocating: %w", err)
		if err := r.patchStatusPending(ctx, network); err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("Successfully marked as pending")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("VNI", vni)
	log.V(1).Info("Successfully allocated")

	if network.Spec.VNI == nil {
		log.V(1).Info("Patching allocated vni into spec")
		if err := r.patchSpecVNI(ctx, network, vni); err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("Patched allocated vni into spec, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Patching as allocated")
	if err := r.patchStatusAllocated(ctx, network); err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info("Patched as allocated")

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) patchSpecVNI(ctx context.Context, network *onmetalapinetv1alpha1.Network, vni int32) error {
	base := network.DeepCopy()
	network.Spec.VNI = &vni
	if err := r.Patch(ctx, network, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching spec vni: %w", err)
	}
	return nil
}

func (r *NetworkReconciler) patchStatusPending(ctx context.Context, network *onmetalapinetv1alpha1.Network) error {
	base := network.DeepCopy()
	onmetalapinetv1alpha1.SetNetworkCondition(&network.Status.Conditions, onmetalapinetv1alpha1.NetworkCondition{
		Type:    onmetalapinetv1alpha1.NetworkAllocated,
		Reason:  "Pending",
		Status:  corev1.ConditionFalse,
		Message: "The network could not yet be allocated.",
	})
	if err := r.Status().Patch(ctx, network, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching status: %w", err)
	}
	return nil
}

func (r *NetworkReconciler) patchStatusAllocated(ctx context.Context, network *onmetalapinetv1alpha1.Network) error {
	base := network.DeepCopy()
	onmetalapinetv1alpha1.SetNetworkCondition(&network.Status.Conditions, onmetalapinetv1alpha1.NetworkCondition{
		Type:    onmetalapinetv1alpha1.NetworkAllocated,
		Reason:  "Allocated",
		Status:  corev1.ConditionTrue,
		Message: "The network was successfully allocated.",
	})
	if err := r.Status().Patch(ctx, network, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching status: %w", err)
	}
	return nil
}

func (r *NetworkReconciler) isAllocated(network *onmetalapinetv1alpha1.Network) bool {
	idx := onmetalapinetv1alpha1.NetworkConditionIndex(network.Status.Conditions, onmetalapinetv1alpha1.NetworkAllocated)
	return idx != -1 && network.Status.Conditions[idx].Status == corev1.ConditionTrue
}

func (r *NetworkReconciler) initialize(ctx context.Context) error {
	if r.MinVNI <= 0 {
		return fmt.Errorf("min vni %d has to be > 0", r.MinVNI)
	}
	if r.MaxVNI < r.MinVNI {
		return fmt.Errorf("max vni %d has to be >= min vni %d", r.MaxVNI, r.MinVNI)
	}

	r.taken = set.New[int32]()
	r.allocationsByKey = make(map[client.ObjectKey]networkAllocation)
	r.released = make(chan struct{}, 1024)

	networkList := &onmetalapinetv1alpha1.NetworkList{}
	if err := r.APIReader.List(ctx, networkList); err != nil {
		return fmt.Errorf("error listing networks: %w", err)
	}

	for _, network := range networkList.Items {
		networkKey := client.ObjectKeyFromObject(&network)
		if !r.isAllocated(&network) {
			continue
		}

		vni := *network.Spec.VNI
		if r.taken.Has(vni) || vni > r.MaxVNI {
			return fmt.Errorf("[network %s] cannot allocate vni %d", networkKey, vni)
		}

		r.taken.Insert(vni)
		r.allocationsByKey[networkKey] = networkAllocation{
			UID: network.UID,
			VNI: vni,
		}
	}

	return nil
}

func (r *NetworkReconciler) determineReconciliationCandidates(networks []onmetalapinetv1alpha1.Network) []onmetalapinetv1alpha1.Network {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if allVNIsTaken(r.taken, r.MinVNI, r.MaxVNI) {
		return nil
	}

	var candidates []onmetalapinetv1alpha1.Network
	for _, network := range networks {
		if !network.DeletionTimestamp.IsZero() {
			continue
		}

		if r.isAllocated(&network) {
			continue
		}

		if network.Spec.VNI != nil && r.taken.Has(*network.Spec.VNI) {
			continue
		}

		candidates = append(candidates, network)
	}
	return candidates
}

func allVNIsTaken(taken set.Set[int32], minVNI, maxVNI int32) bool {
	return int32(taken.Len()) == (maxVNI-minVNI)+1
}

func (r *NetworkReconciler) requeueNetworkCandidates(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithName("network").WithName("requeue-candidates")
	for {
		select {
		case <-ctx.Done():
			log.V(1).Info("Shutting down candidate requeuing")
			return nil
		case <-r.released:
			if err := func() error {
				log.V(1).Info("VNI released")

				log.V(1).Info("Listing networks")
				networkList := &onmetalapinetv1alpha1.NetworkList{}
				if err := r.List(ctx, networkList); err != nil {
					return fmt.Errorf("error listing networks: %w", err)
				}

				log.V(1).Info("Determining candidates")
				candidates := r.determineReconciliationCandidates(networkList.Items)

				var errs []error
				for _, candidate := range candidates {
					log.V(1).Info("Requesting reconciliation", "CandidateKey", client.ObjectKeyFromObject(&candidate))
					if err := PatchAddReconcileAnnotation(ctx, r.Client, &candidate); err != nil {
						errs = append(errs, err)
					}
				}

				if len(errs) > 0 {
					return fmt.Errorf("error(s) requesting candidate reconciliation(s): %v", errs)
				}
				log.V(1).Info("Handled vni release")
				return nil
			}(); err != nil {
				log.Error(err, "Error requeuing network candidates")
			}
		}
	}
}

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		setupLog := ctrl.LoggerFrom(ctx).WithName("network").WithName("setup")
		setupLog.V(1).Info("Initializing")
		if err := r.initialize(ctx); err != nil {
			return fmt.Errorf("error initializing: %w", err)
		}
		setupLog.V(1).Info("Initialized")

		if err := mgr.Add(manager.RunnableFunc(r.requeueNetworkCandidates)); err != nil {
			return fmt.Errorf("error adding requeue network candidates runnable: %w", err)
		}

		return ctrl.NewControllerManagedBy(mgr).
			For(&onmetalapinetv1alpha1.Network{}).
			Complete(r)
	}))
}
