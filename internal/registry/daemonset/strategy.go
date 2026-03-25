// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package daemonset

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
	daemonSet, ok := obj.(*core.DaemonSet)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a DaemonSet")
	}
	return daemonSet.Labels, SelectableFields(daemonSet), nil
}

func MatchDaemonSet(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(daemonSet *core.DaemonSet) fields.Set {
	return generic.ObjectMetaFieldsSet(&daemonSet.ObjectMeta, true)
}

type daemonSetStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) daemonSetStrategy {
	return daemonSetStrategy{typer, names.SimpleNameGenerator}
}

func (daemonSetStrategy) NamespaceScoped() bool {
	return true
}

func (daemonSetStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (daemonSetStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (daemonSetStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	daemonSet := obj.(*core.DaemonSet)
	return validation.ValidateDaemonSet(daemonSet)
}

func (daemonSetStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (daemonSetStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (daemonSetStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (daemonSetStrategy) Canonicalize(obj runtime.Object) {
}

func (daemonSetStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newDaemonSet := obj.(*core.DaemonSet)
	oldDaemonSet := old.(*core.DaemonSet)
	return validation.ValidateDaemonSetUpdate(newDaemonSet, oldDaemonSet)
}

func (daemonSetStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type daemonSetStatusStrategy struct {
	daemonSetStrategy
}

func NewStatusStrategy(typer runtime.ObjectTyper) daemonSetStatusStrategy {
	return daemonSetStatusStrategy{NewStrategy(typer)}
}

func (daemonSetStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apinet.ironcore.dev/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (daemonSetStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newDaemonSet := obj.(*core.DaemonSet)
	oldDaemonSet := old.(*core.DaemonSet)
	newDaemonSet.Spec = oldDaemonSet.Spec
}

func (daemonSetStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newDaemonSet := obj.(*core.DaemonSet)
	oldDaemonSet := old.(*core.DaemonSet)
	return validation.ValidateDaemonSetStatusUpdate(newDaemonSet, oldDaemonSet)
}

func (daemonSetStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
