// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkpolicyrule

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
)

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	networkPolicyRule, ok := obj.(*core.NetworkPolicyRule)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a NetworkPolicyRule")
	}
	return networkPolicyRule.Labels, SelectableFields(networkPolicyRule), nil
}

func MatchNetworkPolicyRule(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(networkPolicyRule *core.NetworkPolicyRule) fields.Set {
	return generic.ObjectMetaFieldsSet(&networkPolicyRule.ObjectMeta, true)
}

type networkPolicyRuleStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) networkPolicyRuleStrategy {
	return networkPolicyRuleStrategy{typer, names.SimpleNameGenerator}
}

func (networkPolicyRuleStrategy) NamespaceScoped() bool {
	return true
}

func (networkPolicyRuleStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (networkPolicyRuleStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (networkPolicyRuleStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	networkPolicyRule := obj.(*core.NetworkPolicyRule)
	return validation.ValidateNetworkPolicyRule(networkPolicyRule)
}

func (networkPolicyRuleStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (networkPolicyRuleStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (networkPolicyRuleStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (networkPolicyRuleStrategy) Canonicalize(obj runtime.Object) {
}

func (networkPolicyRuleStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNetworkPolicyRule := obj.(*core.NetworkPolicyRule)
	oldNetworkPolicyRule := old.(*core.NetworkPolicyRule)
	return validation.ValidateNetworkPolicyRuleUpdate(newNetworkPolicyRule, oldNetworkPolicyRule)
}

func (networkPolicyRuleStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
