// Copyright 2022 OnMetal authors
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

package controllers

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	onmetalapinetv1alpha1 "github.com/onmetal/onmetal-api-net/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IPFamilyBitLen(ipFamily corev1.IPFamily) uint8 {
	switch ipFamily {
	case corev1.IPv4Protocol:
		return 32
	case corev1.IPv6Protocol:
		return 128
	default:
		return 0
	}
}

func APINetV1Alpha1IPsToNetIPAddrs(ips []onmetalapinetv1alpha1.IP) []netip.Addr {
	res := make([]netip.Addr, len(ips))
	for i, ip := range ips {
		res[i] = ip.Addr
	}
	return res
}

func NetIPAddrsToAPINetV1Alpha1IPs(addrs []netip.Addr) []onmetalapinetv1alpha1.IP {
	res := make([]onmetalapinetv1alpha1.IP, len(addrs))
	for i, addr := range addrs {
		res[i] = onmetalapinetv1alpha1.IP{Addr: addr}
	}
	return res
}

func PatchAddReconcileAnnotation(ctx context.Context, c client.Client, obj client.Object) error {
	base := obj.DeepCopyObject().(client.Object)

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[onmetalapinetv1alpha1.ReconcileRequestAnnotation] = time.Now().Format(time.RFC3339Nano)
	obj.SetAnnotations(annotations)

	if err := c.Patch(ctx, obj, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error adding reconcile annotation: %w", err)
	}
	return nil
}

func PatchRemoveReconcileAnnotation(ctx context.Context, c client.Client, obj client.Object) error {
	base := obj.DeepCopyObject().(client.Object)

	annotations := obj.GetAnnotations()
	delete(annotations, onmetalapinetv1alpha1.ReconcileRequestAnnotation)

	if err := c.Patch(ctx, obj, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("error removing reconcile annotation: %w", err)
	}
	return nil
}
