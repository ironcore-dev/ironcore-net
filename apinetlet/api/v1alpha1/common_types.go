// Copyright 2022 IronCore authors
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
