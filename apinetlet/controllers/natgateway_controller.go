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
	"slices"

	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	apinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	apinetletclient "github.com/onmetal/onmetal-api-net/apinetlet/client"
	"github.com/onmetal/onmetal-api-net/apinetlet/handler"
	apinetv1alpha1ac "github.com/onmetal/onmetal-api-net/client-go/applyconfigurations/core/v1alpha1"
	"github.com/onmetal/onmetal-api-net/client-go/onmetalapinet"
	commonv1alpha1 "github.com/onmetal/onmetal-api/api/common/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/api/networking/v1alpha1"
	"github.com/onmetal/onmetal-api/utils/generic"
	"github.com/onmetal/onmetal-api/utils/predicates"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	natGatewayFinalizer = "apinet.api.onmetal.de/natgateway"
)

type NATGatewayReconciler struct {
	client.Client
	APINetClient    client.Client
	APINetInterface onmetalapinet.Interface
	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=natgateways,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=natgateways/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=natgateways/status,verbs=get;update;patch

//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=natgateways,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.api.onmetal.de,resources=natgatewayautoscalers,verbs=get;list;watch;create;update;patch;delete;deletecollection

func (r *NATGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	natGateway := &networkingv1alpha1.NATGateway{}
	if err := r.Get(ctx, req.NamespacedName, natGateway); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error getting nat gateway: %w", err)
		}
		return r.deleteGone(ctx, log, req.NamespacedName)
	}
	return r.reconcileExists(ctx, log, natGateway)
}

func (r *NATGatewayReconciler) deleteGone(ctx context.Context, log logr.Logger, key client.ObjectKey) (ctrl.Result, error) {
	log.V(1).Info("Delete gone")

	log.V(1).Info("Deleting any APINet NAT gateway by key")
	if err := r.APINetClient.DeleteAllOf(ctx, &v1alpha1.NATGateway{},
		client.InNamespace(r.APINetNamespace),
		apinetletclient.MatchingSourceKeyLabels(r.Scheme(), r.RESTMapper(), key, &networkingv1alpha1.NATGateway{}),
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting apinet nat gateways by key: %w", err)
	}

	log.V(1).Info("Deleted gone")
	return ctrl.Result{}, nil
}

func (r *NATGatewayReconciler) reconcileExists(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway) (ctrl.Result, error) {
	if !natGateway.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, natGateway)
	}
	return r.reconcile(ctx, log, natGateway)
}

func (r *NATGatewayReconciler) delete(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(natGateway, natGatewayFinalizer) {
		log.V(1).Info("No finalizer present, nothing to do")
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Finalizer present, running cleanup")

	log.V(1).Info("Deleting APINet NAT gateway")
	apiNetNATGateway := &v1alpha1.NATGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(natGateway.UID),
		},
	}
	if err := r.APINetClient.Delete(ctx, apiNetNATGateway); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting apinet NAT Gateway: %w", err)
		}

		log.V(1).Info("APINet NAT gateway is gone, removing finalizer")
		if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, natGateway, natGatewayFinalizer); err != nil {
			return ctrl.Result{}, fmt.Errorf("error removing NAT gateway finalizer: %w", err)
		}
		log.V(1).Info("Deleted")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Issued APINet NAT gateway deletion")
	return ctrl.Result{Requeue: true}, nil
}

func (r *NATGatewayReconciler) reconcile(ctx context.Context, log logr.Logger, natGateway *networkingv1alpha1.NATGateway) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, natGateway, natGatewayFinalizer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
	}
	if modified {
		log.V(1).Info("Added finalizer, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}
	log.V(1).Info("Finalizer is present")

	networkKey := client.ObjectKey{Namespace: natGateway.Namespace, Name: natGateway.Spec.NetworkRef.Name}
	networkName, err := getAPINetNetworkName(ctx, r.Client, networkKey)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting apinet network name: %w", err)
	}
	if networkName == "" {
		log.V(1).Info("APINet network is not ready")
		return ctrl.Result{}, nil
	}

	apiNetNATGatewayCfg :=
		apinetv1alpha1ac.NATGateway(string(natGateway.UID), r.APINetNamespace).
			WithLabels(apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), natGateway)).
			WithSpec(apinetv1alpha1ac.NATGatewaySpec().
				WithIPFamily(natGateway.Spec.IPFamily).
				WithNetworkRef(corev1.LocalObjectReference{Name: networkName}).
				WithPortsPerNetworkInterface(generic.Deref(
					natGateway.Spec.PortsPerNetworkInterface,
					networkingv1alpha1.DefaultPortsPerNetworkInterface,
				)),
			)
	apiNetNATGateway, err := r.APINetInterface.CoreV1alpha1().
		NATGateways(r.APINetNamespace).
		Apply(ctx, apiNetNATGatewayCfg, metav1.ApplyOptions{FieldManager: string(fieldOwner), Force: true})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying apinet nat gateway: %w", err)
	}

	apiNetNATGatewayAutoscaler := &v1alpha1.NATGatewayAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apinetv1alpha1.SchemeGroupVersion.String(),
			Kind:       "NATGatewayAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(natGateway.UID),
			Labels:    apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), natGateway),
		},
		Spec: v1alpha1.NATGatewayAutoscalerSpec{
			NATGatewayRef: corev1.LocalObjectReference{Name: apiNetNATGateway.Name},
			MinPublicIPs:  generic.Pointer[int32](1),  // TODO: Make this configurable via onmetal-api NAT gateway
			MaxPublicIPs:  generic.Pointer[int32](10), // TODO: Configure depending on onmetal-api NAT gateway
		},
	}
	_ = ctrl.SetControllerReference(apiNetNATGateway, apiNetNATGatewayAutoscaler, r.Scheme())
	if err := r.APINetClient.Patch(ctx, apiNetNATGatewayAutoscaler, client.Apply, client.ForceOwnership, fieldOwner); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying apinet NAT gateway autoscaler: %w", err)
	}

	natGatewayIPs := apiNetIPsToIPs(v1alpha1.GetNATGatewayIPs(apiNetNATGateway))
	if !slices.Equal(natGateway.Status.IPs, natGatewayIPs) {
		if err := r.updateNATGatewayStatus(ctx, natGateway, natGatewayIPs); err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating NAT gateway status IPs: %w", err)
		}
		log.V(1).Info("Updated NAT gateway status IPs", "ips", natGatewayIPs)
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *NATGatewayReconciler) updateNATGatewayStatus(
	ctx context.Context,
	natGateway *networkingv1alpha1.NATGateway,
	ips []commonv1alpha1.IP,
) error {
	base := natGateway.DeepCopy()
	natGateway.Status.IPs = ips
	return r.Status().Patch(ctx, natGateway, client.StrategicMergeFrom(base))
}

func (r *NATGatewayReconciler) SetupWithManager(mgr ctrl.Manager, apiNetCache cache.Cache) error {
	log := ctrl.Log.WithName("natgateway").WithName("setup")

	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1alpha1.NATGateway{},
			builder.WithPredicates(
				predicates.ResourceHasFilterLabel(log, r.WatchFilterValue),
				predicates.ResourceIsNotExternallyManaged(log),
			),
		).
		WatchesRawSource(
			source.Kind(apiNetCache, &v1alpha1.NATGateway{}),
			handler.EnqueueRequestForSource(mgr.GetScheme(), mgr.GetRESTMapper(), &networkingv1alpha1.NATGateway{}),
		).
		WatchesRawSource(
			source.Kind(apiNetCache, &v1alpha1.NATGatewayAutoscaler{}),
			handler.EnqueueRequestForSource(mgr.GetScheme(), mgr.GetRESTMapper(), &networkingv1alpha1.NATGateway{}),
		).
		Complete(r)
}
