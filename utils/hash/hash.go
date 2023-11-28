// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package hash

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/util/rand"
)

// ComputeWithCollisionCount computes a hash including an optional collision count of the given data bytes.
func ComputeWithCollisionCount(collisionCount *int32, data ...[]byte) string {
	h := fnv.New32a()

	for _, p := range data {
		_, _ = h.Write(p)
	}

	// Add collisionCount in the hash if it exists
	if collisionCount != nil {
		collisionCountBytes := make([]byte, 8)
		binary.LittleEndian.PutUint32(collisionCountBytes, uint32(*collisionCount))
		_, _ = h.Write(collisionCountBytes)
	}

	return rand.SafeEncodeString(fmt.Sprint(h.Sum32()))
}
