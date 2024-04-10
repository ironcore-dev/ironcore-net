// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkpolicy

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
	networkPolicy, ok := obj.(*core.NetworkPolicy)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a NetworkPolicy")
	}
	return networkPolicy.Labels, SelectableFields(networkPolicy), nil
}

func MatchNetworkPolicy(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(networkPolicy *core.NetworkPolicy) fields.Set {
	return generic.ObjectMetaFieldsSet(&networkPolicy.ObjectMeta, true)
}

type networkPolicyStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) networkPolicyStrategy {
	return networkPolicyStrategy{typer, names.SimpleNameGenerator}
}
func (networkPolicyStrategy) NamespaceScoped() bool {
	return true
}

func (networkPolicyStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {

}

func (networkPolicyStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (networkPolicyStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	networkPolicy := obj.(*core.NetworkPolicy)
	return validation.ValidateNetworkPolicy(networkPolicy)
}

func (networkPolicyStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (networkPolicyStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (networkPolicyStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (networkPolicyStrategy) Canonicalize(obj runtime.Object) {
}

func (networkPolicyStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNetworkPolicy := obj.(*core.NetworkPolicy)
	oldNetworkPolicy := old.(*core.NetworkPolicy)
	return validation.ValidateNetworkPolicyUpdate(newNetworkPolicy, oldNetworkPolicy)
}

func (networkPolicyStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type networkPolicyStatusStrategy struct {
	networkPolicyStrategy
}

func NewStatusStrategy(typer runtime.ObjectTyper) networkPolicyStatusStrategy {
	return networkPolicyStatusStrategy{NewStrategy(typer)}
}
func (networkPolicyStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apinet.ironcore.dev/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (networkPolicyStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (networkPolicyStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNetworkPolicy := obj.(*core.NetworkPolicy)
	oldNetworkPolicy := old.(*core.NetworkPolicy)
	return validation.ValidateNetworkPolicyUpdate(newNetworkPolicy, oldNetworkPolicy)
}

func (networkPolicyStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
