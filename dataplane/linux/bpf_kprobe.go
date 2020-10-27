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

package intdataplane

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/kprobe"
)

func initKprobe(logLevel string, mc *bpf.MapContext) error {
	err := bpf.MountDebugfs()
	if err != nil {
		log.WithError(err).Panic("Failed to mount debug fs")
	}

	tcpv4Map := kprobe.TcpV4Map(mc)
	err = tcpv4Map.EnsureExists()
	if err != nil {
		log.WithError(err).Panic("Failed to create kprobe tcp v4 BPF map.")
	}

	err = kprobe.Install(logLevel, "tcp", "tcp_sendmsg", tcpv4Map)
	if err != nil {
		return err
	}
	return nil
}
