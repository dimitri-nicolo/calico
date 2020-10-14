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
package bpf

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"io"
	"os"

	"github.com/pkg/errors"
)

const (
	logSize int = 524288 //512KB
)

const (
	BPF_PROG_TYPE_UNSPEC = iota
)

type Loader struct {
	programs map[string]*ProgramInfo
	mapFds   map[string]MapFD
}

type ProgramInfo struct {
	fd   int
	Type uint32
}

func NewLoader(maps ...Map) *Loader {
	BpfLoader := &Loader{
		programs: make(map[string]*ProgramInfo),
		mapFds:   make(map[string]MapFD),
	}
	for _, pinnedMap := range maps {
		BpfLoader.mapFds[pinnedMap.GetName()] = pinnedMap.MapFD()
	}
	return BpfLoader
}

func (l *Loader) GetMapFD(mapName string) MapFD {
	fd, present := l.mapFds[mapName]
	if present {
		return fd
	}
	return 0
}

func (l *Loader) GetFDMap() map[string]MapFD {
	return l.mapFds
}

func (l *Loader) GetProgramMap() map[string]*ProgramInfo {
	return l.programs
}

// Read the license section and return the license string which
// is needed for loading the BPF program
func ReadLicense(file *elf.File) (error, string) {
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
func GetRelocationOffset(data, rdata []byte, file *elf.File) (error, map[uint64]string) {
	var symbol elf.Symbol
	var symMap map[uint64]string
	symbols, err := file.Symbols()
	if err != nil {
		return err, nil
	}
	var rel elf.Rel64
	br := bytes.NewReader(data)
	symMap = make(map[uint64]string)
	for {
		err := binary.Read(br, file.ByteOrder, &rel)
		if err != nil {
			if err == io.EOF {
				return nil, symMap
			}
			return err, symMap
		}
		symNo := rel.Info >> 32
		symbol = symbols[symNo-1]
		symbolSec := file.Sections[symbol.Section]
		if symbolSec.Name == "maps" {
			symMap[rel.Off] = symbol.Name
		} else {
			return errors.Errorf("Invalid relocation section"), symMap
		}
	}
	return nil, symMap

}

// Relocate the imm value in the BPF instruction with map fd
func (l *Loader) Relocate(data, rdata []byte, file *elf.File) error {
	err, symMap := GetRelocationOffset(data, rdata, file)
	if err != nil {
		return err
	}
	for offset, mapName := range symMap {
		fd := l.GetMapFD(mapName)
		if fd == 0 {
			return errors.Errorf("Map fd invalid")
		}
		err = RelocateBpfInsn(fd, rdata, offset)
		if err != nil {
			return err
		}
	}
	return nil
}

// Basic validation of the ELF file
func ValidateElfFile(filename string) (error, *elf.File, *os.File) {
	var fileReader io.ReaderAt
	var file *elf.File
	freader, err := os.Open(filename)
	if err != nil {
		return errors.Errorf("Error opening file"), nil, nil
	}
	fileReader = freader
	file, err = elf.NewFile(fileReader)
	if err != nil {
		return errors.Errorf("Error reading elf file"), nil, nil
	}
	if file.Class != elf.ELFCLASS64 {
		return errors.Errorf("Unsupported file format"), nil, nil
	}
	return nil, file, freader
}

func (l *Loader) Load(filename string) error {
	var err error
	log := make([]byte, logSize)

	err, file, fp := ValidateElfFile(filename)
	defer fp.Close()
	if err != nil {
		return err
	}
	err, license := ReadLicense(file)
	if err != nil {
		return errors.Errorf("Error reading license")
	}
	loaded := make([]bool, len(file.Sections))
	for i, sec := range file.Sections {
		if sec.Type == elf.SHT_REL {
			if loaded[i] == true {
				continue
			}
			data, err := sec.Data()
			if err != nil {
				return errors.Errorf("Error reading section data")
			}
			if len(data) == 0 {
				continue
			}
			relocSec := file.Sections[sec.Info]
			progType := GetProgTypeFromSecName(relocSec.Name)
			if progType != BPF_PROG_TYPE_UNSPEC {
				relData, err := relocSec.Data()
				if err != nil {
					return errors.Errorf("Error reading relocation data")
				}
				if len(relData) == 0 {
					continue
				}
				err = l.Relocate(data, relData, file)
				if err != nil {
					return errors.Errorf("Error handling relocation section")
				}
				err, progfd := LoadBPFProgram(progType, relData, relocSec.Size, license, log, logSize)
				if progfd < 0 {
					return errors.Errorf("Error loading section1 %v %v", relocSec.Name, err)
				}
				loaded[i] = true
				loaded[sec.Info] = true
				l.programs[relocSec.Name] = &ProgramInfo{
					fd:   int(progfd),
					Type: progType,
				}
			}
		}
	}

	for i, sec := range file.Sections {
		if loaded[i] == true {
			continue
		}
		data, err := sec.Data()
		if err != nil {
			return errors.Errorf("Error reading section data")
		}
		if len(data) == 0 {
			continue
		}
		progType := GetProgTypeFromSecName(sec.Name)
		if progType != BPF_PROG_TYPE_UNSPEC {
			err, progfd := LoadBPFProgram(progType, data, sec.Size, license, log, logSize)
			if progfd < 0 {
				return err
			}
			loaded[i] = true
			l.programs[sec.Name] = &ProgramInfo{
				fd:   int(progfd),
				Type: progType,
			}
		}
	}
	return nil
}
