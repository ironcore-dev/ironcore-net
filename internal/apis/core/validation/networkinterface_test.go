// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/internal/apis/core/validation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("NetworkInterface", func() {
	DescribeTable("ValidateNetworkInterfaceSpecUpdate",
		func(oldSpec, newSpec *core.NetworkInterfaceSpec, match types.GomegaMatcher) {
			allErrs := validation.ValidateNetworkInterfaceSpecUpdate(newSpec, oldSpec, field.NewPath("spec"))
			Expect(allErrs).To(match)
		},
		Entry("ips update",
			&core.NetworkInterfaceSpec{IPs: []net.IP{net.MustParseIP("10.0.0.1")}},
			&core.NetworkInterfaceSpec{IPs: []net.IP{net.MustParseIP("192.168.178.1")}},
			ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.ips"),
				"Detail": Equal(apivalidation.FieldImmutableErrorMsg),
			}))),
		),
	)
})
