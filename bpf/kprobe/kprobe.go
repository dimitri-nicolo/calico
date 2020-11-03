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

func progFileName(protocol, logLevel string) string {
	logLevel = strings.ToLower(logLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}
	return fmt.Sprintf("%s_%s_kprobe.o", protocol, logLevel)
}

func NewKprobe(logLevel string, perfEvnt events.Events, mc *bpf.MapContext) error {
	err := bpf.MountDebugfs()
	if err != nil {
		log.WithError(err).Panic("Failed to mount debug fs")
	}

	tcpv4Map := TcpV4Map(mc)
	err = tcpv4Map.EnsureExists()
	if err != nil {
		log.WithError(err).Panic("Failed to create kprobe tcp v4 BPF map.")
	}

	var tcpFns = []string{"tcp_sendmsg", "tcp_cleanup_rbuf"}
	err = installKprobe(logLevel, "tcp", tcpFns, tcpv4Map, perfEvnt.Map(), tcpv4Map)
	if err != nil {
		return fmt.Errorf("Error installing kprobes")
	}
	return nil
}

func installKprobe(logLevel, protocol string, fns []string, maps ...bpf.Map) error {
	filename := path.Join(bpf.ObjectDir, progFileName(protocol, logLevel))
	loader, err := elf.NewLoaderFromFile(filename, maps...)
	if err != nil {
		fmt.Errorf("Error reading elf file")
	}
	insnMap, license, err := loader.Programs()
	if err != nil {
		return fmt.Errorf("Error loading kprobe program %s %w", filename, err)
	}
	for _, fn := range fns {
		sectionName := "kprobe/" + fn
		programInfo, ok := insnMap[sectionName]
		if ok != true {
			return fmt.Errorf("Kprobe section %s not found", fn)
		}
		progFd, err := bpf.LoadBPFProgramFromInsns(programInfo.Insns, license, programInfo.Type)
		if err != nil {
			return err
		}
		err = attachKprobe(progFd, fn)
		if err != nil {
			return fmt.Errorf("Error attaching kprobe to fn %s", fn)
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
			kprobeEventsFileName := "/sys/kernel/debug/tracing/kprobe_events"
			f, err := os.OpenFile(kprobeEventsFileName, os.O_APPEND|os.O_WRONLY, 0666)
			if err != nil {
				return fmt.Errorf("Error opening kprobe events file %w", err)
			}
			defer f.Close()
			cmd := fmt.Sprintf("p:p%s %s", fn, fn)
			if _, err = f.WriteString(cmd); err != nil {
				return fmt.Errorf("Error writing to kprobe events file %w", err)
			}
			kbytes, err = ioutil.ReadFile(kprobeIdFile)
			if err != nil {
				return fmt.Errorf("Error reading kprobe event id %w", err)
			}
		} else {
			return err
		}
	}
	kprobeId, err := strconv.Atoi(strings.TrimSpace(string(kbytes)))
	if err != nil {
		return fmt.Errorf("Not a proper kprobe id %w", err)
	}
	_, err = bpf.PerfEventOpenTracepoint(kprobeId, int(progFd))
	if err != nil {
		return fmt.Errorf("Failed to attach kprobe to %s", fn)
	}
	return nil
}
