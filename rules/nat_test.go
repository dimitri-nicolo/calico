// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.
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
	"net"

	"github.com/tigera/api/pkg/lib/numorstring"
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
			IptablesMarkEgress:          0x4,
			IptablesMarkAccept:          0x8,
			IptablesMarkPass:            0x10,
			IptablesMarkScratch0:        0x20,
			IptablesMarkScratch1:        0x40,
			IptablesMarkDrop:            0x80,
			IptablesMarkEndpoint:        0xff00,
			IptablesMarkNonCaliEndpoint: 0x100,
			EgressIPInterface:           "egress.calico",
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
	It("should render rules when active with an explicit SNAT address", func() {
		snatAddress := "192.168.0.1"
		localConfig := rrConfigNormal
		localConfig.NATOutgoingAddress = net.ParseIP(snatAddress)
		renderer = NewRenderer(localConfig)

		Expect(renderer.NATOutgoingChain(true, 4)).To(Equal(&Chain{
			Name: "cali-nat-outgoing",
			Rules: []Rule{
				{
					Action: SNATAction{ToAddr: snatAddress},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools"),
				},
			},
		}))
	})
	It("should render rules when active with explicit port range", func() {

		//copy struct
		localConfig := rrConfigNormal
		localConfig.NATPortRange, _ = numorstring.PortFromRange(99, 100)
		renderer = NewRenderer(localConfig)

		Expect(renderer.NATOutgoingChain(true, 4)).To(Equal(&Chain{
			Name: "cali-nat-outgoing",
			Rules: []Rule{
				{
					Action: MasqAction{ToPorts: "99-100"},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("tcp"),
				},
				{
					Action: ReturnAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("tcp"),
				},
				{
					Action: MasqAction{ToPorts: "99-100"},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("udp"),
				},
				{
					Action: ReturnAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("udp"),
				},
				{
					Action: MasqAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools"),
				},
			},
		}))
	})
	It("should render rules when active with explicit port range", func() {

		//copy struct
		localConfig := rrConfigNormal
		localConfig.NATPortRange, _ = numorstring.PortFromRange(99, 100)
		localConfig.IptablesNATOutgoingInterfaceFilter = "cali-123"
		renderer = NewRenderer(localConfig)

		Expect(renderer.NATOutgoingChain(true, 4)).To(Equal(&Chain{
			Name: "cali-nat-outgoing",
			Rules: []Rule{
				{
					Action: MasqAction{ToPorts: "99-100"},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("tcp").
						OutInterface("cali-123"),
				},
				{
					Action: ReturnAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("tcp").
						OutInterface("cali-123"),
				},
				{
					Action: MasqAction{ToPorts: "99-100"},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("udp").
						OutInterface("cali-123"),
				},
				{
					Action: ReturnAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("udp").
						OutInterface("cali-123"),
				},
				{
					Action: MasqAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").
						OutInterface("cali-123"),
				},
			},
		}))
	})
	It("should render rules when active with explicit port range and an explicit SNAT address", func() {

		snatAddress := "192.168.0.1"
		//copy struct
		localConfig := rrConfigNormal
		localConfig.NATPortRange, _ = numorstring.PortFromRange(99, 100)
		localConfig.NATOutgoingAddress = net.ParseIP(snatAddress)
		renderer = NewRenderer(localConfig)

		expectedAddress := fmt.Sprintf("%s:%s", snatAddress, "99-100")

		Expect(renderer.NATOutgoingChain(true, 4)).To(Equal(&Chain{
			Name: "cali-nat-outgoing",
			Rules: []Rule{
				{
					Action: SNATAction{ToAddr: expectedAddress},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("tcp"),
				},
				{
					Action: ReturnAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("tcp"),
				},
				{
					Action: SNATAction{ToAddr: expectedAddress},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("udp"),
				},
				{
					Action: ReturnAction{},
					Match: Match().
						SourceIPSet("cali40masq-ipam-pools").
						NotDestIPSet("cali40all-ipam-pools").Protocol("udp"),
				},
				{
					Action: SNATAction{ToAddr: snatAddress},
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
							NotDestIPSet("cali40all-hosts-net"),
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
		It("should render rules when active with explicit port range", func() {

			//copy struct
			localConfig := rrConfigNormal
			localConfig.NATPortRange, _ = numorstring.PortFromRange(99, 100)
			renderer = NewRenderer(localConfig)

			Expect(renderer.NATOutgoingChain(true, 4)).To(Equal(&Chain{
				Name: "cali-nat-outgoing",
				Rules: []Rule{
					{
						Action: MasqAction{ToPorts: "99-100"},
						Match: Match().
							SourceIPSet("cali40masq-ipam-pools").
							NotDestIPSet("cali40all-ipam-pools").
							NotDestIPSet("cali40all-hosts-net").Protocol("tcp"),
					},
					{
						Action: ReturnAction{},
						Match: Match().
							SourceIPSet("cali40masq-ipam-pools").
							NotDestIPSet("cali40all-ipam-pools").
							NotDestIPSet("cali40all-hosts-net").Protocol("tcp"),
					},
					{
						Action: MasqAction{ToPorts: "99-100"},
						Match: Match().
							SourceIPSet("cali40masq-ipam-pools").
							NotDestIPSet("cali40all-ipam-pools").
							NotDestIPSet("cali40all-hosts-net").Protocol("udp"),
					},
					{
						Action: ReturnAction{},
						Match: Match().
							SourceIPSet("cali40masq-ipam-pools").
							NotDestIPSet("cali40all-ipam-pools").
							NotDestIPSet("cali40all-hosts-net").Protocol("udp"),
					},
					{
						Action: MasqAction{},
						Match: Match().
							SourceIPSet("cali40masq-ipam-pools").
							NotDestIPSet("cali40all-ipam-pools").
							NotDestIPSet("cali40all-hosts-net"),
					},
				},
			}))
		})
	})

	Describe("with Egress IP enabled", func() {
		BeforeEach(func() { rrConfigNormal.EgressIPEnabled = true })

		It("should render v4 rules correctly rules when active", func() {
			Expect(renderer.NATOutgoingChain(true, 4)).To(Equal(&Chain{
				Name: "cali-nat-outgoing",
				Rules: []Rule{
					{
						Action: MasqAction{},
						Match: Match().
							SourceIPSet("cali40masq-ipam-pools").
							NotDestIPSet("cali40all-ipam-pools").
							NotOutInterface("egress.calico"),
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
