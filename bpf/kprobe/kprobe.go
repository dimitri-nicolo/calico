// Copyright (c) 2021 Tigera, Inc. All rights reserved.
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
package kprobe

import (
	"fmt"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/events"
	"github.com/projectcalico/felix/bpf/libbpf"
)

const (
	kprobeEventsFileName = "/sys/kernel/debug/tracing/kprobe_events"
)

var tcpFns = []string{"tcp_sendmsg", "tcp_cleanup_rbuf", "tcp_connect"}
var udpFns = []string{"udp_sendmsg", "udp_recvmsg", "udpv6_sendmsg", "udpv6_recvmsg"}
var syscallFns = []string{"__x64_sys_execve"}

type kprobeFDs struct {
	progFD       bpf.ProgFD
	tracePointFD int
}

type bpfKprobe struct {
	logLevel   string
	kpStatsMap bpf.Map
	evnt       events.Events
	objMap     map[string]*libbpf.Obj
	linkMap    map[string]*libbpf.Link
}

func New(logLevel string, evnt events.Events, mc *bpf.MapContext) *bpfKprobe {
	kpStatsMap := MapKpStats(mc)
	err := kpStatsMap.EnsureExists()
	if err != nil {
		log.WithError(err).Errorf("kprobe: failed to create cali_kpstats map")
		return nil
	}

	ePathMap := MapEpath(mc)
	err = ePathMap.EnsureExists()
	if err != nil {
		log.WithError(err).Error("kprobe: failed to create cali_epath map")
		return nil
	}

	execMap := MapExec(mc)
	err = execMap.EnsureExists()
	if err != nil {
		log.WithError(err).Error("kprobe: failed to create cali_exec map")
		return nil
	}

	return &bpfKprobe{
		logLevel:   logLevel,
		evnt:       evnt,
		kpStatsMap: kpStatsMap,
		objMap:     make(map[string]*libbpf.Obj),
		linkMap:    make(map[string]*libbpf.Link),
	}
}

func progFileName(typ, logLevel string) string {
	logLevel = strings.ToLower(logLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}
	return fmt.Sprintf("%s_%s_kprobe.o", typ, logLevel)
}

func (k *bpfKprobe) AttachTCPv4() error {
	err := k.installKprobe("tcp", tcpFns)
	if err != nil {
		return fmt.Errorf("error installing tcp v4 kprobes %w", err)
	}
	return nil
}

func (k *bpfKprobe) AttachUDPv4() error {
	err := k.installKprobe("udp", udpFns)
	if err != nil {
		return fmt.Errorf("error installing udp v4 kprobes")
	}
	return nil
}

func (k *bpfKprobe) AttachSyscall() error {
	err := k.installKprobe("syscall", syscallFns)
	if err != nil {
		return fmt.Errorf("error install exec kprobes %w", err)
	}
	return nil
}

func (k *bpfKprobe) installKprobe(typ string, fns []string) error {
	filename := path.Join(bpf.ObjectDir, progFileName(typ, k.logLevel))
	obj, err := libbpf.OpenObject(filename)
	if err != nil {
		return fmt.Errorf("error loading kprobe program %s :%w", filename, err)
	}
	k.objMap[typ] = obj
	for _, fn := range fns {
		link, err := obj.AttachKprobe(fn, fn)
		if err != nil {
			return fmt.Errorf("error attaching kprobe to fn %s :%w", fn, err)
		}
		k.linkMap[fn] = link
	}
	return nil
}

func (k *bpfKprobe) disableKprobe(link *libbpf.Link) error {
	err := link.Close()
	if err != nil {
		return fmt.Errorf("cannot destroy link: %v", err)
	}
	return nil

}
func (k *bpfKprobe) DetachTCPv4() error {
	for _, fn := range tcpFns {
		err := k.disableKprobe(k.linkMap[fn])
		if err != nil {
			return err
		}
		delete(k.linkMap, fn)
	}
	k.Close("tcp")
	return nil
}

func (k *bpfKprobe) DetachUDPv4() error {
	for _, fn := range udpFns {
		err := k.disableKprobe(k.linkMap[fn])
		if err != nil {
			return err
		}
	}
	k.Close("udp")
	return nil
}

func (k *bpfKprobe) DetachSyscall() error {
	for _, fn := range syscallFns {
		err := k.disableKprobe(k.linkMap[fn])
		if err != nil {
			return err
		}
	}
	k.Close("syscall")
	return nil
}

func (k *bpfKprobe) Close(typ string) {
	obj := k.objMap[typ]
	obj.Close()
	delete(k.objMap, typ)
}
