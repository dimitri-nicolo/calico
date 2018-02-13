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

package iptables

import (
	"fmt"

	"github.com/projectcalico/felix/hashutils"
	"github.com/projectcalico/felix/rules"
	"regexp"
	"strings"
)

type Action interface {
	ToFragment() string
}

type GotoAction struct {
	Target   string
	TypeGoto struct{}
}

func (g GotoAction) ToFragment() string {
	return "--goto " + g.Target
}

func (g GotoAction) String() string {
	return "Goto->" + g.Target
}

type JumpAction struct {
	Target   string
	TypeJump struct{}
}

func (g JumpAction) ToFragment() string {
	return "--jump " + g.Target
}

func (g JumpAction) String() string {
	return "Jump->" + g.Target
}

type ReturnAction struct {
	TypeReturn struct{}
}

func (r ReturnAction) ToFragment() string {
	return "--jump RETURN"
}

func (r ReturnAction) String() string {
	return "Return"
}

type DropAction struct {
	TypeDrop struct{}
}

func (g DropAction) ToFragment() string {
	return "--jump DROP"
}

func (g DropAction) String() string {
	return "Drop"
}

type LogAction struct {
	Prefix  string
	TypeLog struct{}
}

func (g LogAction) ToFragment() string {
	return fmt.Sprintf(`--jump LOG --log-prefix "%s: " --log-level 5`, g.Prefix)
}

func (g LogAction) String() string {
	return "Log"
}

type AcceptAction struct {
	TypeAccept struct{}
}

func (g AcceptAction) ToFragment() string {
	return "--jump ACCEPT"
}

func (g AcceptAction) String() string {
	return "Accept"
}

type NflogAction struct {
	Group  uint16
	Prefix string
}

func (n NflogAction) ToFragment() string {
	// TODO (Matt): Review number of bytes

	// NFLOG prefix which is a combination of action, rule index, policy/profile and tier name
	// separated by `|`s. Example: "D|0|default.deny-icmp|po".
	// We calculate the hash of the prefix's policy/profile name part (which includes tier name and namespace, if applicable)
	// if its length exceeds NFLOG prefix max length which is 64 characters - 9 (9 for first (A|D|N) then a `|` then
	// 3 digits for up to 999 for rule indexes, a `|` after that and 3 more for the `|po` suffix at the end.
	//
	// trimmedPrefix will split the prefix after "D|0|" (ActionID|RuleIndex|) so we get the profile/policy ID which includes tier and namespace.
	//
	// See lookup/lookup_mgr.go PushNFLOGPrefixHash() & PushNFLOGPrefixHash() funcs, these 3 functions need to be in sync,
	// if you are updating the current function, we probably need to change that ones as well.
	trimmedPrefixSlice :=rules.NFLOGPrefixRegexp.Split(n.Prefix, 2)
	trimmedPrefix := trimmedPrefixSlice[0]

	// Remove the `|po` or `|pr` part before calculating the hash.
	sepIdx := strings.Index(trimmedPrefix, "|")
	if sepIdx != -1 {
		trimmedPrefix = trimmedPrefix[:sepIdx]
	}

	prefixHash := hashutils.GetLengthLimitedID("", trimmedPrefix, rules.NFLOGPrefixMaxLength - 9)

	// Reinsert the `|po` or `|pr` suffix before programming the rule.
	return fmt.Sprintf("--jump NFLOG --nflog-group %d --nflog-prefix %s --nflog-range 80", n.Group, fmt.Sprintf("%s|%s", prefixHash, trimmedPrefix[sepIdx:]))
}

func (n NflogAction) String() string {
	return fmt.Sprintf("Nflog:g=%d,p=%s", n.Group, n.Prefix)
}

type DNATAction struct {
	DestAddr string
	DestPort uint16
	TypeDNAT struct{}
}

func (g DNATAction) ToFragment() string {
	if g.DestPort == 0 {
		return fmt.Sprintf("--jump DNAT --to-destination %s", g.DestAddr)
	}
	
	return fmt.Sprintf("--jump DNAT --to-destination %s:%d", g.DestAddr, g.DestPort)
}

func (g DNATAction) String() string {
	return fmt.Sprintf("DNAT->%s:%d", g.DestAddr, g.DestPort)
}

type SNATAction struct {
	ToAddr   string
	TypeSNAT struct{}
}

func (g SNATAction) ToFragment() string {
	return fmt.Sprintf("--jump SNAT --to-source %s", g.ToAddr)
}

func (g SNATAction) String() string {
	return fmt.Sprintf("SNAT->%s", g.ToAddr)
}

type MasqAction struct {
	TypeMasq struct{}
}

func (g MasqAction) ToFragment() string {
	return "--jump MASQUERADE"
}

func (g MasqAction) String() string {
	return "Masq"
}

type ClearMarkAction struct {
	Mark          uint32
	TypeClearMark struct{}
}

func (c ClearMarkAction) ToFragment() string {
	return fmt.Sprintf("--jump MARK --set-mark 0/%#x", c.Mark)
}

func (c ClearMarkAction) String() string {
	return fmt.Sprintf("Clear:%#x", c.Mark)
}

type SetMarkAction struct {
	Mark        uint32
	TypeSetMark struct{}
}

func (c SetMarkAction) ToFragment() string {
	return fmt.Sprintf("--jump MARK --set-mark %#x/%#x", c.Mark, c.Mark)
}

func (c SetMarkAction) String() string {
	return fmt.Sprintf("Set:%#x", c.Mark)
}

type SetMaskedMarkAction struct {
	Mark              uint32
	Mask              uint32
	TypeSetMaskedMark struct{}
}

func (c SetMaskedMarkAction) ToFragment() string {
	return fmt.Sprintf("--jump MARK --set-mark %#x/%#x", c.Mark, c.Mask)
}

func (c SetMaskedMarkAction) String() string {
	return fmt.Sprintf("Set:%#x", c.Mark)
}

type NoTrackAction struct {
	TypeNoTrack struct{}
}

func (g NoTrackAction) ToFragment() string {
	return "--jump NOTRACK"
}

func (g NoTrackAction) String() string {
	return "NOTRACK"
}
