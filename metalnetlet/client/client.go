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
