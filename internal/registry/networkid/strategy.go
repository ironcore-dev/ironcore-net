// Copyright 2023 IronCore authors
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

package networkid

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
	networkID, ok := obj.(*core.NetworkID)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a NetworkID")
	}
	return networkID.Labels, SelectableFields(networkID), nil
}

func MatchNetworkID(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(networkID *core.NetworkID) fields.Set {
	return generic.ObjectMetaFieldsSet(&networkID.ObjectMeta, true)
}

type networkIDStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) networkIDStrategy {
	return networkIDStrategy{typer, names.SimpleNameGenerator}
}

func (networkIDStrategy) NamespaceScoped() bool {
	return false
}

func (networkIDStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (networkIDStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (networkIDStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	networkID := obj.(*core.NetworkID)
	return validation.ValidateNetworkID(networkID)
}

func (networkIDStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (networkIDStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (networkIDStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (networkIDStrategy) Canonicalize(obj runtime.Object) {
}

func (networkIDStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNetworkID := obj.(*core.NetworkID)
	oldNetworkID := old.(*core.NetworkID)
	return validation.ValidateNetworkIDUpdate(newNetworkID, oldNetworkID)
}

func (networkIDStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
