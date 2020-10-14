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

package ut

import (
	"bytes"
	"debug/elf"
	"testing"

	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"

	"github.com/projectcalico/felix/bpf"
)

func TestBpfProgramLoader(t *testing.T) {
	RegisterTestingT(t)

	err, elfFile, fp := bpf.ValidateElfFile("test")
	Expect(err).To(HaveOccurred())
	Expect(elfFile).To(BeNil())
	Expect(fp).To(BeNil())

	err, elfFile, fp = bpf.ValidateElfFile("test_loader.o")
	Expect(err).NotTo(HaveOccurred())
	Expect(elfFile).NotTo(BeNil())
	Expect(fp).NotTo(BeNil())

	err, license := bpf.ReadLicense(elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(license).To(Equal("GPL"))

	err, testMap := CreateTestMap()
	Expect(err).NotTo(HaveOccurred())

	loader := bpf.NewLoader(testMap)
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(1))

	data, rdata, err := GetRelocSectionData(elfFile)
	Expect(data).NotTo(BeNil())
	Expect(rdata).NotTo(BeNil())
	Expect(err).NotTo(HaveOccurred())
	temp := make([]byte, len(rdata))
	copy(temp, rdata)

	err, symMap := bpf.GetRelocationOffset(data, rdata, elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(symMap)).To(Equal(3))

	err = loader.Relocate(data, rdata, elfFile)
	Expect(err).NotTo(HaveOccurred())
	res := bytes.Compare(temp, rdata)
	Expect(res).NotTo(Equal(0))
	CompareRelocationData(temp, rdata, symMap, fdMap["cali_v4_tcp_kp"])

	err = loader.Load("test_loader.o")
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.GetProgramMap()
	Expect(len(programMap)).To(Equal(1))
}

func CompareRelocationData(temp, data []byte, symMap map[uint64]string, fd bpf.MapFD) {
	for offset, _ := range symMap {
		insn := bpf.GetBpfInsn(temp, offset)
		rinsn := bpf.GetBpfInsn(data, offset)
		Expect(insn).NotTo(Equal(rinsn))
		Expect(bpf.GetBpfInsnImm(insn)).NotTo(Equal(bpf.GetBpfInsnImm(rinsn)))
		Expect(bpf.GetBpfInsnImm(rinsn)).To(Equal(uint32(fd)))
	}
}

func GetRelocSectionData(file *elf.File) ([]byte, []byte, error) {
	for _, sec := range file.Sections {
		if sec.Type == elf.SHT_REL {
			data, err := sec.Data()
			if err != nil {
				return nil, nil, err
			}
			if len(data) == 0 {
				continue
			}
			relocSec := file.Sections[sec.Info]
			progType := bpf.GetProgTypeFromSecName(relocSec.Name)
			if progType != 0 {
				relData, err := relocSec.Data()
				if err != nil {
					return nil, nil, err
				}
				if len(relData) == 0 {
					continue
				}
				return data, relData, nil
			}
		}
	}
	return nil, nil, nil
}

func CreateTestMap() (error, bpf.Map) {
	var testMapParams = bpf.MapParameters{
		Filename:   "/sys/fs/bpf/tc/globals/cali_v4_tcp_kp",
		Type:       "hash",
		KeySize:    16,
		ValueSize:  16,
		MaxEntries: 511000,
		Name:       "cali_v4_tcp_kp",
		Flags:      unix.BPF_F_NO_PREALLOC,
		Version:    1,
	}
	mc := &bpf.MapContext{}
	testMap := mc.NewPinnedMap(testMapParams)
	err := testMap.EnsureExists()
	return err, testMap

}
