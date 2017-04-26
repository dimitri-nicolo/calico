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
// source: k8s.io/metrics/pkg/apis/custom_metrics/v1alpha1/generated.proto
// DO NOT EDIT!

/*
	Package v1alpha1 is a generated protocol buffer package.

	It is generated from these files:
		k8s.io/metrics/pkg/apis/custom_metrics/v1alpha1/generated.proto

	It has these top-level messages:
		MetricValue
		MetricValueList
*/
package v1alpha1

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

import strings "strings"
import reflect "reflect"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.GoGoProtoPackageIsVersion1

func (m *MetricValue) Reset()                    { *m = MetricValue{} }
func (*MetricValue) ProtoMessage()               {}
func (*MetricValue) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{0} }

func (m *MetricValueList) Reset()                    { *m = MetricValueList{} }
func (*MetricValueList) ProtoMessage()               {}
func (*MetricValueList) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{1} }

func init() {
	proto.RegisterType((*MetricValue)(nil), "k8s.io.metrics.pkg.apis.custom_metrics.v1alpha1.MetricValue")
	proto.RegisterType((*MetricValueList)(nil), "k8s.io.metrics.pkg.apis.custom_metrics.v1alpha1.MetricValueList")
}
func (m *MetricValue) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *MetricValue) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintGenerated(data, i, uint64(m.DescribedObject.Size()))
	n1, err := m.DescribedObject.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n1
	data[i] = 0x12
	i++
	i = encodeVarintGenerated(data, i, uint64(len(m.MetricName)))
	i += copy(data[i:], m.MetricName)
	data[i] = 0x1a
	i++
	i = encodeVarintGenerated(data, i, uint64(m.Timestamp.Size()))
	n2, err := m.Timestamp.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n2
	if m.WindowSeconds != nil {
		data[i] = 0x20
		i++
		i = encodeVarintGenerated(data, i, uint64(*m.WindowSeconds))
	}
	data[i] = 0x2a
	i++
	i = encodeVarintGenerated(data, i, uint64(m.Value.Size()))
	n3, err := m.Value.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n3
	return i, nil
}

func (m *MetricValueList) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *MetricValueList) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintGenerated(data, i, uint64(m.ListMeta.Size()))
	n4, err := m.ListMeta.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n4
	if len(m.Items) > 0 {
		for _, msg := range m.Items {
			data[i] = 0x12
			i++
			i = encodeVarintGenerated(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func encodeFixed64Generated(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Generated(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintGenerated(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *MetricValue) Size() (n int) {
	var l int
	_ = l
	l = m.DescribedObject.Size()
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.MetricName)
	n += 1 + l + sovGenerated(uint64(l))
	l = m.Timestamp.Size()
	n += 1 + l + sovGenerated(uint64(l))
	if m.WindowSeconds != nil {
		n += 1 + sovGenerated(uint64(*m.WindowSeconds))
	}
	l = m.Value.Size()
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func (m *MetricValueList) Size() (n int) {
	var l int
	_ = l
	l = m.ListMeta.Size()
	n += 1 + l + sovGenerated(uint64(l))
	if len(m.Items) > 0 {
		for _, e := range m.Items {
			l = e.Size()
			n += 1 + l + sovGenerated(uint64(l))
		}
	}
	return n
}

func sovGenerated(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozGenerated(x uint64) (n int) {
	return sovGenerated(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *MetricValue) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&MetricValue{`,
		`DescribedObject:` + strings.Replace(strings.Replace(this.DescribedObject.String(), "ObjectReference", "k8s_io_client_go_pkg_api_v1.ObjectReference", 1), `&`, ``, 1) + `,`,
		`MetricName:` + fmt.Sprintf("%v", this.MetricName) + `,`,
		`Timestamp:` + strings.Replace(strings.Replace(this.Timestamp.String(), "Time", "k8s_io_apimachinery_pkg_apis_meta_v1.Time", 1), `&`, ``, 1) + `,`,
		`WindowSeconds:` + valueToStringGenerated(this.WindowSeconds) + `,`,
		`Value:` + strings.Replace(strings.Replace(this.Value.String(), "Quantity", "k8s_io_apimachinery_pkg_api_resource.Quantity", 1), `&`, ``, 1) + `,`,
		`}`,
	}, "")
	return s
}
func (this *MetricValueList) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&MetricValueList{`,
		`ListMeta:` + strings.Replace(strings.Replace(this.ListMeta.String(), "ListMeta", "k8s_io_apimachinery_pkg_apis_meta_v1.ListMeta", 1), `&`, ``, 1) + `,`,
		`Items:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.Items), "MetricValue", "MetricValue", 1), `&`, ``, 1) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringGenerated(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *MetricValue) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MetricValue: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MetricValue: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DescribedObject", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.DescribedObject.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MetricName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MetricName = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Timestamp.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field WindowSeconds", wireType)
			}
			var v int64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.WindowSeconds = &v
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Value.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MetricValueList) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MetricValueList: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MetricValueList: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ListMeta", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ListMeta.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Items", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Items = append(m.Items, MetricValue{})
			if err := m.Items[len(m.Items)-1].Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipGenerated(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthGenerated
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowGenerated
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipGenerated(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthGenerated = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenerated   = fmt.Errorf("proto: integer overflow")
)

var fileDescriptorGenerated = []byte{
	// 542 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x94, 0x93, 0x4f, 0x6f, 0xd3, 0x30,
	0x18, 0xc6, 0x9b, 0x95, 0x4a, 0xad, 0xab, 0xaa, 0x2c, 0x17, 0xa2, 0x1e, 0xd2, 0x6a, 0xa7, 0x6a,
	0x30, 0x5b, 0x2d, 0x08, 0x38, 0x20, 0x21, 0x45, 0x5c, 0x90, 0x28, 0x88, 0x6c, 0x02, 0x09, 0x90,
	0x26, 0xd7, 0x79, 0x97, 0x9a, 0x36, 0x71, 0x14, 0x3b, 0x9d, 0x76, 0xe3, 0x23, 0xf0, 0x05, 0xf8,
	0x3e, 0x3d, 0x4e, 0x9c, 0x38, 0x55, 0x34, 0x7c, 0x11, 0xe4, 0xfc, 0x59, 0xdb, 0x55, 0x1b, 0xec,
	0x16, 0xdb, 0xef, 0xf3, 0xf3, 0xf3, 0xbe, 0x8f, 0x83, 0x5e, 0x4e, 0x9f, 0x4b, 0xcc, 0x05, 0x09,
	0x40, 0xc5, 0x9c, 0x49, 0x12, 0x4d, 0x7d, 0x42, 0x23, 0x2e, 0x09, 0x4b, 0xa4, 0x12, 0xc1, 0x69,
	0xb9, 0x3f, 0x1f, 0xd0, 0x59, 0x34, 0xa1, 0x03, 0xe2, 0x43, 0x08, 0x31, 0x55, 0xe0, 0xe1, 0x28,
	0x16, 0x4a, 0x98, 0x24, 0x07, 0xe0, 0xa2, 0x10, 0x47, 0x53, 0x1f, 0x6b, 0x00, 0xde, 0x06, 0xe0,
	0x12, 0xd0, 0x39, 0xf2, 0xb9, 0x9a, 0x24, 0x63, 0xcc, 0x44, 0x40, 0x7c, 0xe1, 0x0b, 0x92, 0x71,
	0xc6, 0xc9, 0x59, 0xb6, 0xca, 0x16, 0xd9, 0x57, 0xce, 0xef, 0x3c, 0x29, 0x0c, 0xd2, 0x88, 0x07,
	0x94, 0x4d, 0x78, 0x08, 0xf1, 0x45, 0xe9, 0x92, 0xc4, 0x20, 0x45, 0x12, 0x33, 0xb8, 0xee, 0xea,
	0x56, 0x95, 0xd4, 0xcd, 0x52, 0x32, 0xdf, 0xe9, 0xa5, 0x43, 0x6e, 0x52, 0xc5, 0x49, 0xa8, 0x78,
	0xb0, 0x7b, 0xcd, 0xd3, 0x7f, 0x09, 0x24, 0x9b, 0x40, 0x40, 0x77, 0x74, 0x8f, 0x6f, 0xd2, 0x25,
	0x8a, 0xcf, 0x08, 0x0f, 0x95, 0x54, 0xf1, 0x8e, 0xe8, 0x61, 0x21, 0x62, 0x33, 0x0e, 0xa1, 0x3a,
	0xd2, 0x93, 0x2b, 0xc6, 0xb0, 0xdb, 0xca, 0xc1, 0x8f, 0x2a, 0x6a, 0x8e, 0xb2, 0xd1, 0x7f, 0xa0,
	0xb3, 0x04, 0x4c, 0x81, 0xda, 0x1e, 0x48, 0x16, 0xf3, 0x31, 0x78, 0xef, 0xc6, 0x5f, 0x81, 0x29,
	0xcb, 0xe8, 0x19, 0xfd, 0xe6, 0xf0, 0x11, 0x2e, 0x02, 0xcc, 0xb1, 0xa7, 0x7a, 0xf0, 0x79, 0x84,
	0x78, 0x3e, 0xc0, 0x79, 0xa9, 0x0b, 0x67, 0x10, 0x43, 0xc8, 0xc0, 0x79, 0xb0, 0x58, 0x76, 0x2b,
	0xe9, 0xb2, 0xdb, 0x7e, 0xb5, 0x0d, 0x73, 0xaf, 0xd3, 0xcd, 0x21, 0x42, 0x79, 0xf4, 0x6f, 0x69,
	0x00, 0xd6, 0x5e, 0xcf, 0xe8, 0x37, 0x1c, 0xb3, 0x50, 0xa3, 0xd1, 0xd5, 0x89, 0xbb, 0x51, 0x65,
	0x7e, 0x46, 0x0d, 0x3d, 0x35, 0xa9, 0x68, 0x10, 0x59, 0xd5, 0xcc, 0xde, 0x61, 0x69, 0x6f, 0x73,
	0x54, 0xeb, 0x47, 0xa6, 0x93, 0xd4, 0x3e, 0x4f, 0x78, 0x00, 0xce, 0x7e, 0x81, 0x6f, 0x9c, 0x94,
	0x10, 0x77, 0xcd, 0x33, 0x9f, 0xa1, 0xd6, 0x39, 0x0f, 0x3d, 0x71, 0x7e, 0x0c, 0x4c, 0x84, 0x9e,
	0xb4, 0xee, 0xf5, 0x8c, 0x7e, 0xd5, 0xd9, 0x4f, 0x97, 0xdd, 0xd6, 0xc7, 0xcd, 0x03, 0x77, 0xbb,
	0xce, 0x3c, 0x46, 0xb5, 0xb9, 0x9e, 0xa1, 0x55, 0xcb, 0x1c, 0xe1, 0xdb, 0x1c, 0xe1, 0xf2, 0x45,
	0xe2, 0xf7, 0x09, 0x0d, 0x15, 0x57, 0x17, 0x4e, 0xab, 0x70, 0x55, 0xcb, 0x82, 0x70, 0x73, 0xd6,
	0xc1, 0x4f, 0x03, 0xb5, 0x37, 0xf2, 0x79, 0xc3, 0xa5, 0x32, 0xbf, 0xa0, 0xba, 0xee, 0xc7, 0xa3,
	0x8a, 0x16, 0xe1, 0xe0, 0xff, 0xeb, 0x5e, 0xab, 0x47, 0xa0, 0xa8, 0x73, 0xbf, 0xb8, 0xab, 0x5e,
	0xee, 0xb8, 0x57, 0x44, 0x93, 0xa2, 0x1a, 0x57, 0x10, 0x48, 0x6b, 0xaf, 0x57, 0xed, 0x37, 0x87,
	0x2f, 0xf0, 0x1d, 0x7f, 0x5c, 0xbc, 0x61, 0x77, 0xdd, 0xd4, 0x6b, 0x8d, 0x74, 0x73, 0xb2, 0x73,
	0xb8, 0x58, 0xd9, 0x95, 0xcb, 0x95, 0x5d, 0xf9, 0xb5, 0xb2, 0x2b, 0xdf, 0x52, 0xdb, 0x58, 0xa4,
	0xb6, 0x71, 0x99, 0xda, 0xc6, 0xef, 0xd4, 0x36, 0xbe, 0xff, 0xb1, 0x2b, 0x9f, 0xea, 0x25, 0xed,
	0x6f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xde, 0x29, 0xfe, 0x9a, 0x79, 0x04, 0x00, 0x00,
}
