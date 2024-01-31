// Copyright (c) 2017-2024 Tigera, Inc. All rights reserved.
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
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/felix/ipsets"
	. "github.com/projectcalico/calico/felix/iptables"
	"github.com/projectcalico/calico/felix/proto"
	. "github.com/projectcalico/calico/felix/rules"
)

var _ = Describe("Endpoints", func() {
	const (
		ProtoUDP          = 17
		ProtoIPIP         = 4
		VXLANPort         = 4789
		EgressIPVXLANPort = 4790
		VXLANVNI          = 4096
	)

	for _, trueOrFalse := range []bool{true, false} {
		var denyAction Action
		denyAction = DropAction{}
		denyActionCommand := "DROP"
		denyActionString := "Drop"
		if trueOrFalse {
			denyAction = RejectAction{}
			denyActionCommand = "REJECT"
			denyActionString = "Reject"
		}

		kubeIPVSEnabled := trueOrFalse
		var rrConfigNormalMangleReturn = Config{
			IPIPEnabled:                      true,
			IPIPTunnelAddress:                nil,
			IPSetConfigV4:                    ipsets.NewIPVersionConfig(ipsets.IPFamilyV4, "cali", nil, nil),
			IPSetConfigV6:                    ipsets.NewIPVersionConfig(ipsets.IPFamilyV6, "cali", nil, nil),
			DNSPolicyMode:                    apiv3.DNSPolicyModeDelayDeniedPacket,
			DNSPolicyNfqueueID:               100,
			DNSPacketsNfqueueID:              101,
			IptablesMarkEgress:               0x4,
			IptablesMarkAccept:               0x8,
			IptablesMarkPass:                 0x10,
			IptablesMarkScratch0:             0x20,
			IptablesMarkScratch1:             0x40,
			IptablesMarkDrop:                 0x80,
			IptablesMarkEndpoint:             0xff00,
			IptablesMarkNonCaliEndpoint:      0x0100,
			IptablesMarkDNSPolicy:            0x00001,
			IptablesMarkSkipDNSPolicyNfqueue: 0x400000,
			KubeIPVSSupportEnabled:           kubeIPVSEnabled,
			IptablesMangleAllowAction:        "RETURN",
			IptablesFilterDenyAction:         denyActionCommand,
			VXLANPort:                        4789,
			EgressIPVXLANPort:                4790,
			VXLANVNI:                         4096,
		}

		var rrConfigConntrackDisabledReturnAction = Config{
			IPIPEnabled:                      true,
			IPIPTunnelAddress:                nil,
			IPSetConfigV4:                    ipsets.NewIPVersionConfig(ipsets.IPFamilyV4, "cali", nil, nil),
			IPSetConfigV6:                    ipsets.NewIPVersionConfig(ipsets.IPFamilyV6, "cali", nil, nil),
			DNSPolicyMode:                    apiv3.DNSPolicyModeDelayDeniedPacket,
			DNSPolicyNfqueueID:               100,
			DNSPacketsNfqueueID:              101,
			IptablesMarkEgress:               0x4,
			IptablesMarkAccept:               0x8,
			IptablesMarkPass:                 0x10,
			IptablesMarkScratch0:             0x20,
			IptablesMarkScratch1:             0x40,
			IptablesMarkDrop:                 0x80,
			IptablesMarkEndpoint:             0xff00,
			IptablesMarkNonCaliEndpoint:      0x0100,
			IptablesMarkDNSPolicy:            0x00001,
			IptablesMarkSkipDNSPolicyNfqueue: 0x400000,
			KubeIPVSSupportEnabled:           kubeIPVSEnabled,
			DisableConntrackInvalid:          true,
			IptablesFilterAllowAction:        "RETURN",
			IptablesFilterDenyAction:         denyActionCommand,
			VXLANPort:                        4789,
			EgressIPVXLANPort:                4790,
			VXLANVNI:                         4096,
		}

		var renderer RuleRenderer
		var epMarkMapper EndpointMarkMapper

		dropVXLANRule := Rule{
			Match: Match().ProtocolNum(ProtoUDP).
				DestPorts(uint16(VXLANPort)),
			Action:  denyAction,
			Comment: []string{fmt.Sprintf("%s VXLAN encapped packets originating in workloads", denyActionString)},
		}
		dropIPIPRule := Rule{
			Match:   Match().ProtocolNum(ProtoIPIP),
			Action:  denyAction,
			Comment: []string{fmt.Sprintf("%s IPinIP encapped packets originating in workloads", denyActionString)},
		}

		Context("with normal config", func() {
			BeforeEach(func() {
				renderer = NewRenderer(rrConfigNormalMangleReturn)
				epMarkMapper = NewEndpointMarkMapper(rrConfigNormalMangleReturn.IptablesMarkEndpoint,
					rrConfigNormalMangleReturn.IptablesMarkNonCaliEndpoint)
			})

			It("should render a minimal workload endpoint", func() {
				Expect(renderer.WorkloadEndpointToIptablesChains(
					"cali1234", epMarkMapper,
					true,
					nil,
					nil,
					NotAnEgressGateway,
					0,
					UndefinedIPVersion)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-tw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 1, Prefix: "DRI"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-fw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},
							dropVXLANRule,
							dropIPIPRule,
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 2, Prefix: "DRE"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-sm-cali1234",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
						},
					},
				})))
			})

			It("should render a disabled workload endpoint", func() {
				Expect(renderer.WorkloadEndpointToIptablesChains(
					"cali1234", epMarkMapper,
					false,
					nil,
					nil,
					NotAnEgressGateway,
					8080,
					UndefinedIPVersion,
				)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-tw-cali1234",
						Rules: []Rule{
							{Action: denyAction,
								Comment: []string{"Endpoint admin disabled"}},
						},
					},
					{
						Name: "cali-fw-cali1234",
						Rules: []Rule{
							{Action: denyAction,
								Comment: []string{"Endpoint admin disabled"}},
						},
					},
					{
						Name: "cali-sm-cali1234",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
						},
					},
				})))
			})

			It("should render a fully-loaded workload endpoint", func() {
				Expect(renderer.WorkloadEndpointToIptablesChains(
					"cali1234",
					epMarkMapper,
					true,
					singlePolicyGroups([]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"ai", "bi"},
						EgressPolicies:  []string{"ae", "be"},
					}}),
					[]string{"prof1", "prof2"},
					NotAnEgressGateway,
					0,
					UndefinedIPVersion,
				)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-tw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/ai"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/bi"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 1, Prefix: "DPI|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pri-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pri-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 1, Prefix: "DRI"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-fw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},
							dropVXLANRule,
							dropIPIPRule,

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/ae"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/be"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 2, Prefix: "DPE|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pro-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pro-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-sm-cali1234",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
						},
					},
				})))
			})

			It("should render a workload endpoint with policy groups", func() {
				format.MaxLength = 1000000

				polGrpInABC := &PolicyGroup{
					Tier:        "default",
					Direction:   PolicyDirectionInbound,
					PolicyNames: []string{"a", "b", "c"},
					Selector:    "all()",
				}
				polGrpInEF := &PolicyGroup{
					Tier:        "default",
					Direction:   PolicyDirectionInbound,
					PolicyNames: []string{"e", "f"},
					Selector:    "someLabel == 'bar'",
				}
				polGrpOutAB := &PolicyGroup{
					Tier:        "default",
					Direction:   PolicyDirectionOutbound,
					PolicyNames: []string{"a", "b"},
					Selector:    "all()",
				}
				polGrpOutDE := &PolicyGroup{
					Tier:        "default",
					Direction:   PolicyDirectionOutbound,
					PolicyNames: []string{"d", "e"},
					Selector:    "someLabel == 'bar'",
				}

				Expect(renderer.WorkloadEndpointToIptablesChains(
					"cali1234",
					epMarkMapper,
					true,
					[]*PolicyGroup{
						polGrpInABC,
						polGrpInEF,
					},
					[]*PolicyGroup{
						polGrpOutAB,
						polGrpOutDE,
					},
					[]string{"prof1", "prof2"},
				)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-tw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x18}},

							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: polGrpInABC.ChainName()}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: polGrpInEF.ChainName()}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pri-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pri-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-fw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x18}},
							dropVXLANRule,
							dropIPIPRule,

							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: polGrpOutAB.ChainName()}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: polGrpOutDE.ChainName()}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pro-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pro-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 2, Prefix: "DRE"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-sm-cali1234",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
						},
					},
				})))
			})

			It("should render a fully-loaded workload endpoint - one staged policy, one enforced", func() {
				Expect(renderer.WorkloadEndpointToIptablesChains(
					"cali1234",
					epMarkMapper,
					true,
					[]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"staged:ai", "bi"},
						EgressPolicies:  []string{"ae", "staged:be"},
					}},
					[]string{"prof1", "prof2"},
					NotAnEgressGateway,
					0,
					UndefinedIPVersion,
				)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-tw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/staged:ai"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/bi"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 1, Prefix: "DPI|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pri-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pri-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							{Action: NflogAction{Group: 1, Prefix: "DRI"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-fw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},
							dropVXLANRule,
							dropIPIPRule,

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/ae"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/staged:be"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 2, Prefix: "DPE|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pro-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pro-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							{Action: NflogAction{Group: 2, Prefix: "DRE"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-sm-cali1234",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
						},
					},
				})))
			})

			It("should render a fully-loaded workload endpoint - both staged, end-of-tier action is pass", func() {
				Expect(renderer.WorkloadEndpointToIptablesChains(
					"cali1234",
					epMarkMapper,
					true,
					[]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"staged:ai", "staged:bi"},
						EgressPolicies:  []string{"staged:ae", "staged:be"},
					}},
					[]string{"prof1", "prof2"},
					NotAnEgressGateway,
					0,
					UndefinedIPVersion,
				)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-tw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/staged:ai"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/staged:bi"}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 1, Prefix: "PPI|default"}},

							{Action: JumpAction{Target: "cali-pri-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pri-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 1, Prefix: "DRI"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-fw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},
							dropVXLANRule,
							dropIPIPRule,

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/staged:ae"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/staged:be"}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 2, Prefix: "PPE|default"}},

							{Action: JumpAction{Target: "cali-pro-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pro-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 2, Prefix: "DRE"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-sm-cali1234",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
						},
					},
				})))
			})

			It("should render a host endpoint", func() {
				actual := renderer.HostEndpointToFilterChains("eth0",
					singlePolicyGroups([]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"ai", "bi"},
						EgressPolicies:  []string{"ae", "be"},
					}}),
					singlePolicyGroups([]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"afi", "bfi"},
						EgressPolicies:  []string{"afe", "bfe"},
					}}),
					epMarkMapper,
					[]string{"prof1", "prof2"},
				)
				expected := trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-th-eth0",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							// Host endpoints get extra failsafe rules.
							{Action: JumpAction{Target: "cali-failsafe-out"}},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/ae"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/be"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 2, Prefix: "DPE|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pro-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pro-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 2, Prefix: "DRE"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-fh-eth0",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							// Host endpoints get extra failsafe rules.
							{Action: JumpAction{Target: "cali-failsafe-in"}},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/ai"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/bi"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 1, Prefix: "DPI|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pri-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pri-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 1, Prefix: "DRI"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-thfw-eth0",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/afe"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/bfe"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 2, Prefix: "DPE|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
						},
					},
					{
						Name: "cali-fhfw-eth0",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},
							{Match: Match().ConntrackState("INVALID"),
								Action: denyAction},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/afi"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/bfi"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 1, Prefix: "DPI|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},
						},
					},
					{
						Name: "cali-sm-eth0",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xa200, Mask: 0xff00}},
						},
					},
				})
				Expect(actual).To(Equal(expected), cmp.Diff(actual, expected))
			})

			It("should render host endpoint raw chains with untracked policies", func() {
				Expect(renderer.HostEndpointToRawChains("eth0",
					singlePolicyGroups([]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"c"},
						EgressPolicies:  []string{"c"},
					}}),
				)).To(Equal([]*Chain{
					{
						Name: "cali-th-eth0",
						Rules: []Rule{
							// Host endpoints get extra failsafe rules.
							{Action: JumpAction{Target: "cali-failsafe-out"}},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/c"}},
							// Extra NOTRACK action before returning in raw table.
							{Match: Match().MarkSingleBitSet(0x8),
								Action: NoTrackAction{}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},

							// No drop actions or profiles in raw table.
						},
					},
					{
						Name: "cali-fh-eth0",
						Rules: []Rule{
							// Host endpoints get extra failsafe rules.
							{Action: JumpAction{Target: "cali-failsafe-in"}},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/c"}},
							// Extra NOTRACK action before returning in raw table.
							{Match: Match().MarkSingleBitSet(0x8),
								Action: NoTrackAction{}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},

							// No drop actions or profiles in raw table.
						},
					},
				}))
			})

			It("should render host endpoint mangle chains with pre-DNAT policies", func() {
				Expect(renderer.HostEndpointToMangleIngressChains(
					"eth0",
					singlePolicyGroups([]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"c"},
					}}),
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
								Action: denyAction},

							// Host endpoints get extra failsafe rules.
							{Action: JumpAction{Target: "cali-failsafe-in"}},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/c"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},

							// No drop actions or profiles in raw table.
						},
					},
				}))
			})

			It("should render an egress gateway workload endpoint", func() {
				Expect(renderer.WorkloadEndpointToIptablesChains(
					"cali1234",
					epMarkMapper,
					true,
					[]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"ai", "bi"},
						EgressPolicies:  []string{"ae", "be"},
					}},
					[]string{"prof1", "prof2"},
					IsAnEgressGateway,
					8080,
					4,
				)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-tw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Match: Match().ProtocolNum(ProtoUDP).
								SourceIPSet("cali40all-hosts-net").
								DestPorts(uint16(EgressIPVXLANPort)),
								Action:  AcceptAction{},
								Comment: []string{"Accept VXLAN UDP traffic for egress gateways"}},
							{Match: Match().ProtocolNum(ProtoTCP).
								SourceIPSet("cali40all-hosts-net").
								DestPorts(uint16(8080)),
								Action:  AcceptAction{},
								Comment: []string{"Accept readiness probes for egress gateways"}},
							{Match: Match().ProtocolNum(ProtoUDP).
								SourceIPSet("cali40all-tunnel-net").
								DestPorts(uint16(EgressIPVXLANPort)),
								Action:  AcceptAction{},
								Comment: []string{"Accept VXLAN UDP traffic for egress gateways"}},
							{Match: Match().ProtocolNum(ProtoTCP).
								SourceIPSet("cali40all-tunnel-net").
								DestPorts(uint16(8080)),
								Action:  AcceptAction{},
								Comment: []string{"Accept readiness probes for egress gateways"}},

							{Action: NflogAction{Group: 1, Prefix: "DRI"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s all other ingress traffic to egress gateway.", denyActionString)}},
						},
					},
					{
						Name: "cali-fw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: AcceptAction{}},

							{Action: ClearMarkAction{Mark: 0x88}},
							{Match: Match().ProtocolNum(ProtoUDP).
								DestIPSet("cali40all-hosts-net").
								DestPorts(uint16(EgressIPVXLANPort)),
								Action:  AcceptAction{},
								Comment: []string{"Accept VXLAN UDP traffic for egress gateways"}},
							{Match: Match().ProtocolNum(ProtoUDP).
								DestIPSet("cali40all-tunnel-net").
								DestPorts(uint16(EgressIPVXLANPort)),
								Action:  AcceptAction{},
								Comment: []string{"Accept VXLAN UDP traffic for egress gateways"}},
							dropVXLANRule,
							dropIPIPRule,

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/ae"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-po-default/be"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},
							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000).MarkClear(0x10),
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)},
								Action:  NfqueueAction{QueueNum: 100}},
							{Match: Match().MarkClear(0x10),
								Action: NflogAction{Group: 2, Prefix: "DPE|default"}},
							{Match: Match().MarkClear(0x10),
								Action:  denyAction,
								Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

							{Action: JumpAction{Target: "cali-pro-prof1"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},
							{Action: JumpAction{Target: "cali-pro-prof2"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if profile accepted"}},

							{Action: NflogAction{Group: 2, Prefix: "DRE"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-sm-cali1234",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
						},
					},
				})))
			})
		})

		Describe("with ctstate=INVALID disabled", func() {
			BeforeEach(func() {
				renderer = NewRenderer(rrConfigConntrackDisabledReturnAction)
				epMarkMapper = NewEndpointMarkMapper(rrConfigConntrackDisabledReturnAction.IptablesMarkEndpoint,
					rrConfigConntrackDisabledReturnAction.IptablesMarkNonCaliEndpoint)
			})

			It("should render a minimal workload endpoint", func() {
				Expect(renderer.WorkloadEndpointToIptablesChains(
					"cali1234",
					epMarkMapper,
					true,
					nil,
					nil,
					NotAnEgressGateway,
					0,
					UndefinedIPVersion,
				)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
					{
						Name: "cali-tw-cali1234",
						Rules: []Rule{
							// conntrack rules.
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: SetMarkAction{Mark: 0x8}},
							{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
								Action: ReturnAction{}},

							{Action: ClearMarkAction{Mark: 0x88}},

							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 1, Prefix: "DRI"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
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
							dropVXLANRule,
							dropIPIPRule,

							{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
								Action:  NfqueueAction{QueueNum: 100},
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
							},
							{Action: NflogAction{Group: 2, Prefix: "DRE"}},
							{Action: denyAction,
								Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
						},
					},
					{
						Name: "cali-sm-cali1234",
						Rules: []Rule{
							{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
						},
					},
				})))
			})

			It("should render host endpoint mangle chains with pre-DNAT policies", func() {
				Expect(renderer.HostEndpointToMangleIngressChains(
					"eth0",
					singlePolicyGroups([]*proto.TierInfo{{
						Name:            "default",
						IngressPolicies: []string{"c"},
					}}),
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

							{Comment: []string{"Start of tier default"},
								Action: ClearMarkAction{Mark: 0x10}},
							{Match: Match().MarkClear(0x10),
								Action: JumpAction{Target: "cali-pi-default/c"}},
							{Match: Match().MarkSingleBitSet(0x8),
								Action:  ReturnAction{},
								Comment: []string{"Return if policy accepted"}},

							// No drop actions or profiles in raw table.
						},
					},
				}))
			})
		})
		Describe("Disabling adding drop encap rules", func() {
			Context("VXLAN allowed, IPIP dropped", func() {
				It("should render a minimal workload endpoint without VXLAN drop encap rule and with IPIP drop encap rule", func() {
					rrConfigNormalMangleReturn.AllowVXLANPacketsFromWorkloads = true
					renderer = NewRenderer(rrConfigNormalMangleReturn)
					epMarkMapper = NewEndpointMarkMapper(rrConfigNormalMangleReturn.IptablesMarkEndpoint,
						rrConfigNormalMangleReturn.IptablesMarkNonCaliEndpoint)
					Expect(renderer.WorkloadEndpointToIptablesChains(
						"cali1234", epMarkMapper,
						true,
						nil,
						nil,
						NotAnEgressGateway,
						0,
						UndefinedIPVersion,
					)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
						{
							Name: "cali-tw-cali1234",
							Rules: []Rule{
								// conntrack rules.
								{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
									Action: AcceptAction{}},
								{Match: Match().ConntrackState("INVALID"),
									Action: denyAction},

								{Action: ClearMarkAction{Mark: 0x88}},
								{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
									Action:  NfqueueAction{QueueNum: 100}},
								{Action: NflogAction{Group: 1, Prefix: "DRI"}},
								{Action: denyAction,
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							},
						},
						{
							Name: "cali-fw-cali1234",
							Rules: []Rule{
								// conntrack rules.
								{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
									Action: AcceptAction{}},
								{Match: Match().ConntrackState("INVALID"),
									Action: denyAction},

								{Action: ClearMarkAction{Mark: 0x88}},
								dropIPIPRule,
								{Match: Match().MarkSingleBitSet(0x00001).NotMarkMatchesWithMask(0x400000, 0x400000),
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)},
									Action:  NfqueueAction{QueueNum: 100}},
								{Action: NflogAction{Group: 2, Prefix: "DRE"}},
								{Action: denyAction,
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							},
						},
						{
							Name: "cali-sm-cali1234",
							Rules: []Rule{
								{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
							},
						},
					})))
				})
			})
			Context("VXLAN dropped, IPIP allowed", func() {
				It("should render a minimal workload endpoint with VXLAN drop encap rule and without IPIP drop encap rule", func() {
					rrConfigNormalMangleReturn.AllowIPIPPacketsFromWorkloads = true
					renderer = NewRenderer(rrConfigNormalMangleReturn)
					epMarkMapper = NewEndpointMarkMapper(rrConfigNormalMangleReturn.IptablesMarkEndpoint,
						rrConfigNormalMangleReturn.IptablesMarkNonCaliEndpoint)

					actual := renderer.WorkloadEndpointToIptablesChains(
						"cali1234", epMarkMapper,
						true,
						nil,
						nil,
						NotAnEgressGateway,
						0,
						UndefinedIPVersion,
					)
					expected := trimSMChain(kubeIPVSEnabled, []*Chain{
						{
							Name: "cali-tw-cali1234",
							Rules: []Rule{
								// conntrack rules.
								{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
									Action: AcceptAction{}},
								{Match: Match().ConntrackState("INVALID"),
									Action: denyAction},

								{Action: ClearMarkAction{Mark: 0x88}},
								{Match: Match().MarkSingleBitSet(0x0001).NotMarkMatchesWithMask(0x400000, 0x400000),
									Action:  NfqueueAction{QueueNum: 100},
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
								{Action: NflogAction{Group: 1, Prefix: "DRI"}},
								{Action: denyAction,
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							},
						},
						{
							Name: "cali-fw-cali1234",
							Rules: []Rule{
								// conntrack rules.
								{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
									Action: AcceptAction{}},
								{Match: Match().ConntrackState("INVALID"),
									Action: denyAction},

								{Action: ClearMarkAction{Mark: 0x88}},
								dropVXLANRule,
								{Match: Match().MarkSingleBitSet(0x0001).NotMarkMatchesWithMask(0x400000, 0x400000),
									Action:  NfqueueAction{QueueNum: 100},
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
								{Action: NflogAction{Group: 2, Prefix: "DRE"}},
								{Action: denyAction,
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							},
						},
						{
							Name: "cali-sm-cali1234",
							Rules: []Rule{
								{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
							},
						},
					})
					Expect(actual).To(Equal(expected), cmp.Diff(actual, expected))
				})
			})
			Context("VXLAN and IPIP allowed", func() {
				It("should render a minimal workload endpoint without both VXLAN and IPIP drop encap rule", func() {
					rrConfigNormalMangleReturn.AllowVXLANPacketsFromWorkloads = true
					rrConfigNormalMangleReturn.AllowIPIPPacketsFromWorkloads = true
					renderer = NewRenderer(rrConfigNormalMangleReturn)
					epMarkMapper = NewEndpointMarkMapper(rrConfigNormalMangleReturn.IptablesMarkEndpoint,
						rrConfigNormalMangleReturn.IptablesMarkNonCaliEndpoint)
					Expect(renderer.WorkloadEndpointToIptablesChains(
						"cali1234", epMarkMapper,
						true,
						nil,
						nil,
						NotAnEgressGateway,
						0,
						UndefinedIPVersion,
					)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
						{
							Name: "cali-tw-cali1234",
							Rules: []Rule{
								// conntrack rules.
								{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
									Action: AcceptAction{}},
								{Match: Match().ConntrackState("INVALID"),
									Action: denyAction},

								{Action: ClearMarkAction{Mark: 0x88}},
								{Match: Match().MarkSingleBitSet(0x0001).NotMarkMatchesWithMask(0x400000, 0x400000),
									Action:  NfqueueAction{QueueNum: 100},
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
								{Action: NflogAction{Group: 1, Prefix: "DRI"}},
								{Action: denyAction,
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							},
						},
						{
							Name: "cali-fw-cali1234",
							Rules: []Rule{
								// conntrack rules.
								{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
									Action: AcceptAction{}},
								{Match: Match().ConntrackState("INVALID"),
									Action: denyAction},

								{Action: ClearMarkAction{Mark: 0x88}},
								{Match: Match().MarkSingleBitSet(0x0001).NotMarkMatchesWithMask(0x400000, 0x400000),
									Action:  NfqueueAction{QueueNum: 100},
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
								{Action: NflogAction{Group: 2, Prefix: "DRE"}},
								{Action: denyAction,
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							},
						},
						{
							Name: "cali-sm-cali1234",
							Rules: []Rule{
								{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
							},
						},
					})))
				})
			})
			AfterEach(func() {
				rrConfigNormalMangleReturn.AllowIPIPPacketsFromWorkloads = false
				rrConfigNormalMangleReturn.AllowVXLANPacketsFromWorkloads = false
			})
		})

		// Test the DNSPolicyMode options NoDelay and DelayDNSResponse. The NFQueue rule for the drop rule should not be
		// added.
		for _, dm := range []apiv3.DNSPolicyMode{apiv3.DNSPolicyModeNoDelay, apiv3.DNSPolicyModeDelayDNSResponse} {
			dnsMode := dm
			Context("with normal config and DNSMode set to NoDelay", func() {
				BeforeEach(func() {
					rrConfigNormalMangleReturn.DNSPolicyMode = dnsMode
					renderer = NewRenderer(rrConfigNormalMangleReturn)
					epMarkMapper = NewEndpointMarkMapper(rrConfigNormalMangleReturn.IptablesMarkEndpoint,
						rrConfigNormalMangleReturn.IptablesMarkNonCaliEndpoint)
				})

				AfterEach(func() {
					rrConfigNormalMangleReturn.DNSPolicyMode = apiv3.DNSPolicyModeDelayDeniedPacket
				})

				It("should render a fully-loaded workload endpoint - one staged policy, one enforced", func() {
					Expect(renderer.WorkloadEndpointToIptablesChains(
						"cali1234",
						epMarkMapper,
						true,
						[]*proto.TierInfo{{
							Name:            "default",
							IngressPolicies: []string{"staged:ai", "bi"},
							EgressPolicies:  []string{"ae", "staged:be"},
						}},
						[]string{"prof1", "prof2"},
						NotAnEgressGateway,
						0,
						UndefinedIPVersion,
					)).To(Equal(trimSMChain(kubeIPVSEnabled, []*Chain{
						{
							Name: "cali-tw-cali1234",
							Rules: []Rule{
								// conntrack rules.
								{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
									Action: AcceptAction{}},
								{Match: Match().ConntrackState("INVALID"),
									Action: denyAction},

								{Action: ClearMarkAction{Mark: 0x88}},

								{Comment: []string{"Start of tier default"},
									Action: ClearMarkAction{Mark: 0x10}},
								{Match: Match().MarkClear(0x10),
									Action: JumpAction{Target: "cali-pi-default/staged:ai"}},
								{Match: Match().MarkClear(0x10),
									Action: JumpAction{Target: "cali-pi-default/bi"}},
								{Match: Match().MarkSingleBitSet(0x8),
									Action:  ReturnAction{},
									Comment: []string{"Return if policy accepted"}},
								{Match: Match().MarkClear(0x10),
									Action: NflogAction{Group: 1, Prefix: "DPI|default"}},
								{Match: Match().MarkClear(0x10),
									Action:  denyAction,
									Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

								{Action: JumpAction{Target: "cali-pri-prof1"}},
								{Match: Match().MarkSingleBitSet(0x8),
									Action:  ReturnAction{},
									Comment: []string{"Return if profile accepted"}},
								{Action: JumpAction{Target: "cali-pri-prof2"}},
								{Match: Match().MarkSingleBitSet(0x8),
									Action:  ReturnAction{},
									Comment: []string{"Return if profile accepted"}},

								{Action: NflogAction{Group: 1, Prefix: "DRI"}},
								{Action: denyAction,
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							},
						},
						{
							Name: "cali-fw-cali1234",
							Rules: []Rule{
								// conntrack rules.
								{Match: Match().ConntrackState("RELATED,ESTABLISHED"),
									Action: AcceptAction{}},
								{Match: Match().ConntrackState("INVALID"),
									Action: denyAction},

								{Action: ClearMarkAction{Mark: 0x88}},
								dropVXLANRule,
								dropIPIPRule,

								{Comment: []string{"Start of tier default"},
									Action: ClearMarkAction{Mark: 0x10}},
								{Match: Match().MarkClear(0x10),
									Action: JumpAction{Target: "cali-po-default/ae"}},
								{Match: Match().MarkSingleBitSet(0x8),
									Action:  ReturnAction{},
									Comment: []string{"Return if policy accepted"}},
								{Match: Match().MarkClear(0x10),
									Action: JumpAction{Target: "cali-po-default/staged:be"}},
								{Match: Match().MarkClear(0x10),
									Action: NflogAction{Group: 2, Prefix: "DPE|default"}},
								{Match: Match().MarkClear(0x10),
									Action:  denyAction,
									Comment: []string{fmt.Sprintf("%s if no policies passed packet", denyActionString)}},

								{Action: JumpAction{Target: "cali-pro-prof1"}},
								{Match: Match().MarkSingleBitSet(0x8),
									Action:  ReturnAction{},
									Comment: []string{"Return if profile accepted"}},
								{Action: JumpAction{Target: "cali-pro-prof2"}},
								{Match: Match().MarkSingleBitSet(0x8),
									Action:  ReturnAction{},
									Comment: []string{"Return if profile accepted"}},

								{Action: NflogAction{Group: 2, Prefix: "DRE"}},
								{Action: denyAction,
									Comment: []string{fmt.Sprintf("%s if no profiles matched", denyActionString)}},
							},
						},
						{
							Name: "cali-sm-cali1234",
							Rules: []Rule{
								{Action: SetMaskedMarkAction{Mark: 0xd400, Mask: 0xff00}},
							},
						},
					})))
				})
			})
		}
	}
})

func trimSMChain(ipvsEnable bool, chains []*Chain) []*Chain {
	result := []*Chain{}
	for _, chain := range chains {
		if !ipvsEnable && strings.HasPrefix(chain.Name, "cali-sm") {
			continue
		}
		result = append(result, chain)
	}

	return result
}

func singlePolicyGroups(names []string) (groups []*PolicyGroup) {
	for _, n := range names {
		groups = append(groups, &PolicyGroup{
			Tier:        "default",
			PolicyNames: []string{n},
		})
	}
	return
}

var _ = Describe("PolicyGroups", func() {
	It("should make sensible UIDs", func() {
		pgs := []PolicyGroup{
			{
				Tier:        "default",
				Direction:   PolicyDirectionInbound,
				PolicyNames: nil,
				Selector:    "all()",
			},
			{
				Tier:        "foo",
				Direction:   PolicyDirectionInbound,
				PolicyNames: nil,
				Selector:    "all()",
			},
			{
				Tier:        "default",
				Direction:   PolicyDirectionOutbound,
				PolicyNames: nil,
				Selector:    "all()",
			},
			{
				Tier:        "default",
				Direction:   PolicyDirectionInbound,
				PolicyNames: []string{"a"},
				Selector:    "all()",
			},
			{
				Tier:        "default",
				Direction:   PolicyDirectionInbound,
				PolicyNames: nil,
				Selector:    "a == 'b'",
			},
			{
				Tier:        "default",
				Direction:   PolicyDirectionInbound,
				PolicyNames: []string{"a", "b"},
				Selector:    "all()",
			},
			{
				Tier:        "default",
				Direction:   PolicyDirectionInbound,
				PolicyNames: []string{"ab"},
				Selector:    "all()",
			},
			{
				Tier:        "default",
				Direction:   PolicyDirectionInbound,
				PolicyNames: []string{"aaa", "bbb"},
				Selector:    "all()",
			},
			{
				Tier:      "default",
				Direction: PolicyDirectionInbound,
				// Between this and the entry above, we check that the data
				// sent to the hasher is delimited somehow.
				PolicyNames: []string{"aaab", "bb"},
				Selector:    "all()",
			},
		}

		seenUIDs := map[string]PolicyGroup{}
		for _, pg := range pgs {
			uid := pg.UniqueID()
			Expect(seenUIDs).NotTo(HaveKey(uid), fmt.Sprintf("UID clash with %v", pg))
			Expect(pg.UniqueID()).To(Equal(uid), "UID different on each call")
		}
	})
})

var _ = table.DescribeTable("PolicyGroup chains",
	func(group PolicyGroup, expectedRules []Rule) {
		renderer := NewRenderer(Config{
			IptablesMarkAccept:   0x8,
			IptablesMarkPass:     0x10,
			IptablesMarkScratch0: 0x1,
			IptablesMarkScratch1: 0x2,
			IptablesMarkEndpoint: 0x4,
		})
		chains := renderer.PolicyGroupToIptablesChains(&group)
		Expect(chains).To(HaveLen(1))
		Expect(chains[0].Name).ToNot(BeEmpty())
		Expect(chains[0].Name).To(Equal(group.ChainName()))
		Expect(chains[0].Rules).To(Equal(expectedRules))
	},
	polGroupEntry(
		PolicyGroup{
			Tier:        "default",
			Direction:   PolicyDirectionInbound,
			PolicyNames: []string{"a"},
			Selector:    "all()",
		},
		[]Rule{
			{
				Action: JumpAction{Target: "cali-pi-a"},
			},
		},
	),
	polGroupEntry(
		PolicyGroup{
			Tier:        "default",
			Direction:   PolicyDirectionInbound,
			PolicyNames: []string{"a", "b"},
			Selector:    "all()",
		},
		[]Rule{
			{
				Action: JumpAction{Target: "cali-pi-a"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-b"},
			},
		},
	),
	polGroupEntry(
		PolicyGroup{
			Tier:        "default",
			Direction:   PolicyDirectionInbound,
			PolicyNames: []string{"a", "b", "c"},
			Selector:    "all()",
		},
		[]Rule{
			{
				Action: JumpAction{Target: "cali-pi-a"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-b"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-c"},
			},
		},
	),
	polGroupEntry(
		PolicyGroup{
			Tier:        "default",
			Direction:   PolicyDirectionInbound,
			PolicyNames: []string{"a", "b", "c", "d"},
			Selector:    "all()",
		},
		[]Rule{
			{
				Action: JumpAction{Target: "cali-pi-a"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-b"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-c"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-d"},
			},
		},
	),
	polGroupEntry(
		PolicyGroup{
			Tier:        "default",
			Direction:   PolicyDirectionInbound,
			PolicyNames: []string{"a", "b", "c", "d", "e"},
			Selector:    "all()",
		},
		[]Rule{
			{
				Action: JumpAction{Target: "cali-pi-a"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-b"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-c"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-d"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-e"},
			},
		},
	),
	polGroupEntry(
		PolicyGroup{
			Tier:        "default",
			Direction:   PolicyDirectionInbound,
			PolicyNames: []string{"a", "b", "c", "d", "e", "f"},
			Selector:    "all()",
		},
		[]Rule{
			{
				Action: JumpAction{Target: "cali-pi-a"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-b"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-c"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-d"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-pi-e"},
			},
			{
				// Only get a return action every 5 rules and only if it's
				// not the last action.
				Match:   Match().MarkNotClear(0x18),
				Action:  ReturnAction{},
				Comment: []string{"Return on verdict"},
			},
			{
				Action: JumpAction{Target: "cali-pi-f"},
			},
		},
	),
	polGroupEntry(
		PolicyGroup{
			Tier:        "default",
			Direction:   PolicyDirectionOutbound,
			PolicyNames: []string{"a", "b", "c", "d", "e", "f", "g"},
			Selector:    "all()",
		},
		[]Rule{
			{
				Action: JumpAction{Target: "cali-po-a"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-po-b"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-po-c"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-po-d"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-po-e"},
			},
			{
				Match:   Match().MarkNotClear(0x18),
				Action:  ReturnAction{},
				Comment: []string{"Return on verdict"},
			},
			{
				Action: JumpAction{Target: "cali-po-f"},
			},
			{
				Match:  Match().MarkClear(0x18),
				Action: JumpAction{Target: "cali-po-g"},
			},
		},
	),
)

func polGroupEntry(group PolicyGroup, rules []Rule) table.TableEntry {
	return table.Entry(
		fmt.Sprintf("%v", group),
		group,
		rules,
	)
}
