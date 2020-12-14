// +build !windows

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

package ut

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/events"
	"github.com/projectcalico/felix/bpf/kprobe"
)

func TestKprobe(t *testing.T) {
	t.Skip("XXX flaky")
	RegisterTestingT(t)
	err := bpf.MountDebugfs()
	Expect(err).NotTo(HaveOccurred())
	mc := &bpf.MapContext{}
	bpfEvnt, err := events.New(mc, events.SourcePerfEvents)
	Expect(err).NotTo(HaveOccurred())
	protov4Map := kprobe.MapProtov4(mc)
	err = protov4Map.EnsureExists()
	Expect(err).NotTo(HaveOccurred())
	err = kprobe.AttachTCPv4("debug", bpfEvnt, protov4Map)
	Expect(err).NotTo(HaveOccurred())
	err = kprobe.AttachUDPv4("debug", bpfEvnt, protov4Map)
	Expect(err).NotTo(HaveOccurred())

}
