// Copyright 2023 OnMetal authors
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

	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	metalnetv1alpha1 "github.com/onmetal/metalnet/api/v1alpha1"
	"github.com/onmetal/onmetal-api-net/api/v1alpha1"
	"github.com/onmetal/onmetal-api-net/apiutils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	metalnetNetworkFieldOwner = client.FieldOwner("metalnetlet.apinet.onmetal.de/field-owner")
	networkNamespaceLabel     = "metalnetlet.apinet.onmetal.de/network-namespace"
	networkNameLabel          = "metalnetlet.apinet.onmetal.de/network-name"
	networkUIDLabel           = "metalnet.apinet.onmetal.de/network-uid"
)

func finalizer(name string) string {
	return fmt.Sprintf("%s.metalnetlet.apinet.onmetal.de/network", name)
}

func getNetworkVNI(network *v1alpha1.Network) (int32, bool) {
	if !apiutils.IsNetworkAllocated(network) {
		return 0, false
	}
	vni := network.Spec.VNI
	if vni == nil {
		return 0, false
	}
	return *vni, true
}

type NetworkReconciler struct {
	client.Client
	MetalnetCluster cluster.Cluster
	Name            string
}

//+kubebuilder:rbac:groups=apinet.api.onmetal.de,resources=networks,verbs=get;list;watch

func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	network := &v1alpha1.Network{}
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
	if err := r.MetalnetCluster.GetClient().DeleteAllOf(ctx, &metalnetv1alpha1.Network{},
		client.MatchingLabels{
			networkNamespaceLabel: networkKey.Namespace,
			networkNameLabel:      networkKey.Name,
		},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting metalnet networks matching network label: %w", err)
	}

	log.V(1).Info("Deleted gone")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) reconcileExists(ctx context.Context, log logr.Logger, network *v1alpha1.Network) (ctrl.Result, error) {
	if !network.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, network)
	}
	return r.reconcile(ctx, log, network)
}

func (r *NetworkReconciler) delete(ctx context.Context, log logr.Logger, network *v1alpha1.Network) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(network, finalizer(r.Name)) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Finalizer present, cleaning up")

	vni := *network.Spec.VNI

	log.V(1).Info("Deleting metalnet network if present")
	metalnetNetwork := &metalnetv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("network-%d", vni),
		},
	}
	err := r.MetalnetCluster.GetClient().Delete(ctx, metalnetNetwork)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("error deleting metalnet network: %w", err)
	}
	if err == nil {
		log.V(1).Info("Issued deletion of metalnet network")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Metalnet network is gone, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, network, finalizer(r.Name)); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}
	log.V(1).Info("Removed finalizer")

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) reconcile(ctx context.Context, log logr.Logger, network *v1alpha1.Network) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	vni, ok := getNetworkVNI(network)
	if !ok {
		log.V(1).Info("Network is not yet allocated")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, network, finalizer(r.Name))
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
			Name: fmt.Sprintf("network-%d", vni),
			Labels: map[string]string{
				networkNamespaceLabel: network.Namespace,
				networkNameLabel:      network.Name,
				networkUIDLabel:       string(network.UID),
			},
		},
		Spec: metalnetv1alpha1.NetworkSpec{
			ID:        vni,
			PeeredIDs: network.Spec.PeerVNIs,
		},
	}
	if err := r.MetalnetCluster.GetClient().Patch(ctx, metalnetNetwork, client.Apply, metalnetNetworkFieldOwner); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying network: %w", err)
	}
	log.V(1).Info("Applied metalnet network")

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkReconciler) enqueueNetworkUsingNetworkLabels() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []ctrl.Request {
		metalnetNetwork := obj.(*metalnetv1alpha1.Network)
		networkNamespace, networkName := metalnetNetwork.Labels[networkNamespaceLabel], metalnetNetwork.Labels[networkNameLabel]
		if networkNamespace == "" || networkName == "" {
			return nil
		}
		return []ctrl.Request{
			{
				NamespacedName: client.ObjectKey{
					Namespace: networkNamespace,
					Name:      networkName,
				},
			},
		}
	})
}

var networkHasVNI = predicate.NewPredicateFuncs(func(obj client.Object) bool {
	network := obj.(*v1alpha1.Network)
	_, ok := getNetworkVNI(network)
	return ok
})

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1alpha1.Network{},
			builder.WithPredicates(networkHasVNI),
		).
		Watches(
			source.NewKindWithCache(&metalnetv1alpha1.Network{}, r.MetalnetCluster.GetCache()),
			r.enqueueNetworkUsingNetworkLabels(),
		).
		Complete(r)
}
