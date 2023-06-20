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
