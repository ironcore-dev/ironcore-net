// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package node

import (
	"context"

	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

type NodeStorage struct {
	Node   *REST
	Status *StatusREST
}

type REST struct {
	*genericregistry.Store
}

func NewStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (NodeStorage, error) {
	strategy := NewStrategy(scheme)
	statusStrategy := NewStatusStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &core.Node{}
		},
		NewListFunc: func() runtime.Object {
			return &core.NodeList{}
		},
		PredicateFunc:             MatchNode,
		DefaultQualifiedResource:  core.Resource("nodes"),
		SingularQualifiedResource: core.Resource("node"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		TableConvertor: newTableConvertor(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return NodeStorage{}, err
	}

	statusStore := *store
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy

	return NodeStorage{
		Node:   &REST{store},
		Status: &StatusREST{&statusStore},
	}, nil
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &core.Node{}
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
