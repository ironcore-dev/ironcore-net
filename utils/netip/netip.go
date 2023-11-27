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

package netip

import (
	"fmt"
	"math"
	"math/big"
	"net/netip"
)

func PrefixSize(p netip.Prefix) int64 {
	ones, bits := p.Bits(), p.Addr().BitLen()
	if bits == 32 && (bits-ones) >= 31 || bits == 128 && (bits-ones) >= 127 {
		return 0
	}
	// this checks that we are not overflowing an int64
	if bits-ones >= 63 {
		return math.MaxInt64
	}
	return int64(1) << uint(bits-ones)
}

func AddOffsetAddress(address netip.Addr, offset uint64) (netip.Addr, error) {
	addressBig := big.NewInt(0).SetBytes(address.AsSlice())
	r := big.NewInt(0).Add(addressBig, big.NewInt(int64(offset)))
	addr, ok := netip.AddrFromSlice(r.Bytes())
	if !ok {
		return netip.Addr{}, fmt.Errorf("invalid address %v", r.Bytes())
	}
	return addr, nil
}
