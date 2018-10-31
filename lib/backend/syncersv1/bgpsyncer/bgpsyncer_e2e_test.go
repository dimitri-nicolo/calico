// Copyright (c) 2017 Tigera, Inc. All rights reserved.

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

package bgpsyncer_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/bgpsyncer"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	"github.com/projectcalico/libcalico-go/lib/ipip"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

// These tests validate that the various resources that the BGP watches are
// handled correctly by the syncer.  We don't validate in detail the behavior of
// each of udpate handlers that are invoked, since these are tested more thoroughly
// elsewhere.
var _ = testutils.E2eDatastoreDescribe("BGP syncer tests", testutils.DatastoreAll, func(config apiconfig.CalicoAPIConfig) {

	ctx := context.Background()

	Describe("BGP syncer functionality", func() {
		It("should receive the synced after return all current data", func() {
			// Create a v3 client to drive data changes (luckily because this is the _test module,
			// we don't get circular imports.
			c, err := clientv3.New(config)
			Expect(err).NotTo(HaveOccurred())

			// Create the backend client to obtain a syncer interface.
			be, err := backend.NewClient(config)
			Expect(err).NotTo(HaveOccurred())
			be.Clean()

			// Create a SyncerTester to receive the BGP syncer callback events and to allow us
			// to assert state.
			syncTester := testutils.NewSyncerTester()
			syncer := bgpsyncer.New(be, syncTester, "127.0.0.1")
			syncer.Start()
			expectedCacheSize := 0

			By("Checking status is updated to sync'd at start of day")
			syncTester.ExpectStatusUpdate(api.WaitForDatastore)
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectStatusUpdate(api.ResyncInProgress)
			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				expectedCacheSize += 2
			}
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectStatusUpdate(api.InSync)
			syncTester.ExpectCacheSize(expectedCacheSize)

			// For Kubernetes test the two entries already in the cache - one
			// affinity block and one node.
			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				syncTester.ExpectPath("/calico/resources/v3/projectcalico.org/nodes/127.0.0.1")
				syncTester.ExpectData(model.KVPair{
					Key:   model.BlockAffinityKey{Host: "127.0.0.1", CIDR: net.MustParseCIDR("10.10.10.0/24")},
					Value: &model.BlockAffinity{State: model.StateConfirmed},
				})
			}

			By("Disabling node to node mesh and adding a default ASNumber")
			n2n := false
			asn := numorstring.ASNumber(12345)
			bgpCfg, err := c.BGPConfigurations().Create(
				ctx,
				&apiv3.BGPConfiguration{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
					Spec: apiv3.BGPConfigurationSpec{
						NodeToNodeMeshEnabled: &n2n,
						ASNumber:              &asn,
					},
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			expectedCacheSize += 3

			// We should have entries for each config option that was set (i.e. 2) and the default extensions field
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectData(model.KVPair{
				Key:      model.GlobalBGPConfigKey{"as_num"},
				Value:    "12345",
				Revision: bgpCfg.ResourceVersion,
			})
			syncTester.ExpectData(model.KVPair{
				Key:      model.GlobalBGPConfigKey{"node_mesh"},
				Value:    "{\"enabled\":false}",
				Revision: bgpCfg.ResourceVersion,
			})
			syncTester.ExpectData(model.KVPair{
				Key:      model.GlobalBGPConfigKey{"extensions"},
				Value:    "{}",
				Revision: bgpCfg.ResourceVersion,
			})

			var node *apiv3.Node
			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				// For Kubernetes, update the existing node config to have some BGP configuration.
				By("Configuring a node with BGP configuration")
				node, err = c.Nodes().Get(ctx, "127.0.0.1", options.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				node.Spec.BGP = &apiv3.NodeBGPSpec{
					IPv4Address: "1.2.3.4/24",
					IPv6Address: "aa:bb::cc/120",
				}
				node, err = c.Nodes().Update(ctx, node, options.SetOptions{})
				Expect(err).NotTo(HaveOccurred())

				// The existing Node resource is updated; no change in cache size.
			} else {
				// For non-Kubernetes, add a new node with valid BGP configuration.
				By("Creating a node with BGP configuration")
				node, err = c.Nodes().Create(
					ctx,
					&apiv3.Node{
						ObjectMeta: metav1.ObjectMeta{Name: "127.0.0.1"},
						Spec: apiv3.NodeSpec{
							BGP: &apiv3.NodeBGPSpec{
								IPv4Address: "1.2.3.4/24",
								IPv6Address: "aa:bb::cc/120",
							},
						},
					},
					options.SetOptions{},
				)
				Expect(err).NotTo(HaveOccurred())
				expectedCacheSize += 1
			}

			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectPath("/calico/resources/v3/projectcalico.org/nodes/127.0.0.1")

			By("Updating the BGPConfiguration to remove the default ASNumber")
			bgpCfg.Spec.ASNumber = nil
			_, err = c.BGPConfigurations().Update(ctx, bgpCfg, options.SetOptions{})
			Expect(err).NotTo(HaveOccurred())
			// Removing one config option ( -1 )
			expectedCacheSize -= 1
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectNoData(model.GlobalBGPConfigKey{"as_num"})

			By("Creating an IPPool")
			poolCIDR := "192.124.0.0/21"
			poolCIDRNet := net.MustParseCIDR(poolCIDR)
			pool, err := c.IPPools().Create(
				ctx,
				&apiv3.IPPool{
					ObjectMeta: metav1.ObjectMeta{Name: "mypool"},
					Spec: apiv3.IPPoolSpec{
						CIDR:        poolCIDR,
						IPIPMode:    apiv3.IPIPModeCrossSubnet,
						NATOutgoing: true,
					},
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			// The pool will add as single entry ( +1 )
			poolKeyV1 := model.IPPoolKey{CIDR: net.MustParseCIDR("192.124.0.0/21")}
			expectedCacheSize += 1
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectData(model.KVPair{
				Key: poolKeyV1,
				Value: &model.IPPool{
					CIDR:          poolCIDRNet,
					IPIPInterface: "tunl0",
					IPIPMode:      ipip.CrossSubnet,
					Masquerade:    true,
					IPAM:          true,
					Disabled:      false,
				},
				Revision: pool.ResourceVersion,
			})

			By("Creating a BGPPeer")
			_, err = c.BGPPeers().Create(
				ctx,
				&apiv3.BGPPeer{
					ObjectMeta: metav1.ObjectMeta{Name: "peer1"},
					Spec: apiv3.BGPPeerSpec{
						PeerIP:   "192.124.10.20",
						ASNumber: numorstring.ASNumber(75758),
					},
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())

			// The peer will add as single entry ( +1 )
			expectedCacheSize += 1
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectPath("/calico/resources/v3/projectcalico.org/bgppeers/peer1")

			// For non-kubernetes, check that we can allocate an IP address and get a syncer update
			// for the allocation block.
			var blockAffinityKeyV1 model.BlockAffinityKey
			if config.Spec.DatastoreType != apiconfig.Kubernetes {
				By("Allocating an IP address and checking that we get an allocation block")
				ipV4Nets1, _, err := c.IPAM().AutoAssign(ctx, ipam.AutoAssignArgs{
					Num4:     1,
					Hostname: "127.0.0.1",
				})

				ips1 := make([]net.IP, 0, 0)
				for _, ipnet := range ipV4Nets1 {
					ips1 = append(ips1, net.IP{ipnet.IP})
				}
				Expect(err).NotTo(HaveOccurred())

				// Allocating an IP will create an affinity block that we should be notified of.  Not sure
				// what CIDR will be chosen, so search the cached entries.
				expectedCacheSize += 1
				syncTester.ExpectCacheSize(expectedCacheSize)
				current := syncTester.GetCacheEntries()
				for _, kvp := range current {
					if kab, ok := kvp.Key.(model.BlockAffinityKey); ok {
						if kab.Host == "127.0.0.1" && poolCIDRNet.Contains(kab.CIDR.IP) {
							blockAffinityKeyV1 = kab
							break
						}
					}
				}
				Expect(blockAffinityKeyV1).NotTo(BeNil(), "Did not find affinity block in sync data")

				By("Allocating an IP address on a different host and checking for no updates")
				// The syncer only monitors affine blocks for one host, so IP allocations for a different
				// host should not result in updates.
				ipV4Nets2, _, err := c.IPAM().AutoAssign(ctx, ipam.AutoAssignArgs{
					Num4:     1,
					Hostname: "not-this-host",
				})

				ips2 := make([]net.IP, 0, 0)
				for _, ipnet := range ipV4Nets2 {
					ips2 = append(ips2, net.IP{ipnet.IP})
				}

				Expect(err).NotTo(HaveOccurred())
				syncTester.ExpectCacheSize(expectedCacheSize)

				By("Releasing the IP addresses and checking for no updates")
				// Releasing IPs should leave the affine blocks assigned, so releasing the IPs
				// should result in no updates.
				_, err = c.IPAM().ReleaseIPs(ctx, ips1)
				Expect(err).NotTo(HaveOccurred())
				_, err = c.IPAM().ReleaseIPs(ctx, ips2)
				Expect(err).NotTo(HaveOccurred())
				syncTester.ExpectCacheSize(expectedCacheSize)

				By("Deleting the IPPool and checking for pool and affine block deletion")
				// Deleting the pool will also release all affine blocks associated with the pool.
				_, err = c.IPPools().Delete(ctx, "mypool", options.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				// The pool and the affine block for 127.0.0.1 should have deletion events.
				expectedCacheSize -= 2
				syncTester.ExpectCacheSize(expectedCacheSize)
				syncTester.ExpectNoData(blockAffinityKeyV1)
				syncTester.ExpectNoData(poolKeyV1)
			}

			By("Starting a new syncer and verifying that all current entries are returned before sync status")
			// We need to create a new syncTester and syncer.
			current := syncTester.GetCacheEntries()
			syncTester = testutils.NewSyncerTester()
			syncer = bgpsyncer.New(be, syncTester, "127.0.0.1")
			syncer.Start()

			// Verify the data is the same as the data from the previous cache.  We got the cache in the previous
			// step.
			syncTester.ExpectStatusUpdate(api.WaitForDatastore)
			syncTester.ExpectStatusUpdate(api.ResyncInProgress)
			syncTester.ExpectCacheSize(expectedCacheSize)
			for _, e := range current {
				if config.Spec.DatastoreType == apiconfig.Kubernetes {
					// Don't check revisions for K8s since the node data gets updated constantly.
					e.Revision = ""
				}
				syncTester.ExpectData(e)
			}
			syncTester.ExpectStatusUpdate(api.InSync)
		})
	})
})
