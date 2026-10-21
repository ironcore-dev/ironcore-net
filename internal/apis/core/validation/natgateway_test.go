// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/internal/apis/core/validation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("NATGateway", func() {
	DescribeTable("ValidateNATGatewaySpec portsPerNetworkInterface",
		func(ports int32, match types.GomegaMatcher) {
			spec := &core.NATGatewaySpec{
				IPFamily:                 corev1.IPv4Protocol,
				PortsPerNetworkInterface: ports,
			}
			Expect(validation.ValidateNATGatewaySpec(spec, field.NewPath("spec"))).To(match)
		},
		Entry("zero", int32(0), ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeInvalid),
			"Field": Equal("spec.portsPerNetworkInterface"),
		})))),
		Entry("negative", int32(-1), ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeInvalid),
			"Field": Equal("spec.portsPerNetworkInterface"),
		})))),
		Entry("too large", int32(64513), ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeInvalid),
			"Field": Equal("spec.portsPerNetworkInterface"),
		})))),
		Entry("min int32", int32(-2147483648), Not(BeEmpty())),
		Entry("max int32", int32(2147483647), Not(BeEmpty())),
		Entry("lower bound", int32(1), BeEmpty()),
		Entry("typical", int32(1024), BeEmpty()),
		Entry("upper bound", int32(64512), BeEmpty()),
	)
})
