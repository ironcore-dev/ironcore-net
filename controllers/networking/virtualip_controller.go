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

const finalizer = "net.networking.api.onmetal.de/virtual-ip"

type VirtualIPReconciler struct {
	client.Client
	record.EventRecorder

	Allocator allocator.Allocator
}

//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.api.onmetal.de,resources=virtualips/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;create;watch;update;patch;delete

func (r *VirtualIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	virtualIP := &networkingv1alpha1.VirtualIP{}
	if err := r.Get(ctx, req.NamespacedName, virtualIP); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcileExists(ctx, log, virtualIP)
}

func (r *VirtualIPReconciler) reconcileExists(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (ctrl.Result, error) {
	if virtualIP.Spec.Type != networkingv1alpha1.VirtualIPTypePublic {
		return ctrl.Result{}, nil
	}

	if !virtualIP.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, virtualIP)
	}
	return r.reconcile(ctx, log, virtualIP)
}

func (r *VirtualIPReconciler) delete(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (ctrl.Result, error) {
	log.V(1).Info("Delete")

	if !controllerutil.ContainsFinalizer(virtualIP, finalizer) {
		log.V(1).Info("Finalizer not present")
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Releasing resources for virtual ip")
	if err := r.Allocator.Release(ctx, string(virtualIP.UID)); err != nil {
		return ctrl.Result{}, fmt.Errorf("error releasing resources for virtual ip: %w", err)
	}

	log.V(1).Info("Released resources, removing finalizer")
	if err := clientutils.PatchRemoveFinalizer(ctx, r.Client, virtualIP, finalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("error removing finalizer: %w", err)
	}

	log.V(1).Info("Deleted")
	return ctrl.Result{}, nil
}

func (r *VirtualIPReconciler) reconcile(ctx context.Context, log logr.Logger, virtualIP *networkingv1alpha1.VirtualIP) (ctrl.Result, error) {
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

	log.V(1).Info("Allocating ip", "Family", virtualIP.Spec.IPFamily)
	ip, err := r.Allocator.Allocate(ctx, string(virtualIP.UID), virtualIP.Spec.IPFamily)
	if err != nil {
		switch {
		case errors.Is(err, allocator.ErrCannotHandleIPFamily):
			r.Event(virtualIP, corev1.EventTypeWarning, "IPFamilyNotSupported", "The specified ip family is not supported")
			return ctrl.Result{}, nil
		case errors.Is(err, allocator.ErrNoSpaceLeft):
			r.Event(virtualIP, corev1.EventTypeNormal, "NoIPAvailable", "Currently no ip can be allocated")
			return ctrl.Result{Requeue: true}, nil
		default:
			return ctrl.Result{}, fmt.Errorf("error allocating ip: %w", err)
		}
	}

	log.V(1).Info("Allocated ip, patching status")
	if err := r.patchStatus(ctx, virtualIP, &ip); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching status: %w", err)
	}

	log.V(1).Info("Reconciled")
	return ctrl.Result{}, nil
}

func (r *VirtualIPReconciler) patchStatus(ctx context.Context, virtualIP *networkingv1alpha1.VirtualIP, ip *commonv1alpha1.IP) error {
	base := virtualIP.DeepCopy()
	virtualIP.Status.IP = ip
	return r.Status().Patch(ctx, virtualIP, client.MergeFrom(base))
}

func (r *VirtualIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.VirtualIP{}).
		Complete(r)
}
