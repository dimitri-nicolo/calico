// Code generated by protoc-gen-gogo.
// source: enumprefix.proto
// DO NOT EDIT!

/*
	Package enumprefix is a generated protocol buffer package.

	It is generated from these files:
		enumprefix.proto

	It has these top-level messages:
		MyMessage
*/
package enumprefix

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import test "github.com/gogo/protobuf/test"
import _ "github.com/gogo/protobuf/gogoproto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type MyMessage struct {
	TheField         test.TheTestEnum `protobuf:"varint,1,opt,name=TheField,json=theField,enum=test.TheTestEnum" json:"TheField"`
	XXX_unrecognized []byte           `json:"-"`
}

func (m *MyMessage) Reset()                    { *m = MyMessage{} }
func (m *MyMessage) String() string            { return proto.CompactTextString(m) }
func (*MyMessage) ProtoMessage()               {}
func (*MyMessage) Descriptor() ([]byte, []int) { return fileDescriptorEnumprefix, []int{0} }

func (m *MyMessage) GetTheField() test.TheTestEnum {
	if m != nil {
		return m.TheField
	}
	return test.A
}

func init() {
	proto.RegisterType((*MyMessage)(nil), "enumprefix.MyMessage")
}

func init() { proto.RegisterFile("enumprefix.proto", fileDescriptorEnumprefix) }

var fileDescriptorEnumprefix = []byte{
	// 151 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x12, 0x48, 0xcd, 0x2b, 0xcd,
	0x2d, 0x28, 0x4a, 0x4d, 0xcb, 0xac, 0xd0, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x42, 0x88,
	0x48, 0x69, 0xa7, 0x67, 0x96, 0x64, 0x94, 0x26, 0xe9, 0x25, 0xe7, 0xe7, 0xea, 0xa7, 0xe7, 0xa7,
	0xe7, 0xeb, 0x83, 0x95, 0x24, 0x95, 0xa6, 0xe9, 0x97, 0xa4, 0x16, 0x97, 0xe8, 0x97, 0x64, 0xa4,
	0x82, 0x68, 0x88, 0x46, 0x29, 0x5d, 0x9c, 0x8a, 0x41, 0x3c, 0x30, 0x07, 0xcc, 0x82, 0x28, 0x57,
	0x72, 0xe0, 0xe2, 0xf4, 0xad, 0xf4, 0x4d, 0x2d, 0x2e, 0x4e, 0x4c, 0x4f, 0x15, 0x32, 0xe6, 0xe2,
	0x08, 0xc9, 0x48, 0x75, 0xcb, 0x4c, 0xcd, 0x49, 0x91, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x33, 0x12,
	0xd4, 0x03, 0x1b, 0x1d, 0x92, 0x91, 0x1a, 0x92, 0x5a, 0x5c, 0xe2, 0x9a, 0x57, 0x9a, 0xeb, 0xc4,
	0x72, 0xe2, 0x9e, 0x3c, 0x43, 0x10, 0x47, 0x09, 0x54, 0x21, 0x20, 0x00, 0x00, 0xff, 0xff, 0x8c,
	0xb3, 0x4a, 0xfa, 0xbc, 0x00, 0x00, 0x00,
}
