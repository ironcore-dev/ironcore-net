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

package networkinterface

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
		"apinet.api.onmetal.de/v1alpha1": fieldpath.NewSet(
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
