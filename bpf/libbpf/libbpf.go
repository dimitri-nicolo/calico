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

package libbpf

import (
	"fmt"
	"unsafe"

	"github.com/projectcalico/felix/bpf"
)

// #include "libbpf_api.h"
import "C"

type Obj struct {
	obj *C.struct_bpf_object
}

type Map struct {
	bpfMap *C.struct_bpf_map
	bpfObj *C.struct_bpf_object
}

const MapTypeProgrArray = C.BPF_MAP_TYPE_PROG_ARRAY

type QdiskHook string

const (
	QdiskIngress QdiskHook = "ingress"
	QdiskEgress  QdiskHook = "egress"
)

func (m *Map) Name() string {
	name := C.bpf_map__name(m.bpfMap)
	if name == nil {
		return ""
	}
	return C.GoString(name)
}

func (m *Map) Type() int {
	mapType := C.bpf_map__type(m.bpfMap)
	return int(mapType)
}

func (m *Map) SetPinPath(path string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	err := C.bpf_map__set_pin_path(m.bpfMap, cPath)
	if err != 0 {
		return fmt.Errorf("pinning map failed %w", err)
	}
	return nil
}

func OpenObject(filename string) (*Obj, error) {
	bpf.IncreaseLockedMemoryQuota()
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))
	obj, err := C.bpf_obj_open(cFilename)
	if obj == nil || err != nil {
		return nil, fmt.Errorf("error opening libbpf object %w", err)
	}
	return &Obj{obj: obj}, nil
}

func (o *Obj) Load() error {
	_, err := C.bpf_obj_load(o.obj)
	if err != nil {
		return fmt.Errorf("error loading object %w", err)
	}
	return nil
}

// FirstMap returns first bpf map of the object.
// Returns error if the map is nil.
func (o *Obj) FirstMap() (*Map, error) {
	bpfMap, err := C.bpf_map__next(nil, o.obj)
	if bpfMap == nil || err != nil {
		return nil, fmt.Errorf("error getting first map %w", err)
	}
	return &Map{bpfMap: bpfMap, bpfObj: o.obj}, nil
}

// NextMap returns the successive maps given the first map.
// Returns nil if the map is nil, no error as this is the last map.
func (m *Map) NextMap() (*Map, error) {
	bpfMap, err := C.bpf_map__next(m.bpfMap, m.bpfObj)
	if err != nil {
		return nil, fmt.Errorf("error getting next map %w", err)
	}
	if bpfMap == nil {
		return nil, nil
	}
	return &Map{bpfMap: bpfMap, bpfObj: m.bpfObj}, nil
}

func (o *Obj) AttachClassifier(secName, ifName, hook string) (int, error) {
	isIngress := 0
	cSecName := C.CString(secName)
	cIfName := C.CString(ifName)
	defer C.free(unsafe.Pointer(cSecName))
	defer C.free(unsafe.Pointer(cIfName))
	ifIndex, err := C.if_nametoindex(cIfName)
	if err != nil {
		return -1, err
	}

	if hook == string(QdiskIngress) {
		isIngress = 1
	}

	opts, err := C.bpf_tc_program_attach(o.obj, cSecName, C.int(ifIndex), C.int(isIngress))
	if err != nil {
		return -1, fmt.Errorf("Error attaching tc program %w", err)
	}

	progId, err := C.bpf_tc_query_iface(C.int(ifIndex), opts, C.int(isIngress))
	if err != nil {
		return -1, fmt.Errorf("Error querying interface %s: %w", ifName, err)
	}
	return int(progId), nil
}

func (o *Obj) AttachKprobe(progName, fn string) (*Link, error) {
	cProgName := C.CString(progName)
	cFnName := C.CString(fn)
	defer C.free(unsafe.Pointer(cProgName))
	defer C.free(unsafe.Pointer(cFnName))
	link := C.bpf_program_attach_kprobe(o.obj, cProgName, cFnName)
	if link.link == nil {
		msg := "error attaching kprobe"
		if link.errno != 0 {
			errno := syscall.Errno(-int64(link.errno))
			msg = fmt.Sprintf("error attaching kprobe: %v", errno.Error())
		}
		return nil, fmt.Errorf(msg)
	}
	return &Link{link: link.link}, nil
}

func CreateQDisc(ifName string) error {
	cIfName := C.CString(ifName)
	defer C.free(unsafe.Pointer(cIfName))
	ifIndex, err := C.if_nametoindex(cIfName)
	if err != nil {
		return err
	}
	_, err = C.bpf_tc_create_qdisc(C.int(ifIndex))
	if err != nil {
		return fmt.Errorf("Error creating qdisc %w", err)
	}
	return nil
}

func RemoveQDisc(ifName string) error {
	cIfName := C.CString(ifName)
	defer C.free(unsafe.Pointer(cIfName))
	ifIndex, err := C.if_nametoindex(cIfName)
	if err != nil {
		return err
	}
	_, err = C.bpf_tc_remove_qdisc(C.int(ifIndex))
	if err != nil {
		return fmt.Errorf("Error removing qdisc %w", err)
	}
	return nil
}

func (o *Obj) UpdateJumpMap(mapName, progName string, mapIndex int) error {
	cMapName := C.CString(mapName)
	cProgName := C.CString(progName)
	defer C.free(unsafe.Pointer(cMapName))
	defer C.free(unsafe.Pointer(cProgName))
	_, err := C.bpf_tc_update_jump_map(o.obj, cMapName, cProgName, C.int(mapIndex))
	if err != nil {
		return fmt.Errorf("Error updating %s at index %d: %w", mapName, mapIndex, err)
	}
	return nil
}

func (o *Obj) Close() error {
	if o.obj != nil {
		C.bpf_object__close(o.obj)
		o.obj = nil
		return nil
	}
	return fmt.Errorf("error: libbpf obj nil")
}
