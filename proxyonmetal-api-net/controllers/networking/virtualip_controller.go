// Copyright 2022 OnMetal authors
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

package networking

import (
	"context"
	"errors"
	"fmt"

	"github.com/onmetal/onmetal-api-net/allocator"
	"github.com/onmetal/poollet/broker"
	brokerhandler "github.com/onmetal/poollet/broker/handler"
	brokermeta "github.com/onmetal/poollet/broker/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	commonv1alpha1 "github.com/onmetal/onmetal-api/apis/common/v1alpha1"
	networkingv1alpha1 "github.com/onmetal/onmetal-api/apis/networking/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const finalizer = "net.networking.api.onmetal.de/proxy-virtual-ip"

type ProxyVirtualIPReconciler struct {
	client.Client
	record.EventRecorder
	Scheme *runtime.Scheme

	Allocator allocator.Allocator

	TargetClient client.Client

	ClusterName string
}

//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;create;watch;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func (r *ProxyVirtualIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	virtualIP := &networkingv1alpha1.VirtualIP{}
	if err := r.Get(ctx, req.NamespacedName, virtualIP); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcileExists(ctx, log, virtualIP)
}

func (r *ProxyVirtualIPReconciler) reconcileExists(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (ctrl.Result, error) {
	if virtualIP.Spec.Type != networkingv1alpha1.VirtualIPTypePublic {
		return ctrl.Result{}, nil
	}

	if !virtualIP.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, virtualIP)
	}
	return r.reconcile(ctx, log, virtualIP)
}

func (r *ProxyVirtualIPReconciler) delete(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(virtualIP, finalizer) {
		log.V(1).Info("Finalizer not present")
		return ctrl.Result{}, nil
	}

	var err error
	if brokerCtrl := brokermeta.GetBrokerControllerOf(virtualIP); brokerCtrl != nil {
		err = r.releaseProxy(ctx, log, virtualIP, brokerCtrl)
	} else {
		err = r.releaseDirect(ctx, log, virtualIP)
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error releasing: %w", err)
	}

	log.V(1).Info("Released resources, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, virtualIP, finalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *ProxyVirtualIPReconciler) releaseDirect(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) error {
	log.V(1).Info("Releasing resources for virtual ip")
	if err := r.Allocator.Release(ctx, string(virtualIP.UID)); err != nil {
		return fmt.Errorf("error releasing resources for virtual ip: %w", err)
	}
	return nil
}

func (r *ProxyVirtualIPReconciler) releaseProxy(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP, brokerCtrl *brokermeta.BrokerOwnerReference) error {
	targetVirtualIP := &networkingv1alpha1.VirtualIP{}
	targetVirtualIPKey := client.ObjectKey{Namespace: brokerCtrl.Namespace, Name: brokerCtrl.Name}
	log.V(1).Info("Getting target virtual ip", "TargetVirtualIPKey", targetVirtualIPKey)
	if err := r.TargetClient.Get(ctx, targetVirtualIPKey, targetVirtualIP); err != nil {
		return client.IgnoreNotFound(err)
	}

	log.V(1).Info("Removing broker owner reference", "TargetVirtualIPKey", targetVirtualIPKey)
	baseTargetVirtualIP := targetVirtualIP.DeepCopy()
	if err := brokermeta.RemoveBrokerOwnerReference(r.ClusterName, virtualIP, targetVirtualIP, r.Scheme); err != nil {
		return fmt.Errorf("error removing target %s broker owner reference: %w", targetVirtualIPKey, err)
	}
	if err := r.TargetClient.Patch(ctx, targetVirtualIP, client.MergeFrom(baseTargetVirtualIP)); err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

func (r *ProxyVirtualIPReconciler) reconcile(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (ctrl.Result, error) {
	log.V(1).Info("Reconcile")

	if ip := virtualIP.Status.IP; ip != nil {
		log.V(1).Info("IP already assigned", "IP", ip)
		return ctrl.Result{}, nil
	}

	log.V(1).Info("IP not yet assigned")
	log.V(1).Info("Ensuring finalizer")
	modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, virtualIP, finalizer)
	if err != nil || modified {
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error ensuring finalizer: %w", err)
		}
		return ctrl.Result{}, nil
	}

	var ip *commonv1alpha1.IP
	if brokerCtrl := brokermeta.GetBrokerControllerOf(virtualIP); brokerCtrl != nil {
		ip, err = r.allocateProxy(ctx, log, virtualIP, brokerCtrl)
	} else {
		ip, err = r.allocateDirect(ctx, log, virtualIP)
	}
	if err != nil {
		switch {
		case errors.Is(err, allocator.ErrCannotHandleIPFamily):
			r.Event(virtualIP, corev1.EventTypeWarning, "IPFamilyNotSupported", "The specified ip family is not supported")
			return ctrl.Result{}, nil
		case errors.Is(err, allocator.ErrNoSpaceLeft):
			r.Event(virtualIP, corev1.EventTypeNormal, "NoIPAvailable", "Currently no ip can be allocated")
			return ctrl.Result{}, err
		default:
			return ctrl.Result{}, fmt.Errorf("error allocating ip: %w", err)
		}
	}

	log.V(1).Info("Patching status", "IP", ip)
	if err := r.patchStatus(ctx, virtualIP, ip); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching status: %w", err)
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *ProxyVirtualIPReconciler) allocateDirect(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (*commonv1alpha1.IP, error) {
	log.V(1).Info("Allocating ip", "Family", virtualIP.Spec.IPFamily)
	ip, err := r.Allocator.Allocate(ctx, string(virtualIP.UID), virtualIP.Spec.IPFamily)
	if err != nil {
		return nil, err
	}
	return &ip, nil
}

func (r *ProxyVirtualIPReconciler) allocateProxy(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP, brokerCtrl *brokermeta.BrokerOwnerReference) (*commonv1alpha1.IP, error) {
	targetVirtualIP := &networkingv1alpha1.VirtualIP{}
	targetVirtualIPKey := client.ObjectKey{Namespace: brokerCtrl.Namespace, Name: brokerCtrl.Name}
	log.V(1).Info("Getting target virtual ip", "TargetVirtualIPKey", targetVirtualIPKey)
	if err := r.TargetClient.Get(ctx, targetVirtualIPKey, targetVirtualIP); err != nil {
		return nil, fmt.Errorf("error getting target virtual ip: %w", err)
	}

	log.V(1).Info("Ensuring broker owner reference", "TargetVirtualIPKey", targetVirtualIPKey)
	baseTargetVirtualIP := targetVirtualIP.DeepCopy()
	if err := brokermeta.SetBrokerOwnerReference(r.ClusterName, virtualIP, targetVirtualIP, r.Scheme); err != nil {
		return nil, fmt.Errorf("error setting target %s broker owner reference: %w", targetVirtualIPKey, err)
	}
	if err := r.TargetClient.Patch(ctx, targetVirtualIP, client.MergeFrom(baseTargetVirtualIP)); err != nil {
		return nil, fmt.Errorf("error patching target %s: %w", targetVirtualIPKey, err)
	}

	return targetVirtualIP.Status.IP, nil
}

func (r *ProxyVirtualIPReconciler) patchStatus(ctx context.Context, virtualIP *networkingv1alpha1.VirtualIP, ip *commonv1alpha1.IP) error {
	base := virtualIP.DeepCopy()
	virtualIP.Status.IP = ip
	return r.Status().Patch(ctx, virtualIP, client.MergeFrom(base))
}

func (r *ProxyVirtualIPReconciler) SetupWithManager(mgr broker.Manager) error {
	return broker.NewControllerManagedBy(mgr, r.ClusterName).
		FilterNoTargetNamespace().
		For(&networkingv1alpha1.VirtualIP{}).
		WatchesTarget(
			&source.Kind{Type: &networkingv1alpha1.VirtualIP{}},
			&brokerhandler.EnqueueRequestForBrokerOwner{
				OwnerType:   &networkingv1alpha1.VirtualIP{},
				ClusterName: r.ClusterName,
			},
		).
		Complete(r)
}
