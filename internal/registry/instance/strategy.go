// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package instance

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
	instance, ok := obj.(*core.Instance)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a Instance")
	}
	return instance.Labels, SelectableFields(instance), nil
}

func MatchInstance(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(instance *core.Instance) fields.Set {
	return generic.ObjectMetaFieldsSet(&instance.ObjectMeta, true)
}

type instanceStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) instanceStrategy {
	return instanceStrategy{typer, names.SimpleNameGenerator}
}

func (instanceStrategy) NamespaceScoped() bool {
	return true
}

func (instanceStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (instanceStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (instanceStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	instance := obj.(*core.Instance)
	return validation.ValidateInstance(instance)
}

func (instanceStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (instanceStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (instanceStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (instanceStrategy) Canonicalize(obj runtime.Object) {
}

func (instanceStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newInstance := obj.(*core.Instance)
	oldInstance := old.(*core.Instance)
	return validation.ValidateInstanceUpdate(newInstance, oldInstance)
}

func (instanceStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type instanceStatusStrategy struct {
	instanceStrategy
}

func NewStatusStrategy(typer runtime.ObjectTyper) instanceStatusStrategy {
	return instanceStatusStrategy{NewStrategy(typer)}
}

func (instanceStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apinet.ironcore.dev/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (instanceStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newInstance := obj.(*core.Instance)
	oldInstance := old.(*core.Instance)
	newInstance.Spec = oldInstance.Spec
}

func (instanceStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newInstance := obj.(*core.Instance)
	oldInstance := old.(*core.Instance)
	return validation.ValidateInstanceStatusUpdate(newInstance, oldInstance)
}

func (instanceStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
