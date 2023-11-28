// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package network

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
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	network, ok := obj.(*core.Network)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a Network")
	}
	return network.Labels, SelectableFields(network), nil
}

func MatchNetwork(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(network *core.Network) fields.Set {
	return generic.ObjectMetaFieldsSet(&network.ObjectMeta, true)
}

type networkStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) networkStrategy {
	return networkStrategy{typer, names.SimpleNameGenerator}
}

func (networkStrategy) NamespaceScoped() bool {
	return true
}

func (networkStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (networkStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (networkStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	network := obj.(*core.Network)
	return validation.ValidateNetwork(network)
}

func (networkStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (networkStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (networkStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (networkStrategy) Canonicalize(obj runtime.Object) {
}

func (networkStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNetwork := obj.(*core.Network)
	oldNetwork := old.(*core.Network)
	return validation.ValidateNetworkUpdate(newNetwork, oldNetwork)
}

func (networkStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type networkStatusStrategy struct {
	networkStrategy
}

func NewStatusStrategy(typer runtime.ObjectTyper) networkStatusStrategy {
	return networkStatusStrategy{NewStrategy(typer)}
}

func (networkStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apinet.ironcore.dev/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (networkStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newNetwork := obj.(*core.Network)
	oldNetwork := old.(*core.Network)
	newNetwork.Spec = oldNetwork.Spec
}

func (networkStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNetwork := obj.(*core.Network)
	oldNetwork := old.(*core.Network)
	return validation.ValidateNetworkStatusUpdate(newNetwork, oldNetwork)
}

func (networkStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
