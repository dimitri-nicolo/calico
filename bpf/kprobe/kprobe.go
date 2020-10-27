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

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/elf"
)

func progFileName(protocol, logLevel string) string {
	logLevel = strings.ToLower(logLevel)
	if logLevel == "off" {
		logLevel = "no_log"
	}
	return fmt.Sprintf("%s_%s_kprobe.o", protocol, logLevel)
}

func Install(logLevel, protocol, fn string, tcpV4Map bpf.Map) error {
	filename := path.Join(bpf.ObjectDir, progFileName(protocol, logLevel))
	loader := elf.NewLoader(filename, tcpV4Map)
	insnMap, license, err := loader.Program()
	if err != nil {
		return fmt.Errorf("Error loading kprobe program %s %w", filename, err)
	}
	sectionName := "kprobe/" + fn
	programInfo, ok := insnMap[sectionName]
	if ok != true {
		return fmt.Errorf("Kprobe section not found")
	}
	progFd, err := bpf.LoadBPFProgramFromInsns(programInfo.Insns, license, programInfo.Type)
	if err != nil {
		return err
	}
	return attachKprobe(progFd, fn)
}

func attachKprobe(progFd bpf.ProgFD, fn string) error {
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
	kprobeIdFile := fmt.Sprintf("/sys/kernel/debug/tracing/events/kprobes/p%s/id", fn)
	kbytes, err := ioutil.ReadFile(kprobeIdFile)
	if err != nil {
		return fmt.Errorf("Error reading kprobe event id %w", err)
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
