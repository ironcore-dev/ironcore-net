// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-logr/logr"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	"github.com/ironcore-dev/ironcore-net/apinetlet/handler"
	"github.com/ironcore-dev/ironcore-net/apinetlet/provider"

	"github.com/ironcore-dev/controller-utils/clientutils"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/predicates"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	networkFinalizer = "apinet.ironcore.dev/network"
)

type NetworkReconciler struct {
	client.Client
	APINetClient client.Client

	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=networks,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=networks/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=networks/status,verbs=get;update;patch

//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networks,verbs=get;list;watch;create;update;patch;delete;deletecollection

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

	log.V(1).Info("Deleting any matching APINet networks")
	if err := r.APINetClient.DeleteAllOf(ctx, &apinetv1alpha1.Network{},
		client.InNamespace(r.APINetNamespace),
		apinetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), networkKey, &networkingv1alpha1.Network{}),
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting APINet networks: %w", err)
	}

	log.V(1).Info("Issued delete for any leftover APINet network")
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

	log.V(1).Info("Deleting target APINet network if any")
	apiNetNetwork := &apinetv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(network.UID),
		},
	}
	if err := r.APINetClient.Delete(ctx, apiNetNetwork); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting target APINet network: %w", err)
		}

		log.V(1).Info("Target APINet network is gone, removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, network, networkFinalizer); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Removed finalizer")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Target APINet network is not yet gone, requeueing")
	return ctrl.Result{Requeue: true}, nil
}

func (r *NetworkReconciler) updateNetworkStatus(ctx context.Context, network *networkingv1alpha1.Network, apiNetNetwork *apinetv1alpha1.Network, state networkingv1alpha1.NetworkState) error {
	networkBase := network.DeepCopy()
	network.Status.State = state
	network.Status.Peerings = apiNetNetworkPeeringsStatusToNetworkPeeringsStatus(apiNetNetwork.Status.Peerings, apiNetNetwork.Spec.Peerings)
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

	apiNetNetwork, err := r.applyAPINetNetwork(ctx, log, network)
	if err != nil {
		if network.Status.State != networkingv1alpha1.NetworkStateAvailable {
			if err := r.updateNetworkStatus(ctx, network, apiNetNetwork, networkingv1alpha1.NetworkStatePending); err != nil {
				log.Error(err, "Error updating network state")
			}
		}
		return ctrl.Result{}, fmt.Errorf("error applying APINet network: %w", err)
	}
	log = log.WithValues("ID", apiNetNetwork.Spec.ID)
	log.V(1).Info("Applied APINet network")

	if network.Spec.ProviderID == "" {
		log.V(1).Info("Setting network provider id")
		if err := r.setNetworkProviderID(ctx, network, apiNetNetwork); err != nil {
			return ctrl.Result{}, fmt.Errorf("error setting network provider id: %w", err)
		}

		log.V(1).Info("Set network provider id, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Updating network status")
	if err := r.updateNetworkStatus(ctx, network, apiNetNetwork, networkingv1alpha1.NetworkStateAvailable); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating network status: %w", err)
	}
	log.V(1).Info("Updated network status")

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) setNetworkProviderID(
	ctx context.Context,
	network *networkingv1alpha1.Network,
	apiNetNetwork *apinetv1alpha1.Network,
) error {
	base := network.DeepCopy()
	network.Spec.ProviderID = provider.GetNetworkID(apiNetNetwork.Namespace, apiNetNetwork.Name, apiNetNetwork.Spec.ID, apiNetNetwork.UID)
	if err := r.Patch(ctx, network, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("unable to patch network: %w", err)
	}
	return nil
}

func (r *NetworkReconciler) applyAPINetNetwork(ctx context.Context, log logr.Logger, network *networkingv1alpha1.Network) (*apinetv1alpha1.Network, error) {
	apiNetNetwork := &apinetv1alpha1.Network{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apinetv1alpha1.SchemeGroupVersion.String(),
			Kind:       "Network",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(network.UID),
			Labels:    apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), network),
		},
	}

	var peerings []apinetv1alpha1.NetworkPeering
	for _, peeringClaimRef := range network.Spec.PeeringClaimRefs {
		log.V(1).Info("Get APINet network for target network")
		targetApinetNetwork := &apinetv1alpha1.Network{}
		if err := r.APINetClient.Get(ctx, client.ObjectKey{Namespace: r.APINetNamespace, Name: string(peeringClaimRef.UID)}, targetApinetNetwork); err != nil {
			log.V(1).Info("target APINet network is not created yet")
			break
		}

		idx := slices.IndexFunc(network.Spec.Peerings, func(peering networkingv1alpha1.NetworkPeering) bool {
			peeringNetworkNamespace := peering.NetworkRef.Namespace
			if peeringNetworkNamespace == "" {
				peeringNetworkNamespace = network.Namespace
			}
			return peering.NetworkRef.Name == peeringClaimRef.Name && peeringNetworkNamespace == peeringClaimRef.Namespace
		})
		if idx != -1 {
			peeringPrefixes, err := r.getAPINetNetworkPeeringPrefixes(ctx, network.Spec.Peerings[idx].Prefixes, network.Namespace)
			if err != nil {
				return nil, fmt.Errorf("error getting APINet network peering prefixes: %w", err)
			}

			peerings = append(peerings, apinetv1alpha1.NetworkPeering{
				Name:     network.Spec.Peerings[idx].Name,
				ID:       targetApinetNetwork.Spec.ID,
				Prefixes: peeringPrefixes,
			})
		}
	}
	apiNetNetwork.Spec.Peerings = peerings

	log.V(1).Info("Applying APINet network")
	if err := r.APINetClient.Patch(ctx, apiNetNetwork, client.Apply, fieldOwner, client.ForceOwnership); err != nil {
		return nil, fmt.Errorf("error applying APINet network: %w", err)
	}
	return apiNetNetwork, nil
}

func (r *NetworkReconciler) getAPINetNetworkPeeringPrefixes(ctx context.Context, peeringPrefixes []networkingv1alpha1.PeeringPrefix, networkNamespace string) ([]apinetv1alpha1.PeeringPrefix, error) {
	apinetPeeringPrefixes := []apinetv1alpha1.PeeringPrefix{}
	for _, prefix := range peeringPrefixes {
		if prefix.Prefix != nil {
			apinetPeeringPrefixes = append(apinetPeeringPrefixes, apinetv1alpha1.PeeringPrefix{
				Name:   prefix.Name,
				Prefix: iPPrefixToAPINetIPPrefix(*prefix.Prefix),
			})
		} else if prefix.PrefixRef.Name != "" {
			ipamPrefix := &ipamv1alpha1.Prefix{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: networkNamespace, Name: prefix.PrefixRef.Name}, ipamPrefix); err != nil {
				if !apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("error getting prefix %s: %w", client.ObjectKey{Namespace: networkNamespace, Name: prefix.PrefixRef.Name}, err)
				}
				continue
			}

			if ipamPrefix.Status.Phase != ipamv1alpha1.PrefixPhaseAllocated {
				continue
			}

			apinetPeeringPrefixes = append(apinetPeeringPrefixes, apinetv1alpha1.PeeringPrefix{
				Name:   prefix.Name,
				Prefix: iPPrefixToAPINetIPPrefix(*ipamPrefix.Spec.Prefix),
			})
		}
	}
	return apinetPeeringPrefixes, nil
}

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCache cache.Cache) error {
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
			source.Kind(apiNetCache, &apinetv1alpha1.Network{}),
			handler.EnqueueRequestForSource(mgr.GetScheme(), mgr.GetRESTMapper(), &networkingv1alpha1.Network{}),
		).
		Complete(r)
}
