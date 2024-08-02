// Copyright (c) 2024 Tigera, Inc. All rights reserved.
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

package nftables

import (
	"fmt"

	"github.com/projectcalico/calico/felix/environment"
)

type NflogAction struct {
	Group  uint16
	Prefix string
	Size   int
}

func (n NflogAction) ToFragment(features *environment.Features) string {
	size := 80
	if n.Size != 0 {
		size = n.Size
	}
	if n.Size < 0 {
		return fmt.Sprintf("log prefix %s group %d", n.Prefix, n.Group)
	} else {
		return fmt.Sprintf("log prefix %s snaplen %d group %d", n.Prefix, size, n.Group)
	}
}

func (n NflogAction) String() string {
	return fmt.Sprintf("Nflog:g=%d,p=%s", n.Group, n.Prefix)
}

type NfqueueAction struct {
	QueueNum int64
}

func (n NfqueueAction) ToFragment(features *environment.Features) string {
	return fmt.Sprintf("queue num %d", n.QueueNum)
}

func (n NfqueueAction) String() string {
	return "Nfqueue"
}

type NfqueueWithBypassAction struct {
	QueueNum int64
}

func (n NfqueueWithBypassAction) ToFragment(features *environment.Features) string {
	return fmt.Sprintf("queue num %d bypass", n.QueueNum)
}

func (n NfqueueWithBypassAction) String() string {
	return "NfqueueWithBypass"
}

type ChecksumAction struct {
	TypeChecksum struct{}
}

func (g ChecksumAction) ToFragment(features *environment.Features) string {
	// TODO: This appears to be unsupported in nftables.
	return ""
}

func (g ChecksumAction) String() string {
	return "Checksum-fill"
}

type TProxyAction struct {
	Mark uint32
	Mask uint32
	Port uint16
}

func (tp TProxyAction) ToFragment(_ *environment.Features) string {
	if tp.Mask == 0 {
		return fmt.Sprintf("tproxy <IPV> to :%d meta mark set mark or %#x", tp.Port, tp.Mark)
	}
	return fmt.Sprintf("tproxy <IPV> to :%d meta mark set mark & %#x ^ %#x", tp.Port, (tp.Mask ^ 0xffffffff), tp.Mark)
}

func (tp TProxyAction) String() string {
	return fmt.Sprintf("TProxy mark %#x/%#x port %d", tp.Mark, tp.Mask, tp.Port)
}
