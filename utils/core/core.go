// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"net/netip"

	corev1 "k8s.io/api/core/v1"
)

func IPFamilyForAddr(addr netip.Addr) corev1.IPFamily {
	switch {
	case addr.Is4():
		return corev1.IPv4Protocol
	case addr.Is6():
		return corev1.IPv6Protocol
	default:
		return ""
	}
}
