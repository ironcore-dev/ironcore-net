// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkpolicyrule

import (
	"context"

	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

type NetworkPolicyRuleStorage struct {
	NetworkPolicyRule *REST
}

type REST struct {
	*genericregistry.Store
}

func NewStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (NetworkPolicyRuleStorage, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.NetworkPolicyRule{}
		},
		NewListFunc: func() runtime.Object {
			return &core.NetworkPolicyRuleList{}
		},
		PredicateFunc:             MatchNetworkPolicyRule,
		DefaultQualifiedResource:  core.Resource("networkpolicyrules"),
		SingularQualifiedResource: core.Resource("networkpolicyrule"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return NetworkPolicyRuleStorage{}, err
	}

	return NetworkPolicyRuleStorage{
		NetworkPolicyRule: &REST{store},
	}, nil
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &core.NetworkPolicyRule{}
}

func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

func (r *StatusREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

func (r *StatusREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return r.store.GetResetFields()
}

func (r *StatusREST) Destroy() {}
