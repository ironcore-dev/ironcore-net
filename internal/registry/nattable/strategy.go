// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package nattable

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
	natTable, ok := obj.(*core.NATTable)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a NATTable")
	}
	return natTable.Labels, SelectableFields(natTable), nil
}

func MatchNATTable(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(natTable *core.NATTable) fields.Set {
	return generic.ObjectMetaFieldsSet(&natTable.ObjectMeta, true)
}

type natTableStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) natTableStrategy {
	return natTableStrategy{typer, names.SimpleNameGenerator}
}

func (natTableStrategy) NamespaceScoped() bool {
	return true
}

func (natTableStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (natTableStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (natTableStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	natTable := obj.(*core.NATTable)
	return validation.ValidateNATTable(natTable)
}

func (natTableStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (natTableStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (natTableStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (natTableStrategy) Canonicalize(obj runtime.Object) {
}

func (natTableStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNATTable := obj.(*core.NATTable)
	oldNATTable := old.(*core.NATTable)
	return validation.ValidateNATTableUpdate(newNATTable, oldNATTable)
}

func (natTableStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
