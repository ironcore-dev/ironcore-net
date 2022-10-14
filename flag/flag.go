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

package flag

import (
	"bytes"
	"encoding/csv"
	"flag"
	"io"
	"net/netip"
	"strings"

	"go4.org/netipx"
)

func readAsCSV(val string) ([]string, error) {
	if val == "" {
		return []string{}, nil
	}
	stringReader := strings.NewReader(val)
	csvReader := csv.NewReader(stringReader)
	return csvReader.Read()
}

func writeAsCSV(vals []string) (string, error) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	err := w.Write(vals)
	if err != nil {
		return "", err
	}
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n"), nil
}

type ipPrefixesVar struct {
	value   *[]netip.Prefix
	changed bool
}

func newIPPrefixesVar(val []netip.Prefix, p *[]netip.Prefix) *ipPrefixesVar {
	v := new(ipPrefixesVar)
	v.value = p
	*p = val
	return v
}

func (v *ipPrefixesVar) Set(val string) error {
	// remove all quote characters
	rmQuote := strings.NewReplacer(`"`, "", `'`, "", "`", "")

	// read flag arguments with CSV parser
	ipPrefixStrSlice, err := readAsCSV(rmQuote.Replace(val))
	if err != nil && err != io.EOF {
		return err
	}

	out := make([]netip.Prefix, 0, len(ipPrefixStrSlice))
	for _, ipPrefixStr := range ipPrefixStrSlice {
		prefix := netip.MustParsePrefix(ipPrefixStr)
		out = append(out, prefix)
	}

	if !v.changed {
		*v.value = out
	} else {
		*v.value = append(*v.value, out...)
	}

	v.changed = true
	return nil
}

func (v *ipPrefixesVar) Type() string {
	return "ipPrefixSlice"
}

func (v *ipPrefixesVar) String() string {
	ipPrefixStrSlice := make([]string, 0, len(*v.value))
	for _, prefix := range *v.value {
		ipPrefixStrSlice = append(ipPrefixStrSlice, prefix.String())
	}

	out, _ := writeAsCSV(ipPrefixStrSlice)
	return "[" + out + "]"
}

func IPPrefixesVar(p *[]netip.Prefix, name string, value []netip.Prefix, usage string) {
	flag.Var(newIPPrefixesVar(value, p), name, usage)
}

func IPSetFromPrefixes(prefixes []netip.Prefix) (*netipx.IPSet, error) {
	var bldr netipx.IPSetBuilder
	for _, prefix := range prefixes {
		bldr.AddPrefix(prefix)
	}

	return bldr.IPSet()
}
