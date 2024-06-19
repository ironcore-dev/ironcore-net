// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package netip

import (
	"fmt"
	"math"
	"math/big"
	"net/netip"

	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	corev1 "k8s.io/api/core/v1"
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

func GetIPFamilyFromPrefix(ipPrefix net.IPPrefix) corev1.IPFamily {
	if ipPrefix.Addr().Is6() {
		return corev1.IPv6Protocol
	}
	return corev1.IPv4Protocol
}
