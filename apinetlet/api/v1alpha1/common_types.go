// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"
	"strings"
)

func SourceNameLabel(kind string) string {
	return fmt.Sprintf("apinetlet.ironcore.dev/%s-name", strings.ToLower(kind))
}

func SourceNamespaceLabel(kind string) string {
	return fmt.Sprintf("apinetlet.ironcore.dev/%s-namespace", strings.ToLower(kind))
}

func SourceUIDLabel(kind string) string {
	return fmt.Sprintf("apinetlet.ironcore.dev/%s-uid", strings.ToLower(kind))
}
