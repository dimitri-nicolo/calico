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
	log "github.com/Sirupsen/logrus"

	"github.com/projectcalico/felix/hashutils"
	. "github.com/projectcalico/felix/iptables"
	"github.com/projectcalico/felix/proto"
)

func (r *DefaultRuleRenderer) WorkloadEndpointToIptablesChains(
	ifaceName string,
	adminUp bool,
	tiers []*proto.TierInfo,
	profileIDs []string,
) []*Chain {
	return r.endpointToIptablesChains(
		tiers,
		profileIDs,
		ifaceName,
		PolicyInboundPfx,
		PolicyOutboundPfx,
		ProfileInboundPfx,
		ProfileOutboundPfx,
		WorkloadToEndpointPfx,
		WorkloadFromEndpointPfx,
		"", // No fail-safe chains for workloads.
		"", // No fail-safe chains for workloads.
		chainTypeTracked,
		adminUp,
	)
}

func (r *DefaultRuleRenderer) HostEndpointToFilterChains(
	ifaceName string,
	tiers []*proto.TierInfo,
	profileIDs []string,
) []*Chain {
	log.WithField("ifaceName", ifaceName).Debug("Rendering filter host endpoint chain.")
	return r.endpointToIptablesChains(
		tiers,
		profileIDs,
		ifaceName,
		PolicyOutboundPfx,
		PolicyInboundPfx,
		ProfileOutboundPfx,
		ProfileInboundPfx,
		HostToEndpointPfx,
		HostFromEndpointPfx,
		ChainFailsafeOut,
		ChainFailsafeIn,
		chainTypeTracked,
		true, // Host endpoints are always admin up.
	)
}

func (r *DefaultRuleRenderer) HostEndpointToRawChains(
	ifaceName string,
	untrackedTiers []*proto.TierInfo,
) []*Chain {
	log.WithField("ifaceName", ifaceName).Debug("Rendering raw (untracked) host endpoint chain.")
	log.Debug("Rendering raw (untracked) host endpoint chain. - untrackedTiers %+v", untrackedTiers)
	return r.endpointToIptablesChains(
		untrackedTiers,
		nil, // We don't render profiles into the raw chain.
		ifaceName,
		PolicyOutboundPfx,
		PolicyInboundPfx,
		ProfileOutboundPfx,
		ProfileInboundPfx,
		HostToEndpointPfx,
		HostFromEndpointPfx,
		ChainFailsafeOut,
		ChainFailsafeIn,
		chainTypeUntracked, // Render "untracked" version of chain for the raw table.
		true,               // Host endpoints are always admin up.
	)
}

type endpointChainType int

const (
	chainTypeTracked endpointChainType = iota
	chainTypeUntracked
)

func (r *DefaultRuleRenderer) endpointToIptablesChains(
	tiers []*proto.TierInfo,
	profileIds []string,
	name string,
	toPolicyPrefix PolicyChainNamePrefix,
	fromPolicyPrefix PolicyChainNamePrefix,
	toProfilePrefix ProfileChainNamePrefix,
	fromProfilePrefix ProfileChainNamePrefix,
	toEndpointPrefix string,
	fromEndpointPrefix string,
	toFailsafeChain string,
	fromFailsafeChain string,
	chainType endpointChainType,
	adminUp bool,
) []*Chain {
	toRules := []Rule{}
	fromRules := []Rule{}
	toChainName := EndpointChainName(toEndpointPrefix, name)
	fromChainName := EndpointChainName(fromEndpointPrefix, name)

	if !adminUp {
		// Endpoint is admin-down, drop all traffic to/from it.
		toRules = append(toRules, r.DropRules(Match(), "Endpoint admin disabled")...)
		fromRules = append(fromRules, r.DropRules(Match(), "Endpoint admin disabled")...)
		toEndpointChain := Chain{
			Name:  toChainName,
			Rules: toRules,
		}
		fromEndpointChain := Chain{
			Name:  fromChainName,
			Rules: fromRules,
		}
		return []*Chain{&toEndpointChain, &fromEndpointChain}
	}

	// First set up failsafes.
	if toFailsafeChain != "" {
		toRules = append(toRules, Rule{
			Action: JumpAction{Target: toFailsafeChain},
		})
	}
	if fromFailsafeChain != "" {
		fromRules = append(fromRules, Rule{
			Action: JumpAction{Target: fromFailsafeChain},
		})
	}

	// Start by ensuring that the accept and drop mark bits are clear, policies set those bits to indicate
	// that they accepted / dropped the packet.
	toRules = append(toRules, Rule{
		Action: ClearMarkAction{
			Mark: r.IptablesMarkAccept + r.IptablesMarkDrop,
		},
	})
	fromRules = append(fromRules, Rule{
		Action: ClearMarkAction{
			Mark: r.IptablesMarkAccept + r.IptablesMarkDrop,
		},
	})

	for _, tier := range tiers {
		// Clear the "pass" mark.  If a policy sets that mark, we'll skip the rest of the policies
		// continue processing the profiles, if there are any.
		toRules = append(toRules, Rule{
			Comment: "Start of tier " + tier.Name,
			Action: ClearMarkAction{
				Mark: r.IptablesMarkPass,
			},
		})
		fromRules = append(fromRules, Rule{
			Comment: "Start of tier " + tier.Name,
			Action: ClearMarkAction{
				Mark: r.IptablesMarkPass,
			},
		})

		// Then, jump to each policy in turn.
		for _, polID := range tier.Policies {
			toPolChainName := PolicyChainName(
				toPolicyPrefix,
				&proto.PolicyID{Tier: tier.Name, Name: polID},
			)
			// If a previous policy didn't set the "pass" mark, jump to the policy.
			toRules = append(toRules, Rule{
				Match:  Match().MarkClear(r.IptablesMarkPass),
				Action: JumpAction{Target: toPolChainName},
			})
			// If policy marked packet as accepted, it returns, setting the accept
			// mark bit.
			if chainType == chainTypeUntracked {
				// For an untracked policy, map allow to "NOTRACK and ALLOW".
				toRules = append(toRules, Rule{
					Match:  Match().MarkSet(r.IptablesMarkAccept),
					Action: NoTrackAction{},
				})
			}
			// If accept bit is set, return from this chain.  We don't immediately
			// accept because there may be other policy still to apply.
			toRules = append(toRules, Rule{
				Match:   Match().MarkSet(r.IptablesMarkAccept),
				Action:  ReturnAction{},
				Comment: "Return if policy accepted",
			})

			fromPolChainName := PolicyChainName(
				fromPolicyPrefix,
				&proto.PolicyID{Tier: tier.Name, Name: polID},
			)
			// If a previous policy didn't set the "pass" mark, jump to the policy.
			fromRules = append(fromRules, Rule{
				Match:  Match().MarkClear(r.IptablesMarkPass),
				Action: JumpAction{Target: fromPolChainName},
			})
			// If policy marked packet as accepted, it returns, setting the accept
			// mark bit.
			if chainType == chainTypeUntracked {
				// For an untracked policy, map allow to "NOTRACK and ALLOW".
				fromRules = append(fromRules, Rule{
					Match:  Match().MarkSet(r.IptablesMarkAccept),
					Action: NoTrackAction{},
				})
			}
			// If accept bit is set, return from this chain.  We don't immediately
			// accept because there may be other policy still to apply.
			fromRules = append(fromRules, Rule{
				Match:   Match().MarkSet(r.IptablesMarkAccept),
				Action:  ReturnAction{},
				Comment: "Return if policy accepted",
			})
		}

		if chainType == chainTypeTracked {
			// When rendering normal rules, if no policy marked the packet as "pass", drop the
			// packet.
			//
			// For untracked rules, we don't do that because there may be tracked rules
			// still to be applied to the packet in the filter table.

			toRules = append(toRules, Rule{
				Match: Match().MarkClear(r.IptablesMarkPass),
				Action: NflogAction{
					Group:  1,
					Prefix: "D/0/" + "no-policy-match-inbound/" + tier.Name,
				},
			})
			fromRules = append(fromRules, Rule{
				Match: Match().MarkClear(r.IptablesMarkPass),
				Action: NflogAction{
					Group:  2,
					Prefix: "D/0/" + "no-policy-match-outbound/" + tier.Name,
				},
			})

			toRules = append(toRules, r.DropRules(
				Match().MarkClear(r.IptablesMarkPass),
				"Drop if no policies passed packet")...)
			fromRules = append(fromRules, r.DropRules(
				Match().MarkClear(r.IptablesMarkPass),
				"Drop if no policies passed packet")...)
		}
	}

	if chainType == chainTypeTracked {
		// Then, jump to each profile in turn.
		for _, profileID := range profileIds {
			toProfChainName := ProfileChainName(toProfilePrefix, &proto.ProfileID{Name: profileID})
			fromProfChainName := ProfileChainName(fromProfilePrefix, &proto.ProfileID{Name: profileID})
			toRules = append(toRules,
				Rule{Action: JumpAction{Target: toProfChainName}},
				// If policy marked packet as accepted, it returns, setting the
				// accept mark bit.  If that is set, return from this chain.
				Rule{
					Match:   Match().MarkSet(r.IptablesMarkAccept),
					Action:  ReturnAction{},
					Comment: "Return if profile accepted",
				})
			fromRules = append(fromRules,
				Rule{Action: JumpAction{Target: fromProfChainName}},
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
		toRules = append(toRules, Rule{
			Match: Match(),
			Action: NflogAction{
				Group:  1,
				Prefix: "D/0/no-profile-match-inbound",
			},
		})
		fromRules = append(fromRules, Rule{
			Match: Match(),
			Action: NflogAction{
				Group:  2,
				Prefix: "D/0/no-profile-match-outbound",
			},
		})
		toRules = append(toRules, r.DropRules(Match(), "Drop if no profiles matched")...)
		fromRules = append(fromRules, r.DropRules(Match(), "Drop if no profiles matched")...)
	}

	toEndpointChain := Chain{
		Name:  toChainName,
		Rules: toRules,
	}
	fromEndpointChain := Chain{
		Name:  fromChainName,
		Rules: fromRules,
	}
	return []*Chain{&toEndpointChain, &fromEndpointChain}
}

func EndpointChainName(prefix string, ifaceName string) string {
	return hashutils.GetLengthLimitedID(
		prefix,
		ifaceName,
		MaxChainNameLength,
	)
}
