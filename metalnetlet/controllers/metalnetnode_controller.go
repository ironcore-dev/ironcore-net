// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/controller-utils/clientutils"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/maps"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const metalnetNodeFinalizer = "apinet.ironcore.dev/node"

type MetalnetNodeReconciler struct {
	client.Client
	MetalnetClient client.Client
	PartitionName  string
	NodeLabels     map[string]string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=nodes,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=nodes/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=nodes/status,verbs=get;update;patch

//+cluster=metalnet:kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update;patch
//+cluster=metalnet:kubebuilder:rbac:groups="",resources=nodes/finalizers,verbs=update;patch

// Reconcile ensures that the node is in the expected state.
func (r *MetalnetNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	metalnetNode := &corev1.Node{}
	if err := r.MetalnetClient.Get(ctx, req.NamespacedName, metalnetNode); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting metalnet node: %w", err)
		}

		log.V(1).Info("Metalnet node not found, deleting any corresponding node")
		node := &v1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: PartitionNodeName(r.PartitionName, metalnetNode.Name),
			},
		}
		if err := r.Delete(ctx, node); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("error deleting node: %w", err)
		}
		log.V(1).Info("Deleted corresponding node")
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, metalnetNode)
}

func (r *MetalnetNodeReconciler) reconcileExists(ctx context.Context, log logr.Logger, metalnetNode *corev1.Node) (ctrl.Result, error) {
	if !metalnetNode.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, metalnetNode)
	}
	return r.reconcile(ctx, log, metalnetNode)
}

func (r *MetalnetNodeReconciler) delete(ctx context.Context, log logr.Logger, metalnetNode *corev1.Node) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(metalnetNode, metalnetNodeFinalizer) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Deleting any corresponding node")
	node := &v1alpha1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: PartitionNodeName(r.PartitionName, metalnetNode.Name),
		},
	}
	if err := r.Delete(ctx, node); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting node: %w", err)
		}

		log.V(1).Info("Any corresponding node is gone, removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.MetalnetClient, metalnetNode, metalnetNodeFinalizer); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Deleted")
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Issued node deletion, requeuing")
	return ctrl.Result{Requeue: true}, nil
}

func (r *MetalnetNodeReconciler) reconcile(ctx context.Context, log logr.Logger, metalnetNode *corev1.Node) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.MetalnetClient, metalnetNode, metalnetNodeFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Finalizer is present")

	log.V(1).Info("Applying node")
	node := &v1alpha1.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "Node",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: PartitionNodeName(r.PartitionName, metalnetNode.Name),
			Labels: maps.AppendMap(map[string]string{
				v1alpha1.TopologyPartitionLabel: r.PartitionName,
			}, r.NodeLabels),
		},
	}
	if err := r.Patch(ctx, node, client.Apply, PartitionFieldOwner(r.PartitionName), client.ForceOwnership); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying node: %w", err)
	}
	log.V(1).Info("Applied node")

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *MetalnetNodeReconciler) enqueueByMetalnetNode() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		node := obj.(*corev1.Node)
		return []ctrl.Request{{NamespacedName: client.ObjectKey{Name: node.Name}}}
	})
}

func (r *MetalnetNodeReconciler) logConstructor(mgr ctrl.Manager) func(req *ctrl.Request) logr.Logger {
	log := mgr.GetLogger().WithValues(
		"controller", "node",
		"controllerGroup", metalnetv1alpha1.GroupVersion.Group,
		"controllerKind", "Node",
	)

	return func(req *ctrl.Request) logr.Logger {
		log := log
		if req != nil {
			nodeName := PartitionNodeName(r.PartitionName, req.Name)
			log = log.WithValues(
				"Node", req.Name,
				"APINetNode", klog.KRef("", nodeName),
				"namespace", "", "name", req.Name,
			)
		}
		return log
	}
}

func (r *MetalnetNodeReconciler) enqueueByNode() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		node := obj.(*v1alpha1.Node)
		log := ctrl.LoggerFrom(ctx)

		metalnetNodeName, err := ParseNodeName(r.PartitionName, node.Name)
		if err != nil {
			log.Error(err, "Error parsing node name: %w", err)
			return nil
		}

		return []ctrl.Request{{NamespacedName: client.ObjectKey{Name: metalnetNodeName}}}
	})
}

func (r *MetalnetNodeReconciler) SetupWithManager(mgr ctrl.Manager, metalnetCache cache.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("metalnetnode").
		WithLogConstructor(r.logConstructor(mgr)).
		WatchesRawSource(
			source.Kind[client.Object](
				metalnetCache,
				&corev1.Node{},
				r.enqueueByMetalnetNode(),
			),
		).
		Watches(
			&v1alpha1.Node{},
			r.enqueueByNode(),
			builder.WithPredicates(IsNodeOnPartitionPredicate(r.PartitionName)),
		).
		Complete(r)
}
