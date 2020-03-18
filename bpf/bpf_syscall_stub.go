// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.
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

// +build !cgo

package bpf

import (
	"github.com/projectcalico/felix/bpf/asm"
)

func SyscallSupport() bool {
	return false
}

func GetMapFDByPin(filename string) (MapFD, error) {
	panic("BPF syscall stub")
}

func GetMapFDByID(mapID int) (MapFD, error) {
	panic("BPF syscall stub")
}

func LoadBPFProgramFromInsns(insns asm.Insns, license string) (ProgFD, error) {
	panic("BPF syscall stub")
}

func RunBPFProgram(fd ProgFD, dataIn []byte, repeat int) (pr ProgResult, err error) {
	panic("BPF syscall stub")
}

func PinBPFProgram(fd ProgFD, filename string) error {
	panic("BPF syscall stub")
}

func UpdateMapEntry(mapFD MapFD, k, v []byte) error {
	panic("BPF syscall stub")
}

func GetMapEntry(mapFD MapFD, k []byte, valueSize int) ([]byte, error) {
	panic("BPF syscall stub")
}

func GetMapInfo(fd MapFD) (*MapInfo, error) {
	panic("BPF syscall stub")
}

func DeleteMapEntry(mapFD MapFD, k []byte, valueSize int) error {
	panic("BPF syscall stub")
}
