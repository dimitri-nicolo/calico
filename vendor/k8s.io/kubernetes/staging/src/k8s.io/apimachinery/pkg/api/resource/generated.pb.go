/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by protoc-gen-gogo.
// source: k8s.io/kubernetes/vendor/k8s.io/apimachinery/pkg/api/resource/generated.proto
// DO NOT EDIT!

/*
	Package resource is a generated protocol buffer package.

	It is generated from these files:
		k8s.io/kubernetes/vendor/k8s.io/apimachinery/pkg/api/resource/generated.proto

	It has these top-level messages:
		Quantity
*/
package resource

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

func (m *Quantity) Reset()                    { *m = Quantity{} }
func (*Quantity) ProtoMessage()               {}
func (*Quantity) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{0} }

func init() {
	proto.RegisterType((*Quantity)(nil), "k8s.io.apimachinery.pkg.api.resource.Quantity")
}

func init() {
	proto.RegisterFile("k8s.io/kubernetes/vendor/k8s.io/apimachinery/pkg/api/resource/generated.proto", fileDescriptorGenerated)
}

var fileDescriptorGenerated = []byte{
	// 253 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x74, 0x8f, 0xb1, 0x4a, 0x03, 0x41,
	0x10, 0x86, 0x77, 0x1b, 0x89, 0x57, 0x06, 0x11, 0x49, 0xb1, 0x17, 0xc4, 0x42, 0x04, 0x77, 0x0a,
	0x9b, 0x60, 0x69, 0x6f, 0xa1, 0xa5, 0xdd, 0xdd, 0x65, 0xdc, 0x2c, 0x67, 0x76, 0x8f, 0xd9, 0x59,
	0x21, 0x5d, 0x4a, 0xcb, 0x94, 0x96, 0xb9, 0xb7, 0x49, 0x99, 0xd2, 0xc2, 0xc2, 0x3b, 0x5f, 0x44,
	0x72, 0xc9, 0x81, 0x08, 0x76, 0xf3, 0xfd, 0xc3, 0x37, 0xfc, 0x93, 0xdc, 0x97, 0x93, 0xa0, 0xad,
	0x87, 0x32, 0xe6, 0x48, 0x0e, 0x19, 0x03, 0xbc, 0xa2, 0x9b, 0x7a, 0x82, 0xc3, 0x22, 0xab, 0xec,
	0x3c, 0x2b, 0x66, 0xd6, 0x21, 0x2d, 0xa0, 0x2a, 0xcd, 0x2e, 0x00, 0xc2, 0xe0, 0x23, 0x15, 0x08,
	0x06, 0x1d, 0x52, 0xc6, 0x38, 0xd5, 0x15, 0x79, 0xf6, 0xc3, 0x8b, 0xbd, 0xa5, 0x7f, 0x5b, 0xba,
	0x2a, 0xcd, 0x2e, 0xd0, 0xbd, 0x35, 0xba, 0x36, 0x96, 0x67, 0x31, 0xd7, 0x85, 0x9f, 0x83, 0xf1,
	0xc6, 0x43, 0x27, 0xe7, 0xf1, 0xb9, 0xa3, 0x0e, 0xba, 0x69, 0x7f, 0x74, 0x74, 0xf3, 0x5f, 0x95,
	0xc8, 0xf6, 0x05, 0xac, 0xe3, 0xc0, 0xf4, 0xb7, 0xc9, 0xf9, 0x24, 0x19, 0x3c, 0xc4, 0xcc, 0xb1,
	0xe5, 0xc5, 0xf0, 0x34, 0x39, 0x0a, 0x4c, 0xd6, 0x99, 0x33, 0x39, 0x96, 0x97, 0xc7, 0x8f, 0x07,
	0xba, 0x3d, 0x79, 0x5f, 0xa7, 0xe2, 0xad, 0x4e, 0xc5, 0xaa, 0x4e, 0xc5, 0xba, 0x4e, 0xc5, 0xf2,
	0x73, 0x2c, 0xee, 0xae, 0x36, 0x8d, 0x12, 0xdb, 0x46, 0x89, 0x8f, 0x46, 0x89, 0x65, 0xab, 0xe4,
	0xa6, 0x55, 0x72, 0xdb, 0x2a, 0xf9, 0xd5, 0x2a, 0xb9, 0xfa, 0x56, 0xe2, 0x69, 0xd0, 0x7f, 0xf2,
	0x13, 0x00, 0x00, 0xff, 0xff, 0xdf, 0x3c, 0xf3, 0xc9, 0x3f, 0x01, 0x00, 0x00,
}
