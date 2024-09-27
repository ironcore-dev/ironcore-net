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
	metalnetletclient "github.com/ironcore-dev/ironcore-net/metalnetlet/client"
	metalnetlethandler "github.com/ironcore-dev/ironcore-net/metalnetlet/handler"
	"github.com/ironcore-dev/ironcore-net/networkid"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type NetworkReconciler struct {
	client.Client
	MetalnetClient client.Client

	PartitionName string

	MetalnetNamespace string
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
	if err := r.MetalnetClient.DeleteAllOf(ctx, &metalnetv1alpha1.Network{},
		client.InNamespace(r.MetalnetNamespace),
		metalnetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), networkKey, &apinetv1alpha1.Network{}),
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting metalnet networks matching network label: %w", err)
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

func (r *NetworkReconciler) updateApinetNetworkStatus(ctx context.Context, network *apinetv1alpha1.Network, metalnetNetwork *metalnetv1alpha1.Network) error {
	networkBase := network.DeepCopy()
	network.Status.Peerings = metalnetNetworkPeeringsStatusToNetworkPeeringsStatus(metalnetNetwork.Status.Peerings)
	if err := r.Status().Patch(ctx, network, client.MergeFrom(networkBase)); err != nil {
		return fmt.Errorf("unable to patch network: %w", err)
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

	log.V(1).Info("Applying metalnet network")
	metalnetNetwork := &metalnetv1alpha1.Network{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metalnetv1alpha1.GroupVersion.String(),
			Kind:       "Network",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.MetalnetNamespace,
			Name:      string(network.UID),
			Labels:    metalnetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), network),
		},
		Spec: metalnetv1alpha1.NetworkSpec{
			ID: vni,
		},
	}

	peeredPrefixes := []metalnetv1alpha1.PeeredPrefix{}
	for _, peering := range network.Spec.Peerings {
		id, err := networkid.ParseVNI(peering.ID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to parse peered network ID: %w", err)
		}

		metalnetNetwork.Spec.PeeredIDs = append(metalnetNetwork.Spec.PeeredIDs, id)

		if len(peering.Prefixes) > 0 {
			ipPrefixes := getIPPrefixes(peering.Prefixes)
			peeredPrefix := metalnetv1alpha1.PeeredPrefix{
				ID:       id,
				Prefixes: ipPrefixesToMetalnetPrefixes(ipPrefixes),
			}
			peeredPrefixes = append(peeredPrefixes, peeredPrefix)
		}
	}
	metalnetNetwork.Spec.PeeredPrefixes = peeredPrefixes

	if err := r.MetalnetClient.Patch(ctx, metalnetNetwork, client.Apply, MetalnetFieldOwner, client.ForceOwnership); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying network: %w", err)
	}
	log.V(1).Info("Applied metalnet network")

	log.V(1).Info("Updating apinet network status")
	if err := r.updateApinetNetworkStatus(ctx, network, metalnetNetwork); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating apinet networkstatus: %w", err)
	}
	log.V(1).Info("Updated apinet network status")

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
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
		).
		WatchesRawSource(
			source.Kind(metalnetCache, &metalnetv1alpha1.Network{}),
			metalnetlethandler.EnqueueRequestForSource(r.Scheme(), r.RESTMapper(), &apinetv1alpha1.Network{}),
		).
		Complete(r)
}
