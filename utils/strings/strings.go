// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
