// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/controller-utils/metautils"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteAllOfAndAnyExists(ctx context.Context, c client.Client, obj client.Object, opts ...client.DeleteAllOfOption) (bool, error) {
	o := &client.DeleteAllOfOptions{}
	o.ApplyOptions(opts)

	if err := c.DeleteAllOf(ctx, obj, o); err != nil {
		return false, err
	}

	return AnyExists(ctx, c, obj, o)
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
