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
	"github.com/onmetal/onmetal-api-net/internal/natgateway"
	utilslices "github.com/onmetal/onmetal-api/utils/slices"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

const (
	noOfNATIPGenerateNameChars = 10
)

//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=natgatewayautoscalers,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=natgatewayautoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=natgateways,verbs=get;list;watch;update;patch

type NATGatewayAutoscalerReconciler struct {
	client.Client
}

func (r *NATGatewayAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	natGatewayAutoscaler := &v1alpha1.NATGatewayAutoscaler{}
	if err := r.Get(ctx, req.NamespacedName, natGatewayAutoscaler); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcileExists(ctx, log, natGatewayAutoscaler)
}

func (r *NATGatewayAutoscalerReconciler) reconcileExists(ctx context.Context, log logr.Logger, natGatewayAutoscaler *v1alpha1.NATGatewayAutoscaler) (ctrl.Result, error) {
	natGatewayName := natGatewayAutoscaler.Spec.NATGatewayRef.Name
	log = log.WithValues("NATGatewayName", natGatewayName)

	if !natGatewayAutoscaler.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}
	return r.reconcile(ctx, log, natGatewayAutoscaler)
}

func (r *NATGatewayAutoscalerReconciler) reconcile(ctx context.Context, log logr.Logger, natGatewayAutoscaler *v1alpha1.NATGatewayAutoscaler) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Getting scale target")
	natGateway := &v1alpha1.NATGateway{}
	natGatewayKey := client.ObjectKey{Namespace: natGatewayAutoscaler.Namespace, Name: natGatewayAutoscaler.Spec.NATGatewayRef.Name}
	if err := r.Get(ctx, natGatewayKey, natGateway); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting NAT gateway %s: %w", natGatewayKey.Name, err)
		}
		log.V(1).Info("Scale target not found")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Managing public IPs")
	if err := r.manageNATGatewayIPs(ctx, natGatewayAutoscaler, natGateway); err != nil {
		if !apierrors.IsConflict(err) {
			return ctrl.Result{}, fmt.Errorf("error managing public IPs: %w", err)
		}

		log.V(1).Info("Conflict managing public IPs, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NATGatewayAutoscalerReconciler) generateNewNATGatewayIPs(natGateway *v1alpha1.NATGateway, ct int) []v1alpha1.NATGatewayIP {
	existingNames := utilslices.ToSetFunc(natGateway.Spec.IPs, func(ip v1alpha1.NATGatewayIP) string { return ip.Name })
	newNATGatewayIPs := sets.New[v1alpha1.NATGatewayIP]()
	for newNATGatewayIPs.Len() != ct {
		name := utilrand.String(noOfNATIPGenerateNameChars)
		if !existingNames.Has(name) {
			newNATGatewayIPs.Insert(v1alpha1.NATGatewayIP{Name: name})
		}
	}
	return newNATGatewayIPs.UnsortedList()
}

func (r *NATGatewayAutoscalerReconciler) manageNATGatewayIPs(
	ctx context.Context,
	natGatewayAutoscaler *v1alpha1.NATGatewayAutoscaler,
	natGateway *v1alpha1.NATGateway,
) error {
	totalRequests := natGateway.Status.RequestedNATIPs
	currentNoOfIPs := len(natGateway.Spec.IPs)
	desiredNoOfIPs := r.determineDesiredNumberOfIPs(natGatewayAutoscaler, natGateway, currentNoOfIPs, int(totalRequests))
	diff := currentNoOfIPs - desiredNoOfIPs

	if diff < 0 {
		diff *= -1

		ips := append(make([]v1alpha1.NATGatewayIP, 0, len(natGateway.Spec.IPs)+diff), natGateway.Spec.IPs...)
		ips = append(ips, r.generateNewNATGatewayIPs(natGateway, diff)...)
		return r.updateNATGatewayIPs(ctx, natGateway, ips)
	} else if diff > 0 {
		// Delete IPs from the end since they are the 'newer' ones.
		ips := natGateway.Spec.IPs[:len(natGateway.Spec.IPs)-diff]
		return r.updateNATGatewayIPs(ctx, natGateway, ips)
	}
	return nil
}

func (r *NATGatewayAutoscalerReconciler) updateNATGatewayIPs(
	ctx context.Context,
	natGateway *v1alpha1.NATGateway,
	ips []v1alpha1.NATGatewayIP,
) error {
	base := natGateway.DeepCopy()
	natGateway.Spec.IPs = ips
	return r.Patch(ctx, natGateway, client.MergeFromWithOptions(base, &client.MergeFromWithOptimisticLock{}))
}

func (r *NATGatewayAutoscalerReconciler) determineDesiredNumberOfIPs(
	natGatewayAutoscaler *v1alpha1.NATGatewayAutoscaler,
	natGateway *v1alpha1.NATGateway,
	totalNoOfIPs int,
	totalRequests int,
) int {
	slotsPerIP := int64(natgateway.SlotsPerIP(natGateway.Spec.PortsPerNetworkInterface))
	totalSlots := slotsPerIP * int64(totalNoOfIPs)
	slotsDiff := totalSlots - int64(totalRequests)
	ipsDiff := int(slotsDiff / slotsPerIP)
	desiredNoOfPublicIPs := totalNoOfIPs - ipsDiff

	if minPublicIPs := natGatewayAutoscaler.Spec.MinPublicIPs; minPublicIPs != nil {
		desiredNoOfPublicIPs = max(int(*minPublicIPs), desiredNoOfPublicIPs)
	}
	if maxPublicIPs := natGatewayAutoscaler.Spec.MaxPublicIPs; maxPublicIPs != nil {
		desiredNoOfPublicIPs = min(int(*maxPublicIPs), desiredNoOfPublicIPs)
	}
	return desiredNoOfPublicIPs
}

func (r *NATGatewayAutoscalerReconciler) enqueueByNATGateway() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		natGateway := obj.(*v1alpha1.NATGateway)
		log := ctrl.LoggerFrom(ctx)

		natGatewayAutoscalerList := &v1alpha1.NATGatewayAutoscalerList{}
		if err := r.List(ctx, natGatewayAutoscalerList,
			client.InNamespace(natGateway.Namespace),
		); err != nil {
			log.Error(err, "Error listing NAT gateway autoscalers")
			return nil
		}

		var reqs []ctrl.Request
		for _, natGatewayAutoscaler := range natGatewayAutoscalerList.Items {
			if natGatewayAutoscaler.Spec.NATGatewayRef.Name == natGateway.Name {
				reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&natGatewayAutoscaler)})
			}
		}
		return reqs
	})
}

func (r *NATGatewayAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.NATGatewayAutoscaler{}).
		Watches(
			&v1alpha1.NATGateway{},
			r.enqueueByNATGateway(),
		).
		Complete(r)
}
