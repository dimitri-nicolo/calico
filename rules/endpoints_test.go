// Copyright (c) 2017 Tigera, Inc. All rights reserved.
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

package rules_test

import (
	"github.com/projectcalico/felix/proto"
	. "github.com/projectcalico/felix/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/ipsets"
	. "github.com/projectcalico/felix/iptables"
)

var _ = Describe("Endpoints", func() {
	var rrConfigNormalMangleReturn = Config{
		IPIPEnabled:               true,
		IPIPTunnelAddress:         nil,
		IPSetConfigV4:             ipsets.NewIPVersionConfig(ipsets.IPFamilyV4, "cali", nil, nil),
		IPSetConfigV6:             ipsets.NewIPVersionConfig(ipsets.IPFamilyV6, "cali", nil, nil),
		IptablesMarkAccept:        0x8,
		IptablesMarkPass:          0x10,
		IptablesMarkScratch0:      0x20,
		IptablesMarkScratch1:      0x40,
		IptablesMarkDrop:          0x80,
		IptablesMangleAllowAction: "RETURN",
	}

	var rrConfigConntrackDisabledReturnAction = Config{
		IPIPEnabled:               true,
		IPIPTunnelAddress:         nil,
		IPSetConfigV4:             ipsets.NewIPVersionConfig(ipsets.IPFamilyV4, "cali", nil, nil),
		IPSetConfigV6:             ipsets.NewIPVersionConfig(ipsets.IPFamilyV6, "cali", nil, nil),
		IptablesMarkAccept:        0x8,
		IptablesMarkPass:          0x10,
		IptablesMarkScratch0:      0x20,
		IptablesMarkScratch1:      0x40,
		IptablesMarkDrop:          0x80,
		DisableConntrackInvalid:   true,
		IptablesFilterAllowAction: "RETURN",
	}

	var renderer RuleRenderer
	Context("with normal config", func() {
		BeforeEach(func() {
			renderer = NewRenderer(rrConfigNormalMangleReturn)
		})

		It("should render a minimal workload endpoint", func() {
			Expect(renderer.WorkloadEndpointToIptablesChains("cali1234", true, nil, nil)).To(Equal([]*Chain{
				{
					Name: "cali-tw-cali1234",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						{Action: ClearMarkAction{Mark: 0x88}},
						{Action: NflogAction{
							Group:  1,
							Prefix: "D|0|no-profile-match-inbound|pr",
						}},
						{Action: DropAction{},
							Comment: "Drop if no profiles matched"},
					},
				},
				{
					Name: "cali-fw-cali1234",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						{Action: ClearMarkAction{Mark: 0x88}},
						{Action: NflogAction{
							Group:  2,
							Prefix: "D|0|no-profile-match-outbound|pr",
						}},
						{Action: DropAction{},
							Comment: "Drop if no profiles matched"},
					},
				},
			}))
		})

		It("should render a disabled workload endpoint", func() {
			Expect(renderer.WorkloadEndpointToIptablesChains("cali1234", false, nil, nil)).To(Equal([]*Chain{
				{
					Name: "cali-tw-cali1234",
					Rules: []Rule{
						{Action: DropAction{},
							Comment: "Endpoint admin disabled"},
					},
				},
				{
					Name: "cali-fw-cali1234",
					Rules: []Rule{
						{Action: DropAction{},
							Comment: "Endpoint admin disabled"},
					},
				},
			}))
		})

		It("should render a fully-loaded workload endpoint", func() {
			var endpoint = proto.WorkloadEndpoint{
				Name: "cali1234",
				Tiers: []*proto.TierInfo{
					{Name: "tier1", IngressPolicies: []string{"ai", "bi"}, EgressPolicies: []string{"ae", "be"}},
					{Name: "tier2", IngressPolicies: []string{"ci", "di"}, EgressPolicies: []string{"ce", "de"}},
				},
				ProfileIds: []string{"prof1", "prof2"},
			}
			Expect(renderer.WorkloadEndpointToIptablesChains(
				"cali1234",
				true,
				endpoint.Tiers,
				endpoint.ProfileIds,
			)).To(Equal([]*Chain{
				{
					Name: "cali-tw-cali1234",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier tier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier1/ai"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier1/bi"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: NflogAction{
								Group:  1,
								Prefix: "D|0|tier1.no-policy-match-inbound|po"}},
						{Match: Match().MarkClear(0x10),
							Action:  DropAction{},
							Comment: "Drop if no policies passed packet"},

						{Comment: "Start of tier tier2",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier2/ci"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier2/di"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: NflogAction{
								Group:  1,
								Prefix: "D|0|tier2.no-policy-match-inbound|po"}},
						{Match: Match().MarkClear(0x10),
							Action:  DropAction{},
							Comment: "Drop if no policies passed packet"},

						{Action: JumpAction{Target: "cali-pri-prof1"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if profile accepted"},
						{Action: JumpAction{Target: "cali-pri-prof2"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if profile accepted"},
						{Action: NflogAction{
							Group:  1,
							Prefix: "D|0|no-profile-match-inbound|pr"}},

						{Action: DropAction{},
							Comment: "Drop if no profiles matched"},
					},
				},
				{
					Name: "cali-fw-cali1234",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier tier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-tier1/ae"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-tier1/be"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: NflogAction{
								Group:  2,
								Prefix: "D|0|tier1.no-policy-match-outbound|po"}},
						{Match: Match().MarkClear(0x10),
							Action:  DropAction{},
							Comment: "Drop if no policies passed packet"},

						{Comment: "Start of tier tier2",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-tier2/ce"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-tier2/de"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: NflogAction{
								Group:  2,
								Prefix: "D|0|tier2.no-policy-match-outbound|po"}},
						{Match: Match().MarkClear(0x10),
							Action:  DropAction{},
							Comment: "Drop if no policies passed packet"},

						{Action: JumpAction{Target: "cali-pro-prof1"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if profile accepted"},
						{Action: JumpAction{Target: "cali-pro-prof2"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if profile accepted"},
						{Action: NflogAction{
							Group:  2,
							Prefix: "D|0|no-profile-match-outbound|pr"}},

						{Action: DropAction{},
							Comment: "Drop if no profiles matched"},
					},
				},
			}))
		})

		It("should render a host endpoint", func() {
			var tiers = []*proto.TierInfo{
				{Name: "tier1", IngressPolicies: []string{"ai", "bi"}, EgressPolicies: []string{"ae", "be"}},
			}
			var forwardTiers = []*proto.TierInfo{
				{Name: "fwdTier1", IngressPolicies: []string{"afi", "bfi"}, EgressPolicies: []string{"afe", "bfe"}},
			}

			Expect(renderer.HostEndpointToFilterChains("eth0",
				tiers,
				forwardTiers,
				[]string{"prof1", "prof2"})).To(Equal([]*Chain{
				{
					Name: "cali-th-eth0",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						// Host endpoints get extra failsafe rules.
						{Action: JumpAction{Target: "cali-failsafe-out"}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier tier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-tier1/ae"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-tier1/be"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: NflogAction{
								Group:  2,
								Prefix: "D|0|tier1.no-policy-match-outbound|po"}},
						{Match: Match().MarkClear(0x10),
							Action:  DropAction{},
							Comment: "Drop if no policies passed packet"},

						{Action: JumpAction{Target: "cali-pro-prof1"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if profile accepted"},
						{Action: JumpAction{Target: "cali-pro-prof2"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if profile accepted"},
						{Action: NflogAction{
							Group:  2,
							Prefix: "D|0|no-profile-match-outbound|pr"}},

						{Action: DropAction{},
							Comment: "Drop if no profiles matched"},
					},
				},
				{
					Name: "cali-fh-eth0",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						// Host endpoints get extra failsafe rules.
						{Action: JumpAction{Target: "cali-failsafe-in"}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier tier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier1/ai"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier1/bi"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: NflogAction{
								Group:  1,
								Prefix: "D|0|tier1.no-policy-match-inbound|po"}},
						{Match: Match().MarkClear(0x10),
							Action:  DropAction{},
							Comment: "Drop if no policies passed packet"},

						{Action: JumpAction{Target: "cali-pri-prof1"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if profile accepted"},
						{Action: JumpAction{Target: "cali-pri-prof2"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if profile accepted"},
						{Action: NflogAction{
							Group:  1,
							Prefix: "D|0|no-profile-match-inbound|pr"}},

						{Action: DropAction{},
							Comment: "Drop if no profiles matched"},
					},
				},
				{
					Name: "cali-thfw-eth0",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier fwdTier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-fwdTier1/afe"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-fwdTier1/bfe"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: NflogAction{
								Group:  2,
								Prefix: "D|0|fwdTier1.no-policy-match-outbound|po"}},
						{Match: Match().MarkClear(0x10),
							Action:  DropAction{},
							Comment: "Drop if no policies passed packet"},
					},
				},
				{
					Name: "cali-fhfw-eth0",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier fwdTier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-fwdTier1/afi"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-fwdTier1/bfi"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},
						{Match: Match().MarkClear(0x10),
							Action: NflogAction{
								Group:  1,
								Prefix: "D|0|fwdTier1.no-policy-match-inbound|po"}},
						{Match: Match().MarkClear(0x10),
							Action:  DropAction{},
							Comment: "Drop if no policies passed packet"},
					},
				},
			}))
		})

		It("should render host endpoint raw chains with untracked policies", func() {
			var untrackedTiers = []*proto.TierInfo{
				{Name: "tier1", IngressPolicies: []string{"c"}, EgressPolicies: []string{"c"}},
			}
			Expect(renderer.HostEndpointToRawChains("eth0", untrackedTiers)).To(Equal([]*Chain{
				{
					Name: "cali-th-eth0",
					Rules: []Rule{
						// Host endpoints get extra failsafe rules.
						{Action: JumpAction{Target: "cali-failsafe-out"}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier tier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-po-tier1/c"}},
						// Extra NOTRACK action before returning in raw table.
						{Match: Match().MarkSet(0x8),
							Action: NoTrackAction{}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},

						// No drop actions or profiles in raw table.
					},
				},
				{
					Name: "cali-fh-eth0",
					Rules: []Rule{
						// Host endpoints get extra failsafe rules.
						{Action: JumpAction{Target: "cali-failsafe-in"}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier tier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier1/c"}},
						// Extra NOTRACK action before returning in raw table.
						{Match: Match().MarkSet(0x8),
							Action: NoTrackAction{}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},

						// No drop actions or profiles in raw table.
					},
				},
			}))
		})

		It("should render host endpoint mangle chains with pre-DNAT policies", func() {
			var tiers = []*proto.TierInfo{
				{Name: "tier1", IngressPolicies: []string{"c"}, EgressPolicies: []string{"c"}},
			}
			Expect(renderer.HostEndpointToMangleChains(
				"eth0",
				tiers,
			)).To(Equal([]*Chain{
				{
					Name: "cali-fh-eth0",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: SetMarkAction{Mark: 0x8}},
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: ReturnAction{}},
						{Match: Match().ConntrackState("INVALID"),
							Action: DropAction{}},

						// Host endpoints get extra failsafe rules.
						{Action: JumpAction{Target: "cali-failsafe-in"}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier tier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier1/c"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},

						// No drop actions or profiles in raw table.
					},
				},
			}))
		})
	})

	Describe("with ctstate=INVALID disabled", func() {
		BeforeEach(func() {
			renderer = NewRenderer(rrConfigConntrackDisabledReturnAction)
		})

		It("should render a minimal workload endpoint", func() {
			Expect(renderer.WorkloadEndpointToIptablesChains("cali1234", true, nil, nil)).To(Equal([]*Chain{
				{
					Name: "cali-tw-cali1234",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: SetMarkAction{Mark: 0x8}},
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: ReturnAction{}},

						{Action: ClearMarkAction{Mark: 0x88}},
						{Action: NflogAction{
							Group:  1,
							Prefix: "D|0|no-profile-match-inbound|pr"}},
						{Action: DropAction{},
							Comment: "Drop if no profiles matched"},
					},
				},
				{
					Name: "cali-fw-cali1234",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: SetMarkAction{Mark: 0x8}},
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: ReturnAction{}},

						{Action: ClearMarkAction{Mark: 0x88}},
						{Action: NflogAction{
							Group:  2,
							Prefix: "D|0|no-profile-match-outbound|pr"}},
						{Action: DropAction{},
							Comment: "Drop if no profiles matched"},
					},
				},
			}))
		})

		It("should render host endpoint mangle chains with pre-DNAT policies", func() {
			var tiers = []*proto.TierInfo{
				{Name: "tier1", IngressPolicies: []string{"c"}, EgressPolicies: []string{"c"}},
			}
			Expect(renderer.HostEndpointToMangleChains(
				"eth0",
				tiers,
			)).To(Equal([]*Chain{
				{
					Name: "cali-fh-eth0",
					Rules: []Rule{
						// conntrack rules.
						{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
							Action: AcceptAction{}},

						// Host endpoints get extra failsafe rules.
						{Action: JumpAction{Target: "cali-failsafe-in"}},

						{Action: ClearMarkAction{Mark: 0x88}},

						{Comment: "Start of tier tier1",
							Action: ClearMarkAction{Mark: 0x10}},
						{Match: Match().MarkClear(0x10),
							Action: JumpAction{Target: "cali-pi-tier1/c"}},
						{Match: Match().MarkSet(0x8),
							Action:  ReturnAction{},
							Comment: "Return if policy accepted"},

						// No drop actions or profiles in raw table.
					},
				},
			}))
		})
	})
})
