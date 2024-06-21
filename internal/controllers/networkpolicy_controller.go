// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NetworkPolicyReconciler struct {
	client.Client
}

//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkpolicies,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=networkpolicyrules,verbs=get;list;watch;create;update;patch;delete

func (r *NetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	networkPolicy := &v1alpha1.NetworkPolicy{}
	if err := r.Get(ctx, req.NamespacedName, networkPolicy); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		networkPolicyRule := &v1alpha1.NetworkPolicyRule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			},
		}
		if err := r.Delete(ctx, networkPolicyRule); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("error deleting network policy rule: %w", err)
		}
		return ctrl.Result{}, nil
	}

	return r.reconcileExists(ctx, log, networkPolicy)
}

func (r *NetworkPolicyReconciler) reconcileExists(ctx context.Context, log logr.Logger, networkPolicy *v1alpha1.NetworkPolicy) (ctrl.Result, error) {
	if !networkPolicy.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, networkPolicy)
	}
	return r.reconcile(ctx, log, networkPolicy)
}

func (r *NetworkPolicyReconciler) delete(ctx context.Context, log logr.Logger, networkPolicy *v1alpha1.NetworkPolicy) (ctrl.Result, error) {
	_, _ = ctx, networkPolicy
	log.V(1).Info("Delete")
	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *NetworkPolicyReconciler) reconcile(_ context.Context, log logr.Logger, _ *v1alpha1.NetworkPolicy) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")
	//reconcile logic

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.NetworkPolicy{}).
		Complete(r)
}
