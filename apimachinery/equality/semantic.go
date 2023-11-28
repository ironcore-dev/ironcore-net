// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package equality

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/conversion"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/third_party/forked/golang/reflect"
)

// Semantic checks whether ironcore types are semantically equal.
// It uses equality.Semantic as baseline and adds custom functions on top.
var Semantic conversion.Equalities

func init() {
	base := make(reflect.Equalities)
	for k, v := range equality.Semantic.Equalities {
		base[k] = v
	}
	Semantic = conversion.Equalities{Equalities: base}
	utilruntime.Must(AddFuncs(Semantic))
}

func AddFuncs(equality conversion.Equalities) error {
	return equality.AddFuncs(
		func(a, b net.IP) bool {
			return a == b
		},
		func(a, b net.IPPrefix) bool {
			return a == b
		},
	)
}
