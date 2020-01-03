// Protocol Buffers for Go with Gadgets
//
// Copyright (c) 2015, The GoGo Authors. All rights reserved.
// http://github.com/gogo/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package proto2_maps

import (
	"testing"

	"github.com/gogo/protobuf/proto"
)

func TestNilMaps(t *testing.T) {
	m := &AllMaps{StringToMsgMap: map[string]*FloatingPoint{"a": nil}}
	data, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	size := m.Size()
	protoSize := proto.Size(m)
	marshaledSize := len(data)
	if size != protoSize || marshaledSize != protoSize {
		t.Errorf("size %d != protoSize %d != marshaledSize %d", size, protoSize, marshaledSize)
	}
	m2 := &AllMaps{}
	if err := proto.Unmarshal(data, m2); err != nil {
		t.Fatal(err)
	}
	if v, ok := m2.StringToMsgMap["a"]; !ok {
		t.Error("element not in map")
	} else if v != nil {
		t.Errorf("element should be nil, but its %v", v)
	}
}

func TestNilMapsBytes(t *testing.T) {
	m := &AllMaps{StringToBytesMap: map[string][]byte{"a": nil}}
	data, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	size := m.Size()
	protoSize := proto.Size(m)
	marshaledSize := len(data)
	if size != protoSize || marshaledSize != protoSize {
		t.Errorf("size %d != protoSize %d != marshaledSize %d", size, protoSize, marshaledSize)
	}
	m2 := &AllMaps{}
	if err := proto.Unmarshal(data, m2); err != nil {
		t.Fatal(err)
	}
	if v, ok := m2.StringToBytesMap["a"]; !ok {
		t.Error("element not in map")
	} else if len(v) != 0 {
		t.Errorf("element should be empty, but its %v", v)
	}
}

func TestEmptyMapsBytes(t *testing.T) {
	m := &AllMaps{StringToBytesMap: map[string][]byte{"b": {}}}
	data, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	size := m.Size()
	protoSize := proto.Size(m)
	marshaledSize := len(data)
	if size != protoSize || marshaledSize != protoSize {
		t.Errorf("size %d != protoSize %d != marshaledSize %d", size, protoSize, marshaledSize)
	}
	m2 := &AllMaps{}
	if err := proto.Unmarshal(data, m2); err != nil {
		t.Fatal(err)
	}
	if v, ok := m2.StringToBytesMap["b"]; !ok {
		t.Error("element not in map")
	} else if len(v) != 0 {
		t.Errorf("element should be empty, but its %v", v)
	}
}
