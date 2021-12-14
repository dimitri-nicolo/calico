// Copyright (c) 2016-2017,2020 Tigera, Inc. All rights reserved.
//
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

package calc

import (
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/backend/api"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var (
	tenPointFive = 10.5
)

var _ = DescribeTable("PolKV should stringify correctly",
	func(kv PolKV, expected string) {
		Expect(kv.String()).To(Equal(expected))
	},
	Entry("zero", PolKV{}, "(nil policy)"),
	Entry("nil policy", PolKV{Key: model.PolicyKey{Name: "name"}}, "name(nil policy)"),
	Entry("nil order",
		PolKV{Key: model.PolicyKey{Name: "name"}, Value: &model.Policy{}}, "name(default)"),
	Entry("order set",
		PolKV{Key: model.PolicyKey{Name: "name"}, Value: &model.Policy{Order: &tenPointFive}},
		"name(10.5)"),
)

var _ = Describe("PolicySorter", func() {
	It("should clean up when a policy is removed from non-existent tier", func() {
		ps := NewPolicySorter()
		key := model.PolicyKey{Tier: "default", Name: "foo"}
		pol := &model.Policy{}
		ps.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   key,
				Value: pol,
			},
		})
		Expect(ps.tiers["default"].Valid).To(BeFalse())
		Expect(ps.tiers["default"].Policies).To(Equal(map[model.PolicyKey]*model.Policy{
			key: pol,
		}))

		ps.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   key,
				Value: nil, // deletion
			},
		})
		Expect(ps.tiers).NotTo(HaveKey("default"))
	})
	It("should clean up when a tier is removed", func() {
		ps := NewPolicySorter()
		key := model.TierKey{Name: "default"}
		tier := &model.Tier{}
		ps.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   key,
				Value: tier,
			},
		})
		Expect(ps.tiers["default"].Valid).To(BeTrue())
		Expect(ps.tiers["default"].Policies).To(BeEmpty())

		ps.OnUpdate(api.Update{
			KVPair: model.KVPair{
				Key:   key,
				Value: nil, // deletion
			},
		})
		Expect(ps.tiers).NotTo(HaveKey("default"))
	})
})

var _ = DescribeTable("Tiers should sort correctly",
	func(input, expected []*tierInfo) {
		var reversedInput []*tierInfo
		if input != nil {
			reversedInput = make([]*tierInfo, len(input))
			for i, v := range input {
				reversedInput[len(input)-1-i] = v
			}
		}
		sort.Sort(TierByOrder(input))
		Expect(input).To(Equal(expected))
		sort.Sort(TierByOrder(input))
		Expect(input).To(Equal(expected), "sort should be stable")
		sort.Sort(TierByOrder(reversedInput))
		Expect(reversedInput).To(Equal(expected), "sort work after reversing input")
	},
	Entry("nil", []*tierInfo(nil), []*tierInfo(nil)),
	Entry("empty", []*tierInfo{}, []*tierInfo{}),
	Entry("valid sorts ahead of invalid", []*tierInfo{
		{
			Name:  "bar",
			Valid: false,
			Order: floatPtr(1),
		},
		{
			Name:  "foo",
			Valid: true,
			Order: floatPtr(10),
		},
	}, []*tierInfo{
		{
			Name:  "foo",
			Valid: true,
			Order: floatPtr(10),
		},
		{
			Name:  "bar",
			Valid: false,
			Order: floatPtr(1),
		},
	}),
	Entry("both invalid, both nil order rely on name", []*tierInfo{
		{
			Name:  "foo",
			Valid: false,
		},
		{
			Name:  "bar",
			Valid: false,
		},
	}, []*tierInfo{
		{
			Name:  "bar",
			Valid: false,
		},
		{
			Name:  "foo",
			Valid: false,
		},
	}),
	Entry("both valid, both nil order rely on name", []*tierInfo{
		{
			Name:  "foo",
			Valid: true,
		},
		{
			Name:  "bar",
			Valid: true,
		},
	}, []*tierInfo{
		{
			Name:  "bar",
			Valid: true,
		},
		{
			Name:  "foo",
			Valid: true,
		},
	}),
	Entry("both valid, rely on order", []*tierInfo{
		{
			Name:  "bar",
			Order: floatPtr(10),
			Valid: true,
		},
		{
			Name:  "foo",
			Order: floatPtr(1),
			Valid: true,
		},
	}, []*tierInfo{
		{
			Name:  "foo",
			Order: floatPtr(1),
			Valid: true,
		},
		{
			Name:  "bar",
			Order: floatPtr(10),
			Valid: true,
		},
	}),
	Entry("all valid, non-nil orders sort ahead of nil", []*tierInfo{
		{
			Name:  "bar",
			Valid: true,
		},
		{
			Name:  "baz",
			Order: floatPtr(10),
			Valid: true,
		},
		{
			Name:  "foo",
			Order: floatPtr(1),
			Valid: true,
		},
	}, []*tierInfo{
		{
			Name:  "foo",
			Order: floatPtr(1),
			Valid: true,
		},
		{
			Name:  "baz",
			Order: floatPtr(10),
			Valid: true,
		},
		{
			Name:  "bar",
			Valid: true,
		},
	}),
	Entry("all valid, equal order relies on name", []*tierInfo{
		{
			Name:  "baz",
			Order: floatPtr(10),
			Valid: true,
		},
		{
			Name:  "foo",
			Order: floatPtr(10),
			Valid: true,
		},
	}, []*tierInfo{
		{
			Name:  "baz",
			Order: floatPtr(10),
			Valid: true,
		},
		{
			Name:  "foo",
			Order: floatPtr(10),
			Valid: true,
		},
	}),
)

func floatPtr(f float64) *float64 {
	return &f
}
