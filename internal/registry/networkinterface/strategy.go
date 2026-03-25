// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkinterface

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/internal/apis/core/validation"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	apisrvstorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	networkInterface, ok := obj.(*core.NetworkInterface)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a NetworkInterface")
	}
	return networkInterface.Labels, SelectableFields(networkInterface), nil
}

func MatchNetworkInterface(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(networkInterface *core.NetworkInterface) fields.Set {
	return generic.ObjectMetaFieldsSet(&networkInterface.ObjectMeta, true)
}

type networkInterfaceStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) networkInterfaceStrategy {
	return networkInterfaceStrategy{typer, names.SimpleNameGenerator}
}

func (networkInterfaceStrategy) NamespaceScoped() bool {
	return true
}

func (networkInterfaceStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (networkInterfaceStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (networkInterfaceStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	networkInterface := obj.(*core.NetworkInterface)
	return validation.ValidateNetworkInterface(networkInterface)
}

func (networkInterfaceStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (networkInterfaceStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (networkInterfaceStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (networkInterfaceStrategy) Canonicalize(obj runtime.Object) {
}

func (networkInterfaceStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNetworkInterface := obj.(*core.NetworkInterface)
	oldNetworkInterface := old.(*core.NetworkInterface)
	return validation.ValidateNetworkInterfaceUpdate(newNetworkInterface, oldNetworkInterface)
}

func (networkInterfaceStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type networkInterfaceStatusStrategy struct {
	networkInterfaceStrategy
}

func NewStatusStrategy(typer runtime.ObjectTyper) networkInterfaceStatusStrategy {
	return networkInterfaceStatusStrategy{NewStrategy(typer)}
}

func (networkInterfaceStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apinet.ironcore.dev/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (networkInterfaceStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newNetworkInterface := obj.(*core.NetworkInterface)
	oldNetworkInterface := old.(*core.NetworkInterface)
	newNetworkInterface.Spec = oldNetworkInterface.Spec
}

func (networkInterfaceStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNetworkInterface := obj.(*core.NetworkInterface)
	oldNetworkInterface := old.(*core.NetworkInterface)
	return validation.ValidateNetworkInterfaceStatusUpdate(newNetworkInterface, oldNetworkInterface)
}

func (networkInterfaceStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
