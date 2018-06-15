// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.
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
	. "github.com/projectcalico/felix/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/ipsets"
	. "github.com/projectcalico/felix/iptables"
)

var _ = Describe("NAT", func() {
	var rrConfigNormal Config
	BeforeEach(func() {
		rrConfigNormal = Config{
			IPIPEnabled:                 true,
			IPIPTunnelAddress:           nil,
			IPSetConfigV4:               ipsets.NewIPVersionConfig(ipsets.IPFamilyV4, "cali", nil, nil),
			IPSetConfigV6:               ipsets.NewIPVersionConfig(ipsets.IPFamilyV6, "cali", nil, nil),
			IptablesMarkAccept:          0x8,
			IptablesMarkPass:            0x10,
			IptablesMarkScratch0:        0x20,
			IptablesMarkScratch1:        0x40,
			IptablesMarkDrop:            0x80,
			IptablesMarkEndpoint:        0xff00,
			IptablesMarkNonCaliEndpoint: 0x100,
		}
	})

	var renderer RuleRenderer
	JustBeforeEach(func() {
		renderer = NewRenderer(rrConfigNormal)
	})

	It("should render rules when active", func() {
		Expect(renderer.NATOutgoingChain(true, 4)).To(Equal(&Chain{
			Name: "cali-nat-outgoing",
			Rules: []Rule{
				{
					Action: MasqAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools"),
				},
			},
		}))
	})
	It("should render nothing when inactive", func() {
		Expect(renderer.NATOutgoingChain(false, 4)).To(Equal(&Chain{
			Name:  "cali-nat-outgoing",
			Rules: nil,
		}))
	})

	Describe("with IPsec enabled", func() {
		BeforeEach(func() { rrConfigNormal.IPSecEnabled = true })

		It("should render v4 rules correctly rules when active", func() {
			Expect(renderer.NATOutgoingChain(true, 4)).To(Equal(&Chain{
				Name: "cali-nat-outgoing",
				Rules: []Rule{
					{
						Action: MasqAction{},
						Match: Match().
							SourceIPSet("cali40masq-ipam-pools").
							NotDestIPSet("cali40all-ipam-pools").
							NotDestIPSet("cali40all-hosts"),
					},
				},
			}))
		})
		It("should render v6 rules correctly rules when active", func() {
			Expect(renderer.NATOutgoingChain(true, 6)).To(Equal(&Chain{
				Name: "cali-nat-outgoing",
				Rules: []Rule{
					{
						Action: MasqAction{},
						Match: Match().
							SourceIPSet("cali60masq-ipam-pools").
							NotDestIPSet("cali60all-ipam-pools"),
					},
				},
			}))
		})
		It("should render nothing when inactive", func() {
			Expect(renderer.NATOutgoingChain(false, 4)).To(Equal(&Chain{
				Name:  "cali-nat-outgoing",
				Rules: nil,
			}))
		})
	})
})
