// +build !windows

// Copyright (c) 2021 Tigera, Inc. All rights reserved.
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

	"os"

	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/elf"
	"github.com/projectcalico/felix/bpf/events"
	"github.com/projectcalico/felix/bpf/kprobe"
	"github.com/projectcalico/felix/bpf/perf"
)

func TestBpfProgramLoaderWithMultipleSections(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/ut/loader_test_with_multiple_sections.o"
	loader, err := elf.NewLoaderFromFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(0))

	insnMap, license, err := loader.Programs()
	Expect(err).NotTo(HaveOccurred())

	Expect(len(insnMap)).To(Equal(2))

	_, ok := insnMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
	_, ok = insnMap["kprobe/tcp_recvmsg"]
	Expect(ok).To(Equal(true))

	loadProgram(license, insnMap)
}
func TestBpfProgramLoaderWithoutRelocation(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/ut/loader_test_without_relocation.o"
	loader, err := elf.NewLoaderFromFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(0))

	insnMap, license, err := loader.Programs()
	Expect(err).NotTo(HaveOccurred())

	Expect(len(insnMap)).To(Equal(1))
	_, ok := insnMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
	loadProgram(license, insnMap)
}

func TestBpfProgramLoaderWithMultipleMaps(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/ut/loader_test_multiple_maps.o"

	err, testHashMap := CreateTestMap("cali_test_map1", "hash", 4, 8, 511000, unix.BPF_F_NO_PREALLOC)
	Expect(err).NotTo(HaveOccurred())

	err, testPerfMap := CreateTestMap("cali_test_map2", "hash", 4, 4, 511000, unix.BPF_F_NO_PREALLOC)
	Expect(err).NotTo(HaveOccurred())

	loader, err := elf.NewLoaderFromFile(fileName, testHashMap, testPerfMap)
	Expect(err).NotTo(HaveOccurred())
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(2))

	insnMap, license, err := loader.Programs()
	Expect(err).NotTo(HaveOccurred())

	Expect(len(insnMap)).To(Equal(1))
	_, ok := insnMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
	loadProgram(license, insnMap)
}

func TestBpfProgramLoaderWithSingleMap(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/ut/loader_test_single_map.o"

	err, testMap := CreateTestMap("cali_test_kp", "hash", 4, 8, 511000, unix.BPF_F_NO_PREALLOC)
	Expect(err).NotTo(HaveOccurred())
	loader, err := elf.NewLoaderFromFile(fileName, testMap)
	Expect(err).NotTo(HaveOccurred())
	Expect(loader).NotTo(BeNil())
	fdMap := loader.GetFDMap()
	Expect(len(fdMap)).To(Equal(1))
	insnMap, license, err := loader.Programs()
	Expect(err).NotTo(HaveOccurred())

	Expect(len(insnMap)).To(Equal(1))
	_, ok := insnMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
	loadProgram(license, insnMap)
}

func TestBpfLoaderFailureCases(t *testing.T) {
	RegisterTestingT(t)

	fileName := "dummy_test.o"
	loader, err := elf.NewLoaderFromFile(fileName)
	Expect(err).To(HaveOccurred())
	Expect(loader).To(BeNil())

	fileName = "../../bpf-gpl/ut/loader_test_single_map.o"

	loader, err = elf.NewLoaderFromFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	Expect(loader).NotTo(BeNil())

	insnMap, license, err := loader.Programs()
	Expect(err).To(HaveOccurred())
	Expect(insnMap).To(BeNil())
	Expect(license).To(Equal(""))

	err = corruptElf(fileName)
	Expect(err).NotTo(HaveOccurred())

	loader, err = elf.NewLoaderFromFile("loader_corrupt_elf.o")
	Expect(err).NotTo(HaveOccurred())
	Expect(loader).NotTo(BeNil())
	_, _, err = loader.Programs()
	Expect(err).To(HaveOccurred())
	os.Remove("loader_corrupt_elf.o")
}

func TestBPFLoaderWithTcpKprobe(t *testing.T) {
	RegisterTestingT(t)

	fileName := "../../bpf-gpl/bin/tcp_debug_kprobe.o"
	mc := &bpf.MapContext{}
	perfMap := perf.Map(mc, "perf_evnt", events.MaxCPUs)
	err := perfMap.EnsureExists()
	Expect(err).NotTo(HaveOccurred())
	kpStatMap := kprobe.MapKpStats(mc)
	err = kpStatMap.EnsureExists()
	Expect(err).NotTo(HaveOccurred())

	loader, err := elf.NewLoaderFromFile(fileName, perfMap, kpStatMap)
	Expect(err).NotTo(HaveOccurred())
	Expect(loader).NotTo(BeNil())

	insnMap, license, err := loader.Programs()
	Expect(err).NotTo(HaveOccurred())

	Expect(len(insnMap)).To(Equal(3))
	_, ok := insnMap["kprobe/tcp_sendmsg"]
	Expect(ok).To(Equal(true))
	_, ok = insnMap["kprobe/tcp_cleanup_rbuf"]
	Expect(ok).To(Equal(true))
	_, ok = insnMap["kprobe/tcp_connect"]
	Expect(ok).To(Equal(true))
	loadProgram(license, insnMap)
}

func loadProgram(license string, insnMap map[string]*elf.ProgramInfo) {
	Expect(license).To(Equal("GPL"))
	for _, v := range insnMap {
		_, err := bpf.LoadBPFProgramFromInsns(v.Insns, license, v.Type)
		Expect(err).NotTo(HaveOccurred())
	}
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

func corruptElf(fileName string) error {
	fp, err := os.Open(fileName)
	if err != nil {
		return err
	}

	finfo, err := fp.Stat()
	if err != nil {
		return err
	}
	fsize := finfo.Size()
	buffer := make([]byte, fsize)

	_, err = fp.Read(buffer)
	if err != nil {
		return err
	}
	temp := [4]byte{0xde, 0xad, 0xbe, 0xaf}
	copy(buffer[0:], temp[:])

	fp, err = os.Create("loader_corrupt_elf.o")
	if err != nil {
		return err
	}
	_, err = fp.Write(buffer)
	if err != nil {
		return err
	}
	return nil
}
