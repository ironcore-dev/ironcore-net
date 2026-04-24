// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/ironcore-dev/controller-utils/metautils"
	"github.com/ironcore-dev/ironcore-net/utils/origin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StemmingFrom struct {
	Origin *origin.Origin
	Source client.Object
}

func (s *StemmingFrom) ApplyToDeleteAllOf(opts *client.DeleteAllOfOptions) {
	client.MatchingLabels(s.Origin.Labels(s.Source)).ApplyToDeleteAllOf(opts)
}

func (s *StemmingFrom) ApplyToList(opts *client.ListOptions) {
	client.MatchingLabels(s.Origin.Labels(s.Source)).ApplyToList(opts)
}

func (s *StemmingFrom) Matches(obj client.Object) bool {
	return s.Origin.StemsFrom(obj, s.Source)
}

type StemmingFromKey struct {
	Origin    *origin.Origin
	SourceKey client.ObjectKey
}

func (s *StemmingFromKey) selector() client.MatchingLabelsSelector {
	sel := labels.NewSelector()

	sourceUIDLabelExistsReq, err := labels.NewRequirement(s.Origin.UIDLabelKey(), selection.Exists, nil)
	if err != nil {
		panic(fmt.Errorf("creating source uid exists requirement: %w", err))
	}

	sel = sel.Add(*sourceUIDLabelExistsReq)

	if s.Origin.Namespaced {
		sourceNamespaceReq, err := labels.NewRequirement(s.Origin.NamespaceLabelKey(), selection.Equals, []string{s.SourceKey.Namespace})
		if err != nil {
			panic(fmt.Errorf("creating source namespace requirement: %w", err))
		}

		sel = sel.Add(*sourceNamespaceReq)
	}

	return client.MatchingLabelsSelector{Selector: sel}
}

func (s *StemmingFromKey) ApplyToDeleteAllOf(opts *client.DeleteAllOfOptions) {
	s.selector().ApplyToDeleteAllOf(opts)
}

func (s *StemmingFromKey) ApplyToList(opts *client.ListOptions) {
	s.selector().ApplyToList(opts)
}

func (s *StemmingFromKey) Matches(obj client.Object) bool {
	return s.Origin.StemsFromKey(obj, s.SourceKey)
}

func AnyExists(ctx context.Context, c client.Client, obj client.Object, opts ...client.ListOption) (bool, error) {
	_, list, err := metautils.NewListForObject(c.Scheme(), obj)
	if err != nil {
		return false, fmt.Errorf("error creating new list for object: %w", err)
	}

	o := &client.ListOptions{}
	o.ApplyOptions(opts)
	o.Limit = 1

	for {
		if err := c.List(ctx, list, o); err != nil {
			return false, err
		}
		if meta.LenList(list) > 0 {
			// If we see at least one element, something still exists.
			return true, nil
		}
		if o.Continue == "" {
			// If we cannot continue doing List requests, return false (nothing exists)
			return false, nil
		}
	}
}

func ListAnd(c client.Client, list client.ObjectList, opts ...client.ListOption) *ListAndOperation {
	return &ListAndOperation{
		client: c,
		list:   list,
		opts:   opts,
	}
}

type ListAndOperation struct {
	client client.Client
	list   client.ObjectList
	opts   []client.ListOption
}

type ObjectPredicate interface {
	Matches(obj client.Object) bool
}

func (o *ListAndOperation) DeletePredicate(ctx context.Context, pred ObjectPredicate) (n int, err error) {
	if err := o.client.List(ctx, o.list, o.opts...); err != nil {
		return 0, fmt.Errorf("list: %w", err)
	}

	var errs []error
	if err := metautils.EachListItem(o.list, func(obj client.Object) error {
		if !pred.Matches(obj) {
			return nil
		}

		if err := o.client.Delete(ctx, obj); err != nil {
			if !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("delete %s: %w", client.ObjectKeyFromObject(obj), err))
			}
			return nil
		}

		n++
		return nil
	}); err != nil {
		errs = append(errs, fmt.Errorf("iterating list: %w", err))
	}
	return n, errors.Join(errs...)
}
