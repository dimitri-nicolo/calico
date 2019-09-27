// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v3_test

import (
	. "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/set"
)

var (
	// stagednpExtraFields is the set of fields that should be in StagedNetworkPolicy but not
	// NetworkPolicy.
	stagednpExtraFields = set.From("StagedAction")

	// networkPolicyExtraFields is the set of fields that should be in NetworkPolicy but not
	// StagedNetworkPolicy.
	networkPolicyExtraFields = set.From()
)

// These tests verify that the StagedNetworkPolicySpec struct and the NetworkPolicySpec struct
// are kept in sync.
var _ = Describe("StagedGlobalNetworkPolicySpec", func() {
	var snpFieldsByName map[string]reflect.StructField
	var npFieldsByName map[string]reflect.StructField

	BeforeEach(func() {
		snpFieldsByName = fieldsByName(StagedNetworkPolicySpec{})
		npFieldsByName = fieldsByName(NetworkPolicySpec{})
	})

	It("and NetworkPolicySpec shared fields should have the same tags", func() {
		for n, f := range snpFieldsByName {
			if gf, ok := npFieldsByName[n]; ok {
				if f.Name != "Selector" { //selector tags are not same. selector is not required for staged policy
					Expect(f.Tag).To(Equal(gf.Tag), "Field "+n+" had different tag")
				}
			}
		}
	})

	It("and NetworkPolicySpec shared fields should have the same types", func() {
		for n, f := range snpFieldsByName {
			if gf, ok := npFieldsByName[n]; ok {
				Expect(f.Type).To(Equal(gf.Type), "Field "+n+" had different type")
			}
		}
	})

	It("should not have any unexpected fields that NetworkPolicySpec doesn't have", func() {
		for n := range snpFieldsByName {
			if stagednpExtraFields.Contains(n) {
				continue
			}
			Expect(npFieldsByName).To(HaveKey(n))
		}
	})

	It("should contain all expected fields of NetworkPolicySpec", func() {
		for n := range npFieldsByName {
			if networkPolicyExtraFields.Contains(n) {
				continue
			}
			Expect(snpFieldsByName).To(HaveKey(n))
		}
	})
})
