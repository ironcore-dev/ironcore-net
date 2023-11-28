// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package flag

import (
	"bytes"
	"encoding/csv"
	"io"
	"net/netip"
	"strings"

	"github.com/spf13/pflag"
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

func IPPrefixesVar(fs *pflag.FlagSet, p *[]netip.Prefix, name string, value []netip.Prefix, usage string) {
	fs.Var(newIPPrefixesVar(value, p), name, usage)
}

func IPSetFromPrefixes(prefixes []netip.Prefix) (*netipx.IPSet, error) {
	var bldr netipx.IPSetBuilder
	for _, prefix := range prefixes {
		bldr.AddPrefix(prefix)
	}

	return bldr.IPSet()
}
