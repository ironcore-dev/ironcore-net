// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ipaddress

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
	ipAddress, ok := obj.(*core.IPAddress)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a IPAddress")
	}
	return ipAddress.Labels, SelectableFields(ipAddress), nil
}

func MatchIPAddress(label labels.Selector, field fields.Selector) apisrvstorage.SelectionPredicate {
	return apisrvstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func SelectableFields(ipAddress *core.IPAddress) fields.Set {
	return generic.ObjectMetaFieldsSet(&ipAddress.ObjectMeta, true)
}

type ipAddressStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) ipAddressStrategy {
	return ipAddressStrategy{typer, names.SimpleNameGenerator}
}

func (ipAddressStrategy) NamespaceScoped() bool {
	return false
}

func (ipAddressStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (ipAddressStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (ipAddressStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	ipAddress := obj.(*core.IPAddress)
	return validation.ValidateIPAddress(ipAddress)
}

func (ipAddressStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (ipAddressStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (ipAddressStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (ipAddressStrategy) Canonicalize(obj runtime.Object) {
}

func (ipAddressStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newIPAddress := obj.(*core.IPAddress)
	oldIPAddress := old.(*core.IPAddress)
	return validation.ValidateIPAddressUpdate(newIPAddress, oldIPAddress)
}

func (ipAddressStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
