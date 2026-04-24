// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/controller-utils/clientutils"
	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore-net/apimachinery/equality"
	"github.com/ironcore-dev/ironcore-net/networkid"
	netclientutils "github.com/ironcore-dev/ironcore-net/utils/client"
	utilhandler "github.com/ironcore-dev/ironcore-net/utils/handler"
	"github.com/ironcore-dev/ironcore-net/utils/origin"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	metalnetv1alpha1ac "github.com/ironcore-dev/metalnet/api/v1alpha1/applyconfiguration/api/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	NetworkOrigin = &origin.Origin{
		Name:       "metalnetlet.ironcore.dev/network",
		Namespaced: true,
	}
)

type NetworkReconciler struct {
	client.Client
	MetalnetClient client.Client

	PartitionName string

	MetalnetNamespace string

	NetworkPeeringDisabled bool
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networks,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networks/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networks/status,verbs=get;update;patch

//+cluster=metalnet:kubebuilder:rbac:groups=networking.metalnet.ironcore.dev,resources=networks,verbs=get;list;watch;create;update;patch;delete;deletecollection

func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	network := &apinetv1alpha1.Network{}
	if err := r.Get(ctx, req.NamespacedName, network); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting network: %w", err)
		}
		return r.deleteGone(ctx, log, req.NamespacedName)
	}

	return r.reconcileExists(ctx, log, network)
}

func (r *NetworkReconciler) deleteGone(ctx context.Context, log logr.Logger, networkKey client.ObjectKey) (ctrl.Result, error) {
	log.V(1).Info("Delete gone")

	log.V(1).Info("Deleting all metalnet networks that match the network label")
	stemmingFromKey := &netclientutils.StemmingFromKey{Origin: NetworkOrigin, SourceKey: networkKey}
	if _, err := netclientutils.ListAnd(r.MetalnetClient, &metalnetv1alpha1.NetworkList{},
		client.InNamespace(r.MetalnetNamespace),
		stemmingFromKey.UIDExistsSelector(),
	).DeletePredicate(ctx, stemmingFromKey); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting metalnet networks stemming from network key: %w", err)
	}

	log.V(1).Info("Deleted gone")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) reconcileExists(ctx context.Context, log logr.Logger, network *apinetv1alpha1.Network) (ctrl.Result, error) {
	if !network.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, network)
	}
	return r.reconcile(ctx, log, network)
}

func (r *NetworkReconciler) delete(ctx context.Context, log logr.Logger, network *apinetv1alpha1.Network) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(network, PartitionFinalizer(r.PartitionName)) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Finalizer present, cleaning up")

	log.V(1).Info("Deleting metalnet network if present")
	metalnetNetwork := &metalnetv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.MetalnetNamespace,
			Name:      string(network.UID),
		},
	}
	err := r.MetalnetClient.Delete(ctx, metalnetNetwork)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("error deleting metalnet network: %w", err)
	}
	if err == nil {
		log.V(1).Info("Issued deletion of metalnet network")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Metalnet network is gone, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, network, PartitionFinalizer(r.PartitionName)); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}
	log.V(1).Info("Removed finalizer")

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) updateApinetNetworkStatus(ctx context.Context, log logr.Logger, network *apinetv1alpha1.Network, metalnetNetwork *metalnetv1alpha1.Network) error {
	if !r.NetworkPeeringDisabled {
		apinetStatusPeerings := metalnetNetworkPeeringsStatusToNetworkPeeringsStatus(metalnetNetwork.Status.Peerings)
		if !equality.Semantic.DeepEqual(network.Status.Peerings[r.PartitionName], apinetStatusPeerings) {
			log.V(1).Info("Patching apinet network status", "status", apinetStatusPeerings)
			networkBase := network.DeepCopy()
			if network.Status.Peerings == nil {
				network.Status.Peerings = make(map[string][]apinetv1alpha1.NetworkPeeringStatus)
			}
			network.Status.Peerings[r.PartitionName] = apinetStatusPeerings
			if err := r.Status().Patch(ctx, network, client.MergeFrom(networkBase)); err != nil {
				return fmt.Errorf("unable to patch network: %w", err)
			}
			log.V(1).Info("Patched apinet network status")
		}
	}
	return nil
}

func (r *NetworkReconciler) reconcile(ctx context.Context, log logr.Logger, network *apinetv1alpha1.Network) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	vni, err := networkid.ParseVNI(network.Spec.ID)
	if err != nil {
		log.Error(err, "Network has invalid ID", "ID", network.Spec.ID)
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, network, PartitionFinalizer(r.PartitionName))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer")
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Finalizer is present")

	var peeredIDs []int32
	var peeredPrefixes []metalnetv1alpha1.PeeredPrefix

	if !r.NetworkPeeringDisabled {
		for _, peering := range network.Spec.Peerings {
			id, err := networkid.ParseVNI(peering.ID)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to parse peered network ID: %w", err)
			}

			peeredIDs = append(peeredIDs, id)

			if len(peering.Prefixes) > 0 {
				ipPrefixes := getIPPrefixes(peering.Prefixes)
				peeredPrefixes = append(peeredPrefixes, metalnetv1alpha1.PeeredPrefix{
					ID:       id,
					Prefixes: ipPrefixesToMetalnetPrefixes(ipPrefixes),
				})
			}
		}
	}

	log.V(1).Info("Applying metalnet network")

	networkName := string(network.UID)

	metalnetNetworkapplyCfg := metalnetv1alpha1ac.Network(networkName, r.MetalnetNamespace).
		WithAnnotations(NetworkOrigin.Annotations(network)).
		WithLabels(NetworkOrigin.Labels(network)).
		WithSpec(
			metalnetv1alpha1ac.NetworkSpec().
				WithID(vni).
				WithPeeredIDs(peeredIDs...).
				WithPeeredPrefixes(convertPeeredPrefixesToApply(peeredPrefixes)...),
		)

	if err := r.MetalnetClient.Apply(ctx, metalnetNetworkapplyCfg, MetalnetFieldOwner, client.ForceOwnership); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying metalnet network: %w", err)
	}

	log.V(1).Info("Applied metalnet network")

	// --- Fetch latest for status update ---
	metalnetNetwork := &metalnetv1alpha1.Network{}
	key := client.ObjectKey{Namespace: r.MetalnetNamespace, Name: networkName}
	if err := r.MetalnetClient.Get(ctx, key, metalnetNetwork); err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting metalnet network: %w", err)
	}

	log.V(1).Info("Updating apinet network status")
	if err := r.updateApinetNetworkStatus(ctx, log, network, metalnetNetwork); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating apinet networkstatus: %w", err)
	}

	log.V(1).Info("Updated apinet network status")
	log.V(1).Info("Reconciled")

	return ctrl.Result{}, nil
}

func convertPeeredPrefixesToApply(in []metalnetv1alpha1.PeeredPrefix) []*metalnetv1alpha1ac.PeeredPrefixApplyConfiguration {
	var out []*metalnetv1alpha1ac.PeeredPrefixApplyConfiguration

	for _, p := range in {
		cfg := metalnetv1alpha1ac.PeeredPrefix().
			WithID(p.ID).
			WithPrefixes(p.Prefixes...) // assuming slice of primitives or already correct type

		out = append(out, cfg)
	}

	return out
}

func getIPPrefixes(peeringPrefixes []apinetv1alpha1.PeeringPrefix) []net.IPPrefix {
	ipPrefixes := []net.IPPrefix{}
	for _, peeringPrefix := range peeringPrefixes {
		if peeringPrefix.Prefix != nil {
			ipPrefixes = append(ipPrefixes, *peeringPrefix.Prefix)
		}
	}
	return ipPrefixes
}

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager, metalnetCache cache.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&apinetv1alpha1.Network{},
			// builder.WithPredicates(r.networkStatusChangedPredicate()),
		).
		WatchesRawSource(
			source.Kind[client.Object](
				metalnetCache,
				&metalnetv1alpha1.Network{},
				utilhandler.EnqueueRequestByOrigin(NetworkOrigin),
			),
		).
		Complete(r)
}
