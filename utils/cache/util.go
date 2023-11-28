// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
