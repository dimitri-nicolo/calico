// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

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

package polprog

import (
	"fmt"
	"math"
	"math/bits"
	"strings"

	"github.com/projectcalico/felix/bpf/ipsets"

	"github.com/projectcalico/felix/bpf"

	log "github.com/sirupsen/logrus"

	. "github.com/projectcalico/felix/bpf/asm"
	"github.com/projectcalico/felix/bpf/state"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/rules"
)

type Builder struct {
	b               *Block
	tierID          int
	policyID        int
	ruleID          int
	rulePartID      int
	ipSetIDProvider ipSetIDProvider

	ipSetMapFD bpf.MapFD
	stateMapFD bpf.MapFD
	jumpMapFD  bpf.MapFD
}

type ipSetIDProvider interface {
	GetNoAlloc(ipSetID string) uint64
}

func NewBuilder(ipSetIDProvider ipSetIDProvider, ipsetMapFD, stateMapFD, jumpMapFD bpf.MapFD) *Builder {
	b := &Builder{
		ipSetIDProvider: ipSetIDProvider,
		ipSetMapFD:      ipsetMapFD,
		stateMapFD:      stateMapFD,
		jumpMapFD:       jumpMapFD,
	}
	return b
}

var offset int = 0

func nextOffset(size int, align int) int16 {
	offset -= size
	remainder := offset % align
	if remainder != 0 {
		// For negative numbers, the remainder is negative (e.g. -9 % 8 == -1)
		offset = offset - remainder - align
	}
	return int16(offset)
}

const (
	stateEventHdrSize int16 = 8
)

var (
	// Stack offsets.  These are defined locally.
	offStateKey    = nextOffset(4, 4)
	offSrcIPSetKey = nextOffset(ipsets.IPSetEntrySize, 8)
	offDstIPSetKey = nextOffset(ipsets.IPSetEntrySize, 8)

	// Offsets within the cal_tc_state struct.
	// WARNING: must be kept in sync with the definitions in bpf/include/jump.h.
	stateOffIPSrc          int16 = stateEventHdrSize + 0
	stateOffIPDst          int16 = stateEventHdrSize + 4
	_                            = stateOffIPDst
	stateOffPreNATIPDst    int16 = stateEventHdrSize + 8
	_                            = stateOffPreNATIPDst
	stateOffPostNATIPDst   int16 = stateEventHdrSize + 12
	stateOffPolResult      int16 = stateEventHdrSize + 20
	stateOffSrcPort        int16 = stateEventHdrSize + 24
	stateOffDstPort        int16 = stateEventHdrSize + 26
	stateOffICMPType             = stateOffDstPort
	stateOffPreNATDstPort  int16 = stateEventHdrSize + 28
	_                            = stateOffPreNATDstPort
	stateOffPostNATDstPort int16 = stateEventHdrSize + 30
	stateOffIPProto        int16 = stateEventHdrSize + 32
	stateOffFlags          int16 = stateEventHdrSize + 33

	stateOffRulesHit int16 = stateEventHdrSize + 36
	stateOffRuleIDs  int16 = stateEventHdrSize + 40

	// Compile-time check that IPSetEntrySize hasn't changed; if it changes, the code will need to change.
	_ = [1]struct{}{{}}[20-ipsets.IPSetEntrySize]

	// Offsets within struct ip4_set_key.
	// WARNING: must be kept in sync with the definitions in bpf/ipsets/map.go.
	// WARNING: must be kept in sync with the definitions in bpf/include/policy.h.
	ipsKeyPrefix int16 = 0
	ipsKeyID     int16 = 4
	ipsKeyAddr   int16 = 12
	ipsKeyPort   int16 = 16
	ipsKeyProto  int16 = 18
	ipsKeyPad    int16 = 19

	// Bits in the state flags field.
	FlagDestIsHost uint8 = 1 << 2
	FlagSrcIsHost  uint8 = 1 << 3
)

type Rule struct {
	*proto.Rule
	MatchID RuleMatchID
}

type Policy struct {
	Name      string
	Rules     []Rule
	NoMatchID RuleMatchID
	Staged    bool
}

type Tier struct {
	Name      string
	EndRuleID RuleMatchID
	EndAction TierEndAction
	Policies  []Policy
}

type Rules struct {
	// Both workload and host interfaces can enforce host endpoint policy (carried here in the
	// Host... fields); in the case of a workload interface, that can only come from the
	// wildcard host endpoint, aka "host-*".
	//
	// However, only a workload interface can have any workload policy (carried here in the
	// Tiers and Profiles fields), and workload interfaces also Deny by default when there is no
	// workload policy at all.  ForHostInterface (with reversed polarity) is the boolean that
	// tells us whether or not to implement workload policy and that default Deny.
	ForHostInterface bool

	// Indicates to suppress normal host policy because it's trumped by the setting of
	// DefaultEndpointToHostAction.
	SuppressNormalHostPolicy bool

	// Workload policy.
	Tiers            []Tier
	Profiles         []Profile
	NoProfileMatchID RuleMatchID

	// Host endpoint policy.
	HostPreDnatTiers []Tier
	HostForwardTiers []Tier
	HostNormalTiers  []Tier
	HostProfiles     []Profile
}

type RuleMatchID = uint64

type Profile = Policy

type TierEndAction string

const (
	TierEndUndef TierEndAction = ""
	TierEndDeny                = "deny"
	TierEndPass                = "pass"
)

func (p *Builder) Instructions(rules Rules) (Insns, error) {
	p.b = NewBlock()
	p.writeProgramHeader()

	// Pre-DNAT policy: on a host interface, or host-* policy on a workload interface.  Traffic
	// is allowed to continue if there is no applicable pre-DNAT policy.
	p.writeTiers(rules.HostPreDnatTiers, legDestPreNAT, "allowed_by_host_policy")

	// If traffic is to or from the local host, skip over any apply-on-forward policy.  Note
	// that this case can be:
	// - on a workload interface, workload <--> own host
	// - on a host interface, this host (not a workload) <--> anywhere outside this host
	//
	// When rules.SuppressNormalHostPolicy is true, we also skip normal host policy; this is
	// the case when we're building the policy program for workload -> host and
	// DefaultEndpointToHostAction is ACCEPT or DROP; or for host -> workload.
	if rules.SuppressNormalHostPolicy {
		p.writeJumpIfToOrFromHost("allowed_by_host_policy")
	} else {
		p.writeJumpIfToOrFromHost("to_or_from_host")
	}

	// At this point we know we have traffic that is being forwarded through the host's root
	// network namespace.  Note that this case can be:
	// - workload interface, workload <--> another local workload
	// - workload interface, workload <--> anywhere outside this host
	// - host interface, workload <--> anywhere outside this host
	// - host interface, anywhere outside this host <--> anywhere outside this host

	// Apply-On-Forward policy: on a host interface, or host-* policy on a workload interface.
	// Traffic is allowed to continue if there is no applicable AoF policy.
	p.writeTiers(rules.HostForwardTiers, legDest, "allowed_by_host_policy")

	// Now skip over normal host policy and jump to where we apply possible workload policy.
	p.b.Jump("allowed_by_host_policy")

	if !rules.SuppressNormalHostPolicy {
		// "Normal" host policy, i.e. for non-forwarded traffic.
		p.b.LabelNextInsn("to_or_from_host")
		p.writeTiers(rules.HostNormalTiers, legDest, "allowed_by_host_policy")
		p.writeProfiles(rules.HostProfiles, rules.NoProfileMatchID, "allowed_by_host_policy")
	}

	// End of host policy.
	p.b.LabelNextInsn("allowed_by_host_policy")

	if rules.ForHostInterface {
		// On a host interface there is no workload policy, so we are now done.
		p.b.Jump("allow")
	} else {
		// Workload policy.
		p.writeTiers(rules.Tiers, legDest, "allow")
		p.writeProfiles(rules.Profiles, rules.NoProfileMatchID, "allow")
	}

	p.writeProgramFooter()
	return p.b.Assemble()
}

// writeProgramHeader emits instructions to load the state from the state map, leaving
// R6 = program context
// R9 = pointer to state map
func (p *Builder) writeProgramHeader() {
	// Pre-amble to the policy program.
	p.b.LabelNextInsn("start")
	p.b.Mov64(R6, R1) // Save R1 (context) in R6.
	// Zero-out the map key
	p.b.MovImm64(R1, 0) // R1 = 0
	p.b.StoreStack32(R1, offStateKey)
	// Get pointer to map key in R2.
	p.b.Mov64(R2, R10) // R2 = R10
	p.b.AddImm64(R2, int32(offStateKey))
	// Load map file descriptor into R1.
	// clang uses a 64-bit load so copy that for now.
	p.b.LoadMapFD(R1, uint32(p.stateMapFD)) // R1 = 0 (64-bit immediate)
	p.b.Call(HelperMapLookupElem)           // Call helper
	// Check return value for NULL.
	p.b.JumpEqImm64(R0, 0, "exit")
	// Save state pointer in R9.
	p.b.Mov64(R9, R0)
	p.b.LabelNextInsn("policy")
}

const (
	jumpIdxPolicy = iota
	jumpIdxEpilogue
	jumpIdxICMP
	jumpIdxDrop

	_ = jumpIdxPolicy
	_ = jumpIdxICMP
	_ = jumpIdxDrop
)

func (p *Builder) writeJumpIfToOrFromHost(label string) {
	// Load state flags.
	p.b.Load8(R1, R9, stateOffFlags)

	// Mask against host bits.
	p.b.AndImm32(R1, int32(FlagDestIsHost|FlagSrcIsHost))

	// If non-zero, jump to specified label.
	p.b.JumpNEImm64(R1, 0, label)
}

// writeProgramFooter emits the program exit jump targets.
func (p *Builder) writeProgramFooter() {
	// Fall through here if there's no match.  Also used when we hit an error or if policy rejects packet.
	p.b.LabelNextInsn("deny")

	// Store the policy result in the state for the next program to see.
	p.b.MovImm32(R1, int32(state.PolicyDeny))
	p.b.Store32(R9, R1, stateOffPolResult)

	// Execute the tail call.
	p.b.Mov64(R1, R6)                      // First arg is the context.
	p.b.LoadMapFD(R2, uint32(p.jumpMapFD)) // Second arg is the map.
	p.b.MovImm32(R3, jumpIdxDrop)          // Third arg is the index (rather than a pointer to the index).
	p.b.Call(HelperTailCall)

	// Fall through if tail call fails.
	p.b.LabelNextInsn("exit")
	p.b.MovImm64(R0, 2 /* TC_ACT_SHOT */)

	p.b.Exit()

	if p.b.TargetIsUsed("allow") {
		p.b.LabelNextInsn("allow")
		// Store the policy result in the state for the next program to see.
		p.b.MovImm32(R1, int32(state.PolicyAllow))
		p.b.Store32(R9, R1, stateOffPolResult)
		// Execute the tail call.
		p.b.Mov64(R1, R6)                      // First arg is the context.
		p.b.LoadMapFD(R2, uint32(p.jumpMapFD)) // Second arg is the map.
		p.b.MovImm32(R3, jumpIdxEpilogue)      // Third arg is the index (rather than a pointer to the index).
		p.b.Call(HelperTailCall)

		// Fall through if tail call fails.
		p.b.MovImm32(R1, state.PolicyTailCallFailed)
		p.b.Store32(R9, R1, stateOffPolResult)
		p.b.MovImm64(R0, 2 /* TC_ACT_SHOT */)
		p.b.Exit()
	}
}

func (p *Builder) writeRecordRuleID(id RuleMatchID, skipLabel string) {
	// Load the hit count
	p.b.Load8(R1, R9, stateOffRulesHit)

	// Make sure we do not hit too many rules, if so skip to action without
	// recording the rule ID
	p.b.JumpGEImm64(R1, state.MaxRuleIDs, skipLabel)

	// Increment the hit count
	p.b.Mov64(R2, R1)
	p.b.AddImm64(R2, 1)
	// Store the new count
	p.b.Store8(R9, R2, stateOffRulesHit)

	// Store the rule ID in the rule ids array
	p.b.ShiftLImm64(R1, 3) // x8
	p.b.AddImm64(R1, int32(stateOffRuleIDs))
	p.b.LoadImm64(R2, int64(id))
	p.b.Add64(R1, R9)
	p.b.Store64(R1, R2, 0)
}

func (p *Builder) writeRecordRuleHit(r Rule, skipLabel string) {
	log.Debugf("Hit rule ID 0x%x", r.MatchID)
	p.writeRecordRuleID(r.MatchID, skipLabel)
}

func (p *Builder) setUpIPSetKey(ipsetID uint64, keyOffset, ipOffset, portOffset int16) {
	// TODO track whether we've already done an initialisation and skip the parts that don't change.
	// Zero the padding.
	p.b.MovImm64(R1, 0) // R1 = 0
	p.b.StoreStack8(R1, keyOffset+ipsKeyPad)
	p.b.MovImm64(R1, 128) // R1 = 128
	p.b.StoreStack32(R1, keyOffset+ipsKeyPrefix)

	// Store the IP address, port and protocol.
	p.b.Load32(R1, R9, ipOffset)
	p.b.StoreStack32(R1, keyOffset+ipsKeyAddr)
	p.b.Load16(R1, R9, portOffset)
	p.b.StoreStack16(R1, keyOffset+ipsKeyPort)
	p.b.Load8(R1, R9, stateOffIPProto)
	p.b.StoreStack8(R1, keyOffset+ipsKeyProto)

	// Store the IP set ID.  It is 64-bit but, since it's a packed struct, we have to write it in two
	// 32-bit chunks.
	beIPSetID := bits.ReverseBytes64(ipsetID)
	p.b.MovImm32(R1, int32(beIPSetID))
	p.b.StoreStack32(R1, keyOffset+ipsKeyID)
	p.b.MovImm32(R1, int32(beIPSetID>>32))
	p.b.StoreStack32(R1, keyOffset+ipsKeyID+4)
}

func (p *Builder) writeTiers(tiers []Tier, destLeg matchLeg, allowLabel string) {
	actionLabels := map[string]string{
		"allow": allowLabel,
		"deny":  "deny",
	}
	for _, tier := range tiers {
		endOfTierLabel := fmt.Sprint("end_of_tier_", p.tierID)
		actionLabels["pass"] = endOfTierLabel
		actionLabels["next-tier"] = endOfTierLabel

		log.Debugf("Start of tier %d %q", p.tierID, tier.Name)
		for _, pol := range tier.Policies {
			p.writePolicy(pol, actionLabels, destLeg)
		}

		// End of tier rule.
		action := tier.EndAction
		if action == TierEndUndef {
			action = TierEndDeny
		}
		log.Debugf("End of tier %d %q: %s", p.tierID, tier.Name, action)
		p.writeRule(Rule{
			Rule:    &proto.Rule{},
			MatchID: tier.EndRuleID,
		}, actionLabels[string(action)], destLeg)
		p.b.LabelNextInsn(endOfTierLabel)
		p.tierID++
	}
}

func (p *Builder) writeProfiles(profiles []Policy, noProfileMatchID uint64, allowLabel string) {
	log.Debugf("Start of profiles")
	for idx, prof := range profiles {
		p.writeProfile(prof, idx, allowLabel)
	}

	log.Debugf("End of profiles drop")
	p.writeRule(Rule{
		Rule:    &proto.Rule{},
		MatchID: noProfileMatchID,
	}, "deny", legDest)
}

func (p *Builder) writePolicyRules(policy Policy, actionLabels map[string]string, destLeg matchLeg) {
	endOfPolicyLabel := fmt.Sprintf("end_of_policy_%d", p.policyID)

	if policy.Staged {
		// When a pass or allow rule matches in a staged policy then we want to skip
		// the rest of the rules in the staged policy and continue processing the next
		// policy in the same tier.  Note: don't modify the caller's map here!
		actionLabels = map[string]string{
			"pass":      endOfPolicyLabel,
			"next-tier": endOfPolicyLabel,
			"allow":     endOfPolicyLabel,
			"deny":      endOfPolicyLabel,
		}
	}

	for ruleIdx, rule := range policy.Rules {
		log.Debugf("Start of rule %d", ruleIdx)
		action := strings.ToLower(rule.Action)
		p.writeRule(rule, actionLabels[action], destLeg)
		log.Debugf("End of rule %d", ruleIdx)
	}

	if policy.Staged {
		log.Debugf("NoMatch policy ID 0x%x", policy.NoMatchID)
		p.writeRecordRuleID(policy.NoMatchID, endOfPolicyLabel)
		p.b.LabelNextInsn(endOfPolicyLabel)
	}
}

func (p *Builder) writePolicy(policy Policy, actionLabels map[string]string, destLeg matchLeg) {
	log.Debugf("Start of policy %q %d", policy.Name, p.policyID)
	p.writePolicyRules(policy, actionLabels, destLeg)
	log.Debugf("End of policy %q %d", policy.Name, p.policyID)
	p.policyID++
}

func (p *Builder) writeProfile(profile Profile, idx int, allowLabel string) {
	actionLabels := map[string]string{
		"allow":     allowLabel,
		"deny":      "deny",
		"pass":      "deny",
		"next-tier": "deny",
	}
	log.Debugf("Start of profile %q %d", profile.Name, idx)
	p.writePolicyRules(profile, actionLabels, legDest)
	log.Debugf("End of profile %q %d", profile.Name, idx)
	p.policyID++
}

type matchLeg string

const (
	legSource     matchLeg = "source"
	legDest       matchLeg = "dest"
	legDestPreNAT matchLeg = "destPreNAT"
)

func (leg matchLeg) offsetToStateIPAddressField() (offset int16) {
	if leg == legSource {
		offset = stateOffIPSrc
	} else if leg == legDestPreNAT {
		offset = stateOffPreNATIPDst
	} else {
		offset = stateOffPostNATIPDst
	}
	return
}

func (leg matchLeg) offsetToStatePortField() (portOffset int16) {
	if leg == legSource {
		portOffset = stateOffSrcPort
	} else if leg == legDestPreNAT {
		portOffset = stateOffPreNATDstPort
	} else {
		portOffset = stateOffPostNATDstPort
	}
	return
}

func (leg matchLeg) stackOffsetToIPSetKey() (keyOffset int16) {
	if leg == legSource {
		keyOffset = offSrcIPSetKey
	} else {
		keyOffset = offDstIPSetKey
	}
	return
}

func (p *Builder) writeRule(r Rule, actionLabel string, destLeg matchLeg) {
	if actionLabel == "" {
		log.Panic("empty action label")
	}

	rule := rules.FilterRuleToIPVersion(4, r.Rule)
	if rule == nil {
		log.Debugf("Version mismatch, skipping rule")
		return
	}
	p.writeStartOfRule()

	if rule.Protocol != nil {
		log.WithField("proto", rule.Protocol).Debugf("Protocol match")
		p.writeProtoMatch(false, rule.Protocol)
	}
	if rule.NotProtocol != nil {
		log.WithField("proto", rule.NotProtocol).Debugf("NotProtocol match")
		p.writeProtoMatch(true, rule.NotProtocol)
	}

	if len(rule.SrcNet) != 0 {
		log.WithField("cidrs", rule.SrcNet).Debugf("SrcNet match")
		p.writeCIDRSMatch(false, legSource, rule.SrcNet)
	}
	if len(rule.NotSrcNet) != 0 {
		log.WithField("cidrs", rule.NotSrcNet).Debugf("NotSrcNet match")
		p.writeCIDRSMatch(true, legSource, rule.NotSrcNet)
	}

	if len(rule.DstNet) != 0 {
		log.WithField("cidrs", rule.DstNet).Debugf("DstNet match")
		p.writeCIDRSMatch(false, destLeg, rule.DstNet)
	}
	if len(rule.NotDstNet) != 0 {
		log.WithField("cidrs", rule.NotDstNet).Debugf("NotDstNet match")
		p.writeCIDRSMatch(true, destLeg, rule.NotDstNet)
	}

	if len(rule.SrcIpSetIds) > 0 {
		log.WithField("ipSetIDs", rule.SrcIpSetIds).Debugf("SrcIpSetIds match")
		p.writeIPSetMatch(false, legSource, rule.SrcIpSetIds)
	}
	if len(rule.NotSrcIpSetIds) > 0 {
		log.WithField("ipSetIDs", rule.NotSrcIpSetIds).Debugf("NotSrcIpSetIds match")
		p.writeIPSetMatch(true, legSource, rule.NotSrcIpSetIds)
	}

	if len(rule.DstIpSetIds) > 1 {
		log.WithField("rule", rule).Panic("proto.Rule has more than one DstIpSetIds")
	}
	if len(rule.DstDomainIpSetIds) > 1 {
		log.WithField("rule", rule).Panic("proto.Rule has more than one DstDomainIpSetIds")
	}
	dstIPSetIDs := append(rule.DstDomainIpSetIds, rule.DstIpSetIds...)
	if len(dstIPSetIDs) > 0 {
		log.WithField("ipSetIDs", rule.DstDomainIpSetIds).Debugf("DstDomainIpSetIds match")
		log.WithField("ipSetIDs", rule.DstIpSetIds).Debugf("DstIpSetIds match")
		p.writeIPSetOrMatch(destLeg, dstIPSetIDs)
	}
	if len(rule.NotDstIpSetIds) > 0 {
		log.WithField("ipSetIDs", rule.NotDstIpSetIds).Debugf("NotDstIpSetIds match")
		p.writeIPSetMatch(true, destLeg, rule.NotDstIpSetIds)
	}

	if len(rule.SrcPorts) > 0 || len(rule.SrcNamedPortIpSetIds) > 0 {
		log.WithField("ports", rule.SrcPorts).Debugf("SrcPorts match")
		p.writePortsMatch(false, legSource, rule.SrcPorts, rule.SrcNamedPortIpSetIds)
	}
	if len(rule.NotSrcPorts) > 0 || len(rule.NotSrcNamedPortIpSetIds) > 0 {
		log.WithField("ports", rule.NotSrcPorts).Debugf("NotSrcPorts match")
		p.writePortsMatch(true, legSource, rule.NotSrcPorts, rule.NotSrcNamedPortIpSetIds)
	}

	if len(rule.DstPorts) > 0 || len(rule.DstNamedPortIpSetIds) > 0 {
		log.WithField("ports", rule.DstPorts).Debugf("DstPorts match")
		p.writePortsMatch(false, destLeg, rule.DstPorts, rule.DstNamedPortIpSetIds)
	}
	if len(rule.NotDstPorts) > 0 || len(rule.NotDstNamedPortIpSetIds) > 0 {
		log.WithField("ports", rule.NotDstPorts).Debugf("NotDstPorts match")
		p.writePortsMatch(true, destLeg, rule.NotDstPorts, rule.NotDstNamedPortIpSetIds)
	}

	if rule.Icmp != nil {
		log.WithField("icmpv4", rule.Icmp).Debugf("ICMP type/code match")
		switch icmp := rule.Icmp.(type) {
		case *proto.Rule_IcmpTypeCode:
			p.writeICMPTypeCodeMatch(false, uint8(icmp.IcmpTypeCode.Type), uint8(icmp.IcmpTypeCode.Code))
		case *proto.Rule_IcmpType:
			p.writeICMPTypeMatch(false, uint8(icmp.IcmpType))
		}
	}
	if rule.NotIcmp != nil {
		log.WithField("icmpv4", rule.Icmp).Debugf("Not ICMP type/code match")
		switch icmp := rule.NotIcmp.(type) {
		case *proto.Rule_NotIcmpTypeCode:
			p.writeICMPTypeCodeMatch(true, uint8(icmp.NotIcmpTypeCode.Type), uint8(icmp.NotIcmpTypeCode.Code))
		case *proto.Rule_NotIcmpType:
			p.writeICMPTypeMatch(true, uint8(icmp.NotIcmpType))
		}
	}

	p.writeEndOfRule(r, actionLabel)
	p.ruleID++
	p.rulePartID = 0
}

func (p *Builder) writeStartOfRule() {
}

func (p *Builder) writeEndOfRule(rule Rule, actionLabel string) {
	// If all the match criteria are met, we fall through to the end of the rule
	// so all that's left to do is to jump to the relevant action.
	// TODO log and log-and-xxx actions

	p.writeRecordRuleHit(rule, actionLabel)

	p.b.Jump(actionLabel)

	p.b.LabelNextInsn(p.endOfRuleLabel())
}

func (p *Builder) writeProtoMatch(negate bool, protocol *proto.Protocol) {
	p.b.Load8(R1, R9, stateOffIPProto)
	protoNum := protocolToNumber(protocol)
	if negate {
		p.b.JumpEqImm64(R1, int32(protoNum), p.endOfRuleLabel())
	} else {
		p.b.JumpNEImm64(R1, int32(protoNum), p.endOfRuleLabel())
	}
}

func (p *Builder) writeICMPTypeMatch(negate bool, icmpType uint8) {
	p.b.Load8(R1, R9, stateOffICMPType)
	if negate {
		p.b.JumpEqImm64(R1, int32(icmpType), p.endOfRuleLabel())
	} else {
		p.b.JumpNEImm64(R1, int32(icmpType), p.endOfRuleLabel())
	}
}

func (p *Builder) writeICMPTypeCodeMatch(negate bool, icmpType, icmpCode uint8) {
	p.b.Load16(R1, R9, stateOffICMPType)
	if negate {
		p.b.JumpEqImm64(R1, ((int32(icmpCode) << 8) | int32(icmpType)), p.endOfRuleLabel())
	} else {
		p.b.JumpNEImm64(R1, ((int32(icmpCode) << 8) | int32(icmpType)), p.endOfRuleLabel())
	}
}
func (p *Builder) writeCIDRSMatch(negate bool, leg matchLeg, cidrs []string) {
	p.b.Load32(R1, R9, leg.offsetToStateIPAddressField())

	var onMatchLabel string
	if negate {
		// Match negated, if we match any CIDR then we jump to the next rule.
		onMatchLabel = p.endOfRuleLabel()
	} else {
		// Match is non-negated, if we match, got to the next match criteria.
		onMatchLabel = p.freshPerRuleLabel()
	}
	for _, cidrStr := range cidrs {
		cidr := ip.MustParseCIDROrIP(cidrStr)
		addrU32 := bits.ReverseBytes32(cidr.Addr().(ip.V4Addr).AsUint32()) // TODO IPv6
		maskU32 := bits.ReverseBytes32(math.MaxUint32 << (32 - cidr.Prefix()) & math.MaxUint32)

		p.b.MovImm32(R2, int32(maskU32))
		p.b.And32(R2, R1)
		p.b.JumpEqImm32(R2, int32(addrU32), onMatchLabel)
	}
	if !negate {
		// If we fall through then none of the CIDRs matched so the rule doesn't match.
		p.b.Jump(p.endOfRuleLabel())
		// Label the next match so we can skip to it on success.
		p.b.LabelNextInsn(onMatchLabel)
	}
}

func (p *Builder) writeIPSetMatch(negate bool, leg matchLeg, ipSets []string) {
	// IP sets are different to CIDRs, if we have multiple IP sets then they all have to match
	// so we treat them as independent match criteria.
	for _, ipSetID := range ipSets {
		id := p.ipSetIDProvider.GetNoAlloc(ipSetID)
		if id == 0 {
			log.WithField("setID", ipSetID).Panic("Failed to look up IP set ID.")
		}

		keyOffset := leg.stackOffsetToIPSetKey()
		p.setUpIPSetKey(id, keyOffset, leg.offsetToStateIPAddressField(), leg.offsetToStatePortField())
		p.b.LoadMapFD(R1, uint32(p.ipSetMapFD))
		p.b.Mov64(R2, R10)
		p.b.AddImm64(R2, int32(keyOffset))
		p.b.Call(HelperMapLookupElem)

		if negate {
			// Negated; if we got a hit (non-0) then the rule doesn't match.
			// (Otherwise we fall through to the next match criteria.)
			p.b.JumpNEImm64(R0, 0, p.endOfRuleLabel())
		} else {
			// Non-negated; if we got a miss (0) then the rule can't match.
			// (Otherwise we fall through to the next match criteria.)
			p.b.JumpEqImm64(R0, 0, p.endOfRuleLabel())
		}
	}
}

// Match if packet matches ANY of the given IP sets.
func (p *Builder) writeIPSetOrMatch(leg matchLeg, ipSets []string) {

	onMatchLabel := p.freshPerRuleLabel()

	for _, ipSetID := range ipSets {
		id := p.ipSetIDProvider.GetNoAlloc(ipSetID)
		if id == 0 {
			log.WithField("setID", ipSetID).Panic("Failed to look up IP set ID.")
		}

		keyOffset := leg.stackOffsetToIPSetKey()
		p.setUpIPSetKey(id, keyOffset, leg.offsetToStateIPAddressField(), leg.offsetToStatePortField())
		p.b.LoadMapFD(R1, uint32(p.ipSetMapFD))
		p.b.Mov64(R2, R10)
		p.b.AddImm64(R2, int32(keyOffset))
		p.b.Call(HelperMapLookupElem)

		// If we got a hit (non-0) then packet matches one of the IP sets.
		// (Otherwise we fall through to try the next IP set.)
		p.b.JumpNEImm64(R0, 0, onMatchLabel)
	}

	// If packet reaches here, it hasn't matched any of the IP sets.
	p.b.Jump(p.endOfRuleLabel())
	// Label the next match so we can skip to it on success.
	p.b.LabelNextInsn(onMatchLabel)
}

func (p *Builder) writePortsMatch(negate bool, leg matchLeg, ports []*proto.PortRange, namedPorts []string) {
	// For a ports match, numeric ports and named ports are ORed together.  Check any
	// numeric ports first and then any named ports.
	var onMatchLabel string
	if negate {
		// Match negated, if we match any port then we jump to the next rule.
		onMatchLabel = p.endOfRuleLabel()
	} else {
		// Match is non-negated, if we match, go to the next match criteria.
		onMatchLabel = p.freshPerRuleLabel()
	}

	// R1 = port to test against.
	p.b.Load16(R1, R9, leg.offsetToStatePortField())

	for _, portRange := range ports {
		if portRange.First == portRange.Last {
			// Optimisation, single port, just do a comparison.
			p.b.JumpEqImm64(R1, portRange.First, onMatchLabel)
		} else {
			// Port range,
			var skipToNextPortLabel string
			if portRange.First > 0 {
				// If port is too low, skip to next port.
				skipToNextPortLabel = p.freshPerRuleLabel()
				p.b.JumpLTImm64(R1, portRange.First, skipToNextPortLabel)
			}
			// If port is in range, got a match, otherwise fall through to next port.
			p.b.JumpLEImm64(R1, portRange.Last, onMatchLabel)
			if portRange.First > 0 {
				p.b.LabelNextInsn(skipToNextPortLabel)
			}
		}
	}

	for _, ipSetID := range namedPorts {
		id := p.ipSetIDProvider.GetNoAlloc(ipSetID)
		if id == 0 {
			log.WithField("setID", ipSetID).Panic("Failed to look up IP set ID.")
		}

		keyOffset := leg.stackOffsetToIPSetKey()
		p.setUpIPSetKey(id, keyOffset, leg.offsetToStateIPAddressField(), leg.offsetToStatePortField())
		p.b.LoadMapFD(R1, uint32(p.ipSetMapFD))
		p.b.Mov64(R2, R10)
		p.b.AddImm64(R2, int32(keyOffset))
		p.b.Call(HelperMapLookupElem)

		p.b.JumpNEImm64(R0, 0, onMatchLabel)
	}

	if !negate {
		// If we fall through then none of the ports matched so the rule doesn't match.
		p.b.Jump(p.endOfRuleLabel())
		// Label the next match so we can skip to it on success.
		p.b.LabelNextInsn(onMatchLabel)
	}
}

func (p *Builder) freshPerRuleLabel() string {
	part := p.rulePartID
	p.rulePartID++
	return fmt.Sprintf("rule_%d_part_%d", p.ruleID, part)
}

func (p *Builder) endOfRuleLabel() string {
	return fmt.Sprintf("rule_%d_no_match", p.ruleID)
}

func protocolToNumber(protocol *proto.Protocol) uint8 {
	var pcol uint8
	switch p := protocol.NumberOrName.(type) {
	case *proto.Protocol_Name:
		switch strings.ToLower(p.Name) {
		case "tcp":
			pcol = 6
		case "udp":
			pcol = 17
		case "icmp":
			pcol = 1
		case "sctp":
			pcol = 132
		}
	case *proto.Protocol_Number:
		pcol = uint8(p.Number)
	}
	return pcol
}
