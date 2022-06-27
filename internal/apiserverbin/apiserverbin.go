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

package apiserverbin

import (
	"path/filepath"
	"runtime"

	"github.com/onmetal/controller-utils/modutils"
)

var Path string

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("apiserverbin: unable to determine filename")
	}

	Path = filepath.Join(filename, "..", "..", "..", "testbin", "apiserver")
	modutils.Build(Path, "github.com/onmetal/onmetal-api", "cmd", "apiserver")
}
