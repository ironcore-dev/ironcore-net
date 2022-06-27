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
	"fmt"
	"io"
	"strings"

	"inet.af/netaddr"
	corev1 "k8s.io/api/core/v1"
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
	value   *[]netaddr.IPPrefix
	changed bool
}

func newIPPrefixesVar(val []netaddr.IPPrefix, p *[]netaddr.IPPrefix) *ipPrefixesVar {
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

	out := make([]netaddr.IPPrefix, 0, len(ipPrefixStrSlice))
	for _, ipPrefixStr := range ipPrefixStrSlice {
		prefix, err := netaddr.ParseIPPrefix(ipPrefixStr)
		if err != nil {
			return err
		}

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

func IPPrefixesVar(p *[]netaddr.IPPrefix, name string, value []netaddr.IPPrefix, usage string) {
	flag.Var(newIPPrefixesVar(value, p), name, usage)
}

func IPFamilySetFromPrefixes(ipFamily corev1.IPFamily, prefixes []netaddr.IPPrefix) (*netaddr.IPSet, error) {
	if len(prefixes) == 0 {
		return nil, nil
	}

	var validatePrefix func(prefix netaddr.IPPrefix) error
	switch ipFamily {
	case corev1.IPv4Protocol:
		validatePrefix = func(prefix netaddr.IPPrefix) error {
			if !prefix.IP().Is4() {
				return fmt.Errorf("invalid non ipv4-prefix: %s", prefix)
			}
			return nil
		}
	case corev1.IPv6Protocol:
		validatePrefix = func(prefix netaddr.IPPrefix) error {
			if !prefix.IP().Is6() {
				return fmt.Errorf("invalid non ipv6-prefix: %s", prefix)
			}
			return nil
		}
	default:
		return nil, fmt.Errorf("invalid ip family %s", ipFamily)
	}

	var bldr netaddr.IPSetBuilder
	for _, prefix := range prefixes {
		if err := validatePrefix(prefix); err != nil {
			return nil, err
		}

		bldr.AddPrefix(prefix)
	}

	return bldr.IPSet()
}
