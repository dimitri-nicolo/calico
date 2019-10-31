// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calc_test

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"net"

	. "github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var _ = DescribeTable("Check Inserting CIDR and compare with network set names",
	func(key model.NetworkSetKey, netset *model.NetworkSet) {
		it := NewIpTrie()
		c := key.Name
		ed := &EndpointData{
			Key:        key,
			Networkset: netset,
		}

		for _, cidr := range netset.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertNetworkset(cidrb, ed)
		}
		for _, cidr := range netset.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			eds, ok := it.GetNetworksets(cidrb)
			for _, edl := range eds {
				Expect(ok).To(Equal(true))
				Expect(c).To(Equal(edl.Key.(model.NetworkSetKey).Name))
				//log.Infof("Test1: C:%s string:%s\n", c, edl.Key.(model.NetworkSetKey).Name)
			}
		}
	},
	Entry("Insert network CIDR and match with ns name", netSet1Key, &netSet1),
	Entry("Insert network CIDR and match with ns name", netSet2Key, &netSet2),
)

var _ = DescribeTable("Insert and Delete CIDRs and compare with network set names",
	func(key model.NetworkSetKey, netset *model.NetworkSet, key1 model.NetworkSetKey, netset1 *model.NetworkSet) {
		it := NewIpTrie()
		c := key.Name
		ed := &EndpointData{
			Key:        key,
			Networkset: netset,
		}
		ed1 := &EndpointData{
			Key:        key1,
			Networkset: netset1,
		}

		for _, cidr := range netset1.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertNetworkset(cidrb, ed1)
		}
		for _, cidr := range netset.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertNetworkset(cidrb, ed)
		}
		for _, cidr := range netset1.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.DeleteNetworkset(cidrb, key1)
		}
		for _, cidr := range netset.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			eds, ok := it.GetNetworksets(cidrb)
			for _, edl := range eds {
				Expect(ok).To(Equal(true))
				Expect(c).To(Equal(edl.Key.(model.NetworkSetKey).Name))
				//log.Infof("Test2: C:%s string:%s\n", c, edl.Key.(model.NetworkSetKey).Name)
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
		ed1 := &EndpointData{
			Key:        key1,
			Networkset: netset1,
		}
		ed2 := &EndpointData{
			Key:        key2,
			Networkset: netset2,
		}

		for _, cidr := range netset1.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertNetworkset(cidrb, ed1)
		}
		for _, cidr := range netset2.Nets {
			cidrb := ip.CIDRFromCalicoNet(cidr)
			it.InsertNetworkset(cidrb, ed2)
		}

		edl, ok := it.GetLongestPrefixCidr(ipaddr)
		Expect(ok).To(Equal(true))
		Expect(edl.Key.(model.NetworkSetKey).Name).To(Equal(res.Name))
		//log.Infof("Test3: string:%s res:%s\n", edl.Key.(model.NetworkSetKey).Name, res.Name)
	},
	Entry("Longest Prefix Match find ns name", netSet1Key, netSet3Key, &netSet1, &netSet3, netset3Ip1a, netSet1Key),
	Entry("Longest Prefix Match find ns name", netSet1Key, netSet3Key, &netSet1, &netSet3, netset3Ip1b, netSet3Key),
)
