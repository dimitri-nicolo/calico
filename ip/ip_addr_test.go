// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.
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

package ip_test

import (
	. "github.com/projectcalico/felix/ip"

	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("IpAddr",
	func(version int, inputIP, canonical string, bytes []byte, binvalue string) {
		ip := FromString(inputIP)
		Expect([]byte(ip.AsNetIP())).To(Equal(bytes))
		Expect(ip.String()).To(Equal(canonical))
		Expect(int(ip.Version())).To(Equal(version))
		Expect(MustParseCIDROrIP(inputIP).Addr().String()).To(Equal(canonical))
		Expect(ip.AsBinary()).To(Equal(binvalue))
	},
	Entry("IPv4", 4, "10.0.0.1", "10.0.0.1", []byte{0xa, 0, 0, 1}, "00001010000000000000000000000001"),
	Entry("IPv6", 6, "dead::beef", "dead::beef", []byte{
		0xde, 0xad, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0xbe, 0xef},
		"11011110101011010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001011111011101111",
	),
	Entry("IPv6 non-canon", 6, "deaf:0:0::beef", "deaf::beef", []byte{
		0xde, 0xaf, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0xbe, 0xef},
		"11011110101011110000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001011111011101111",
	),
)

var _ = DescribeTable("CIDR",
	func(version int, inputCIDR, canonical string, bytes []byte, len int, binvalue string) {
		cidr := MustParseCIDROrIP(inputCIDR)
		Expect([]byte(cidr.Addr().AsNetIP())).To(Equal(bytes))
		Expect(int(cidr.Prefix())).To(Equal(len))
		Expect(cidr.String()).To(Equal(canonical))
		Expect(int(cidr.Version())).To(Equal(version))
		Expect(cidr.AsBinary()).To(Equal(binvalue))
	},
	Entry("IPv4", 4, "10.0.0.0/16", "10.0.0.0/16", []byte{0xa, 0, 0, 0}, 16, "0000101000000000"),
	Entry("IPv4 should be masked", 4, "10.0.0.1/16", "10.0.0.0/16", []byte{0xa, 0, 0, 0}, 16, "0000101000000000"),
	Entry("IPv6", 6, "dead::/16", "dead::/16", []byte{
		0xde, 0xad, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0},
		16,
		"1101111010101101",
	),
	Entry("IPv6 non-canon", 6, "dead:0:0::beef/16", "dead::/16", []byte{
		0xde, 0xad, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 00},
		16,
		"1101111010101101",
	),
)
