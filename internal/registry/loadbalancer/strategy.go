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

package loadbalancer

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
	loadBalancer, ok := obj.(*core.LoadBalancer)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a LoadBalancer")
	}
	return loadBalancer.Labels, SelectableFields(loadBalancer), nil
}

func MatchLoadBalancer(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(loadBalancer *core.LoadBalancer) fields.Set {
	return generic.ObjectMetaFieldsSet(&loadBalancer.ObjectMeta, true)
}

type loadBalancerStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) loadBalancerStrategy {
	return loadBalancerStrategy{typer, names.SimpleNameGenerator}
}

func (loadBalancerStrategy) NamespaceScoped() bool {
	return true
}

func (loadBalancerStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (loadBalancerStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (loadBalancerStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	loadBalancer := obj.(*core.LoadBalancer)
	return validation.ValidateLoadBalancer(loadBalancer)
}

func (loadBalancerStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (loadBalancerStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (loadBalancerStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (loadBalancerStrategy) Canonicalize(obj runtime.Object) {
}

func (loadBalancerStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newLoadBalancer := obj.(*core.LoadBalancer)
	oldLoadBalancer := old.(*core.LoadBalancer)
	return validation.ValidateLoadBalancerUpdate(newLoadBalancer, oldLoadBalancer)
}

func (loadBalancerStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type loadBalancerStatusStrategy struct {
	loadBalancerStrategy
}

func NewStatusStrategy(typer runtime.ObjectTyper) loadBalancerStatusStrategy {
	return loadBalancerStatusStrategy{NewStrategy(typer)}
}

func (loadBalancerStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apinet.api.onmetal.de/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (loadBalancerStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newLoadBalancer := obj.(*core.LoadBalancer)
	oldLoadBalancer := old.(*core.LoadBalancer)
	newLoadBalancer.Spec = oldLoadBalancer.Spec
}

func (loadBalancerStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newLoadBalancer := obj.(*core.LoadBalancer)
	oldLoadBalancer := old.(*core.LoadBalancer)
	return validation.ValidateLoadBalancerStatusUpdate(newLoadBalancer, oldLoadBalancer)
}

func (loadBalancerStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
