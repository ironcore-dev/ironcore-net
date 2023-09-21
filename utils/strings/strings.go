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

package strings

import (
	"fmt"
	"strings"
)

type Joiner struct {
	sb      strings.Builder
	needSep bool
	sep     string
}

func NewJoiner(sep string) *Joiner {
	return &Joiner{
		sep: sep,
	}
}

func (j *Joiner) Add(v any) {
	if j.needSep {
		j.sb.WriteString(j.sep)
	}
	j.needSep = true
	_, _ = fmt.Fprint(&j.sb, v)
}

func (j *Joiner) String() string {
	return j.sb.String()
}

func (j *Joiner) Reset() {
	j.sb.Reset()
	j.needSep = false
}
