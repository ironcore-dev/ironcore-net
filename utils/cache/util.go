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

package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PollGetInformer(ctx context.Context, c cache.Cache, object client.Object) (cache.Informer, error) {
	var (
		log     = ctrl.LoggerFrom(ctx)
		i       cache.Informer
		lastErr error
	)
	// Tries to get an informer until it returns true,
	// an error or the specified context is cancelled or expired.
	if err := wait.PollUntilContextCancel(ctx, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		// Lookup the Informer from the Cache and add an EventHandler which populates the Queue
		i, lastErr = c.GetInformer(ctx, object)
		if lastErr != nil {
			kindMatchErr := &meta.NoKindMatchError{}
			switch {
			case errors.As(lastErr, &kindMatchErr):
				log.Error(lastErr, "if kind is a CRD, it should be installed before calling Start",
					"kind", kindMatchErr.GroupKind)
			case runtime.IsNotRegisteredError(lastErr):
				log.Error(lastErr, "kind must be registered to the Scheme")
			default:
				log.Error(lastErr, "failed to get informer from cache")
			}
			return false, nil // Retry.
		}
		return true, nil
	}); err != nil {
		if lastErr != nil {
			return nil, fmt.Errorf("failed to get informer from cache: %w", lastErr)
		}
		return nil, err
	}
	return i, nil
}
