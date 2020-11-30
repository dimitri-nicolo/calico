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
package kprobe

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/elf"
	"github.com/projectcalico/felix/bpf/events"
)

const (
	kprobeEventsFileName = "/sys/kernel/debug/tracing/kprobe_events"
)

func progFileName(protocol, logLevel string) string {
	logLevel = strings.ToLower(logLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}
	return fmt.Sprintf("%s_%s_kprobe.o", protocol, logLevel)
}

func AttachTCPv4(logLevel string, evnt events.Events, mc *bpf.MapContext) error {
	tcpv4Map := MapTCPv4(mc)
	err := tcpv4Map.EnsureExists()
	if err != nil {
		log.WithError(err).Panic("Failed to create kprobe tcp v4 BPF map.")
	}
	var tcpFns = []string{"tcp_sendmsg", "tcp_cleanup_rbuf"}
	err = installKprobe(logLevel, "tcp", tcpFns, tcpv4Map, evnt.Map())
	if err != nil {
		return fmt.Errorf("error installing tcp v4 kprobes")
	}
	return nil
}

func AttachUDPv4(logLevel string, evnt events.Events, mc *bpf.MapContext) error {
	udpv4Map := MapUDPv4(mc)
	err := udpv4Map.EnsureExists()
	if err != nil {
		log.WithError(err).Panic("Failed to create kprobe udp v4 BPF map.")
	}
	var udpFns = []string{"udp_sendmsg", "udp_recvmsg"}
	err = installKprobe(logLevel, "udp", udpFns, udpv4Map, evnt.Map())
	if err != nil {
		return fmt.Errorf("error installing udp v4 kprobes")
	}
	return nil
}

func installKprobe(logLevel, protocol string, fns []string, maps ...bpf.Map) error {
	filename := path.Join(bpf.ObjectDir, progFileName(protocol, logLevel))
	loader, err := elf.NewLoaderFromFile(filename, maps...)
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
		err = attachKprobe(progFd, fn)
		if err != nil {
			return fmt.Errorf("error attaching kprobe to fn %s :%w", fn, err)
		}
	}
	return nil
}

func attachKprobe(progFd bpf.ProgFD, fn string) error {
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
	_, err = bpf.PerfEventOpenTracepoint(kprobeId, int(progFd))
	if err != nil {
		return fmt.Errorf("failed to attach kprobe to %s", fn)
	}
	return nil
}
