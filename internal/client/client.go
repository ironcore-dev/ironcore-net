// Copyright 2023 OnMetal authors
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

	"github.com/onmetal/onmetal-api-net/api/core/v1alpha1"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ClaimNetworkInterfaceNAT(
	ctx context.Context,
	c client.Client,
	nic *v1alpha1.NetworkInterface,
	ipFamily corev1.IPFamily,
	claimRef v1alpha1.NetworkInterfaceNATClaimRef,
) error {
	if existing := v1alpha1.GetNetworkInterfaceNATClaimer(nic, ipFamily); existing != nil {
		return fmt.Errorf("cannot claim a claimed network interface")
	}

	var (
		key         = client.ObjectKeyFromObject(nic)
		uid         = nic.UID
		needGet     bool
		newConflict = func() error {
			return apierrors.NewConflict(schema.GroupResource{
				Group:    v1alpha1.GroupName,
				Resource: "networkinterfaces",
			}, key.Name, fmt.Errorf("network interface has been modified, please modify and reapply your changes"))
		}
		lastErr error
	)
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (done bool, err error) {
		if needGet {
			if err := c.Get(ctx, key, nic); err != nil {
				return false, err
			}
		}
		needGet = true

		if nic.UID != uid {
			return false, newConflict()
		}
		if existing := v1alpha1.GetNetworkInterfaceNATClaimer(nic, ipFamily); existing != nil {
			if *existing == claimRef {
				// Somehow it was claimed correctly - take it as it is.
				return true, nil
			}

			// Taken by different claimer in the meantime.
			return false, newConflict()
		}

		base := nic.DeepCopy()
		nic.Spec.NATs = append(nic.Spec.NATs, v1alpha1.NetworkInterfaceNAT{IPFamily: ipFamily, ClaimRef: claimRef})
		if err := c.Patch(ctx, nic, client.MergeFromWithOptions(base, &client.MergeFromWithOptimisticLock{})); err != nil {
			if !apierrors.IsConflict(err) {
				// No conflict - return immediately.
				return false, err
			}

			lastErr = err
			return false, nil
		}
		// Successfully patched and claimed.
		return true, nil
	})
	if wait.Interrupted(err) {
		return lastErr
	}
	return err
}

// ReleaseNetworkInterfaceNAT releases an IP address NAT.
// Since we use CRDs, we unfortunately cannot implement a custom 'release-Verb' that would handle
// the correct release procedure in the API server. To work around this, this method acts on the conflict
// that might arise when deleting / patching the network interface.
func ReleaseNetworkInterfaceNAT(ctx context.Context, c client.Client, nic *v1alpha1.NetworkInterface, ipFamily corev1.IPFamily) error {
	existing := v1alpha1.GetNetworkInterfaceNATClaimer(nic, ipFamily)
	if existing == nil {
		// Short circuit if no claim ref is specified.
		return nil
	}

	// Store key, UID and claim ref in order not to alter any other IP address.
	var (
		key          = client.ObjectKeyFromObject(nic)
		uid          = nic.UID
		initClaimRef = *existing
		needGet      bool
	)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if needGet {
			if err := c.Get(ctx, key, nic); err != nil {
				return err
			}
		}
		needGet = true

		if nic.UID != uid {
			// UID has changed - different object counts as released.
			return nil
		}

		if actualClaimRef := v1alpha1.GetNetworkInterfaceNATClaimer(nic, ipFamily); actualClaimRef == nil || *actualClaimRef != initClaimRef {
			// Claimer has already changed, counts as released.
			return nil
		}

		idx := slices.IndexFunc(nic.Spec.NATs, func(nicNAT v1alpha1.NetworkInterfaceNAT) bool { return nicNAT.IPFamily == ipFamily })
		nic.Spec.NATs = slices.Delete(nic.Spec.NATs, idx, idx+1)
		if err := c.Patch(ctx, nic, client.MergeFromWithOptions(nic, &client.MergeFromWithOptimisticLock{})); err != nil {
			return fmt.Errorf("error releasing network interface: %w", err)
		}
		return nil
	})
}
