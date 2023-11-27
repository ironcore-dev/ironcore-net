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

package ip

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
	"k8s.io/client-go/tools/cache"
)

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	ip, ok := obj.(*core.IP)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a IP")
	}
	return ip.Labels, SelectableFields(ip), nil
}

func MatchIP(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:       label,
		Field:       field,
		GetAttrs:    GetAttrs,
		IndexFields: []string{"spec.ip"},
	}
}

func SelectableFields(ip *core.IP) fields.Set {
	fieldSet := fields.Set{
		"spec.ip": ip.Spec.IP.String(),
	}
	return generic.AddObjectMetaFieldsSet(fieldSet, &ip.ObjectMeta, true)
}

func IPTriggerFunc(obj runtime.Object) string {
	return obj.(*core.IP).Spec.IP.String()
}

func IPIndexFunc(obj any) ([]string, error) {
	ip, ok := obj.(*core.IP)
	if !ok {
		return nil, fmt.Errorf("not an ip")
	}
	return []string{ip.Spec.IP.String()}, nil
}

func Indexers() *cache.Indexers {
	return &cache.Indexers{
		apisrvstorage.FieldIndex("spec.ip"): IPIndexFunc,
	}
}

type ipStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) ipStrategy {
	return ipStrategy{typer, names.SimpleNameGenerator}
}

func (ipStrategy) NamespaceScoped() bool {
	return true
}

func (ipStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (ipStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (ipStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	ip := obj.(*core.IP)
	return validation.ValidateIP(ip)
}

func (ipStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (ipStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (ipStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (ipStrategy) Canonicalize(obj runtime.Object) {
}

func (ipStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newIP := obj.(*core.IP)
	oldIP := old.(*core.IP)
	return validation.ValidateIPUpdate(newIP, oldIP)
}

func (ipStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
