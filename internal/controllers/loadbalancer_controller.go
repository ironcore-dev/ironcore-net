// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	corev1alpha1apply "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LoadBalancerReconciler struct {
	client.Client
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=loadbalancers,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=loadbalancers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=loadbalancerroutings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=daemonsets,verbs=get;list;watch;create;update;patch

func (r *LoadBalancerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	loadBalancer := &v1alpha1.LoadBalancer{}
	if err := r.Get(ctx, req.NamespacedName, loadBalancer); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		loadBalancerRouting := &v1alpha1.LoadBalancerRouting{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			},
		}
		if err := r.Delete(ctx, loadBalancerRouting); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("error deleting load balancer routing: %w", err)
		}
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, loadBalancer)
}

func (r *LoadBalancerReconciler) reconcileExists(ctx context.Context, log logr.Logger, loadBalancer *v1alpha1.LoadBalancer) (ctrl.Result, error) {
	if !loadBalancer.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, loadBalancer)
	}
	return r.reconcile(ctx, log, loadBalancer)
}

func (r *LoadBalancerReconciler) delete(ctx context.Context, log logr.Logger, loadBalancer *v1alpha1.LoadBalancer) (ctrl.Result, error) {
	_, _ = ctx, loadBalancer
	log.V(1).Info("Delete")
	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *LoadBalancerReconciler) reconcile(ctx context.Context, log logr.Logger, loadBalancer *v1alpha1.LoadBalancer) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	if err := r.applyDaemonSetForLoadBalancer(ctx, loadBalancer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying daemon set: %w", err)
	}
	log.V(1).Info("Applied daemon set")

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *LoadBalancerReconciler) applyDaemonSetForLoadBalancer(ctx context.Context, loadBalancer *v1alpha1.LoadBalancer) error {
	daemonsetApplyconfig := corev1alpha1apply.DaemonSet(loadBalancer.Namespace, v1alpha1.LoadBalancerDaemonSetName(loadBalancer.Name)).
		WithOwnerReferences(v1.OwnerReference().
			WithAPIVersion(v1alpha1.SchemeGroupVersion.String()).
			WithKind("LoadBalancer").
			WithName(loadBalancer.Name).
			WithUID(loadBalancer.UID)).
		WithSpec(corev1alpha1apply.DaemonSetSpec().
			WithSelector(v1.LabelSelector().
				WithMatchLabels(loadBalancer.Spec.Selector.MatchLabels)).
			WithTemplate(corev1alpha1apply.InstanceTemplate().
				WithLabels(loadBalancer.Spec.Template.ObjectMeta.Labels).
				WithSpec(corev1alpha1apply.InstanceSpec().
					WithType(v1alpha1.InstanceTypeLoadBalancer).
					WithLoadBalancerType(loadBalancer.Spec.Type).
					WithNetworkRef(corev1.LocalObjectReference{Name: loadBalancer.Spec.NetworkRef.Name}).
					WithIPs(v1alpha1.GetLoadBalancerIPs(loadBalancer)...))))
	err := r.Apply(ctx, daemonsetApplyconfig, fieldOwner, client.ForceOwnership)
	return err
}

func (r *LoadBalancerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.LoadBalancer{}).
		Owns(&v1alpha1.DaemonSet{}).
		Complete(r)
}
