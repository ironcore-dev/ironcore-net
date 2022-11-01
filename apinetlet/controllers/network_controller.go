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
	"github.com/onmetal/controller-utils/metautils"
	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	apinetletclient "github.com/onmetal/onmetal-api-net/apinetlet/client"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/apis/networking/v1alpha1"
	"github.com/onmetal/onmetal-api/util/predicates"
	brokerclient "github.com/onmetal/poollet/broker/client"
	brokerhandler "github.com/onmetal/poollet/broker/handler"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	networkFinalizer = "apinet.api.onmetal.de/network"
)

type NetworkReconciler struct {
	client.Client
	APINetClient client.Client

	ClusterName     string
	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=networks,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=networks/finalizers,verbs=update

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

	log.V(1).Info("Listing if any apinet networks are present for gone network")
	networkList := &onmetalapinetv1alpha1.NetworkList{}
	if err := r.APINetClient.List(ctx, networkList,
		client.InNamespace(r.APINetNamespace),
		client.MatchingFields{apinetletclient.NetworkNetworkController: networkKey.String()},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing networks: %w", err)
	}

	var errs []error
	for _, network := range networkList.Items {
		log.V(1).Info("Deleting apinet network", "APINetNetworkKey", client.ObjectKeyFromObject(&network))
		if err := r.Delete(ctx, &network); client.IgnoreNotFound(err) != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return ctrl.Result{}, fmt.Errorf("error deleting related apinet network(s): %v", errs)
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
		return ctrl.Result{}, fmt.Errorf("error getting / applying apinet network: %w", err)
	}
	if vni == 0 {
		log.V(1).Info("APINet network is not yet allocated, patching vni annotation")
		if err := r.patchAnnotationUnallocated(ctx, network); err != nil {
			return ctrl.Result{}, err
		}
		log.V(1).Info("Patched vni annotation")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("VNI", vni)
	log.V(1).Info("APINet network is allocated")

	log.V(1).Info("Patching network vni annotation")
	if err := r.patchAnnotationAllocated(ctx, network, vni); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching network vni annotation")
	}
	log.V(1).Info("Patched network vni annotation to vni")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) applyAPINetNetwork(ctx context.Context, log logr.Logger, network *networkingv1alpha1.Network) (int32, error) {
	apiNetNetwork := &onmetalapinetv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(network.UID),
		},
	}
	log.V(1).Info("Applying apinet network")
	if _, err := brokerclient.BrokerControlledCreateOrPatch(ctx, r.APINetClient, r.ClusterName, network, apiNetNetwork, func() error {
		return nil
	}); err != nil {
		return 0, fmt.Errorf("error applying apinet network: %w", err)
	}
	log.V(1).Info("Applied public ip")

	return apiNetNetwork.Status.VNI, nil
}

func (r *NetworkReconciler) patchAnnotationAllocated(ctx context.Context, network *networkingv1alpha1.Network, vni int32) error {
	base := network.DeepCopy()
	metautils.SetAnnotation(network, onmetalapinetv1alpha1.OnmetalAPINetworkVNIAnnotation, strconv.FormatInt(int64(vni), 10))
	if err := r.Patch(ctx, network, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching network vni annotation: %w", err)
	}
	return nil
}

func (r *NetworkReconciler) patchAnnotationUnallocated(ctx context.Context, network *networkingv1alpha1.Network) error {
	base := network.DeepCopy()

	delete(network.Annotations, onmetalapinetv1alpha1.OnmetalAPINetworkVNIAnnotation)
	if err := r.Patch(ctx, network, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error patching network vni annotation: %w", err)
	}
	return nil
}

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCluster cluster.Cluster) error {
	log := ctrl.Log.WithName("network").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.Network{}).
		WithEventFilter(predicates.ResourceHasFilterLabel(log, r.WatchFilterValue)).
		Watches(
			source.NewKindWithCache(&onmetalapinetv1alpha1.Network{}, apiNetCluster.GetCache()),
			&brokerhandler.EnqueueRequestForBrokerOwner{
				ClusterName:  r.ClusterName,
				OwnerType:    &networkingv1alpha1.Network{},
				IsController: true,
			},
		).
		Complete(r)
}