// Code generated by protoc-gen-go.
// source: google/protobuf/empty.proto
// DO NOT EDIT!

package google_protobuf

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// A generic empty message that you can re-use to avoid defining duplicated
// empty messages in your APIs. A typical example is to use it as the request
// or the response type of an API method. For instance:
//
//     service Foo {
//       rpc Bar(google.protobuf.Empty) returns (google.protobuf.Empty);
//     }
//
// The JSON representation for `Empty` is empty JSON object `{}`.
type Empty struct {
}

func (m *Empty) Reset()                    { *m = Empty{} }
func (m *Empty) String() string            { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()               {}
func (*Empty) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{0} }
func (*Empty) XXX_WellKnownType() string   { return "Empty" }

func init() {
	proto.RegisterType((*Empty)(nil), "google.protobuf.Empty")
}

func init() { proto.RegisterFile("google/protobuf/empty.proto", fileDescriptor3) }

var fileDescriptor3 = []byte{
	// 124 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x92, 0x4e, 0xcf, 0xcf, 0x4f,
	0xcf, 0x49, 0xd5, 0x2f, 0x28, 0xca, 0x2f, 0xc9, 0x4f, 0x2a, 0x4d, 0xd3, 0x4f, 0xcd, 0x2d, 0x28,
	0xa9, 0xd4, 0x03, 0x73, 0x85, 0xf8, 0x21, 0x92, 0x7a, 0x30, 0x49, 0x25, 0x76, 0x2e, 0x56, 0x57,
	0x90, 0xbc, 0x53, 0x00, 0x97, 0x70, 0x72, 0x7e, 0xae, 0x1e, 0x9a, 0xbc, 0x13, 0x17, 0x58, 0x36,
	0x00, 0xc4, 0x0d, 0x60, 0x5c, 0xc0, 0xc8, 0xf8, 0x83, 0x91, 0x71, 0x11, 0x13, 0xb3, 0x7b, 0x80,
	0xd3, 0x2a, 0x26, 0x39, 0x77, 0x88, 0xda, 0x00, 0xa8, 0x5a, 0xbd, 0xf0, 0xd4, 0x9c, 0x1c, 0xef,
	0xbc, 0xfc, 0xf2, 0xbc, 0x90, 0xca, 0x82, 0xd4, 0xe2, 0x24, 0x36, 0xb0, 0x21, 0xc6, 0x80, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xac, 0xca, 0x5b, 0xd0, 0x91, 0x00, 0x00, 0x00,
}
