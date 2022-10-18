// Copyright (c) 2016-2022 Tigera, Inc. All rights reserved.
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
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"

	"github.com/projectcalico/calico/felix/hashutils"
	. "github.com/projectcalico/calico/felix/iptables"
	"github.com/projectcalico/calico/felix/proto"
)

const (
	ingressPolicy         = "ingress"
	egressPolicy          = "egress"
	dropEncap             = true
	dontDropEncap         = false
	NotAnEgressGateway    = false
	IsAnEgressGateway     = true
	alwaysAllowVXLANEncap = true
	alwaysAllowIPIPEncap  = true
	UndefinedIPVersion    = 0
)

func (r *DefaultRuleRenderer) WorkloadEndpointToIptablesChains(
	ifaceName string,
	epMarkMapper EndpointMarkMapper,
	adminUp bool,
	tiers []*proto.TierInfo,
	profileIDs []string,
	isEgressGateway bool,
	ipVersion uint8,
) []*Chain {
	allowVXLANEncapFromWorkloads := r.Config.AllowVXLANPacketsFromWorkloads
	allowIPIPEncapFromWorkloads := r.Config.AllowIPIPPacketsFromWorkloads
	result := []*Chain{}
	result = append(result,
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
			NFLOGInboundGroup,
			RuleDirIngress,
			ingressPolicy,
			r.filterAllowAction, // Workload endpoint chains are only used in the filter table
			alwaysAllowVXLANEncap,
			alwaysAllowIPIPEncap,
			isEgressGateway,
			ipVersion,
		),
		// Chain for traffic _from_ the endpoint.
		// Encap traffic is blocked by default from workload endpoints
		// unless explicitly overridden.
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
			NFLOGOutboundGroup,
			RuleDirEgress,
			egressPolicy,
			r.filterAllowAction, // Workload endpoint chains are only used in the filter table
			allowVXLANEncapFromWorkloads,
			allowIPIPEncapFromWorkloads,
			isEgressGateway,
			ipVersion,
		),
	)

	if r.KubeIPVSSupportEnabled {
		// Chain for setting endpoint mark of an endpoint.
		result = append(result,
			r.endpointSetMarkChain(
				ifaceName,
				epMarkMapper,
				SetEndPointMarkPfx,
			),
		)
	}

	return result
}

func (r *DefaultRuleRenderer) HostEndpointToFilterChains(
	ifaceName string,
	tiers []*proto.TierInfo,
	forwardTiers []*proto.TierInfo,
	epMarkMapper EndpointMarkMapper,
	profileIDs []string,
) []*Chain {
	log.WithField("ifaceName", ifaceName).Debug("Rendering filter host endpoint chain.")
	result := []*Chain{}
	result = append(result,
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
			NFLOGOutboundGroup,
			RuleDirEgress,
			egressPolicy,
			r.filterAllowAction,
			alwaysAllowVXLANEncap,
			alwaysAllowIPIPEncap,
			NotAnEgressGateway,
			UndefinedIPVersion,
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
			NFLOGInboundGroup,
			RuleDirIngress,
			ingressPolicy,
			r.filterAllowAction,
			alwaysAllowVXLANEncap,
			alwaysAllowIPIPEncap,
			NotAnEgressGateway,
			UndefinedIPVersion,
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
			NFLOGOutboundGroup,
			RuleDirEgress,
			egressPolicy,
			r.filterAllowAction,
			alwaysAllowVXLANEncap,
			alwaysAllowIPIPEncap,
			NotAnEgressGateway,
			UndefinedIPVersion,
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
			NFLOGInboundGroup,
			RuleDirIngress,
			ingressPolicy,
			r.filterAllowAction,
			alwaysAllowVXLANEncap,
			alwaysAllowIPIPEncap,
			NotAnEgressGateway,
			UndefinedIPVersion,
		),
	)

	if r.KubeIPVSSupportEnabled {
		// Chain for setting endpoint mark of an endpoint.
		result = append(result,
			r.endpointSetMarkChain(
				ifaceName,
				epMarkMapper,
				SetEndPointMarkPfx,
			),
		)
	}

	return result
}

func (r *DefaultRuleRenderer) HostEndpointToMangleEgressChains(
	ifaceName string,
	tiers []*proto.TierInfo,
	profileIDs []string,
) []*Chain {
	log.WithField("ifaceName", ifaceName).Debug("Render host endpoint mangle egress chain.")
	return []*Chain{
		// Chain for output traffic _to_ the endpoint.  Note, we use RETURN here rather than
		// ACCEPT because the mangle table is typically used, if at all, for packet
		// manipulations that might need to apply to our allowed traffic.
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
			NFLOGOutboundGroup,
			RuleDirEgress,
			egressPolicy,
			ReturnAction{},
			alwaysAllowVXLANEncap,
			alwaysAllowIPIPEncap,
			NotAnEgressGateway,
			UndefinedIPVersion,
		),
	}
}

func (r *DefaultRuleRenderer) HostEndpointToRawEgressChain(
	ifaceName string,
	untrackedTiers []*proto.TierInfo,
) *Chain {
	log.WithField("ifaceName", ifaceName).Debug("Rendering raw (untracked) host endpoint egress chain.")
	return r.endpointIptablesChain(
		untrackedTiers,
		nil, // We don't render profiles into the raw table.
		ifaceName,
		PolicyOutboundPfx,
		ProfileOutboundPfx,
		HostToEndpointPfx,
		ChainFailsafeOut,
		chainTypeUntracked,
		true, // Host endpoints are always admin up.
		NFLOGOutboundGroup,
		RuleDirEgress,
		egressPolicy,
		AcceptAction{},
		alwaysAllowVXLANEncap,
		alwaysAllowIPIPEncap,
		NotAnEgressGateway,
		UndefinedIPVersion,
	)
}

func (r *DefaultRuleRenderer) HostEndpointToRawChains(
	ifaceName string,
	untrackedTiers []*proto.TierInfo,
) []*Chain {
	log.WithField("ifaceName", ifaceName).Debugf("Rendering raw (untracked) host endpoint chain. - untrackedTiers %+v", untrackedTiers)
	return []*Chain{
		// Chain for traffic _to_ the endpoint.
		r.HostEndpointToRawEgressChain(ifaceName, untrackedTiers),
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
			NFLOGInboundGroup,
			RuleDirIngress,
			ingressPolicy,
			AcceptAction{},
			alwaysAllowVXLANEncap,
			alwaysAllowIPIPEncap,
			NotAnEgressGateway,
			UndefinedIPVersion,
		),
	}
}

func (r *DefaultRuleRenderer) HostEndpointToMangleIngressChains(
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
			NFLOGInboundGroup,
			RuleDirIngress,
			ingressPolicy,
			r.mangleAllowAction,
			alwaysAllowVXLANEncap,
			alwaysAllowIPIPEncap,
			NotAnEgressGateway,
			UndefinedIPVersion,
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

func (r *DefaultRuleRenderer) endpointSetMarkChain(
	name string,
	epMarkMapper EndpointMarkMapper,
	endpointPrefix string,
) *Chain {
	rules := []Rule{}
	chainName := EndpointChainName(endpointPrefix, name)

	if endPointMark, err := epMarkMapper.GetEndpointMark(name); err == nil {
		// Set endpoint mark.
		rules = append(rules, Rule{
			Action: SetMaskedMarkAction{
				Mark: endPointMark,
				Mask: epMarkMapper.GetMask()},
		})
	}
	return &Chain{
		Name:  chainName,
		Rules: rules,
	}
}

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
	dir RuleDir,
	policyType string,
	allowAction Action,
	allowVXLANEncap bool,
	allowIPIPEncap bool,
	isEgressGateway bool,
	ipVersion uint8,
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
		rules = r.appendConntrackRules(
			rules,
			allowAction,
			// Allow CtState INVALID for traffic _from_ an egress gateway.
			// This is because the return path from an egress gateway is different from the
			// VXLAN-tunnelled forwards path.

			// We also need to allow CtState INVALID for traffic _to_ an egress gateway,
			// when using IP-IP and VXLAN. However, we do not understand why we need it yet.
			isEgressGateway,
		)
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

	// Accept the UDP VXLAN traffic for egress gateways
	endOfChainDropComment := "Drop if no profiles matched"
	if !r.BPFEnabled && ipVersion == 4 && isEgressGateway {
		programEgwRule := func(ipset string) {
			baseMatch := func(proto uint8) MatchCriteria {
				if dir == RuleDirIngress {
					return Match().ProtocolNum(proto).SourceIPSet(r.IPSetConfigV4.
						NameForMainIPSet(ipset))
				} else {
					return Match().ProtocolNum(proto).DestIPSet(r.IPSetConfigV4.
						NameForMainIPSet(ipset))
				}
			}

			rules = append(rules,
				Rule{
					Match: baseMatch(ProtoUDP).
						DestPorts(
							uint16(r.Config.EgressIPVXLANPort), // egress.calico
						),
					Action:  AcceptAction{},
					Comment: []string{"Accept VXLAN UDP traffic for egress gateways"},
				},
			)
			if dir == RuleDirIngress {
				rules = append(rules,
					Rule{
						Match: baseMatch(ProtoTCP).
							DestPorts(8080), // FIXME make configurable?
						Action:  AcceptAction{},
						Comment: []string{"Accept readiness probes for egress gateways"},
					},
				)
			}
		}
		// Auto-allow VXLAN UDP traffic for egress gateways from/to host IPs
		programEgwRule(IPSetIDAllHostNets)
		// Auto-allow VXLAN UDP traffic for egress gateways from/to tunnel IPs in case of overlay
		if r.VXLANEnabled || r.IPIPEnabled || r.WireguardEnabled {
			programEgwRule(IPSetIDAllTunnelNets)
		}

	}

	if !r.BPFEnabled && isEgressGateway {
		if dir == RuleDirIngress {
			// Block any other traffic _to_ egress gateways; zero out the policy and profiles so that we'll
			// just render an end-of-chain drop.
			tiers = nil
			profileIds = nil
			endOfChainDropComment = "Drop all other ingress traffic to egress gateway."
		}
	}

	if !allowVXLANEncap {
		// VXLAN encapped packets that originated in a pod should be dropped, as the encapsulation can be used to
		// bypass restrictive egress policies.
		rules = append(rules, Rule{
			Match: Match().ProtocolNum(ProtoUDP).
				DestPorts(uint16(r.Config.VXLANPort)),
			Action:  DropAction{},
			Comment: []string{"Drop VXLAN encapped packets originating in workloads"},
		})
	}
	if !allowIPIPEncap {
		// IPinIP encapped packets that originated in a pod should be dropped, as the encapsulation can be used to
		// bypass restrictive egress policies.
		rules = append(rules, Rule{
			Match:   Match().ProtocolNum(ProtoIPIP),
			Action:  DropAction{},
			Comment: []string{"Drop IPinIP encapped packets originating in workloads"},
		})
	}

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
				Comment: []string{"Start of tier " + tier.Name},
				Action: ClearMarkAction{
					Mark: r.IptablesMarkPass,
				},
			})

			// Track if any of the policies are not staged. If all of the policies in a tier are staged
			// then the default end of tier behavior should be pass rather than drop.
			endOfTierDrop := false

			// Then, jump to each policy in turn.
			for _, polID := range policies {
				isStaged := model.PolicyIsStaged(polID)

				// If this is not a staged policy then end of tier behavior should be drop.
				if !isStaged {
					endOfTierDrop = true
				}

				polChainName := PolicyChainName(
					policyPrefix,
					&proto.PolicyID{Tier: tier.Name, Name: polID},
				)

				// If a previous policy didn't set the "pass" mark, jump to the policy.
				rules = append(rules, Rule{
					Match:  Match().MarkClear(r.IptablesMarkPass),
					Action: JumpAction{Target: polChainName},
				})

				// Only handle actions for non-staged policies.
				if !isStaged {
					// If policy marked packet as accepted, it returns, setting the accept
					// mark bit.
					if chainType == chainTypeUntracked {
						// For an untracked policy, map allow to "NOTRACK and ALLOW".
						rules = append(rules, Rule{
							Match:  Match().MarkSingleBitSet(r.IptablesMarkAccept),
							Action: NoTrackAction{},
						})
					}
					// If accept bit is set, return from this chain.  We don't immediately
					// accept because there may be other policy still to apply.
					rules = append(rules, Rule{
						Match:   Match().MarkSingleBitSet(r.IptablesMarkAccept),
						Action:  ReturnAction{},
						Comment: []string{"Return if policy accepted"},
					})
				}
			}

			if chainType == chainTypeNormal || chainType == chainTypeForward {
				if endOfTierDrop {
					nfqueueRule := r.NfqueueRuleDelayDeniedPacket(
						Match().MarkClear(r.IptablesMarkPass),
						"Drop if no policies passed packet",
					)
					if nfqueueRule != nil {
						rules = append(rules, *nfqueueRule)
					}

					// When rendering normal and forward rules, if no policy marked the packet as "pass", drop the
					// packet.
					//
					// For untracked and pre-DNAT rules, we don't do that because there may be
					// normal rules still to be applied to the packet in the filter table.
					rules = append(rules, Rule{
						Match: Match().MarkClear(r.IptablesMarkPass),
						Action: NflogAction{
							Group:  nflogGroup,
							Prefix: CalculateEndOfTierDropNFLOGPrefixStr(dir, tier.Name),
						},
					})

					rules = append(rules, r.DropRules(Match().MarkClear(r.IptablesMarkPass), "Drop if no policies passed packet")...)
				} else {
					// If we do not require an end of tier drop (i.e. because all of the policies in the tier are
					// staged), then add an end of tier pass nflog action so that we can at least track that we
					// would hit end of tier drop. This simplifies the processing in the collector.
					rules = append(rules, Rule{
						Match: Match().MarkClear(r.IptablesMarkPass),
						Action: NflogAction{
							Group:  nflogGroup,
							Prefix: CalculateEndOfTierPassNFLOGPrefixStr(dir, tier.Name),
						},
					})
				}
			}
		}
	}

	if len(tiers) == 0 && chainType == chainTypeForward {
		// Forwarded traffic is allowed when there are no policies with
		// applyOnForward that apply to this endpoint (and in this direction).
		rules = append(rules, Rule{
			Action:  SetMarkAction{Mark: r.IptablesMarkAccept},
			Comment: []string{"Allow forwarded traffic by default"},
		})
		rules = append(rules, Rule{
			Action:  ReturnAction{},
			Comment: []string{"Return for accepted forward traffic"},
		})
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
					Match:   Match().MarkSingleBitSet(r.IptablesMarkAccept),
					Action:  ReturnAction{},
					Comment: []string{"Return if profile accepted"},
				})
		}

		if !isEgressGateway { // We don't support DNS policy on EGWs so no point in queueing.
			nfqueueRule := r.NfqueueRuleDelayDeniedPacket(Match(), endOfChainDropComment)
			if nfqueueRule != nil {
				rules = append(rules, *nfqueueRule)
			}
		}

		// When rendering normal rules, if no profile marked the packet as accepted, drop
		// the packet.
		//
		// For untracked rules, we don't do that because there may be tracked rules
		// still to be applied to the packet in the filter table.
		// TODO (Matt): This (and the policy equivalent just above) can probably be refactored.
		//              At least the magic 1 and 2 need to be combined with the equivalent in CalculateActions.
		// No profile matched the packet: drop it.
		// if dropIfNoProfilesMatched {
		rules = append(rules, Rule{
			Match: Match(),
			Action: NflogAction{
				Group:  nflogGroup,
				Prefix: CalculateNoMatchProfileNFLOGPrefixStr(dir),
			},
		})

		rules = append(rules, r.DropRules(Match(), endOfChainDropComment)...)
		// }
	}

	return &Chain{
		Name:  chainName,
		Rules: rules,
	}
}

func (r *DefaultRuleRenderer) appendConntrackRules(rules []Rule, allowAction Action, allowInvalid bool) []Rule {
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
	if !(r.Config.DisableConntrackInvalid || allowInvalid) {
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
