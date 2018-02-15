// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.
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
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/hashutils"
	. "github.com/projectcalico/felix/iptables"
	"github.com/projectcalico/felix/proto"
)

const (
	ingressPolicy = "ingress"
	egressPolicy  = "egress"
)

func (r *DefaultRuleRenderer) WorkloadEndpointToIptablesChains(
	ifaceName string,
	adminUp bool,
	tiers []*proto.TierInfo,
	profileIDs []string,
) []*Chain {
	return []*Chain{
		// Chain for traffic _to_ the endpoint.
		r.endpointIptablesChain(
			tiers,
			profileIDs,
			ifaceName,
			PolicyInboundPfx,
			ProfileInboundPfx,
			WorkloadToEndpointPfx,
			"", // No fail-safe chains for workloads.
			chainTypeNormal,
			adminUp,
			uint16(1),
			"inbound",
			ingressPolicy,
			r.filterAllowAction, // Workload endpoint chains are only used in the filter table
		),
		// Chain for traffic _from_ the endpoint.
		r.endpointIptablesChain(
			tiers,
			profileIDs,
			ifaceName,
			PolicyOutboundPfx,
			ProfileOutboundPfx,
			WorkloadFromEndpointPfx,
			"", // No fail-safe chains for workloads.
			chainTypeNormal,
			adminUp,
			uint16(2),
			"outbound",
			egressPolicy,
			r.filterAllowAction, // Workload endpoint chains are only used in the filter table
		),
	}
}

func (r *DefaultRuleRenderer) HostEndpointToFilterChains(
	ifaceName string,
	tiers []*proto.TierInfo,
	forwardTiers []*proto.TierInfo,
	profileIDs []string,
) []*Chain {
	log.WithField("ifaceName", ifaceName).Debug("Rendering filter host endpoint chain.")
	return []*Chain{
		// Chain for output traffic _to_ the endpoint.
		r.endpointIptablesChain(
			tiers,
			profileIDs,
			ifaceName,
			PolicyOutboundPfx,
			ProfileOutboundPfx,
			HostToEndpointPfx,
			ChainFailsafeOut,
			chainTypeNormal,
			true, // Host endpoints are always admin up.
			uint16(2),
			"outbound",
			egressPolicy,
			r.filterAllowAction,
		),
		// Chain for input traffic _from_ the endpoint.
		r.endpointIptablesChain(
			tiers,
			profileIDs,
			ifaceName,
			PolicyInboundPfx,
			ProfileInboundPfx,
			HostFromEndpointPfx,
			ChainFailsafeIn,
			chainTypeNormal,
			true, // Host endpoints are always admin up.
			uint16(1),
			"inbound",
			ingressPolicy,
			r.filterAllowAction,
		),
		// Chain for forward traffic _to_ the endpoint.
		r.endpointIptablesChain(
			forwardTiers,
			profileIDs,
			ifaceName,
			PolicyOutboundPfx,
			ProfileOutboundPfx,
			HostToEndpointForwardPfx,
			"", // No fail-safe chains for forward traffic.
			chainTypeForward,
			true, // Host endpoints are always admin up.
			uint16(2),
			"outbound",
			egressPolicy,
			r.filterAllowAction,
		),
		// Chain for forward traffic _from_ the endpoint.
		r.endpointIptablesChain(
			forwardTiers,
			profileIDs,
			ifaceName,
			PolicyInboundPfx,
			ProfileInboundPfx,
			HostFromEndpointForwardPfx,
			"", // No fail-safe chains for forward traffic.
			chainTypeForward,
			true, // Host endpoints are always admin up.
			uint16(1),
			"inbound",
			ingressPolicy,
			r.filterAllowAction,
		),
	}
}

func (r *DefaultRuleRenderer) HostEndpointToRawChains(
	ifaceName string,
	untrackedTiers []*proto.TierInfo,
) []*Chain {
	log.WithField("ifaceName", ifaceName).Debugf("Rendering raw (untracked) host endpoint chain. - untrackedTiers %+v", untrackedTiers)
	return []*Chain{
		// Chain for traffic _to_ the endpoint.
		r.endpointIptablesChain(
			untrackedTiers,
			nil, // We don't render profiles into the raw table.
			ifaceName,
			PolicyOutboundPfx,
			ProfileOutboundPfx,
			HostToEndpointPfx,
			ChainFailsafeOut,
			chainTypeUntracked,
			true, // Host endpoints are always admin up.
			uint16(2),
			"outbound",
			egressPolicy,
			AcceptAction{},
		),
		// Chain for traffic _from_ the endpoint.
		r.endpointIptablesChain(
			untrackedTiers,
			nil, // We don't render profiles into the raw table.
			ifaceName,
			PolicyInboundPfx,
			ProfileInboundPfx,
			HostFromEndpointPfx,
			ChainFailsafeIn,
			chainTypeUntracked,
			true, // Host endpoints are always admin up.
			uint16(1),
			"inbound",
			ingressPolicy,
			AcceptAction{},
		),
	}
}

func (r *DefaultRuleRenderer) HostEndpointToMangleChains(
	ifaceName string,
	preDNATTiers []*proto.TierInfo,
) []*Chain {
	log.WithField("ifaceName", ifaceName).Debug("Rendering pre-DNAT host endpoint chain.")
	return []*Chain{
		// Chain for traffic _from_ the endpoint.  Pre-DNAT policy does not apply to
		// outgoing traffic through a host endpoint.
		r.endpointIptablesChain(
			preDNATTiers,
			nil, // We don't render profiles into the raw table.
			ifaceName,
			PolicyInboundPfx,
			ProfileInboundPfx,
			HostFromEndpointPfx,
			ChainFailsafeIn,
			chainTypePreDNAT,
			true, // Host endpoints are always admin up.
			uint16(1),
			"inbound",
			ingressPolicy,
			r.mangleAllowAction,
		),
	}
}

type endpointChainType int

const (
	chainTypeNormal endpointChainType = iota
	chainTypeUntracked
	chainTypePreDNAT
	chainTypeForward
)

func (r *DefaultRuleRenderer) endpointIptablesChain(
	tiers []*proto.TierInfo,
	profileIds []string,
	name string,
	policyPrefix PolicyChainNamePrefix,
	profilePrefix ProfileChainNamePrefix,
	endpointPrefix string,
	failsafeChain string,
	chainType endpointChainType,
	adminUp bool,
	nflogGroup uint16,
	directionSuffix string,
	policyType string,
	allowAction Action,
) *Chain {
	rules := []Rule{}
	chainName := EndpointChainName(endpointPrefix, name)

	if !adminUp {
		// Endpoint is admin-down, drop all traffic to/from it.
		rules = append(rules, r.DropRules(Match(), "Endpoint admin disabled")...)
		return &Chain{
			Name:  chainName,
			Rules: rules,
		}
	}

	if chainType != chainTypeUntracked {
		// Tracked chain: install conntrack rules, which implement our stateful connections.
		// This allows return traffic associated with a previously-permitted request.
		rules = r.appendConntrackRules(rules, allowAction)
	}

	// First set up failsafes.
	if failsafeChain != "" {
		rules = append(rules, Rule{
			Action: JumpAction{Target: failsafeChain},
		})
	}

	// Start by ensuring that the accept mark bit is clear, policies set that bit to indicate
	// that they accepted / dropped the packet.
	rules = append(rules, Rule{
		Action: ClearMarkAction{
			Mark: r.IptablesMarkAccept + r.IptablesMarkDrop,
		},
	})

	for _, tier := range tiers {
		var policies []string
		if policyType == ingressPolicy {
			policies = tier.IngressPolicies
		} else {
			policies = tier.EgressPolicies
		}
		if len(policies) > 0 {
			// Clear the "pass" mark.  If a policy sets that mark, we'll skip the rest of the policies and
			// continue processing the profiles, if there are any.
			rules = append(rules, Rule{
				Comment: "Start of tier " + tier.Name,
				Action: ClearMarkAction{
					Mark: r.IptablesMarkPass,
				},
			})

			// Then, jump to each policy in turn.
			for _, polID := range policies {
				polChainName := PolicyChainName(
					policyPrefix,
					&proto.PolicyID{Tier: tier.Name, Name: polID},
				)
				// If a previous policy didn't set the "pass" mark, jump to the policy.
				rules = append(rules, Rule{
					Match:  Match().MarkClear(r.IptablesMarkPass),
					Action: JumpAction{Target: polChainName},
				})
				// If policy marked packet as accepted, it returns, setting the accept
				// mark bit.
				if chainType == chainTypeUntracked {
					// For an untracked policy, map allow to "NOTRACK and ALLOW".
					rules = append(rules, Rule{
						Match:  Match().MarkSet(r.IptablesMarkAccept),
						Action: NoTrackAction{},
					})
				}
				// If accept bit is set, return from this chain.  We don't immediately
				// accept because there may be other policy still to apply.
				rules = append(rules, Rule{
					Match:   Match().MarkSet(r.IptablesMarkAccept),
					Action:  ReturnAction{},
					Comment: "Return if policy accepted",
				})
			}

			if chainType == chainTypeNormal || chainType == chainTypeForward {
				// When rendering normal and forward rules, if no policy marked the packet as "pass",
				// drop the packet.
				//
				// For untracked and pre-DNAT rules, we don't do that because there may be
				// normal rules still to be applied to the packet in the filter table.
				rules = append(rules, Rule{
					Match: Match().MarkClear(r.IptablesMarkPass),
					Action: NflogAction{
						Group:  nflogGroup,
						Prefix: fmt.Sprintf("D|0|no-policy-match-%s|po", directionSuffix),
					},
				})
				rules = append(rules, r.DropRules(
					Match().MarkClear(r.IptablesMarkPass),
					"Drop if no policies passed packet")...)
			}
		}
	}

	if chainType == chainTypeNormal {
		// Then, jump to each profile in turn.
		for _, profileID := range profileIds {
			profChainName := ProfileChainName(profilePrefix, &proto.ProfileID{Name: profileID})
			rules = append(rules,
				Rule{Action: JumpAction{Target: profChainName}},
				// If policy marked packet as accepted, it returns, setting the
				// accept mark bit.  If that is set, return from this chain.
				Rule{
					Match:   Match().MarkSet(r.IptablesMarkAccept),
					Action:  ReturnAction{},
					Comment: "Return if profile accepted",
				})
		}

		// When rendering normal rules, if no profile marked the packet as accepted, drop
		// the packet.
		//
		// For untracked rules, we don't do that because there may be tracked rules
		// still to be applied to the packet in the filter table.
		// TODO (Matt): This (and the policy equivalent just above) can probably be refactored.
		//              At least the magic 1 and 2 need to be combined with the equivalent in CalculateActions.
		// No profile matched the packet: drop it.
		rules = append(rules, Rule{
			Match: Match(),
			Action: NflogAction{
				Group:  nflogGroup,
				Prefix: fmt.Sprintf("D|0|no-profile-match-%s|pr", directionSuffix),
			},
		})
		rules = append(rules, r.DropRules(Match(), "Drop if no profiles matched")...)
	}

	return &Chain{
		Name:  chainName,
		Rules: rules,
	}
}

func (r *DefaultRuleRenderer) appendConntrackRules(rules []Rule, allowAction Action) []Rule {
	// Allow return packets for established connections.
	if allowAction != (AcceptAction{}) {
		// If we've been asked to return instead of accept the packet immediately,
		// make sure we flag the packet as allowed.
		rules = append(rules,
			Rule{
				Match:  Match().ConntrackState("RELATED,ESTABLISHED"),
				Action: SetMarkAction{Mark: r.IptablesMarkAccept},
			},
		)
	}
	rules = append(rules,
		Rule{
			Match:  Match().ConntrackState("RELATED,ESTABLISHED"),
			Action: allowAction,
		},
	)
	if !r.Config.DisableConntrackInvalid {
		// Drop packets that aren't either a valid handshake or part of an established
		// connection.
		rules = append(rules, Rule{
			Match:  Match().ConntrackState("INVALID"),
			Action: DropAction{},
		})
	}
	return rules
}

func EndpointChainName(prefix string, ifaceName string) string {
	return hashutils.GetLengthLimitedID(
		prefix,
		ifaceName,
		MaxChainNameLength,
	)
}
