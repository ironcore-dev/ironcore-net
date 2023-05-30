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
	"strconv"

	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	apinetletv1alpha1 "github.com/onmetal/onmetal-api-net/apinetlet/api/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	"github.com/onmetal/onmetal-api/utils/predicates"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	networkFinalizer = "apinet.api.onmetal.de/network"
)

type NetworkReconciler struct {
	client.Client
	APINetClient client.Client

	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=networks,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=networks/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=networks/status,verbs=get;update;patch

func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	network := &networkingv1alpha1.Network{}
	if err := r.Get(ctx, req.NamespacedName, network); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting network %s: %w", req.NamespacedName, err)
		}

		return r.deleteGone(ctx, log, req.NamespacedName)
	}

	return r.reconcileExists(ctx, log, network)
}

func (r *NetworkReconciler) deleteGone(ctx context.Context, log logr.Logger, networkKey client.ObjectKey) (ctrl.Result, error) {
	log.V(1).Info("Delete gone")

	log.V(1).Info("Deleting any matching apinet networks")
	if err := r.APINetClient.DeleteAllOf(ctx, &onmetalapinetv1alpha1.Network{},
		client.InNamespace(r.APINetNamespace),
		client.MatchingLabels{
			apinetletv1alpha1.NetworkNamespaceLabel: networkKey.Namespace,
			apinetletv1alpha1.NetworkNameLabel:      networkKey.Name,
		},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting apinet networks: %w", err)
	}

	log.V(1).Info("Issued delete for any leftover apinet network")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) reconcileExists(ctx context.Context, log logr.Logger, network *networkingv1alpha1.Network) (ctrl.Result, error) {
	log = log.WithValues("UID", network.UID)
	if !network.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, network)
	}
	return r.reconcile(ctx, log, network)
}

func (r *NetworkReconciler) delete(ctx context.Context, log logr.Logger, network *networkingv1alpha1.Network) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(network, networkFinalizer) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Deleting target apinet network if any")
	if err := r.APINetClient.Delete(ctx, &onmetalapinetv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(network.UID),
		},
	}); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting target apinet network: %w", err)
		}

		log.V(1).Info("Target apinet network is gone, removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, network, networkFinalizer); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Removed finalizer")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Target apinet network is not yet gone, requeueing")
	return ctrl.Result{Requeue: true}, nil
}

func (r *NetworkReconciler) updateNetworkStatus(ctx context.Context, network *networkingv1alpha1.Network, state networkingv1alpha1.NetworkState) error {
	networkBase := network.DeepCopy()
	network.Status.State = state
	if err := r.Status().Patch(ctx, network, client.MergeFrom(networkBase)); err != nil {
		return fmt.Errorf("unable to patch network: %w", err)
	}
	return nil
}

func (r *NetworkReconciler) reconcile(ctx context.Context, log logr.Logger, network *networkingv1alpha1.Network) (ctrl.Result, error) {
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

	vni, err := r.applyAPINetNetwork(ctx, log, network)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying apinet network: %w", err)
	}
	if vni == nil {
		log.V(1).Info("APINet network is not yet allocated, setting network to pending")
		if err := r.updateNetworkStatus(ctx, network, networkingv1alpha1.NetworkStatePending); err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating network status: %w", err)
		}
		return ctrl.Result{}, nil
	}

	log = log.WithValues("VNI", *vni)
	log.V(1).Info("APINet network is allocated")

	if network.Spec.ProviderID == "" {
		log.V(1).Info("Setting network provider id")
		if err := r.setNetworkProviderID(ctx, network, *vni); err != nil {
			return ctrl.Result{}, fmt.Errorf("error setting network provider id: %w", err)
		}

		log.V(1).Info("Set network provider id, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Updating network status")
	if err := r.updateNetworkStatus(ctx, network, networkingv1alpha1.NetworkStateAvailable); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating network status: %w", err)
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) setNetworkProviderID(ctx context.Context, network *networkingv1alpha1.Network, vni int32) error {
	networkBase := network.DeepCopy()
	network.Spec.ProviderID = strconv.FormatInt(int64(vni), 10)
	if err := r.Patch(ctx, network, client.MergeFrom(networkBase)); err != nil {
		return fmt.Errorf("unable to patch network: %w", err)
	}
	return nil
}

func isAPINetNetworkAllocated(apiNetNetwork *onmetalapinetv1alpha1.Network) bool {
	apiNetNetworkConditions := apiNetNetwork.Status.Conditions
	idx := onmetalapinetv1alpha1.NetworkConditionIndex(apiNetNetworkConditions, onmetalapinetv1alpha1.NetworkAllocated)
	if idx == -1 || apiNetNetworkConditions[idx].Status != corev1.ConditionTrue {
		return false
	}

	return apiNetNetwork.Spec.VNI != nil
}

func (r *NetworkReconciler) applyAPINetNetwork(ctx context.Context, log logr.Logger, network *networkingv1alpha1.Network) (*int32, error) {
	var vni *int32
	if providerID := network.Spec.ProviderID; providerID != "" {
		v, err := strconv.ParseInt(providerID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("error parsing network provider id %s: %w", providerID, err)
		}

		vni = pointer.Int32(int32(v))
	}

	apiNetNetwork := &onmetalapinetv1alpha1.Network{
		TypeMeta: metav1.TypeMeta{
			APIVersion: onmetalapinetv1alpha1.GroupVersion.String(),
			Kind:       "Network",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(network.UID),
			Labels: map[string]string{
				apinetletv1alpha1.NetworkNamespaceLabel: network.Namespace,
				apinetletv1alpha1.NetworkNameLabel:      network.Name,
				apinetletv1alpha1.NetworkUIDLabel:       string(network.UID),
			},
		},
		Spec: onmetalapinetv1alpha1.NetworkSpec{
			VNI: vni,
		},
	}

	log.V(1).Info("Applying apinet network")
	if err := r.APINetClient.Patch(ctx, apiNetNetwork, client.Apply,
		client.FieldOwner(apinetletv1alpha1.FieldOwner),
		client.ForceOwnership,
	); err != nil {
		return nil, fmt.Errorf("error applying apinet network: %w", err)
	}
	log.V(1).Info("Applied apinet network")

	if !isAPINetNetworkAllocated(apiNetNetwork) {
		return nil, nil
	}
	return apiNetNetwork.Spec.VNI, nil
}

var apiNetNetworkAllocationChanged = predicate.Funcs{
	UpdateFunc: func(event event.UpdateEvent) bool {
		oldAPINetNetwork, newAPINetNetwork := event.ObjectOld.(*onmetalapinetv1alpha1.Network), event.ObjectNew.(*onmetalapinetv1alpha1.Network)
		return isAPINetNetworkAllocated(oldAPINetNetwork) != isAPINetNetworkAllocated(newAPINetNetwork)
	},
}

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCluster cluster.Cluster) error {
	log := ctrl.Log.WithName("network").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1alpha1.Network{},
			builder.WithPredicates(
				predicates.ResourceHasFilterLabel(log, r.WatchFilterValue),
				predicates.ResourceIsNotExternallyManaged(log),
			),
		).
		WatchesRawSource(
			source.Kind(apiNetCluster.GetCache(), &onmetalapinetv1alpha1.Network{}),
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
				apiNetNetwork := obj.(*onmetalapinetv1alpha1.Network)

				if apiNetNetwork.Namespace != r.APINetNamespace {
					return nil
				}

				namespace, ok := apiNetNetwork.Labels[apinetletv1alpha1.NetworkNamespaceLabel]
				if !ok {
					return nil
				}

				name, ok := apiNetNetwork.Labels[apinetletv1alpha1.NetworkNameLabel]
				if !ok {
					return nil
				}

				return []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: namespace, Name: name}}}
			}),
			builder.WithPredicates(
				apiNetNetworkAllocationChanged,
			),
		).
		Complete(r)
}
