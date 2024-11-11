// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-logr/logr"

	apinetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apinetletclient "github.com/ironcore-dev/ironcore-net/apinetlet/client"
	"github.com/ironcore-dev/ironcore-net/apinetlet/handler"
	apinetv1alpha1ac "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/core/v1alpha1"
	ironcorenet "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned"

	"github.com/ironcore-dev/controller-utils/clientutils"
	commonv1alpha1 "github.com/ironcore-dev/ironcore/api/common/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	"github.com/ironcore-dev/ironcore/utils/generic"
	"github.com/ironcore-dev/ironcore/utils/predicates"
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
	natGatewayFinalizer = "apinet.ironcore.dev/natgateway"
)

type NATGatewayReconciler struct {
	client.Client
	APINetClient    client.Client
	APINetInterface ironcorenet.Interface
	APINetNamespace string

	WatchFilterValue string
}

//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=natgateways,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=natgateways/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=networking.ironcore.dev,resources=natgateways/status,verbs=get;update;patch

//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=natgateways,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+cluster=apinet:kubebuilder:rbac:groups=core.apinet.ironcore.dev,resources=natgatewayautoscalers,verbs=get;list;watch;create;update;patch;delete;deletecollection

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
	if err := r.APINetClient.DeleteAllOf(ctx, &apinetv1alpha1.NATGateway{},
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
	apiNetNATGateway := &apinetv1alpha1.NATGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(natGateway.UID),
		},
	}
	if err := r.APINetClient.Delete(ctx, apiNetNATGateway); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error deleting APINet NAT Gateway: %w", err)
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

	apiNetNATGatewayCfg := apinetv1alpha1ac.NATGateway(string(natGateway.UID), r.APINetNamespace).
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

	apiNetNATGatewayAutoscaler := &apinetv1alpha1.NATGatewayAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apinetv1alpha1.SchemeGroupVersion.String(),
			Kind:       "NATGatewayAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.APINetNamespace,
			Name:      string(natGateway.UID),
			Labels:    apinetletclient.SourceLabels(r.Scheme(), r.RESTMapper(), natGateway),
		},
		Spec: apinetv1alpha1.NATGatewayAutoscalerSpec{
			NATGatewayRef: corev1.LocalObjectReference{Name: apiNetNATGateway.Name},
			MinPublicIPs:  generic.Pointer[int32](1),  // TODO: Make this configurable via ironcore NAT gateway
			MaxPublicIPs:  generic.Pointer[int32](10), // TODO: Configure depending on ironcore NAT gateway
		},
	}
	_ = ctrl.SetControllerReference(apiNetNATGateway, apiNetNATGatewayAutoscaler, r.Scheme())
	if err := r.APINetClient.Patch(ctx, apiNetNATGatewayAutoscaler, client.Apply, client.ForceOwnership, fieldOwner); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying APINet NAT gateway autoscaler: %w", err)
	}

	natGatewayIPs := apiNetIPsToIPs(apinetv1alpha1.GetNATGatewayIPs(apiNetNATGateway))
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
			source.Kind[client.Object](
				apiNetCache,
				&apinetv1alpha1.NATGateway{},
				handler.EnqueueRequestForSource(mgr.GetScheme(), mgr.GetRESTMapper(), &networkingv1alpha1.NATGateway{}),
			),
		).
		WatchesRawSource(
			source.Kind[client.Object](
				apiNetCache,
				&apinetv1alpha1.NATGatewayAutoscaler{},
				handler.EnqueueRequestForSource(mgr.GetScheme(), mgr.GetRESTMapper(), &networkingv1alpha1.NATGateway{}),
			),
		).
		Complete(r)
}
