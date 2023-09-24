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
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LoadBalancerReconciler struct {
	client.Client
}

//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=loadbalancers,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=loadbalancers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=loadbalancerroutings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=daemonsets,verbs=get;list;watch;create;update;patch

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
	daemonSet := &v1alpha1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: loadBalancer.Namespace,
			Name:      v1alpha1.LoadBalancerDaemonSetName(loadBalancer.Name),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(loadBalancer, v1alpha1.SchemeGroupVersion.WithKind("LoadBalancer")),
			},
		},
		Spec: v1alpha1.DaemonSetSpec{
			Selector: loadBalancer.Spec.Selector,
			Template: loadBalancer.Spec.Template,
		},
	}
	daemonSet.Spec.Template.Spec.Type = v1alpha1.InstanceTypeLoadBalancer
	daemonSet.Spec.Template.Spec.LoadBalancerType = loadBalancer.Spec.Type
	daemonSet.Spec.Template.Spec.IPs = v1alpha1.GetLoadBalancerIPs(loadBalancer)
	daemonSet.Spec.Template.Spec.NetworkRef = loadBalancer.Spec.NetworkRef
	daemonSet.Spec.Template.Spec.LoadBalancerPorts = loadBalancer.Spec.Ports
	err := r.Patch(ctx, daemonSet, client.Apply, fieldOwner, client.ForceOwnership)
	return err
}

func (r *LoadBalancerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.LoadBalancer{}).
		Owns(&v1alpha1.DaemonSet{}).
		Complete(r)
}
