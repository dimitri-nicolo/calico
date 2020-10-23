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

package elf

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/asm"
)

type SectionInfo struct {
	name string
	Type elf.SectionType
	info uint32
	data []byte
}

type Loader struct {
	mapFds map[string]bpf.MapFD
	eInfo  *elfInfo
}

type ProgramInfo struct {
	Insns asm.Insns
	Type  uint32
}

type elfInfo struct {
	elfFile  *elf.File
	filePtr  *os.File
	fileName string
	sections map[string]*SectionInfo
}

func NewLoader(fileName string, maps ...bpf.Map) *Loader {
	BpfLoader := &Loader{
		mapFds: make(map[string]bpf.MapFD),
	}
	for _, pinnedMap := range maps {
		BpfLoader.mapFds[pinnedMap.GetName()] = pinnedMap.MapFD()
	}
	BpfLoader.eInfo = &elfInfo{
		fileName: fileName,
		sections: make(map[string]*SectionInfo),
	}

	return BpfLoader
}

func (l *Loader) MapFD(mapName string) (bpf.MapFD, error) {
	fd, present := l.mapFds[mapName]
	if present {
		return fd, nil
	}
	return 0, fmt.Errorf("Map FD not found %s", mapName)
}

func (l *Loader) ElfInfo() *elfInfo {
	return (l.eInfo)
}

func (l *Loader) GetFDMap() map[string]bpf.MapFD {
	return l.mapFds
}

// Read the license section and return the license string which
// is needed for loading the BPF program
func (e *elfInfo) readLicense() (error, string) {
	file := e.elfFile
	lsec := file.Section("license")
	if lsec != nil {
		data, err := lsec.Data()
		if err == nil {
			return nil, string(data[0 : len(data)-1])
		}
		return err, ""
	}
	return nil, ""
}

// Get the relocation offset and the name of the map whose FD needs to be added to the
// BPF instruction
func (e *elfInfo) getMapRelocations(data []byte) (map[uint64]string, error) {
	var symbol elf.Symbol
	var symMap map[uint64]string
	file := e.elfFile
	symbols, err := file.Symbols()
	if err != nil {
		return nil, err
	}
	var rel elf.Rel64
	br := bytes.NewReader(data)
	symMap = make(map[uint64]string)
	for {
		err := binary.Read(br, file.ByteOrder, &rel)
		if err != nil {
			if err == io.EOF {
				return symMap, nil
			}
			return symMap, err
		}
		symNo := rel.Info >> 32
		symbol = symbols[symNo-1]
		symbolSec := file.Sections[symbol.Section]
		if symbolSec.Name == "maps" {
			symMap[rel.Off] = symbol.Name
		} else {
			return symMap, fmt.Errorf("Invalid relocation section %s", symbolSec.Name)
		}
	}
	return symMap, nil

}

// Relocate the imm value in the BPF instruction with map fd
func (e *elfInfo) relocate(data, rdata []byte, bpfMap map[string]bpf.MapFD) error {
	symMap, err := e.getMapRelocations(data)
	if err != nil {
		return err
	}
	for offset, mapName := range symMap {
		fd, ok := bpfMap[mapName]
		if ok != true {
			return fmt.Errorf("Map FD not found %s", mapName)
		}
		err = asm.RelocateBpfInsn(uint32(fd), rdata, offset)
		if err != nil {
			return err
		}
	}
	return nil
}

// Basic validation of the ELF file
func (e *elfInfo) readElfFile() error {
	var fileReader io.ReaderAt
	var file *elf.File
	freader, err := os.Open(e.fileName)
	if err != nil {
		return fmt.Errorf("Error opening file %w", err)
	}
	fileReader = freader
	file, err = elf.NewFile(fileReader)
	if err != nil {
		return fmt.Errorf("Error reading elf file %w", err)
	}
	if file.Class != elf.ELFCLASS64 {
		return fmt.Errorf("Unsupported file format")
	}
	e.elfFile = file
	e.filePtr = freader
	return nil
}

func (e *elfInfo) readSectionData() error {
	for _, sec := range e.elfFile.Sections {
		data, err := sec.Data()
		if err != nil {
			return err
		}
		if len(data) == 0 {
			continue
		}
		if unix.BPF_PROG_TYPE_UNSPEC != GetProgTypeFromSecName(sec.Name) {
			e.sections[sec.Name] = &SectionInfo{name: sec.Name, Type: sec.Type, info: sec.Info}
			e.sections[sec.Name].data = make([]byte, len(data))
			copy(e.sections[sec.Name].data[:], data[:])
		}
	}
	return nil
}

func (l *Loader) Program() (map[string]*ProgramInfo, string, error) {
	var err error
	eInfo := l.ElfInfo()
	err = eInfo.readElfFile()
	defer eInfo.filePtr.Close()
	if err != nil {
		return nil, "", err
	}
	err, license := eInfo.readLicense()
	if err != nil {
		return nil, "", fmt.Errorf("Error reading license %w", err)
	}

	err = eInfo.readSectionData()
	if err != nil {
		return nil, "", err
	}

	file := eInfo.elfFile
	for _, sInfo := range eInfo.sections {
		if sInfo.Type == elf.SHT_REL {
			rName := file.Sections[sInfo.info].Name
			rSec, ok := eInfo.sections[rName]
			if ok != true {
				return nil, "", fmt.Errorf("Failed to retrieve relocation section data: %s", sInfo.name)
			}
			err = eInfo.relocate(sInfo.data, rSec.data, l.GetFDMap())
			if err != nil {
				return nil, "", fmt.Errorf("Failed to relocate section %s", sInfo.name)
			}
		}
	}
	insnMap := make(map[string]*ProgramInfo)

	for _, sInfo := range eInfo.sections {
		if sInfo.Type != elf.SHT_REL {
			progType := GetProgTypeFromSecName(sInfo.name)
			insnMap[sInfo.name] = &ProgramInfo{Insns: asm.GetBPFInsns(sInfo.data), Type: progType}
		}
	}
	return insnMap, license, nil
}

func GetProgTypeFromSecName(secName string) uint32 {
	if strings.Contains(secName, "kprobe") {
		return uint32(unix.BPF_PROG_TYPE_KPROBE)
	}
	return uint32(unix.BPF_PROG_TYPE_UNSPEC)
}
