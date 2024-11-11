// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/controller-utils/clientutils"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	metalnetletclient "github.com/ironcore-dev/ironcore-net/metalnetlet/client"
	utilhandler "github.com/ironcore-dev/ironcore-net/metalnetlet/handler"
	"github.com/ironcore-dev/ironcore/utils/generic"
	metalnetv1alpha1 "github.com/ironcore-dev/metalnet/api/v1alpha1"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type InstanceReconciler struct {
	client.Client
	MetalnetClient client.Client

	PartitionName string

	MetalnetNamespace string
}

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=instances,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=instances/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=instances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networks,verbs=get;list;watch

//+cluster=metalnet:kubebuilder:rbac:groups=networking.metalnet.ironcore.dev,resources=loadbalancers,verbs=get;list;watch;create;update;patch;delete;deletecollection

func (r *InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	instance := &v1alpha1.Instance{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		log.V(1).Info("Deleting any leftover metalnet load balancers")
		anyExists, err := r.deleteMetalnetLoadBalancersByLoadBalancerInstanceKeyAndAnyExists(ctx, req.NamespacedName)
		if err != nil {
			return ctrl.Result{}, err
		}
		if anyExists {
			log.V(1).Info("Some metalnet load balancers still exist, requeueing")
			return ctrl.Result{Requeue: true}, nil
		}
		log.V(1).Info("Any potential leftover metalnet load balancer is gone")
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, instance)
}

func (r *InstanceReconciler) reconcileExists(ctx context.Context, log logr.Logger, loadBalancerInstance *v1alpha1.Instance) (ctrl.Result, error) {
	if !loadBalancerInstance.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, loadBalancerInstance)
	}
	return r.reconcile(ctx, log, loadBalancerInstance)
}

func (r *InstanceReconciler) deleteMetalnetLoadBalancersByLoadBalancerInstanceKeyAndAnyExists(ctx context.Context, key client.ObjectKey) (bool, error) {
	return metalnetletclient.DeleteAllOfAndAnyExists(ctx, r.MetalnetClient, &metalnetv1alpha1.LoadBalancer{},
		client.InNamespace(r.MetalnetNamespace),
		metalnetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), key, &v1alpha1.Instance{}),
	)
}

func (r *InstanceReconciler) deleteMetalnetLoadBalancersByLoadBalancerInstanceAndAnyExists(ctx context.Context, inst *v1alpha1.Instance) (bool, error) {
	return metalnetletclient.DeleteAllOfAndAnyExists(ctx, r.MetalnetClient, &metalnetv1alpha1.LoadBalancer{},
		client.InNamespace(r.MetalnetNamespace),
		metalnetletclient.MatchingSourceLabels(r.Scheme(), r.RESTMapper(), inst),
	)
}

func (r *InstanceReconciler) delete(ctx context.Context, log logr.Logger, loadBalancerInstance *v1alpha1.Instance) (ctrl.Result, error) {
	log.V(1).Info("Delete")
	if !controllerutil.ContainsFinalizer(loadBalancerInstance, PartitionFinalizer(r.PartitionName)) {
		log.V(1).Info("Finalizer not present, nothing to do")
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Finalizer present, doing cleanup")
	anyExists, err := r.deleteMetalnetLoadBalancersByLoadBalancerInstanceAndAnyExists(ctx, loadBalancerInstance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if anyExists {
		log.V(1).Info("Some metalnet load balancers are still present, requeuing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("All metalnet load balancers gone, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, loadBalancerInstance, PartitionFinalizer(r.PartitionName)); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *InstanceReconciler) getMetalnetLoadBalancersForLoadBalancerInstance(
	ctx context.Context,
	inst *v1alpha1.Instance,
) ([]metalnetv1alpha1.LoadBalancer, error) {
	metalnetLoadBalancerList := &metalnetv1alpha1.LoadBalancerList{}
	if err := r.MetalnetClient.List(ctx, metalnetLoadBalancerList,
		client.InNamespace(r.MetalnetNamespace),
		metalnetletclient.MatchingSourceLabels(r.Scheme(), r.RESTMapper(), inst),
	); err != nil {
		return nil, fmt.Errorf("error listing metalnet load balancer instances: %w", err)
	}
	return metalnetLoadBalancerList.Items, nil
}

func (r *InstanceReconciler) manageMetalnetLoadBalancers(
	ctx context.Context,
	log logr.Logger,
	inst *v1alpha1.Instance,
	metalnetNodeName string,
) (bool, error) {
	network := &v1alpha1.Network{}
	networkKey := client.ObjectKey{Namespace: inst.Namespace, Name: inst.Spec.NetworkRef.Name}
	if err := r.Get(ctx, networkKey, network); err != nil {
		return false, client.IgnoreNotFound(err)
	}

	metalnetLoadBalancerType, err := loadBalancerTypeToMetalnetLoadBalancerType(inst.Spec.LoadBalancerType)
	if err != nil {
		return false, err
	}

	metalnetLoadBalancers, err := r.getMetalnetLoadBalancersForLoadBalancerInstance(ctx, inst)
	if err != nil {
		return false, err
	}

	var (
		unsatisfiedIPs = sets.New(inst.Spec.IPs...)
		errs           []error
	)
	for _, metalnetLoadBalancer := range metalnetLoadBalancers {
		if !metalnetLoadBalancer.DeletionTimestamp.IsZero() {
			continue
		}

		ip := metalnetIPToIP(metalnetLoadBalancer.Spec.IP)
		if unsatisfiedIPs.Has(ip) {
			unsatisfiedIPs.Delete(ip)
			continue
		}

		if err := r.MetalnetClient.Delete(ctx, &metalnetLoadBalancer); client.IgnoreNotFound(err) != nil {
			errs = append(errs, err)
			continue
		}
	}

	var bumpCollisionCount bool
	for ip := range unsatisfiedIPs {
		metalnetLoadBalancerHash := computeMetalnetLoadBalancerHash(ip, inst.Status.CollisionCount)
		metalnetLoadBalancerSpec := metalnetv1alpha1.LoadBalancerSpec{
			NetworkRef: corev1.LocalObjectReference{Name: string(network.UID)},
			LBtype:     metalnetLoadBalancerType,
			IPFamily:   ip.Family(),
			IP:         ipToMetalnetIP(ip),
			Ports:      loadBalancerPortsToMetalnetLoadBalancerPorts(inst.Spec.LoadBalancerPorts),
			NodeName:   &metalnetNodeName,
		}
		metalnetLoadBalancerName := string(inst.UID) + "-" + metalnetLoadBalancerHash
		metalnetLoadBalancer := &metalnetv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: r.MetalnetNamespace,
				Name:      metalnetLoadBalancerName,
				Labels:    metalnetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), inst),
			},
			Spec: metalnetLoadBalancerSpec,
		}
		createMetalnetLoadBalancer := metalnetLoadBalancer.DeepCopy()
		if err := r.MetalnetClient.Create(ctx, metalnetLoadBalancer); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				errs = append(errs, err)
				continue
			}

			// We may end up hitting this due to a slow cache or a fast resync of the load balancer instance.
			metalnetLoadBalancerKey := client.ObjectKey{Namespace: r.MetalnetNamespace, Name: metalnetLoadBalancerName}
			if err := r.MetalnetClient.Get(ctx, metalnetLoadBalancerKey, metalnetLoadBalancer); err != nil {
				errs = append(errs, err)
				continue
			}

			if metalnetLoadBalancer.DeletionTimestamp.IsZero() &&
				EqualMetalnetLoadBalancers(createMetalnetLoadBalancer, metalnetLoadBalancer) {
				continue
			}

			// Issue collision count bump and return original already exists error.
			bumpCollisionCount = true
			errs = append(errs, err)
		}
	}
	if bumpCollisionCount {
		if err := r.bumpLoadBalancerInstanceCollisionCount(ctx, inst); err != nil {
			log.Error(err, "Error bumping collision count")
		} else {
			log.V(1).Info("Bumped collision count")
		}
	}
	return true, errors.Join(errs...)
}

func EqualMetalnetLoadBalancers(inst1, inst2 *metalnetv1alpha1.LoadBalancer) bool {
	return inst1.Spec.IP == inst2.Spec.IP &&
		inst1.Spec.IPFamily == inst2.Spec.IPFamily &&
		inst1.Spec.NetworkRef == inst2.Spec.NetworkRef &&
		inst1.Spec.LBtype == inst2.Spec.LBtype &&
		slices.Equal(inst1.Spec.Ports, inst2.Spec.Ports)
}

func (r *InstanceReconciler) bumpLoadBalancerInstanceCollisionCount(ctx context.Context, loadBalancerInstance *v1alpha1.Instance) error {
	oldCollisionCount := generic.Deref(loadBalancerInstance.Status.CollisionCount, 0)
	base := loadBalancerInstance.DeepCopy()
	loadBalancerInstance.Status.CollisionCount = generic.Pointer(oldCollisionCount + 1)
	return r.Status().Patch(ctx, loadBalancerInstance, client.MergeFrom(base))
}

func computeMetalnetLoadBalancerHash(ip net.IP, collisionCount *int32) string {
	h := fnv.New32a()

	_, _ = h.Write(ip.AsSlice())

	// Add collisionCount in the hash if it exists
	if collisionCount != nil {
		collisionCountBytes := make([]byte, 8)
		binary.LittleEndian.PutUint32(collisionCountBytes, uint32(*collisionCount))
		_, _ = h.Write(collisionCountBytes)
	}

	return rand.SafeEncodeString(fmt.Sprint(h.Sum32()))
}

func (r *InstanceReconciler) reconcile(ctx context.Context, log logr.Logger, inst *v1alpha1.Instance) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	metalnetNode, err := GetMetalnetNode(ctx, r.PartitionName, r.MetalnetClient, inst.Spec.NodeRef.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if metalnetNode == nil || !metalnetNode.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(inst, PartitionFinalizer(r.PartitionName)) {
			log.V(1).Info("Finalizer not present and metalnet node not found / deleting, nothing to do")
			return ctrl.Result{}, nil
		}

		anyExists, err := r.deleteMetalnetLoadBalancersByLoadBalancerInstanceAndAnyExists(ctx, inst)
		if err != nil {
			return ctrl.Result{}, err
		}
		if anyExists {
			log.V(1).Info("Not yet all metalnet load balancers gone, requeueing")
			return ctrl.Result{Requeue: true}, nil
		}

		log.V(1).Info("All metalnet load balancers gone, removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, inst, PartitionFinalizer(r.PartitionName)); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
		}
		log.V(1).Info("Removed finalizer")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Metalnet node present and not deleting, ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, inst, PartitionFinalizer(r.PartitionName))
	if err != nil {
		return ctrl.Result{}, err
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Finalizer present")

	ok, err := r.manageMetalnetLoadBalancers(ctx, log, inst, metalnetNode.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error managing metalnet load balancers: %w", err)
	}
	if !ok {
		log.V(1).Info("Not all load balancer instance dependencies are ready")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *InstanceReconciler) isPartitionLoadBalancerInstance() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		loadBalancerInstance := obj.(*v1alpha1.Instance)
		nodeRef := loadBalancerInstance.Spec.NodeRef
		if nodeRef == nil {
			return false
		}

		_, err := ParseNodeName(r.PartitionName, nodeRef.Name)
		return err == nil
	})
}

func (r *InstanceReconciler) SetupWithManager(mgr ctrl.Manager, metalnetCache cache.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1alpha1.Instance{},
			builder.WithPredicates(r.isPartitionLoadBalancerInstance()),
		).
		WatchesRawSource(
			source.Kind[client.Object](
				metalnetCache,
				&metalnetv1alpha1.LoadBalancer{},
				utilhandler.EnqueueRequestForSource(r.Scheme(), r.RESTMapper(), &v1alpha1.Instance{}),
			),
		).
		Complete(r)
}
