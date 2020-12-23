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

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/elf"
	"github.com/projectcalico/felix/bpf/events"
)

const (
	kprobeEventsFileName = "/sys/kernel/debug/tracing/kprobe_events"
)

var tcpFns = []string{"tcp_sendmsg", "tcp_cleanup_rbuf"}
var udpFns = []string{"udp_sendmsg", "udp_recvmsg"}

type fds struct {
	progFD       bpf.ProgFD
	tracePointFD int
}

var fdMap map[string]fds

func GetFDMap() map[string]fds {
	return fdMap
}

func InitFDMap() {
	fdMap = make(map[string]fds)
}

func progFileName(protocol, logLevel string) string {
	logLevel = strings.ToLower(logLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}
	return fmt.Sprintf("%s_%s_kprobe.o", protocol, logLevel)
}

func AttachTCPv4(logLevel string, evnt events.Events, protov4Map bpf.Map) error {
	err := installKprobe(logLevel, "tcp", tcpFns, protov4Map, evnt.Map())
	if err != nil {
		return fmt.Errorf("error installing tcp v4 kprobes")
	}
	return nil
}

func AttachUDPv4(logLevel string, evnt events.Events, protov4Map bpf.Map) error {
	err := installKprobe(logLevel, "udp", udpFns, protov4Map, evnt.Map())
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
		err = attachKprobe(progFd, fn, fdMap)
		if err != nil {
			return fmt.Errorf("error attaching kprobe to fn %s :%w", fn, err)
		}
	}
	return nil
}

func attachKprobe(progFd bpf.ProgFD, fn string, fdMap map[string]fds) error {
	var fd fds
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
	fdMap[fn] = fd
	return nil
}

func disableKprobe(fn string) error {
	fMap := GetFDMap()
	err := bpf.PerfEventDisableTracepoint(fMap[fn].tracePointFD)
	if err != nil {
		return fmt.Errorf("Error disabling perf event")
	}
	syscall.Close(int(fMap[fn].progFD))
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

func DetachTCPv4() error {
	for _, fn := range tcpFns {
		err := disableKprobe(fn)
		if err != nil {
			return err
		}
	}
	return nil
}

func DetachUDPv4() error {
	for _, fn := range udpFns {
		err := disableKprobe(fn)
		if err != nil {
			return err
		}
	}
	return nil
}
