// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/internal/controllers/scheduler"
	"github.com/ironcore-dev/ironcore-net/internal/nodeaffinity"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	outOfCapacity = "OutOfCapacity"
)

type SchedulerReconciler struct {
	client.Client
	record.EventRecorder
	Cache *scheduler.Cache

	snapshot *scheduler.Snapshot
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=instances,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=nodes,verbs=get;list;watch

func (r *SchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	instance := &v1alpha1.Instance{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if r.skipSchedule(log, instance) {
		log.V(1).Info("Skipping scheduling for instance")
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, instance)
}

func (r *SchedulerReconciler) skipSchedule(log logr.Logger, instance *v1alpha1.Instance) bool {
	if !instance.DeletionTimestamp.IsZero() {
		return true
	}

	isAssumed, err := r.Cache.IsAssumedInstance(instance)
	if err != nil {
		log.Error(err, "Error checking whether instance has been assumed")
		return false
	}
	return isAssumed
}

func (r *SchedulerReconciler) updateSnapshot() {
	if r.snapshot == nil {
		r.snapshot = r.Cache.Snapshot()
	} else {
		r.snapshot.Update()
	}
}

func (r *SchedulerReconciler) filterNodesByAffinity(
	log logr.Logger,
	inst *v1alpha1.Instance,
	nodes []*scheduler.ContainerInfo,
) ([]*scheduler.ContainerInfo, error) {
	if inst.Spec.Affinity == nil || inst.Spec.Affinity.NodeAffinity == nil {
		// Short circuit if no affinity is specified.
		return nodes, nil
	}

	nodeAffinity := nodeaffinity.GetRequiredNodeAffinity(inst)

	var filtered []*scheduler.ContainerInfo
	for _, node := range nodes {
		ok, err := nodeAffinity.Match(node.Node())
		if err != nil {
			log.Info("Node affinity match error", "Error", err)
			continue
		}
		if !ok {
			continue
		}

		filtered = append(filtered, node)
	}
	return filtered, nil
}

func (r *SchedulerReconciler) getExistingAntiAffinityCounts(inst *v1alpha1.Instance, nodes []*scheduler.ContainerInfo) (map[topologyPair]int, error) {
	tpCount := make(map[topologyPair]int)
	for _, n := range nodes {
		node := n.Node()

		for _, i := range n.Instances() {
			existingInst := i.Instance()
			if existingInst.Namespace != inst.Namespace {
				// Don't include instances from different namespaces.
				continue
			}

			if existingInst.Spec.Affinity == nil || existingInst.Spec.Affinity.InstanceAntiAffinity == nil {
				// Don't include instances that have no instance anti-affinity.
				continue
			}

			antiAffinity := existingInst.Spec.Affinity.InstanceAntiAffinity
			for _, term := range antiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				sel, err := metav1.LabelSelectorAsSelector(term.LabelSelector)
				if err != nil {
					return nil, err
				}

				if sel.Matches(labels.Set(inst.Labels)) {
					tpValue, ok := node.Labels[term.TopologyKey]
					if !ok {
						continue
					}

					tpCount[topologyPair{term.TopologyKey, tpValue}] += 1
				}
			}
		}
	}
	return tpCount, nil
}

func (r *SchedulerReconciler) getIncomingAntiAffinityCounts(inst *v1alpha1.Instance, nodes []*scheduler.ContainerInfo) (map[topologyPair]int, error) {
	tpCount := make(map[topologyPair]int)
	if inst.Spec.Affinity == nil || inst.Spec.Affinity.InstanceAntiAffinity == nil {
		return tpCount, nil
	}

	antiAffinity := inst.Spec.Affinity.InstanceAntiAffinity

	for _, n := range nodes {
		node := n.Node()

		for _, i := range n.Instances() {
			existingInst := i.Instance()
			if existingInst.Namespace != inst.Namespace {
				// Don't include instances from different namespaces.
				continue
			}

			for _, term := range antiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				sel, err := metav1.LabelSelectorAsSelector(term.LabelSelector)
				if err != nil {
					return nil, err
				}

				if sel.Matches(labels.Set(existingInst.Labels)) {
					tpValue, ok := node.Labels[term.TopologyKey]
					if !ok {
						continue
					}

					tpCount[topologyPair{term.TopologyKey, tpValue}] += 1
				}
			}
		}
	}
	return tpCount, nil
}

func (r *SchedulerReconciler) filterNodesByInstanceAntiAffinity(
	log logr.Logger,
	inst *v1alpha1.Instance,
	nodes []*scheduler.ContainerInfo,
) ([]*scheduler.ContainerInfo, error) {
	existingAntiAffinityCounts, err := r.getExistingAntiAffinityCounts(inst, nodes)
	if err != nil {
		return nil, err
	}

	incomingAntiAffinityCounts, err := r.getIncomingAntiAffinityCounts(inst, nodes)
	if err != nil {
		return nil, err
	}

	var filtered []*scheduler.ContainerInfo
	for _, n := range nodes {
		if !satisfyInstanceAntiAffinity(inst, incomingAntiAffinityCounts, n) {
			continue
		}

		if !satisfyExistingInstanceAntiAffinity(existingAntiAffinityCounts, n) {
			continue
		}

		filtered = append(filtered, n)
	}
	return filtered, nil
}

func satisfyExistingInstanceAntiAffinity(
	existingAntiAffinityCounts map[topologyPair]int,
	nodeInfo *scheduler.ContainerInfo,
) bool {
	if len(existingAntiAffinityCounts) == 0 {
		return true
	}

	for topologyKey, topologyValue := range nodeInfo.Node().Labels {
		tp := topologyPair{key: topologyKey, value: topologyValue}
		if existingAntiAffinityCounts[tp] > 0 {
			return false
		}
	}
	return true
}

func satisfyInstanceAntiAffinity(
	inst *v1alpha1.Instance,
	antiAffinityCounts map[topologyPair]int,
	nodeInfo *scheduler.ContainerInfo,
) bool {
	if len(antiAffinityCounts) == 0 {
		return true
	}

	if inst.Spec.Affinity == nil || inst.Spec.Affinity.InstanceAntiAffinity == nil {
		return true
	}

	for _, term := range inst.Spec.Affinity.InstanceAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
		if topologyValue, ok := nodeInfo.Node().Labels[term.TopologyKey]; ok {
			tp := topologyPair{key: term.TopologyKey, value: topologyValue}
			if antiAffinityCounts[tp] > 0 {
				return false
			}
		}
	}
	return true
}

// topologySpreadConstraint is an internal version for v1.TopologySpreadConstraint
// and where the selector is parsed.
type topologySpreadConstraint struct {
	MaxSkew     int32
	TopologyKey string
	Selector    labels.Selector
	MinDomains  int32
}

// nodeLabelsMatchSpreadConstraints checks if ALL topology keys in spread Constraints are present in node labels.
func nodeLabelsMatchSpreadConstraints(nodeLabels map[string]string, constraints []topologySpreadConstraint) bool {
	for _, c := range constraints {
		if _, ok := nodeLabels[c.TopologyKey]; !ok {
			return false
		}
	}
	return true
}

func buildTopologySpreadConstraint(constraint *v1alpha1.TopologySpreadConstraint) (*topologySpreadConstraint, error) {
	sel, err := metav1.LabelSelectorAsSelector(constraint.LabelSelector)
	if err != nil {
		return nil, err
	}

	return &topologySpreadConstraint{
		MaxSkew:     constraint.MaxSkew,
		TopologyKey: constraint.TopologyKey,
		Selector:    sel,
		MinDomains:  1,
	}, nil
}

func buildTopologySpreadConstraints(constraints []v1alpha1.TopologySpreadConstraint) ([]topologySpreadConstraint, error) {
	var res []topologySpreadConstraint
	for i := range constraints {
		constraint := &constraints[i]
		c, err := buildTopologySpreadConstraint(constraint)
		if err != nil {
			return nil, err
		}

		res = append(res, *c)
	}
	return res, nil
}

type topologyPair struct {
	key   string
	value string
}

type criticalPath struct {
	value string
	count int
}

func (r *SchedulerReconciler) filterNodesByTopology(
	log logr.Logger,
	inst *v1alpha1.Instance,
	nodes []*scheduler.ContainerInfo,
) ([]*scheduler.ContainerInfo, error) {
	_ = log
	if inst.Spec.TopologySpreadConstraints == nil {
		// Short circuit if no topology spread constraints are specified.
		return nodes, nil
	}

	constraints, err := buildTopologySpreadConstraints(inst.Spec.TopologySpreadConstraints)
	if err != nil {
		return nil, err
	}

	var (
		tpCounts = make(map[topologyPair]int)
		tpNodes  []*scheduler.ContainerInfo
	)
	for _, n := range nodes {
		node := n.Node()
		if !nodeLabelsMatchSpreadConstraints(node.Labels, constraints) {
			continue
		}

		tpNodes = append(tpNodes, n)
		for _, c := range constraints {
			pair := topologyPair{key: c.TopologyKey, value: node.Labels[c.TopologyKey]}
			count := countInstancesMatchSelector(n.Instances(), c.Selector, inst.Namespace)
			tpCounts[pair] += count
		}
	}

	var (
		tpKeyToCriticalPath = make(map[string]criticalPath)
	)
	for pair, count := range tpCounts {
		cur, ok := tpKeyToCriticalPath[pair.key]
		if ok && cur.count < count {
			continue
		}

		tpKeyToCriticalPath[pair.key] = criticalPath{
			value: pair.value,
			count: count,
		}
	}

	var filtered []*scheduler.ContainerInfo
	for _, n := range tpNodes {
		node := n.Node()

		ok := true
		for _, c := range constraints {
			tpKey := c.TopologyKey
			tpVal := node.Labels[tpKey]

			minCount := tpKeyToCriticalPath[tpKey].count
			selfCount := 0
			if c.Selector.Matches(labels.Set(inst.Labels)) {
				selfCount = 1
			}

			pair := topologyPair{key: tpKey, value: tpVal}
			matchCount := 0
			if tpCount, ok := tpCounts[pair]; ok {
				matchCount = tpCount
			}

			skew := matchCount + selfCount - minCount
			if skew > int(c.MaxSkew) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		filtered = append(filtered, n)
	}
	return filtered, nil
}

func countInstancesMatchSelector(instInfos []*scheduler.InstanceInfo, selector labels.Selector, namespace string) int {
	if selector.Empty() {
		return 0
	}
	count := 0
	for _, i := range instInfos {
		if !i.Instance().DeletionTimestamp.IsZero() || i.Instance().Namespace != namespace {
			continue
		}
		if selector.Matches(labels.Set(i.Instance().Labels)) {
			count++
		}
	}
	return count
}

func (r *SchedulerReconciler) getNodesForInstance(
	ctx context.Context,
	log logr.Logger,
	inst *v1alpha1.Instance,
) ([]*scheduler.ContainerInfo, error) {
	_ = ctx
	nodes := r.snapshot.ListNodes()

	var instCt int
	for _, node := range nodes {
		for _, i := range node.Instances() {
			if i.Instance().Namespace != inst.Namespace {
				continue
			}
			instCt++
		}
	}

	filters := []func(logr.Logger, *v1alpha1.Instance, []*scheduler.ContainerInfo) ([]*scheduler.ContainerInfo, error){
		r.filterNodesByAffinity,
		r.filterNodesByInstanceAntiAffinity,
		r.filterNodesByTopology,
	}

	// Initialize matching nodes with all available nodes.
	matchingNodes := sets.New(nodes...)
	for _, filter := range filters {
		res, err := filter(log, inst, nodes)
		if err != nil {
			return nil, err
		}

		// Intersect with the intermediate result to see what nodes are
		// still matching.
		matchingNodes = matchingNodes.Intersection(sets.New(res...))
		if matchingNodes.Len() == 0 {
			// Short circuit in case no node matches.
			return nil, nil
		}
	}

	return matchingNodes.UnsortedList(), nil
}

func (r *SchedulerReconciler) reconcileExists(
	ctx context.Context,
	log logr.Logger,
	inst *v1alpha1.Instance,
) (ctrl.Result, error) {
	r.updateSnapshot()

	nodes, err := r.getNodesForInstance(ctx, log, inst)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting nodes for instance: %w", err)
	}
	if len(nodes) == 0 {
		r.EventRecorder.Event(inst, corev1.EventTypeNormal, outOfCapacity, "No nodes available to schedule instance on")
		return ctrl.Result{}, nil
	}

	minUsedNode := nodes[0]
	for _, node := range nodes[1:] {
		if node.NumInstances() < minUsedNode.NumInstances() {
			minUsedNode = node
		}
	}
	log.Info("Determined node to schedule on",
		"NodeName", minUsedNode.Node().Name,
		"Usage", minUsedNode.NumInstances(),
	)

	log.V(1).Info("Assuming instance to be on node")
	if err := r.assume(inst, minUsedNode.Node().Name); err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("Running binding asynchronously")
	go func() {
		if err := r.bindingCycle(ctx, log, inst); err != nil {
			if err := r.Cache.ForgetInstance(inst); err != nil {
				log.Error(err, "Error forgetting instance")
			}
		}
	}()
	return ctrl.Result{}, nil
}

func (r *SchedulerReconciler) assume(assumed *v1alpha1.Instance, nodeName string) error {
	assumed.Spec.NodeRef = &corev1.LocalObjectReference{Name: nodeName}
	if err := r.Cache.AssumeInstance(assumed.DeepCopy()); err != nil {
		return err
	}
	return nil
}

func (r *SchedulerReconciler) bindingCycle(ctx context.Context, log logr.Logger, assumedInstance *v1alpha1.Instance) error {
	if err := r.bind(ctx, log, assumedInstance); err != nil {
		return fmt.Errorf("error binding: %w", err)
	}
	return nil
}

func (r *SchedulerReconciler) bind(ctx context.Context, log logr.Logger, assumed *v1alpha1.Instance) error {
	defer func() {
		if err := r.Cache.FinishBinding(assumed); err != nil {
			log.Error(err, "Error finishing cache binding")
		}
	}()

	nonAssumed := assumed.DeepCopy()
	nonAssumed.Spec.NodeRef = nil

	if err := r.Patch(ctx, assumed, client.MergeFrom(nonAssumed)); err != nil {
		return fmt.Errorf("error patching instance: %w", err)
	}
	return nil
}

func (r *SchedulerReconciler) instanceNotAssignedPredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		instance := obj.(*v1alpha1.Instance)
		return instance.Spec.NodeRef == nil
	})
}

func (r *SchedulerReconciler) handleNode() handler.EventHandler {
	return handler.Funcs{
		CreateFunc: func(ctx context.Context, evt event.CreateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			node := evt.Object.(*v1alpha1.Node)
			log := ctrl.LoggerFrom(ctx)

			r.Cache.AddContainer(node)

			// TODO: Setup an index for listing unscheduled load balancer instances for the target partition.
			instanceList := &v1alpha1.InstanceList{}
			if err := r.List(ctx, instanceList); err != nil {
				log.Error(err, "Error listing load balancer instances")
				return
			}

			for _, instance := range instanceList.Items {
				if !instance.DeletionTimestamp.IsZero() {
					continue
				}
				if instance.Spec.NodeRef != nil {
					continue
				}

				queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&instance)})
			}
		},
		UpdateFunc: func(ctx context.Context, evt event.UpdateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			oldNode := evt.ObjectOld.(*v1alpha1.Node)
			newNode := evt.ObjectNew.(*v1alpha1.Node)
			r.Cache.UpdateContainer(oldNode, newNode)
		},
		DeleteFunc: func(ctx context.Context, evt event.DeleteEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			node := evt.Object.(*v1alpha1.Node)
			log := ctrl.LoggerFrom(ctx)

			if err := r.Cache.RemoveContainer(node); err != nil {
				log.Error(err, "Error removing container from cache")
			}
		},
	}
}

func (r *SchedulerReconciler) isInstanceAssigned() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		instance := obj.(*v1alpha1.Instance)
		return instance.Spec.NodeRef != nil
	})
}

func (r *SchedulerReconciler) handleAssignedInstances() handler.EventHandler {
	return handler.Funcs{
		CreateFunc: func(ctx context.Context, evt event.CreateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			instance := evt.Object.(*v1alpha1.Instance)
			log := ctrl.LoggerFrom(ctx)

			if err := r.Cache.AddInstance(instance); err != nil {
				log.Error(err, "Error adding instance to cache")
			}
		},
		UpdateFunc: func(ctx context.Context, evt event.UpdateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			oldInstance := evt.ObjectOld.(*v1alpha1.Instance)
			newInstance := evt.ObjectNew.(*v1alpha1.Instance)
			log := ctrl.LoggerFrom(ctx)

			// Only add or update are possible - node updates are not allowed by admission.
			oldInstanceAssigned := oldInstance.Spec.NodeRef != nil
			newInstanceAssigned := newInstance.Spec.NodeRef != nil

			switch {
			case oldInstanceAssigned && newInstanceAssigned:
				if err := r.Cache.UpdateInstance(oldInstance, newInstance); err != nil {
					log.Error(err, "Error updating instance in cache")
				}
			case !oldInstanceAssigned && newInstanceAssigned:
				if err := r.Cache.AddInstance(newInstance); err != nil {
					log.Error(err, "Error adding instance to cache")
				}
			}
		},
		DeleteFunc: func(ctx context.Context, evt event.DeleteEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			instance := evt.Object.(*v1alpha1.Instance)
			log := ctrl.LoggerFrom(ctx)

			if err := r.Cache.RemoveInstance(instance); err != nil {
				log.Error(err, "Error adding instance to cache")
			}
		},
	}
}

func (r *SchedulerReconciler) handleUnassignedInstance() handler.EventHandler {
	return handler.Funcs{
		CreateFunc: func(ctx context.Context, evt event.CreateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			instance := evt.Object.(*v1alpha1.Instance)
			queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(instance)})
		},
		UpdateFunc: func(ctx context.Context, evt event.UpdateEvent, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			oldInstance := evt.ObjectOld.(*v1alpha1.Instance)
			newInstance := evt.ObjectNew.(*v1alpha1.Instance)
			log := ctrl.LoggerFrom(ctx)

			if oldInstance.ResourceVersion == newInstance.ResourceVersion {
				return
			}

			isAssumed, err := r.Cache.IsAssumedInstance(newInstance)
			if err != nil {
				log.Error(err, "Error checking whether instance is assumed", "Instance", klog.KObj(newInstance))
			}
			if isAssumed {
				return
			}

			queue.Add(ctrl.Request{NamespacedName: client.ObjectKeyFromObject(newInstance)})
		},
	}
}

func (r *SchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			// Only a single concurrent reconcile since it is serialized on the scheduling algorithm's node fitting.
			MaxConcurrentReconciles: 1,
		}).
		Named("instance-scheduler").
		Watches(
			&v1alpha1.Instance{},
			r.handleUnassignedInstance(),
			builder.WithPredicates(
				r.instanceNotAssignedPredicate(),
			),
		).
		Watches(
			&v1alpha1.Instance{},
			r.handleAssignedInstances(),
			builder.WithPredicates(
				r.isInstanceAssigned(),
			),
		).
		Watches(
			&v1alpha1.Node{},
			r.handleNode(),
		).
		Complete(r)
}
