// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package api_test

import (
	"fmt"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/tigera/lma/pkg/api"
)

type testPolicyHit struct {
	action      api.Action
	index       int
	name        string
	flowLogName string
	namespace   string
	tier        string
	isKNP       bool
	isKNS       bool
	isStaged    bool
	count       int64
}

var _ = Describe("PolicyHitFromFlowLogPolicyString", func() {
	DescribeTable("Successful PolicyHit parsing",
		func(policyStr string, docCount int, expectedPolicyHit testPolicyHit) {
			policyHit, err := api.PolicyHitFromFlowLogPolicyString(policyStr, int64(docCount))
			Expect(err).ShouldNot(HaveOccurred())

			Expect(policyHit.Action()).Should(Equal(expectedPolicyHit.action))
			Expect(policyHit.Index()).Should(Equal(expectedPolicyHit.index))
			Expect(policyHit.Tier()).Should(Equal(expectedPolicyHit.tier))
			Expect(policyHit.FlowLogName()).Should(Equal(expectedPolicyHit.flowLogName))
			Expect(policyHit.Namespace()).Should(Equal(expectedPolicyHit.namespace))
			Expect(policyHit.Count()).Should(Equal(expectedPolicyHit.count))
			Expect(policyHit.IsKubernetes()).Should(Equal(expectedPolicyHit.isKNP))
			Expect(policyHit.IsProfile()).Should(Equal(expectedPolicyHit.isKNS))
			Expect(policyHit.IsStaged()).Should(Equal(expectedPolicyHit.isStaged))
			Expect(policyHit.ToFlowLogPolicyString()).Should(Equal(policyStr))
		},
		Entry(
			"properly handles a network policy",
			"4|tierName|namespaceName/tierName.policyName|allow", 5,
			testPolicyHit{
				action: api.ActionAllow, index: 4, tier: "tierName", name: "policyName", flowLogName: "namespaceName/tierName.policyName", namespace: "namespaceName", count: 5,
				isKNP: false, isKNS: false, isStaged: false,
			}),
		Entry(
			"properly handles a staged network policy",
			"4|tierName|namespaceName/tierName.staged:policyName|deny", 5,
			testPolicyHit{
				action: api.ActionDeny, index: 4, tier: "tierName", name: "policyName", flowLogName: "namespaceName/tierName.staged:policyName", namespace: "namespaceName", count: 5,
				isKNP: false, isKNS: false, isStaged: true,
			}),
		Entry(
			"properly handles a global network policy",
			"4|tierName|tierName.policyName|allow", 5,
			testPolicyHit{
				action: api.ActionAllow, index: 4, tier: "tierName", name: "policyName", flowLogName: "tierName.policyName", namespace: "", count: 5,
				isKNP: false, isKNS: false, isStaged: false,
			}),
		Entry(
			"properly handles a staged global network policy",
			"4|tierName|tierName.staged:policyName|allow", 5,
			testPolicyHit{
				action: api.ActionAllow, index: 4, tier: "tierName", name: "policyName", flowLogName: "tierName.staged:policyName", namespace: "", count: 5,
				isKNP: false, isKNS: false, isStaged: true,
			}),
		Entry(
			"properly handles a kubernetes network policy",
			"4|default|namespaceName/knp.default.policyName|allow", 5,
			testPolicyHit{
				action: api.ActionAllow, index: 4, tier: "default", name: "policyName", flowLogName: "namespaceName/knp.default.policyName", namespace: "namespaceName", count: 5,
				isKNP: true, isStaged: false,
			}),
		Entry(
			"properly handles a staged kubernetes network policy",
			"4|default|namespaceName/staged:knp.default.policyName|deny", 5,
			testPolicyHit{
				action: api.ActionDeny, index: 4, tier: "default", name: "policyName", flowLogName: "namespaceName/staged:knp.default.policyName", namespace: "namespaceName", count: 5,
				isKNP: true, isKNS: false, isStaged: true,
			}),
		Entry(
			"properly handles a kubernetes namespace profile",
			"4|__PROFILE__|__PROFILE__.kns.namespaceName|allow", 5,
			testPolicyHit{
				action: api.ActionAllow, index: 4, tier: "__PROFILE__", name: "namespaceName", flowLogName: "__PROFILE__.kns.namespaceName", namespace: "", count: 5,
				isKNP: false, isKNS: true, isStaged: false,
			}),
	)

	DescribeTable("Unsuccessful PolicyHit parsing",
		func(policyStr string, docCount int, expectedErr error) {
			_, err := api.PolicyHitFromFlowLogPolicyString(policyStr, int64(docCount))
			Expect(err).Should(Equal(expectedErr))
		},
		Entry(
			"fails to parse a policy string with extra pipes",
			"4|tier1|namespace1/policy1|allow|extra|extra", 5,
			fmt.Errorf("invalid policy string '4|tier1|namespace1/policy1|allow|extra|extra': pipe count must equal 4")),
		Entry(
			"fails to parse a policy string with an invalid index",
			"x|tier1|namespace1/policy1|allow", 5,
			fmt.Errorf("invalid policy index: %w", &strconv.NumError{Func: "Atoi", Num: "x", Err: fmt.Errorf("invalid syntax")})),
		Entry(
			"fails to parse a policy string with an invalid index",
			"4|tier1|namespace1/policy1|badaction", 5,
			fmt.Errorf("invalid action 'badaction'")),
	)

	When("changing fields with the Set functions", func() {
		It("returns an updated copy of the original PolicyHit while keep the original unmodified", func() {
			policyHit, err := api.PolicyHitFromFlowLogPolicyString("4|tierName|namespaceName/tierName.policyName|allow", int64(7))
			Expect(err).ShouldNot(HaveOccurred())

			updatedPolicyHit := policyHit.SetIndex(2).SetAction(api.ActionDeny).SetCount(20)

			Expect(updatedPolicyHit.Index()).Should(Equal(2))
			Expect(updatedPolicyHit.Action()).Should(Equal(api.ActionDeny))
			Expect(updatedPolicyHit.Count()).Should(Equal(int64(20)))

			Expect(policyHit.Index()).Should(Equal(4))
			Expect(policyHit.Action()).Should(Equal(api.ActionAllow))
			Expect(policyHit.Count()).Should(Equal(int64(7)))
		})
	})
})

var _ = Describe("NewPolicyHit", func() {
	DescribeTable("Creating a valid policy hit", func(
		action api.Action, count int, index int, isStaged bool, name, namespace, tier string,
		fullName, policyString string) {

		policyHit, err := api.NewPolicyHit(action, int64(count), index, isStaged, name, namespace, tier)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(policyHit.FlowLogName()).Should(Equal(fullName))
		Expect(policyHit.ToFlowLogPolicyString()).Should(Equal(policyString))
		Expect(policyHit.Count()).Should(Equal(int64(count)))
	},
		Entry(
			"properly handles a network policy",
			api.ActionAllow, 5, 4, false, "tierName.policyName", "namespaceName", "tierName",
			"namespaceName/tierName.policyName", "4|tierName|namespaceName/tierName.policyName|allow",
		),
		Entry(
			"properly handles a staged network policy",
			api.ActionDeny, 5, 4, true, "tierName.staged:policyName", "namespaceName", "tierName",
			"namespaceName/tierName.staged:policyName", "4|tierName|namespaceName/tierName.staged:policyName|deny"),
		Entry(
			"properly handles a global network policy",
			api.ActionAllow, 5, 4, false, "tierName.policyName", "", "tierName",
			"tierName.policyName", "4|tierName|tierName.policyName|allow"),
		Entry(
			"properly handles a staged global network policy",
			api.ActionAllow, 5, 4, true, "tierName.policyName", "", "tierName",
			"tierName.staged:policyName", "4|tierName|tierName.staged:policyName|allow"),
		Entry(
			"properly handles a kubernetes network policy",
			api.ActionAllow, 5, 4, false, "knp.default.policyName", "namespaceName", "default",
			"namespaceName/knp.default.policyName", "4|default|namespaceName/knp.default.policyName|allow"),
		Entry(
			"properly handles a staged kubernetes network policy",
			api.ActionDeny, 5, 4, true, "knp.default.policyName", "namespaceName", "default",
			"namespaceName/staged:knp.default.policyName", "4|default|namespaceName/staged:knp.default.policyName|deny"),
		Entry(
			"properly handles a kubernetes namespace profile",
			api.ActionAllow, 5, 4, false, "__PROFILE__.kns.namespaceName", "", "__PROFILE__",
			"__PROFILE__.kns.namespaceName", "4|__PROFILE__|__PROFILE__.kns.namespaceName|allow"),
	)

	DescribeTable("Creating an invalid policy hit", func(
		action api.Action, count int, index int, isStaged bool, name, namespace, tier string,
		expectedErr error) {

		_, err := api.NewPolicyHit(action, int64(count), index, isStaged, name, namespace, tier)
		Expect(err).Should(Equal(expectedErr))
	},
		Entry(
			"returns an error when action the is empty",
			api.ActionInvalid, 5, 4, false, "tierName.policyName", "namespaceName", "tierName",
			fmt.Errorf("a none empty Action must be provided")),
		Entry(
			"returns an error when the index is negative",
			api.ActionDeny, 5, -1, false, "tierName.policyName", "namespaceName", "tierName",
			fmt.Errorf("index must be a positive integer")),
		Entry(
			"returns an error when the count is negative",
			api.ActionAllow, -1, 4, false, "policyName", "namespaceName", "tierName",
			fmt.Errorf("count must be a positive integer")),
	)
})
