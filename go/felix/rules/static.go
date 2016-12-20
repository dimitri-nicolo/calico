// Copyright (c) 2016 Tigera, Inc. All rights reserved.
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

package rules

import (
	"github.com/Sirupsen/logrus"
	. "github.com/projectcalico/felix/go/felix/iptables"
	"strings"
)

func (r *ruleRenderer) StaticFilterTableChains() (chains []*Chain) {
	chains = append(chains, r.StaticFilterForwardChains()...)
	chains = append(chains, r.StaticFilterInputChains()...)
	chains = append(chains, r.StaticFilterOutputChains()...)
	return
}

func (r *ruleRenderer) StaticFilterInputChains() []*Chain {
	// TODO(smc) fitler input chain
	return []*Chain{}
}

func (r *ruleRenderer) StaticFilterForwardChains() []*Chain {
	rules := []Rule{}

	for _, prefix := range r.WorkloadIfacePrefixes {
		logrus.WithField("ifacePrefix", prefix).Debug("Adding workload match rules")
		ifaceMatch := prefix + "+"
		rules = append(rules, r.DropRules(Match().InInterface(ifaceMatch).ConntrackState("INVALID"))...)
		rules = append(rules,
			Rule{
				Match:  Match().InInterface(ifaceMatch).ConntrackState("RELATED,ESTABLISHED"),
				Action: AcceptAction{},
			},
			Rule{
				Match:  Match().OutInterface(ifaceMatch).ConntrackState("RELATED,ESTABLISHED"),
				Action: AcceptAction{},
			},
			Rule{
				Match:  Match().InInterface(ifaceMatch),
				Action: JumpAction{Target: DispatchFromWorkloadEndpoint},
			},
			Rule{
				Match:  Match().OutInterface(ifaceMatch),
				Action: JumpAction{Target: DispatchToWorkloadEndpoint},
			},
			Rule{
				Match:  Match().InInterface(ifaceMatch),
				Action: AcceptAction{},
			},
			Rule{
				Match:  Match().OutInterface(ifaceMatch),
				Action: AcceptAction{},
			})
	}

	return []*Chain{{
		Name:  FilterForwardChainName,
		Rules: rules,
	}}
}

func (r *ruleRenderer) StaticFilterOutputChains() []*Chain {
	// TODO(smc) filter output chain
	return []*Chain{}
}

func (r *ruleRenderer) StaticNATTableChains(ipVersion uint8) (chains []*Chain) {
	chains = append(chains, r.StaticNATPreroutingChains(ipVersion)...)
	chains = append(chains, r.StaticNATPostroutingChains(ipVersion)...)
	return
}

func (r *ruleRenderer) StaticNATPreroutingChains(ipVersion uint8) []*Chain {
	rules := []Rule{}

	if ipVersion == 4 && r.OpenStackMetadataIP != nil {
		rules = append(rules, Rule{
			Match: Match().
				Protocol("tcp").
				DestPorts(80).
				DestNet("169.254.169.254/32"),
			Action: DNATAction{
				DestAddr: r.OpenStackMetadataIP.String(),
				DestPort: r.OpenStackMetadataPort,
			},
		})
	}

	return []*Chain{{
		Name:  NATPreroutingChainName,
		Rules: rules,
	}}
}

func (r *ruleRenderer) StaticNATPostroutingChains(ipVersion uint8) []*Chain {
	return []*Chain{{
		Name: NATPostroutingChainName,
		Rules: []Rule{
			{
				Action: JumpAction{Target: NATOutgoingChainName},
			},
		},
	}}
}

func (t ruleRenderer) DropRules(matchCriteria MatchCriteria, comments ...string) []Rule {
	return []Rule{
		{
			Match:   matchCriteria,
			Action:  DropAction{},
			Comment: strings.Join(comments, "; "),
		},
	}
}
