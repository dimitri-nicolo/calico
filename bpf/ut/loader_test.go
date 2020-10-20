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
	"testing"

	. "github.com/onsi/gomega"

	"golang.org/x/sys/unix"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/elf"
)

func TestBpfProgramLoaderWithMultipleSections(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/ut/loader_test_with_multiple_sections.o"
	err, elfFile, fp := elf.ValidateElfFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	Expect(elfFile).NotTo(BeNil())
	Expect(fp).NotTo(BeNil())

	err, license := elf.ReadLicense(elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(license).To(Equal("GPL"))

	loader := elf.NewLoader()
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(0))

	err = loader.Load(fileName)
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.Programs()
	Expect(len(programMap)).To(Equal(2))

	_, ok := programMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
	_, ok = programMap["kprobe/tcp_recvmsg"]
	Expect(ok).To(Equal(true))
}

func TestBpfProgramLoaderWithoutRelocation(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/ut/loader_test_without_relocation.o"
	err, elfFile, fp := elf.ValidateElfFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	Expect(elfFile).NotTo(BeNil())
	Expect(fp).NotTo(BeNil())

	err, license := elf.ReadLicense(elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(license).To(Equal("GPL"))

	loader := elf.NewLoader()
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(0))

	err = loader.Load(fileName)
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.Programs()
	Expect(len(programMap)).To(Equal(1))
	_, ok := programMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))

}

func TestBpfProgramLoaderWithMultipleMaps(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/ut/loader_test_multiple_maps.o"
	err, elfFile, fp := elf.ValidateElfFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	Expect(elfFile).NotTo(BeNil())
	Expect(fp).NotTo(BeNil())

	err, license := elf.ReadLicense(elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(license).To(Equal("GPL"))

	err, testHashMap := CreateTestMap("cali_test_map1", "hash", 4, 8, 511000, unix.BPF_F_NO_PREALLOC)
	Expect(err).NotTo(HaveOccurred())

	err, testPerfMap := CreateTestMap("cali_test_map2", "hash", 4, 4, 511000, unix.BPF_F_NO_PREALLOC)
	Expect(err).NotTo(HaveOccurred())

	loader := elf.NewLoader(testHashMap, testPerfMap)
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(2))

	err = loader.Load(fileName)
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.Programs()
	Expect(len(programMap)).To(Equal(1))
	_, ok := programMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
}

func TestBpfProgramLoaderWithSingleMap(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/ut/loader_test_single_map.o"
	err, elfFile, fp := elf.ValidateElfFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	Expect(elfFile).NotTo(BeNil())
	Expect(fp).NotTo(BeNil())

	err, license := elf.ReadLicense(elfFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(license).To(Equal("GPL"))

	err, testMap := CreateTestMap("cali_test_kp", "hash", 4, 8, 511000, unix.BPF_F_NO_PREALLOC)
	Expect(err).NotTo(HaveOccurred())
	loader := elf.NewLoader(testMap)
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(1))
	err = loader.Load(fileName)
	Expect(err).NotTo(HaveOccurred())

	programMap := loader.Programs()
	Expect(len(programMap)).To(Equal(1))
	_, ok := programMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
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
