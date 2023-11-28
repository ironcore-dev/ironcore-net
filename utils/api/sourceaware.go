// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package api

import "strings"

type SourceAwareSystem struct {
	SourceUIDLabel       func(kind string) string
	SourceNameLabel      func(kind string) string
	SourceNamespaceLabel func(kind string) string
}

func PrefixSourceAwareSystem(prefix string) SourceAwareSystem {
	mkLabelFunc := func(name string) func(kind string) string {
		return func(kind string) string {
			var sb strings.Builder
			// "<prefix><kind>-<name>"
			sb.Grow(len(prefix) + len(kind) + 1 + len(name))
			sb.WriteString(prefix)
			sb.WriteString(strings.ToLower(kind))
			sb.WriteRune('-')
			sb.WriteString(name)
			return sb.String()
		}
	}

	return SourceAwareSystem{
		SourceUIDLabel:       mkLabelFunc("uid"),
		SourceNameLabel:      mkLabelFunc("name"),
		SourceNamespaceLabel: mkLabelFunc("namespace"),
	}
}
