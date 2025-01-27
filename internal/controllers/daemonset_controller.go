// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/controller-utils/metautils"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/internal/nodeaffinity"
	"github.com/ironcore-dev/ironcore-net/utils/controller"
	"github.com/ironcore-dev/ironcore-net/utils/expectations"
	utilhandler "github.com/ironcore-dev/ironcore-net/utils/handler"
	utilslices "github.com/ironcore-dev/ironcore/utils/slices"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type DaemonSetReconciler struct {
	client.Client
	Expectations *expectations.Expectations
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=daemonsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=instances,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=nodes,verbs=get;list;watch

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	ds := &v1alpha1.DaemonSet{}
	if err := r.Get(ctx, req.NamespacedName, ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcileExists(ctx, log, ds)
}

func (r *DaemonSetReconciler) reconcileExists(
	ctx context.Context,
	log logr.Logger,
	ds *v1alpha1.DaemonSet,
) (ctrl.Result, error) {
	if !ds.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, ds)
	}
	return r.reconcile(ctx, log, ds)
}

func (r *DaemonSetReconciler) delete(
	ctx context.Context,
	log logr.Logger,
	ds *v1alpha1.DaemonSet,
) (ctrl.Result, error) {
	log.V(1).Info("Delete")
	_, _ = ctx, ds
	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *DaemonSetReconciler) instanceNeedsUpdate(ds *v1alpha1.DaemonSet, inst *v1alpha1.Instance) bool {
	return !slices.Equal(inst.Spec.IPs, ds.Spec.Template.Spec.IPs)
}

func (r *DaemonSetReconciler) updateInstance(ctx context.Context, ds *v1alpha1.DaemonSet, inst *v1alpha1.Instance) error {
	base := inst.DeepCopy()
	inst.Spec.IPs = ds.Spec.Template.Spec.IPs
	return r.Patch(ctx, inst, client.StrategicMergeFrom(base))
}

func (r *DaemonSetReconciler) getDaemonInstances(ctx context.Context, ds *v1alpha1.DaemonSet) ([]*v1alpha1.Instance, error) {
	sel, err := metav1.LabelSelectorAsSelector(ds.Spec.Selector)
	if err != nil {
		return nil, err
	}

	instanceList := &v1alpha1.InstanceList{}
	if err := r.List(ctx, instanceList,
		client.InNamespace(ds.Namespace),
	); err != nil {
		return nil, err
	}

	var (
		claimMgr = controller.NewRefManager(r.Client, ds, controller.MatchLabelSelectorFunc[*v1alpha1.Instance](sel))
		insts    []*v1alpha1.Instance
		errs     []error
	)
	for i := range instanceList.Items {
		inst := &instanceList.Items[i]
		ok, err := claimMgr.ClaimObject(ctx, inst)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !ok {
			continue
		}
		// TODO: Determine whether this is the right place to do this.
		// Maybe an instance should only have a single IP.
		// Maybe the update should be done in a later step (e.g. where we decide which instance to delete / keep).
		if r.instanceNeedsUpdate(ds, inst) {
			if err := r.updateInstance(ctx, ds, inst); err != nil {
				errs = append(errs, err)
				continue
			}
		}

		insts = append(insts, inst)
	}
	return insts, errors.Join(errs...)
}

func (r *DaemonSetReconciler) getNodesToDaemonInstances(ctx context.Context, log logr.Logger, ds *v1alpha1.DaemonSet) (map[string][]*v1alpha1.Instance, error) {
	claimedInsts, err := r.getDaemonInstances(ctx, ds)
	if err != nil {
		return nil, err
	}

	nodeToDaemonInsts := make(map[string][]*v1alpha1.Instance)
	for _, inst := range claimedInsts {
		nodeName, err := GetTargetNodeName(inst)
		if err != nil {
			log.Info("Failed to get target node name of Instance in DaemonSet",
				"Instance", klog.KObj(inst),
			)
			continue
		}

		nodeToDaemonInsts[nodeName] = append(nodeToDaemonInsts[nodeName], inst)
	}

	return nodeToDaemonInsts, nil
}

func (r *DaemonSetReconciler) nodeShouldRunDaemonInstance(
	node *v1alpha1.Node,
	ds *v1alpha1.DaemonSet,
) bool {
	// If the daemon set specifies a node name, and it does not match with our node name bail out immediately.
	if nodeRef := ds.Spec.Template.Spec.NodeRef; nodeRef != nil && nodeRef.Name != node.Name {
		return false
	}

	inst := &v1alpha1.Instance{
		ObjectMeta: ds.Spec.Template.ObjectMeta,
		Spec:       ds.Spec.Template.Spec,
	}
	inst.Namespace = ds.Namespace
	inst.Spec.NodeRef = &corev1.LocalObjectReference{Name: node.Name}

	fitsNodeAffinity, _ := nodeaffinity.GetRequiredNodeAffinity(inst).Match(node)
	return fitsNodeAffinity
}

func (r *DaemonSetReconciler) instancesShouldBeOnNode(
	log logr.Logger,
	node *v1alpha1.Node,
	nodeToDaemonInsts map[string][]*v1alpha1.Instance,
	ds *v1alpha1.DaemonSet,
	hash string,
) (nodesNeedingDaemonInsts []string, instsToDelete []string) {
	_, _ = log, hash
	shouldRun := r.nodeShouldRunDaemonInstance(node, ds)
	insts, exists := nodeToDaemonInsts[node.Name]

	switch {
	case shouldRun && !exists:
		// If a daemon instance is supposed to be running on a node but isn't, create one.
		nodesNeedingDaemonInsts = append(nodesNeedingDaemonInsts, node.Name)
	case shouldRun:
		var filtered []*v1alpha1.Instance
		for _, inst := range insts {
			if !inst.DeletionTimestamp.IsZero() {
				continue
			}
			filtered = append(filtered, inst)
		}
		if len(filtered) == 0 {
			nodesNeedingDaemonInsts = append(nodesNeedingDaemonInsts, node.Name)
		} else if len(filtered) > 1 {
			// Delete any unnecessary instance, keeping the oldest ones.
			slices.SortFunc(filtered, func(a, b *v1alpha1.Instance) int {
				return a.CreationTimestamp.Compare(b.CreationTimestamp.Time)
			})
			for _, inst := range filtered[1:] {
				instsToDelete = append(instsToDelete, inst.Name)
			}
		}
	case !shouldRun:
		for _, inst := range insts {
			instsToDelete = append(instsToDelete, inst.Name)
		}
	}

	return nodesNeedingDaemonInsts, instsToDelete
}

func (r *DaemonSetReconciler) getUnscheduledInstancesWithoutNode(
	nodes []v1alpha1.Node,
	nodeToDaemonInsts map[string][]*v1alpha1.Instance,
) []string {
	var (
		res       []string
		nodeNames = utilslices.ToSetFunc(nodes, func(n v1alpha1.Node) string { return n.Name })
	)

	for nodeName, insts := range nodeToDaemonInsts {
		if nodeNames.Has(nodeName) {
			continue
		}

		for _, inst := range insts {
			if inst.Spec.NodeRef == nil {
				res = append(res, inst.Name)
			}
		}
	}
	return res
}

func (r *DaemonSetReconciler) manage(
	ctx context.Context,
	log logr.Logger,
	ds *v1alpha1.DaemonSet,
	nodes []v1alpha1.Node,
	hash string,
) error {
	nodeToDaemonInsts, err := r.getNodesToDaemonInstances(ctx, log, ds)
	if err != nil {
		return fmt.Errorf("error getting node to daemon instance mapping: %w", err)
	}

	var (
		nodesNeedingDaemonInsts []string
		instsToDelete           []string
	)
	for i := range nodes {
		node := &nodes[i]
		nodeDaemonInsts, nodeInstsToDelete := r.instancesShouldBeOnNode(log, node, nodeToDaemonInsts, ds, hash)

		nodesNeedingDaemonInsts = append(nodesNeedingDaemonInsts, nodeDaemonInsts...)
		instsToDelete = append(instsToDelete, nodeInstsToDelete...)
	}

	instsToDelete = append(instsToDelete, r.getUnscheduledInstancesWithoutNode(nodes, nodeToDaemonInsts)...)

	if err := r.syncNodes(ctx, log, ds, instsToDelete, nodesNeedingDaemonInsts, hash); err != nil {
		return err
	}

	return nil
}

func (r *DaemonSetReconciler) createInstance(
	ctx context.Context,
	ds *v1alpha1.DaemonSet,
	nodeName string,
	instName string,
	hash string,
) (*v1alpha1.Instance, error) {
	templ := ds.Spec.Template.DeepCopy()
	inst := &v1alpha1.Instance{
		ObjectMeta: templ.ObjectMeta,
		Spec:       templ.Spec,
	}
	inst.Namespace = ds.Namespace
	inst.Name = instName
	metautils.SetLabel(inst, v1alpha1.ControllerRevisionHashLabel, hash)
	inst.Spec.Affinity = ReplaceDaemonSetInstanceNodeNameNodeAffinity(inst.Spec.Affinity, nodeName)
	if err := ctrl.SetControllerReference(ds, inst, r.Scheme()); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, inst); err != nil {
		return nil, err
	}
	return inst, nil
}

func (r *DaemonSetReconciler) syncNodes(
	ctx context.Context,
	log logr.Logger,
	ds *v1alpha1.DaemonSet,
	instsToDelete []string,
	nodesNeedingDaemonInsts []string,
	hash string,
) error {
	var (
		ctrlKey     = client.ObjectKeyFromObject(ds)
		createNames = expectations.GenerateCreateNames(ds.Name, len(nodesNeedingDaemonInsts))
	)
	r.Expectations.ExpectCreationsAndDeletions(ctrlKey,
		expectations.ObjectKeysFromNames(ds.Namespace, createNames),
		expectations.ObjectKeysFromNames(ds.Namespace, instsToDelete),
	)
	log.V(1).Info("Expecting creations / deletions",
		"createNames", createNames,
		"deleteInstances", instsToDelete,
	)

	var errs []error

	for i, createName := range createNames {
		nodeName := nodesNeedingDaemonInsts[i]
		if _, err := r.createInstance(ctx, ds, nodeName, createName, hash); err != nil {
			r.Expectations.CreationObserved(ctrlKey, client.ObjectKey{Namespace: ds.Namespace, Name: createName})
			errs = append(errs, err)
		}
	}

	for _, deleteName := range instsToDelete {
		inst := &v1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ds.Namespace,
				Name:      deleteName,
			},
		}
		instKey := client.ObjectKeyFromObject(inst)
		if err := r.Delete(ctx, inst); err != nil {
			r.Expectations.DeletionObserved(ctrlKey, instKey)
			if !apierrors.IsNotFound(err) {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

func (r *DaemonSetReconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	ds *v1alpha1.DaemonSet,
) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	nodeList := &v1alpha1.NodeList{}
	if err := r.List(ctx, nodeList); err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing nodes: %w", err)
	}

	hash := ComputeHash(&ds.Spec.Template, ds.Status.CollisionCount)

	if r.Expectations.Satisfied(client.ObjectKeyFromObject(ds)) {
		log.V(1).Info("Managing daemon set")
		if err := r.manage(ctx, log, ds, nodeList.Items, hash); err != nil {
			return ctrl.Result{}, fmt.Errorf("error managing daemon set: %w", err)
		}
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *DaemonSetReconciler) enqueueByNode() handler.EventHandler {
	enqueueAllDaemonSets := func(ctx context.Context, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
		log := ctrl.LoggerFrom(ctx)
		dsList := &v1alpha1.DaemonSetList{}

		if err := r.List(ctx, dsList); err != nil {
			log.Error(err, "Error listing daemon sets")
			return
		}

		for _, ds := range dsList.Items {
			queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&ds)})
		}
	}

	return handler.Funcs{
		CreateFunc: func(ctx context.Context, evt event.CreateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			enqueueAllDaemonSets(ctx, queue)
		},
		DeleteFunc: func(ctx context.Context, evt event.DeleteEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			enqueueAllDaemonSets(ctx, queue)
		},
	}
}

func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DaemonSet{}).
		Owns(&v1alpha1.Instance{}).
		Watches(
			&v1alpha1.Instance{},
			utilhandler.ObserveExpectationsForController(r.Scheme(), r.RESTMapper(), &v1alpha1.DaemonSet{}, r.Expectations),
		).
		Watches(
			&v1alpha1.Node{},
			r.enqueueByNode(),
		).
		Complete(r)
}
