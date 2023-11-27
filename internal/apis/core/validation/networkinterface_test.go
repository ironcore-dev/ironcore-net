// Copyright 2023 IronCore authors
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
