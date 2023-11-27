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

package node

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
	node, ok := obj.(*core.Node)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a Node")
	}
	return node.Labels, SelectableFields(node), nil
}

func MatchNode(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(node *core.Node) fields.Set {
	return generic.ObjectMetaFieldsSet(&node.ObjectMeta, true)
}

type nodeStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) nodeStrategy {
	return nodeStrategy{typer, names.SimpleNameGenerator}
}

func (nodeStrategy) NamespaceScoped() bool {
	return false
}

func (nodeStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (nodeStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (nodeStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	node := obj.(*core.Node)
	return validation.ValidateNode(node)
}

func (nodeStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (nodeStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (nodeStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (nodeStrategy) Canonicalize(obj runtime.Object) {
}

func (nodeStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNode := obj.(*core.Node)
	oldNode := old.(*core.Node)
	return validation.ValidateNodeUpdate(newNode, oldNode)
}

func (nodeStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type nodeStatusStrategy struct {
	nodeStrategy
}

func NewStatusStrategy(typer runtime.ObjectTyper) nodeStatusStrategy {
	return nodeStatusStrategy{NewStrategy(typer)}
}

func (nodeStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apinet.ironcore.dev/v1alpha1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (nodeStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newNode := obj.(*core.Node)
	oldNode := old.(*core.Node)
	newNode.Spec = oldNode.Spec
}

func (nodeStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newNode := obj.(*core.Node)
	oldNode := old.(*core.Node)
	return validation.ValidateNodeStatusUpdate(newNode, oldNode)
}

func (nodeStatusStrategy) WarningsOnUpdate(cxt context.Context, obj, old runtime.Object) []string {
	return nil
}
