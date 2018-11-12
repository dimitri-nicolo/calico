// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

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

package ipam

import (
	"context"
	//"errors"
	"fmt"
	"net"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/backend"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

// Implement an IP pools accessor for the IPAM client.  This is a "mock" version
// of the accessor that we populate directly, rather than requiring the pool
// data to be persisted in etcd.
type ipPoolAccessorWindows struct {
	pools map[string]poolWindows
}

type poolWindows struct {
	cidr      string
	blockSize int
	enabled   bool
}

func (i *ipPoolAccessorWindows) GetEnabledPools(ipVersion int) ([]v3.IPPool, error) {
	sorted := make([]string, 0)
	// Get a sorted list of enabled pool CIDR strings.
	for p, e := range i.pools {
		if e.enabled {
			sorted = append(sorted, p)
		}
	}
	sort.Strings(sorted)

	// Convert to IPNets and sort out the correct IP versions.  Sorting the results
	// mimics more closely the behavior of etcd and allows the tests to be
	// deterministic.
	pools := make([]v3.IPPool, 0)
	for _, p := range sorted {
		c := cnet.MustParseCIDR(p)
		if c.Version() == ipVersion {
			pool := v3.IPPool{Spec: v3.IPPoolSpec{CIDR: p}}
			if ipVersion == 4 {
				pool.Spec.BlockSize = 26
			} else {
				pool.Spec.BlockSize = 122
			}
			pools = append(pools, pool)
		}
	}

	log.Infof("GetEnabledPools returns: %s", pools)

	return pools, nil
}

var (
	ipPoolsWindows = &ipPoolAccessorWindows{pools: map[string]poolWindows{}}
)

type testArgsClaimAff1 struct {
	inNet, host                 string
	cleanEnv                    bool
	pool                        []string
	assignIP                    net.IP
	expClaimedIPs, expFailedIPs int
	expError                    error
}

var _ = testutils.E2eDatastoreDescribe("Windows: IPAM tests", testutils.DatastoreEtcdV3, func(config apiconfig.CalicoAPIConfig) {
	// Create a new backend client and an IPAM Client using the IP Pools Accessor.
	// Tests that need to ensure a clean datastore should invokke Clean() on the datastore at the start of the
	// tests.
	bc, err := backend.NewClient(config)
	if err != nil {
		panic(err)
	}
	ic := NewIPAMClient(bc, ipPoolsWindows)

        //Request for 256 IPs from a pool, say "10.0.0.0/24", with a blocksize of 26, allocates only 240 IPs as
        //the pool of 256 IPs is splitted into 4 blocks of 64 IPs each and 4 IPs, i.e,
        //gateway IP, the first IP of the block, the second IP of the block and the broadcast IP are reserved.
        //So 256 - (4 * 4) = 240.

	//Test Case: Reserved IPs should not be allocated
	//           test case is only written for IPv4
	DescribeTable("Windows: IPAM AutoAssign should not assign reserved IPs",
		func(host string, pool []poolWindows, usePool string, inv4 int, expv4 int, expError error) {
			bc.Clean()
			deleteAllPoolsWindows()

			for _, v := range pool {
				ipPoolsWindows.pools[v.cidr] = poolWindows{cidr: v.cidr, enabled: v.enabled, blockSize: v.blockSize}
			}

			fromPool := cnet.MustParseNetwork(usePool)
			args := AutoAssignArgs{
				Num4:      inv4,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{fromPool},
			}

			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			outv4, _, outErr := ic.AutoAssign(ctx, args)
			if expError != nil {
				Expect(outErr).To(HaveOccurred())
			} else {
				Expect(outErr).ToNot(HaveOccurred())
			}
			Expect(len(outv4)).To(Equal(expv4))

			for _, ip := range outv4 {
				// for a block size of 64 as per unit test
				if int(ip.IP[3]) > -1 && int(ip.IP[3]) < 64 {
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.0"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.1"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.2"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.63"))
				} else if int(ip.IP[3]) > 63 && int(ip.IP[3]) < 128 {
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.64"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.65"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.66"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.127"))
				} else if int(ip.IP[3]) > 127 && int(ip.IP[3]) < 192 {
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.128"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.129"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.130"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.191"))
				} else if int(ip.IP[3]) > 191 && int(ip.IP[3]) < 256 {
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.192"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.193"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.194"))
					Expect(ip.IP.String()).ToNot(Equal("10.0.0.255"))
				}
			}
		},

		// Test 1: AutoAssign 256 IPv4 - expect NOT to assign 10.0.0.0, 10.0.0.1, 10.0.0.2, 10.0.0.63,
		//	   					      10.0.0.64, 10.0.0.65, 10.0.0.66, 10.0.0.127,
		//                                                    10.0.0.128, 10.0.0.129, 10.0.0.130, 10.0.0.191,
		//                                                    10.0.0.192, 10.0.0.193, 10.0.0.194, 10.0.0.255 IPs.
		Entry("256 v4 ", "testHost", []poolWindows{{"10.0.0.0/24", 26, true}}, "10.0.0.0/24", 256, 240, nil),
	)

	// We're assigning one IP which should be from the only ipPool created at the time, second one
	// should be from the same /26 block since they're both from the same host, then delete
	// the ipPool and create a new ipPool, and AutoAssign 1 more IP for the same host - expect the
	// assigned IP to be from the new ipPool that was created, this is to make sure the assigned IP
	// doesn't come from the old affinedBlock even after the ipPool was deleted.
	Describe("Windows: IPAM AutoAssign from the default pool then delete the pool and assign again", func() {
		hostA := "host-A"
		hostB := "host-B"
		pool1 := cnet.MustParseNetwork("10.0.0.0/24")
		pool2 := cnet.MustParseNetwork("20.0.0.0/24")
		var block cnet.IPNet

		Context("Windows: AutoAssign a single IP without specifying a pool", func() {
			bc.Clean()
			deleteAllPoolsWindows()

			It("Windows: should auto-assign from the only available pool", func() {
				bc.Clean()
				deleteAllPoolsWindows()
				applyPoolWindows("10.0.0.0/24", true)

				args := AutoAssignArgs{
					Num4:     1,
					Num6:     0,
					Hostname: hostA,
				}
				ctx := context.WithValue(context.Background(), "windowsHost", "windows")
				v4, _, outErr := ic.AutoAssign(ctx, args)
				blocks := getAffineBlocksWindows(bc, hostA)
				for _, b := range blocks {
					if pool1.Contains(b.IPNet.IP) {
						block = b
					}
				}
				Expect(outErr).NotTo(HaveOccurred())
				Expect(pool1.IPNet.Contains(v4[0].IP)).To(BeTrue())
			})

			It("Windows: should auto-assign another IP from the same pool into the same allocation block", func() {
				args := AutoAssignArgs{
					Num4:     1,
					Num6:     0,
					Hostname: hostA,
				}

				ctx := context.WithValue(context.Background(), "windowsHost", "windows")
				v4, _, outErr := ic.AutoAssign(ctx, args)
				Expect(outErr).NotTo(HaveOccurred())
				Expect(block.IPNet.Contains(v4[0].IP)).To(BeTrue())
			})

			It("Windows: should assign from a new pool for a new host (old pool is removed)", func() {
				deleteAllPoolsWindows()
				applyPoolWindows("20.0.0.0/24", true)

				p, _ := ipPoolsWindows.GetEnabledPools(4)
				Expect(len(p)).To(Equal(1))
				Expect(p[0].Spec.CIDR).To(Equal(pool2.String()))
				p, _ = ipPoolsWindows.GetEnabledPools(6)
				Expect(len(p)).To(BeZero())

				args := AutoAssignArgs{
					Num4:     1,
					Num6:     0,
					Hostname: hostB,
				}

				ctx := context.WithValue(context.Background(), "windowsHost", "windows")
				v4, _, outErr := ic.AutoAssign(ctx, args)
				Expect(outErr).NotTo(HaveOccurred())
				Expect(pool2.IPNet.Contains(v4[0].IP)).To(BeTrue())
			})

			It("Windows: should not assign from an existing affine block for the first host since the pool is removed)", func() {
				args := AutoAssignArgs{
					Num4:     1,
					Num6:     0,
					Hostname: hostA,
				}

				ctx := context.WithValue(context.Background(), "windowsHost", "windows")
				v4, _, outErr := ic.AutoAssign(ctx, args)
				Expect(outErr).NotTo(HaveOccurred())
				Expect(pool2.IPNet.Contains(v4[0].IP)).To(BeTrue())
			})

		})
	})

	Describe("Windows: IPAM AutoAssign from any pool", func() {
		// Assign an IP address, don't pass a pool, make sure we can get an
		// address.
		args := AutoAssignArgs{
			Num4:     1,
			Num6:     0,
			Hostname: "test-host",
		}
		// Call once in order to assign an IP address and create a block.
		It("Windows: should have assigned an IP address with no error", func() {
			deleteAllPoolsWindows()
			applyPoolWindows("10.0.0.0/24", true)
			applyPoolWindows("20.0.0.0/24", true)

			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			Expect(outErr).NotTo(HaveOccurred())
			Expect(len(v4) == 1).To(BeTrue())
		})

		// Call again to trigger an assignment from the newly created block.
		It("Windows: should have assigned an IP address with no error", func() {
			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			Expect(outErr).NotTo(HaveOccurred())
			Expect(len(v4)).To(Equal(1))
		})

	})

	Describe("Windows: IPAM AutoAssign from different pools", func() {
		host := "host-A"
		pool1 := cnet.MustParseNetwork("10.0.0.0/24")
		pool2 := cnet.MustParseNetwork("20.0.0.0/24")
		var block1, block2 cnet.IPNet

		It("Windows: Should get an IP from pool1 when explicitly requesting from that pool", func() {
			bc.Clean()
			deleteAllPoolsWindows()
			applyPoolWindows("10.0.0.0/24", true)
			applyPoolWindows("20.0.0.0/24", true)

			args := AutoAssignArgs{
				Num4:      1,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool1},
			}

			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			blocks := getAffineBlocksWindows(bc, host)
			for _, b := range blocks {
				if pool1.Contains(b.IPNet.IP) {
					block1 = b
				}
			}

			Expect(outErr).NotTo(HaveOccurred())
			Expect(pool1.IPNet.Contains(v4[0].IP)).To(BeTrue())
		})

		It("Windows: Should get an IP from pool2 when explicitly requesting from that pool", func() {
			args := AutoAssignArgs{
				Num4:      1,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool2},
			}

			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			blocks := getAffineBlocksWindows(bc, host)
			for _, b := range blocks {
				if pool2.Contains(b.IPNet.IP) {
					block2 = b
				}
			}

			Expect(outErr).NotTo(HaveOccurred())
			Expect(block2.IPNet.Contains(v4[0].IP)).To(BeTrue())
		})

		It("Windows: Should get an IP from pool1 in the same allocation block as the first IP from pool1", func() {
			args := AutoAssignArgs{
				Num4:      1,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool1},
			}

			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			Expect(outErr).NotTo(HaveOccurred())
			Expect(block1.IPNet.Contains(v4[0].IP)).To(BeTrue())
		})

		It("Windows: Should get an IP from pool2 in the same allocation block as the first IP from pool2", func() {
			args := AutoAssignArgs{
				Num4:      1,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool2},
			}

			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			Expect(outErr).NotTo(HaveOccurred())
			Expect(block2.IPNet.Contains(v4[0].IP)).To(BeTrue())
		})

		It("Windows: Should have strict IP pool affinity", func() {
			// Assign the rest of the addresses in pool2.
			// A /24 has 256 addresses and block size is 26 so we would have 16 reserved ips and We've assigned 2 already, so assign (256-18) 238 more.
			args := AutoAssignArgs{
				Num4:      238,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool2},
			}

			By("Windows: Allocating the rest of the IPs in the pool", func() {
				ctx := context.WithValue(context.Background(), "windowsHost", "windows")
				v4, _, outErr := ic.AutoAssign(ctx, args)
				Expect(outErr).NotTo(HaveOccurred())
				Expect(len(v4)).To(Equal(238))

				// Expect all the IPs to be in pool2.
				for _, a := range v4 {
					Expect(pool2.IPNet.Contains(a.IP)).To(BeTrue(), fmt.Sprintf("%s not in pool %s", a.IP, pool2))
				}
			})

			By("Windows: Attempting to allocate an IP when there are no more left in the pool", func() {
				args.Num4 = 1
				ctx := context.WithValue(context.Background(), "windowsHost", "windows")
				v4, _, outErr := ic.AutoAssign(ctx, args)

				Expect(outErr).NotTo(HaveOccurred())
				Expect(len(v4)).To(Equal(0))
			})
		})

	})

	Describe("Windows: IPAM AutoAssign from different pools - multi", func() {
		host := "host-A"
		pool1 := cnet.MustParseNetwork("10.0.0.0/24")
		pool2 := cnet.MustParseNetwork("20.0.0.0/24")
		pool3 := cnet.MustParseNetwork("30.0.0.0/24")
		pool4_v6 := cnet.MustParseNetwork("fe80::11/120")
		pool5_doesnot_exist := cnet.MustParseNetwork("40.0.0.0/24")

		It("Windows: should fail to AutoAssign 1 IPv4 when requesting a disabled IPv4 in the list of requested pools", func() {
			args := AutoAssignArgs{
				Num4:      1,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool1, pool3},
			}
			bc.Clean()
			deleteAllPoolsWindows()
			applyPoolWindows(pool1.String(), true)
			applyPoolWindows(pool2.String(), true)
			applyPoolWindows(pool3.String(), false)
			applyPoolWindows(pool4_v6.String(), true)
			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			_, _, outErr := ic.AutoAssign(ctx, args)
			Expect(outErr).To(HaveOccurred())
		})

		It("Windows: should fail to AutoAssign when specifying an IPv6 pool in the IPv4 requested pools", func() {
			args := AutoAssignArgs{
				Num4:      0,
				Num6:      1,
				Hostname:  host,
				IPv6Pools: []cnet.IPNet{pool4_v6, pool1},
			}
			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			_, _, outErr := ic.AutoAssign(ctx, args)
			Expect(outErr).To(HaveOccurred())
		})

		It("Windows: should allocate an IP from the first requested pool when two valid pools are requested", func() {
			args := AutoAssignArgs{
				Num4:      1,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool1, pool2},
			}
			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			log.Println("IPAM returned: %v", v4)

			Expect(outErr).NotTo(HaveOccurred())
			Expect(len(v4)).To(Equal(1))
			Expect(pool1.Contains(v4[0].IP)).To(BeTrue())
		})

		It("Windows: should allocate 300 IP addresses from two enabled pools that contain sufficient addresses", func() {
			args := AutoAssignArgs{
				Num4:      300,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool1, pool2},
			}
			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			log.Println("v4: %d IPs", len(v4))

			Expect(outErr).NotTo(HaveOccurred())
			Expect(len(v4)).To(Equal(300))
		})

		It("Windows: should fail to allocate another 300 IP addresses from the same pools due to lack of addresses (partial allocation)", func() {
			args := AutoAssignArgs{
				Num4:      300,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool1, pool2},
			}
			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, outErr := ic.AutoAssign(ctx, args)
			log.Println("v4: %d IPs", len(v4))

			// Expect 179 entries since we have a total of 512, out of which 4*4=16(from each pool, that means 32 reserved ips from both the pools) are reserved and we requested 1 + 300 already.
			Expect(outErr).NotTo(HaveOccurred())
			Expect(v4).To(HaveLen(179))
		})

		It("Windows: should fail to allocate any address when requesting an invalid pool and a valid pool", func() {
			args := AutoAssignArgs{
				Num4:      1,
				Num6:      0,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{pool1, pool5_doesnot_exist},
			}
			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			v4, _, err := ic.AutoAssign(ctx, args)
			log.Println("v4: %d IPs", len(v4))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("the given pool (40.0.0.0/24) does not exist, or is not enabled"))
			Expect(len(v4)).To(Equal(0))
		})

	})

	DescribeTable("Windows: AutoAssign: requested IPs vs returned IPs",
		func(host string, cleanEnv bool, pool []poolWindows, usePool string, inv4, inv6, expv4, expv6 int, expError error) {
			if cleanEnv {
				bc.Clean()
				deleteAllPoolsWindows()
			}
			for _, v := range pool {
				ipPoolsWindows.pools[v.cidr] = poolWindows{cidr: v.cidr, enabled: v.enabled, blockSize: v.blockSize}
			}

			fromPool := cnet.MustParseNetwork(usePool)
			args := AutoAssignArgs{
				Num4:      inv4,
				Num6:      inv6,
				Hostname:  host,
				IPv4Pools: []cnet.IPNet{fromPool},
			}

			ctx := context.WithValue(context.Background(), "windowsHost", "windows")
			outv4, outv6, outErr := ic.AutoAssign(ctx, args)
			if expError != nil {
				Expect(outErr).To(HaveOccurred())
			} else {
				Expect(outErr).ToNot(HaveOccurred())
			}
			Expect(outv4).To(HaveLen(expv4))
			Expect(outv6).To(HaveLen(expv6))
		},


		// Test 1: AutoAssign 256 IPv4, 256 IPv6 - expect 240 IPv4 + IPv6 addresses.
		Entry("256 v4 256 v6", "testHost", true, []poolWindows{{"192.168.1.0/24", 26, true}, {"fd80:24e2:f998:72d6::/120", 128, true}}, "192.168.1.0/24", 256, 256, 240, 240, nil),

		// Test 2: AutoAssign 257 IPv4, 0 IPv6 - expect 240 IPv4 addresses, no IPv6, and no error.
		Entry("257 v4 0 v6", "testHost", true, []poolWindows{{"192.168.1.0/24", 26, true}, {"fd80:24e2:f998:72d6::/120", 128, true}}, "192.168.1.0/24", 257, 0, 240, 0, nil),

		// Test 3: AutoAssign 0 IPv4, 257 IPv6 - expect 240 IPv6 addresses, no IPv4, and no error.
		Entry("0 v4 257 v6", "testHost", true, []poolWindows{{"192.168.1.0/24", 26, true}, {"fd80:24e2:f998:72d6::/120", 128, true}}, "192.168.1.0/24", 0, 257, 0, 240, nil),

	)

})

// getAffineBlocksWindows gets all the blocks affined to the host passed in.
func getAffineBlocksWindows(backend bapi.Client, host string) []cnet.IPNet {
	opts := model.BlockAffinityListOptions{Host: host, IPVersion: 4}
	datastoreObjs, err := backend.List(context.Background(), opts, "")
	if err != nil {
		if _, ok := err.(cerrors.ErrorResourceDoesNotExist); ok {
			log.Printf("Windows: No affined blocks found")
		} else {
			Expect(err).NotTo(HaveOccurred(), "Windows: Error getting affine blocks: %v", err)
		}
	}

	// Iterate through and extract the block CIDRs.
	blocks := []cnet.IPNet{}
	for _, o := range datastoreObjs.KVPairs {
		k := o.Key.(model.BlockAffinityKey)
		blocks = append(blocks, k.CIDR)
	}
	return blocks
}

func deleteAllPoolsWindows() {
	log.Infof("Windows: Deleting all pools")
	ipPoolsWindows.pools = map[string]poolWindows{}
}

func applyPoolWindows(cidr string, enabled bool) {
	log.Infof("Windows: Adding pool: %s, enabled: %v", cidr, enabled)
	ipPoolsWindows.pools[cidr] = poolWindows{enabled: enabled}
}
