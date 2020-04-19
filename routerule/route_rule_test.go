// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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

package routerule_test

import (
	. "github.com/projectcalico/felix/routerule"

	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/testutils"
	"github.com/projectcalico/libcalico-go/lib/set"
)

var (
	FelixRouteProtocol = syscall.RTPROT_BOOT

	simulatedError = errors.New("dummy error")
	notFound       = errors.New("not found")
	alreadyExists  = errors.New("already exists")

	mac1 = testutils.MustParseMAC("00:11:22:33:44:51")
	mac2 = testutils.MustParseMAC("00:11:22:33:44:52")

	ip1  = ip.MustParseCIDROrIP("10.0.0.1/32").ToIPNet()
	ip2  = ip.MustParseCIDROrIP("10.0.0.2/32").ToIPNet()
	ip13 = ip.MustParseCIDROrIP("10.0.1.3/32").ToIPNet()
)

var _ = Describe("RouteRules Construct", func() {
	var dataplane *mockDataplane
	BeforeEach(func() {
		dataplane = &mockDataplane{
			ruleKeyToRule:   map[string]netlink.Rule{},
			addedRuleKeys:   set.New(),
			deletedRuleKeys: set.New(),
			updatedRuleKeys: set.New(),
		}

	})

	It("should not be constructable with no table index", func() {
		tableIndexSet := set.New()
		_, err := NewWithShims(
			4,
			100,
			tableIndexSet,
			RulesMatchSrcFWMarkTable,
			RulesMatchSrcFWMark,
			10*time.Second,
			dataplane.NewNetlinkHandle,
		)
		Expect(err).To(HaveOccurred())
	})

	It("should not be constructable with wrong table index", func() {
		tableIndexSet := set.New()
		tableIndexSet.Add(0)
		tableIndexSet.Add(10)
		_, err := NewWithShims(
			4,
			100,
			tableIndexSet,
			RulesMatchSrcFWMarkTable,
			RulesMatchSrcFWMark,
			10*time.Second,
			dataplane.NewNetlinkHandle,
		)
		Expect(err).To(HaveOccurred())

		tableIndexSet.Discard(0)
		tableIndexSet.Add(252)
		_, err = NewWithShims(
			4,
			100,
			tableIndexSet,
			RulesMatchSrcFWMarkTable,
			RulesMatchSrcFWMark,
			10*time.Second,
			dataplane.NewNetlinkHandle,
		)
		Expect(err).To(HaveOccurred())
	})

	It("should be constructable", func() {
		tableIndexSet := set.New()
		tableIndexSet.Add(1)
		tableIndexSet.Add(10)
		tableIndexSet.Add(250)
		_, err := NewWithShims(
			4,
			100,
			tableIndexSet,
			RulesMatchSrcFWMarkTable,
			RulesMatchSrcFWMark,
			10*time.Second,
			dataplane.NewNetlinkHandle,
		)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("RouteRules", func() {
	var dataplane *mockDataplane
	var rrs *RouteRules

	BeforeEach(func() {
		dataplane = &mockDataplane{
			ruleKeyToRule:   map[string]netlink.Rule{},
			addedRuleKeys:   set.New(),
			deletedRuleKeys: set.New(),
			updatedRuleKeys: set.New(),
		}

		tableIndexSet := set.New()
		tableIndexSet.Add(1)
		tableIndexSet.Add(10)
		tableIndexSet.Add(250)
		rrs, err := NewWithShims(
			4,
			100,
			tableIndexSet,
			RulesMatchSrcFWMarkTable,
			RulesMatchSrcFWMark,
			10*time.Second,
			dataplane.NewNetlinkHandle,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(rrs).ToNot(BeNil())
	})

	Describe("with existing cali and nonCali rules", func() {
		var cali1Rule, cali2Rule, nonCaliRule netlink.Rule
		BeforeEach(func() {
			// Calico rule in both control plane and dataplane.
			cali1Rule = netlink.Rule{
				Priority: 100,
				Family:   4,
				Src:      mustParseCIDR("10.0.0.1/32"),
				Mark:     0x100,
				Mask:     0x100,
				Table:    1,
				Invert:   true,
			}
			dataplane.addMockRule(&cali1Rule)
			rrs.SetRule(FromNetlinkRule(&cali1Rule))

			// Calico rule in dataplane only.
			cali2Rule = netlink.Rule{
				Priority: 100,
				Family:   4,
				Src:      mustParseCIDR("10.0.0.2/32"),
				Mark:     0x200,
				Mask:     0x200,
				Table:    10,
				Invert:   true,
			}
			dataplane.addMockRule(&cali2Rule)

			// Non Calico rule in dataplane only.
			nonCaliRule = netlink.Rule{
				Priority: 100,
				Family:   4,
				Src:      mustParseCIDR("10.0.0.1/32"),
				Mark:     0x800,
				Mask:     0x800,
				Table:    90,
				Invert:   true,
			}
			dataplane.addMockRule(&nonCaliRule)
		})

		It("should remove Calico rule not in control plane", func() {
			err := rrs.Apply()
			Expect(err).ToNot(HaveOccurred())
			Expect(dataplane.ruleKeyToRule).To(ConsistOf(cali1Rule, nonCaliRule))
		})
		It("should add rule with correct table index", func() {
			rule := NewRule(4, 100).
				MatchSrcAddress(*mustParseCIDR("10.0.0.3/32")).
				MatchFWMark(0x400).
				GoToTable(250)
			rrs.SetRule(rule)
			err := rrs.Apply()
			Expect(err).ToNot(HaveOccurred())
			Expect(dataplane.ruleKeyToRule).To(ConsistOf(cali1Rule, rule.NetLinkRule(), nonCaliRule))
		})
		It("should panic addimg rule with table index not managed by calico", func() {
			err := rrs.Apply()
			Expect(err).ToNot(HaveOccurred())
			Expect(dataplane.ruleKeyToRule).To(ConsistOf(nonCaliRule))
			rule := NewRule(4, 100).
				MatchSrcAddress(*mustParseCIDR("10.0.0.3/32")).
				MatchFWMark(0x400).
				GoToTable(249)
			Expect(func() { rrs.SetRule(rule) }).To(Panic())
		})
		It("should remove rule", func() {
			rule := NewRule(4, 100).
				MatchSrcAddress(*mustParseCIDR("10.0.0.1/32")).
				MatchFWMark(0x100)
			rrs.RemoveRule(rule)
			err := rrs.Apply()
			Expect(err).ToNot(HaveOccurred())
			Expect(dataplane.ruleKeyToRule).To(ConsistOf(nonCaliRule))
		})

		Describe("with a persistent failure to connect", func() {
			BeforeEach(func() {
				dataplane.PersistentlyFailToConnect = true
			})

			It("should panic after all its retries are exhausted", func() {
				for i := 0; i < 3; i++ {
					Expect(rrs.Apply()).To(Equal(ConnectFailed))
				}
				Expect(func() { _ = rrs.Apply() }).To(Panic())
			})
		})

		// We do the following tests in different failure (and non-failure) scenarios.  In
		// each case, we make the failure transient so that only the first Apply() should
		// fail.  Then, at most, the second call to Apply() should succeed.
		for _, failFlags := range failureScenarios {
			failFlags := failFlags
			desc := fmt.Sprintf("with some rules added and failures: %v", failFlags)
			Describe(desc, func() {
				BeforeEach(func() {
					rrs.SetRule(NewRule(4, 100))
					dataplane.failuresToSimulate = failFlags
				})
				JustBeforeEach(func() {
					maxTries := 1
					if failFlags != 0 {
						maxTries = 2
					}
					for try := 0; try < maxTries; try++ {
						err := rrs.Apply()
						if err == nil {
							// We should only need to retry if Apply returns an error.
							log.Info("Apply returned no error, breaking out of loop")
							break
						}
					}
				})
				It("should have consumed all failures", func() {
					// Check that all the failures we simulated were hit.
					Expect(dataplane.failuresToSimulate).To(Equal(failNone))
				})
				It("should keep correct rule", func() {
					Expect(dataplane.ruleKeyToRule["1-10.0.0.1/32"]).To(Equal(netlink.Rule{
						LinkIndex: 1,
						Dst:       &ip1,
						Type:      syscall.RTN_UNICAST,
						Protocol:  FelixRuleProtocol,
						Scope:     netlink.SCOPE_LINK,
					}))
					Expect(dataplane.addedRuleKeys.Contains("1-10.0.0.1/32")).To(BeFalse())
				})
				It("should add new rule", func() {
					Expect(dataplane.ruleKeyToRule["2-10.0.0.2/32"]).To(Equal(netlink.Rule{
						LinkIndex: 2,
						Dst:       &ip2,
						Type:      syscall.RTN_UNICAST,
						Protocol:  FelixRuleProtocol,
						Scope:     netlink.SCOPE_LINK,
					}))
				})
				It("should update changed rule", func() {
					Expect(dataplane.ruleKeyToRule["3-10.0.1.3/32"]).To(Equal(netlink.Rule{
						LinkIndex: 3,
						Dst:       &ip13,
						Type:      syscall.RTN_UNICAST,
						Protocol:  FelixRuleProtocol,
						Scope:     netlink.SCOPE_LINK,
					}))
					Expect(dataplane.deletedRuleKeys.Contains("3-10.0.0.3/32")).To(BeTrue())
				})
				It("should have expected number of rules at the end", func() {
					Expect(len(dataplane.ruleKeyToRule)).To(Equal(4),
						fmt.Sprintf("Wrong number of rules %v: %v",
							len(dataplane.ruleKeyToRule),
							dataplane.ruleKeyToRule))
				})
				if failFlags&(failNextSetSocketTimeout|
					failNextNewNetlinkHandle|
					failNextRuleAdd|
					failNextRuleDel|
					failNextRuleList) != 0 {
					It("should reconnect to netlink", func() {
						Expect(dataplane.NumNewNetlinkCalls).To(Equal(2))
					})
				} else {
					It("should not reconnect to netlink", func() {
						Expect(dataplane.NumNewNetlinkCalls).To(Equal(1))
					})
				}
			})
		}
	})
})

var _ = Describe("Tests to verify netlink interface", func() {
	It("Should give expected error for missing interface", func() {
		_, err := netlink.LinkByName("dsfhjakdhfjk")
		Expect(err.Error()).To(ContainSubstring("not found"))
	})
})

func mustParseCIDR(cidr string) *net.IPNet {
	_, c, err := net.ParseCIDR(cidr)
	Expect(err).NotTo(HaveOccurred())
	return c
}

type failFlags uint32

const (
	failNextRuleList failFlags = 1 << iota
	failNextRuleAdd
	failNextRuleDel
	failNextNewNetlinkHandle
	failNextSetSocketTimeout
	failNone failFlags = 0
)

var failureScenarios = []failFlags{
	failNone,
	failNextRuleList,
	failNextRuleAdd,
	failNextRuleDel,
	failNextNewNetlinkHandle,
	failNextSetSocketTimeout,
}

func (f failFlags) String() string {
	parts := []string{}
	if f&failNextRuleList != 0 {
		parts = append(parts, "failNextRuleList")
	}
	if f&failNextRuleAdd != 0 {
		parts = append(parts, "failNextRuleAdd")
	}
	if f&failNextRuleDel != 0 {
		parts = append(parts, "failNextRuleDel")
	}
	if f&failNextNewNetlinkHandle != 0 {
		parts = append(parts, "failNextNewNetlinkHandle")
	}
	if f&failNextSetSocketTimeout != 0 {
		parts = append(parts, "failNextSetSocketTimeout")
	}
	if f == 0 {
		parts = append(parts, "failNone")
	}
	return strings.Join(parts, "|")
}

type mockDataplane struct {
	ruleKeyToRule   map[string]netlink.Rule
	addedRuleKeys   set.Set
	deletedRuleKeys set.Set
	updatedRuleKeys set.Set

	NumNewNetlinkCalls int
	NetlinkOpen        bool

	PersistentlyFailToConnect bool

	failuresToSimulate failFlags
}

func (d *mockDataplane) shouldFail(flag failFlags) bool {
	flagPresent := d.failuresToSimulate&flag != 0
	d.failuresToSimulate &^= flag
	if flagPresent {
		log.WithField("flag", flag).Warn("Mock dataplane: triggering failure")
	}
	return flagPresent
}

func (d *mockDataplane) NewNetlinkHandle() (HandleIface, error) {
	d.NumNewNetlinkCalls++
	if d.PersistentlyFailToConnect || d.shouldFail(failNextNewNetlinkHandle) {
		return nil, simulatedError
	}
	Expect(d.NetlinkOpen).To(BeFalse())
	d.NetlinkOpen = true
	return d, nil
}

func (d *mockDataplane) Delete() {
	Expect(d.NetlinkOpen).To(BeTrue())
	d.NetlinkOpen = false
}

func (d *mockDataplane) SetSocketTimeout(to time.Duration) error {
	Expect(d.NetlinkOpen).To(BeTrue())
	if d.shouldFail(failNextSetSocketTimeout) {
		return simulatedError
	}
	return nil
}

func (d *mockDataplane) RuleList(family int) ([]netlink.Rule, error) {
	Expect(d.NetlinkOpen).To(BeTrue())
	if d.shouldFail(failNextRuleList) {
		return nil, simulatedError
	}
	var rules []netlink.Rule
	for _, rule := range d.ruleKeyToRule {
		if rule.Family == family {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (d *mockDataplane) addMockRule(rule *netlink.Rule) {
	key := keyForRule(rule)
	d.ruleKeyToRule[key] = *rule
}

func (d *mockDataplane) removeMockRule(rule *netlink.Rule) {
	key := keyForRule(rule)
	delete(d.ruleKeyToRule, key)
}

func (d *mockDataplane) RuleAdd(rule *netlink.Rule) error {
	Expect(d.NetlinkOpen).To(BeTrue())
	if d.shouldFail(failNextRuleAdd) {
		return simulatedError
	}
	key := keyForRule(rule)
	log.WithField("ruleKey", key).Info("Mock dataplane: RuleAdd called")
	d.addedRuleKeys.Add(key)
	if _, ok := d.ruleKeyToRule[key]; ok {
		return alreadyExists
	} else {
		d.ruleKeyToRule[key] = *rule
		return nil
	}
}

func (d *mockDataplane) RuleDel(rule *netlink.Rule) error {
	Expect(d.NetlinkOpen).To(BeTrue())
	if d.shouldFail(failNextRuleDel) {
		return simulatedError
	}
	key := keyForRule(rule)
	log.WithField("ruleKey", key).Info("Mock dataplane: RuleDel called")
	d.deletedRuleKeys.Add(key)
	// Rule was deleted, but is planned on being readded
	if _, ok := d.ruleKeyToRule[key]; ok {
		delete(d.ruleKeyToRule, key)
		d.updatedRuleKeys.Add(key)
		return nil
	} else {
		return nil
	}
}

func keyForRule(rule *netlink.Rule) string {
	key := fmt.Sprintf("%s-%#x-%#x", rule.Src.String(), rule.Mark, rule.Mask)
	log.WithField("ruleKey", key).Debug("Calculated rule key")
	return key
}
