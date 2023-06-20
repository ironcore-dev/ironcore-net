// Copyright 2023 OnMetal authors
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

package natgatewayautoscaler

import (
	"context"
	"fmt"

	"github.com/onmetal/onmetal-api-net/internal/apis/core"
	"github.com/onmetal/onmetal-api-net/internal/apis/core/validation"
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
	natGatewayAutoscaler, ok := obj.(*core.NATGatewayAutoscaler)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a NATGatewayAutoscaler")
	}
	return natGatewayAutoscaler.Labels, SelectableFields(natGatewayAutoscaler), nil
}

func MatchNATGatewayAutoscaler(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(natGatewayAutoscaler *core.NATGatewayAutoscaler) fields.Set {
	return generic.ObjectMetaFieldsSet(&natGatewayAutoscaler.ObjectMeta, true)
}

type natGatewayAutoscalerStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) natGatewayAutoscalerStrategy {
	return natGatewayAutoscalerStrategy{typer, names.SimpleNameGenerator}
}

func (natGatewayAutoscalerStrategy) NamespaceScoped() bool {
	return true
}

func (natGatewayAutoscalerStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (natGatewayAutoscalerStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (natGatewayAutoscalerStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	natGatewayAutoscaler := obj.(*core.NATGatewayAutoscaler)
	return validation.ValidateNATGatewayAutoscaler(natGatewayAutoscaler)
}

func (natGatewayAutoscalerStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (natGatewayAutoscalerStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (natGatewayAutoscalerStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (natGatewayAutoscalerStrategy) Canonicalize(obj runtime.Object) {
}

func (natGatewayAutoscalerStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNATGatewayAutoscaler := obj.(*core.NATGatewayAutoscaler)
	oldNATGatewayAutoscaler := old.(*core.NATGatewayAutoscaler)
	return validation.ValidateNATGatewayAutoscalerUpdate(newNATGatewayAutoscaler, oldNATGatewayAutoscaler)
}

func (natGatewayAutoscalerStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type natGatewayAutoscalerStatusStrategy struct {
	natGatewayAutoscalerStrategy
}

func NewStatusStrategy(typer runtime.ObjectTyper) natGatewayAutoscalerStatusStrategy {
	return natGatewayAutoscalerStatusStrategy{NewStrategy(typer)}
}

func (natGatewayAutoscalerStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apinet.api.onmetal.de/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (natGatewayAutoscalerStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newNATGatewayAutoscaler := obj.(*core.NATGatewayAutoscaler)
	oldNATGatewayAutoscaler := old.(*core.NATGatewayAutoscaler)
	newNATGatewayAutoscaler.Spec = oldNATGatewayAutoscaler.Spec
}

func (natGatewayAutoscalerStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNATGatewayAutoscaler := obj.(*core.NATGatewayAutoscaler)
	oldNATGatewayAutoscaler := old.(*core.NATGatewayAutoscaler)
	return validation.ValidateNATGatewayAutoscalerStatusUpdate(newNATGatewayAutoscaler, oldNATGatewayAutoscaler)
}

func (natGatewayAutoscalerStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
