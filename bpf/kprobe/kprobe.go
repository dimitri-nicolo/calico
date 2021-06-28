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
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/elf"
	"github.com/projectcalico/felix/bpf/events"
)

const (
	kprobeEventsFileName = "/sys/kernel/debug/tracing/kprobe_events"
)

var tcpFns = []string{"tcp_sendmsg", "tcp_cleanup_rbuf", "tcp_connect"}
var udpFns = []string{"udp_sendmsg", "udp_recvmsg", "udpv6_sendmsg", "udpv6_recvmsg"}

type kprobeFDs struct {
	progFD       bpf.ProgFD
	tracePointFD int
}

type bpfKprobe struct {
	logLevel   string
	kpStatsMap bpf.Map
	evnt       events.Events
	fdMap      map[string]kprobeFDs
}

func New(logLevel string, evnt events.Events, mc *bpf.MapContext) *bpfKprobe {
	kpStatsMap := MapKpStats(mc)
	err := kpStatsMap.EnsureExists()
	if err != nil {
		return nil
	}

	return &bpfKprobe{
		logLevel:   logLevel,
		evnt:       evnt,
		kpStatsMap: kpStatsMap,
		fdMap:      make(map[string]kprobeFDs),
	}
}

func progFileName(protocol, logLevel string) string {
	logLevel = strings.ToLower(logLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}
	return fmt.Sprintf("%s_%s_kprobe.o", protocol, logLevel)
}

func (k *bpfKprobe) AttachTCPv4() error {
	err := k.installKprobe("tcp", tcpFns)
	if err != nil {
		return fmt.Errorf("error installing tcp v4 kprobes")
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

func (k *bpfKprobe) installKprobe(protocol string, fns []string) error {
	filename := path.Join(bpf.ObjectDir, progFileName(protocol, k.logLevel))
	loader, err := elf.NewLoaderFromFile(filename, k.evnt.Map(), k.kpStatsMap)
	if err != nil {
		fmt.Errorf("error reading elf file")
	}
	insnMap, license, err := loader.Programs()
	if err != nil {
		return fmt.Errorf("error loading kprobe program %s :%w", filename, err)
	}
	for _, fn := range fns {
		sectionName := "kprobe/" + fn
		programInfo, ok := insnMap[sectionName]
		if ok != true {
			return fmt.Errorf("kprobe section %s not found", fn)
		}
		progFd, err := bpf.LoadBPFProgramFromInsns(programInfo.Insns, license, programInfo.Type)
		if err != nil {
			return err
		}
		err = k.attachKprobe(progFd, fn)
		if err != nil {
			return fmt.Errorf("error attaching kprobe to fn %s :%w", fn, err)
		}
	}
	return nil
}

func (k *bpfKprobe) attachKprobe(progFd bpf.ProgFD, fn string) error {
	var fd kprobeFDs
	kprobeIdFile := fmt.Sprintf("/sys/kernel/debug/tracing/events/kprobes/p%s/id", fn)
	kbytes, err := ioutil.ReadFile(kprobeIdFile)
	if err != nil {
		pathErr, ok := err.(*os.PathError)
		if ok && pathErr.Err == syscall.ENOENT {
			f, err := os.OpenFile(kprobeEventsFileName, os.O_APPEND|os.O_WRONLY, 0666)
			if err != nil {
				return fmt.Errorf("error opening kprobe events file :%w", err)
			}
			defer f.Close()
			cmd := fmt.Sprintf("p:p%s %s", fn, fn)
			if _, err = f.WriteString(cmd); err != nil {
				return fmt.Errorf("error writing to kprobe events file :%w", err)
			}
			kbytes, err = ioutil.ReadFile(kprobeIdFile)
			if err != nil {
				return fmt.Errorf("error reading kprobe event id : %w", err)
			}
		} else {
			return err
		}
	}
	kprobeId, err := strconv.Atoi(strings.TrimSpace(string(kbytes)))
	if err != nil {
		return fmt.Errorf("not a proper kprobe id :%w", err)
	}
	tfd, err := bpf.PerfEventOpenTracepoint(kprobeId, int(progFd))
	if err != nil {
		return fmt.Errorf("failed to attach kprobe to %s", fn)
	}
	fd.progFD = progFd
	fd.tracePointFD = tfd
	k.fdMap[fn] = fd
	return nil
}

func (k *bpfKprobe) disableKprobe(fn string) error {
	err := bpf.PerfEventDisableTracepoint(k.fdMap[fn].tracePointFD)
	if err != nil {
		return fmt.Errorf("Error disabling perf event")
	}
	syscall.Close(int(k.fdMap[fn].progFD))
	f, err := os.OpenFile(kprobeEventsFileName, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot open kprobe_events: %v", err)
	}
	defer f.Close()
	cmd := fmt.Sprintf("-:p%s\n", fn)
	if _, err = f.WriteString(cmd); err != nil {
		return fmt.Errorf("cannot write %q to kprobe_events: %v", cmd, err)
	}
	return nil
}

func (k *bpfKprobe) DetachTCPv4() error {
	for _, fn := range tcpFns {
		err := k.disableKprobe(fn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *bpfKprobe) DetachUDPv4() error {
	for _, fn := range udpFns {
		err := k.disableKprobe(fn)
		if err != nil {
			return err
		}
	}
	return nil
}
