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
