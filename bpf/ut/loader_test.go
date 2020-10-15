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
	"github.com/projectcalico/felix/bpf/asm"
)

func TestBpfProgramLoaderWithMultipleSections(t *testing.T) {
	RegisterTestingT(t)

	err, elfFile, fp := bpf.ValidateElfFile("test_loader_multiple_sections.o")
	Expect(err).NotTo(HaveOccurred())
	Expect(elfFile).NotTo(BeNil())
	Expect(fp).NotTo(BeNil())

	err, license := bpf.ReadLicense(elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(license).To(Equal("GPL"))

	loader := bpf.NewLoader()
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(0))

	err = loader.Load("test_loader_multiple_sections.o")
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.GetProgramMap()
	Expect(len(programMap)).To(Equal(2))

	_, ok := programMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
	_, ok = programMap["kprobe/tcp_recvmsg"]
	Expect(ok).To(Equal(true))
}

func TestBpfProgramLoaderWithoutRelocation(t *testing.T) {
	RegisterTestingT(t)

	err, elfFile, fp := bpf.ValidateElfFile("test_loader_without_relocation.o")
	Expect(err).NotTo(HaveOccurred())
	Expect(elfFile).NotTo(BeNil())
	Expect(fp).NotTo(BeNil())

	err, license := bpf.ReadLicense(elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(license).To(Equal("GPL"))
	data, rdata, err := GetRelocSectionData(elfFile)
	Expect(data).To(BeNil())
	Expect(rdata).To(BeNil())
	Expect(err).NotTo(HaveOccurred())

	loader := bpf.NewLoader()
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(0))

	err = loader.Load("test_loader_without_relocation.o")
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.GetProgramMap()
	Expect(len(programMap)).To(Equal(1))
	_, ok := programMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))

}
func TestBpfProgramLoaderWithMultipleMaps(t *testing.T) {
	RegisterTestingT(t)

	err, elfFile, fp := bpf.ValidateElfFile("test_loader_multiple_maps.o")
	Expect(err).NotTo(HaveOccurred())
	Expect(elfFile).NotTo(BeNil())
	Expect(fp).NotTo(BeNil())

	err, license := bpf.ReadLicense(elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(license).To(Equal("GPL"))

	err, testHashMap := CreateTestMap("cali_v4_tcp_kp", "hash", 16, 16, 511000, unix.BPF_F_NO_PREALLOC)
	Expect(err).NotTo(HaveOccurred())

	err, testPerfMap := CreateTestMap("cali_perf_evnt", "perf_event_array", 4, 4, 128, 0)
	Expect(err).NotTo(HaveOccurred())

	loader := bpf.NewLoader(testHashMap, testPerfMap)
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(2))

	data, rdata, err := GetRelocSectionData(elfFile)
	Expect(data).NotTo(BeNil())
	Expect(rdata).NotTo(BeNil())
	Expect(err).NotTo(HaveOccurred())
	temp := make([]byte, len(rdata))
	copy(temp, rdata)

	err, symMap := bpf.GetRelocationOffset(data, rdata, elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(symMap)).To(Equal(4))

	err = loader.Relocate(data, rdata, elfFile)
	Expect(err).NotTo(HaveOccurred())
	res := bytes.Compare(temp, rdata)
	Expect(res).NotTo(Equal(0))
	CompareRelocationData(temp, rdata, symMap, fdMap)

	err = loader.Load("test_loader_multiple_maps.o")
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.GetProgramMap()
	Expect(len(programMap)).To(Equal(1))
	_, ok := programMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
}

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

	err, testMap := CreateTestMap("cali_v4_tcp_kp", "hash", 16, 16, 511000, unix.BPF_F_NO_PREALLOC)
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
	CompareRelocationData(temp, rdata, symMap, fdMap)

	err = loader.Load("test_loader.o")
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.GetProgramMap()
	Expect(len(programMap)).To(Equal(1))
	_, ok := programMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
}

func CompareRelocationData(temp, data []byte, symMap map[uint64]string, fdMap map[string]bpf.MapFD) {
	for offset, mapName := range symMap {
		insn := asm.GetBpfInsn(temp, offset)
		rinsn := asm.GetBpfInsn(data, offset)
		fd := fdMap[mapName]
		res := (insn == rinsn)
		Expect(res).NotTo(Equal(0))
		Expect(insn.Imm()).NotTo(Equal(rinsn.Imm()))
		Expect(rinsn.Imm()).To(Equal(int32(fd)))
	}
}

func GetRelocSectionData(file *elf.File) ([]byte, []byte, error) {
	for _, sec := range file.Sections {
		err, data, rdata, rsec := bpf.GetRelocationSectionData(sec, file)
		if err != nil {
			return nil, nil, err
		}
		if data == nil && rdata == nil && rsec == nil {
			continue
		}
		return data, rdata, nil
	}
	return nil, nil, nil
}

func CreateTestMap(mapName, mapType string, keySize, valueSize, maxEntries, flags int) (error, bpf.Map) {
	var testMapParams = bpf.MapParameters{
		Filename:   "/sys/fs/bpf/tc/globals/" + mapName,
		Type:       mapType,
		KeySize:    keySize,
		ValueSize:  valueSize,
		MaxEntries: maxEntries,
		Name:       mapName,
		Flags:      flags,
		Version:    1,
	}
	mc := &bpf.MapContext{}
	testMap := mc.NewPinnedMap(testMapParams)
	err := testMap.EnsureExists()
	return err, testMap

}
