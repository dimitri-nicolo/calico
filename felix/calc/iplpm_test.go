// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package calc_test

import (
	"net"

	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	. "github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

var _ = DescribeTable("Check Inserting CIDR and compare with network set names",
	func(key model.NetworkSetKey, netset *model.NetworkSet) {
		it := NewIpTrie()

		for _, cidr := range netset.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertKey(cidrb, key)
		}
		for _, cidr := range netset.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			keys, ok := it.GetKeys(cidrb)
			for _, ekey := range keys {
				Expect(ok).To(Equal(true))
				Expect(ekey).To(Equal(key))
			}
		}
	},
	Entry("Insert network CIDR and match with ns name", netSet1Key, &netSet1),
	Entry("Insert network CIDR and match with ns name", netSet2Key, &netSet2),
)

var _ = DescribeTable("Insert and Delete CIDRs and compare with network set names",
	func(key model.NetworkSetKey, netset *model.NetworkSet, key1 model.NetworkSetKey, netset1 *model.NetworkSet) {
		it := NewIpTrie()

		for _, cidr := range netset1.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertKey(cidrb, key1)
		}
		for _, cidr := range netset.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertKey(cidrb, key)
		}
		for _, cidr := range netset1.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.DeleteKey(cidrb, key1)
		}
		for _, cidr := range netset.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			keys, ok := it.GetKeys(cidrb)
			for _, ekey := range keys {
				Expect(ok).To(Equal(true))
				Expect(ekey).To(Equal(key))
			}
		}
	},
	Entry("Insert network CIDR and match with ns name", netSet1Key, &netSet1, netSet2Key, &netSet2),
	Entry("Insert network CIDR and match with ns name", netSet2Key, &netSet2, netSet1Key, &netSet1),
)

var _ = DescribeTable("Test by finding Longest Prefix Match CIDR's name for given IP Address",
	func(key1 model.NetworkSetKey, key2 model.NetworkSetKey, netset1 *model.NetworkSet, netset2 *model.NetworkSet, ipAddr net.IP, res model.NetworkSetKey) {
		it := NewIpTrie()
		ipaddr := ip.FromNetIP(ipAddr)

		for _, cidr := range netset1.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertKey(cidrb, key1)
		}
		for _, cidr := range netset2.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertKey(cidrb, key2)
		}

		key, ok := it.GetLongestPrefixCidr(ipaddr)
		Expect(ok).To(Equal(true))
		Expect(key).To(Equal(res))
	},
	Entry("Longest Prefix Match find ns name", netSet1Key, netSet3Key, &netSet1, &netSet3, netset3Ip1a, netSet1Key),
	Entry("Longest Prefix Match find ns name", netSet1Key, netSet3Key, &netSet1, &netSet3, netset3Ip1b, netSet3Key),
)
