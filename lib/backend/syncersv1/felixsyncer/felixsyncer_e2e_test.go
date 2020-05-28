// Copyright (c) 2017-2020 Tigera, Inc. All rights reserved.

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

package felixsyncer_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/backend/encap"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/felixsyncer"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/resources"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

// Kubernetes will have a profile for each of the namespaces that is configured.
// We expect:  default, kube-system, kube-public, namespace-1, namespace-2, kube-node-lease.
func defaultKubernetesResource() []model.KVPair {
	var out []model.KVPair
	// Add resources for the namespaces we expect in the cluster.
	for _, ns := range []string{"default", "kube-public", "kube-system", "namespace-1", "namespace-2", "kube-node-lease"} {
		// Expect profile rules for each namespace providing default allow behavior.
		out = append(out, model.KVPair{
			Key: model.ProfileRulesKey{ProfileKey: model.ProfileKey{Name: "kns." + ns}},
			Value: &model.ProfileRules{
				InboundRules:  []model.Rule{{Action: "allow"}},
				OutboundRules: []model.Rule{{Action: "allow"}},
			},
		})

		// Expect profile labels for each namespace as well. The labels should include the name
		// of the namespace.
		out = append(out, model.KVPair{
			Key: model.ProfileLabelsKey{ProfileKey: model.ProfileKey{Name: "kns." + ns}},
			Value: map[string]string{
				"pcns.projectcalico.org/name": ns,
			},
		})

		// Expect corresponding v3 Profile resource.
		out = append(out, model.KVPair{
			Key: model.ResourceKey{Name: "kns." + ns, Kind: apiv3.KindProfile},
			Value: &apiv3.Profile{
				Spec: apiv3.ProfileSpec{
					Ingress: []apiv3.Rule{{Action: "Allow"}},
					Egress:  []apiv3.Rule{{Action: "Allow"}},
					LabelsToApply: map[string]string{
						"pcns.projectcalico.org/name": ns,
					},
				},
			},
		})

		// Expect profile rules for the default serviceaccount in each namespace.
		out = append(out, model.KVPair{
			Key: model.ProfileRulesKey{ProfileKey: model.ProfileKey{Name: "ksa." + ns + ".default"}},
			Value: &model.ProfileRules{
				InboundRules:  nil,
				OutboundRules: nil,
			},
		})

		// Expect profile labels for each default serviceaccount as well. The labels should include the name
		// of the service account.
		out = append(out, model.KVPair{
			Key: model.ProfileLabelsKey{ProfileKey: model.ProfileKey{Name: "ksa." + ns + ".default"}},
			Value: map[string]string{
				"pcsa.projectcalico.org/name": "default",
			},
		})

		// Expect corresponding v3 Profile resource.
		out = append(out, model.KVPair{
			Key: model.ResourceKey{Name: "ksa." + ns + ".default", Kind: apiv3.KindProfile},
			Value: &apiv3.Profile{
				Spec: apiv3.ProfileSpec{
					LabelsToApply: map[string]string{
						"pcsa.projectcalico.org/name": "default",
					},
				},
			},
		})
	}

	// in addition, there is a hard-coded default profile that allows everything.
	out = append(out, model.KVPair{
		Key: model.ProfileRulesKey{ProfileKey: model.ProfileKey{Name: "default"}},
		Value: &model.ProfileRules{
			InboundRules:  []model.Rule{{Action: "allow"}},
			OutboundRules: []model.Rule{{Action: "allow"}},
		},
	})
	out = append(out, model.KVPair{
		Key: model.ResourceKey{Name: "default", Kind: apiv3.KindProfile},
		Value: &apiv3.Profile{
			Spec: apiv3.ProfileSpec{
				Ingress: []apiv3.Rule{{Action: "Allow"}},
				Egress:  []apiv3.Rule{{Action: "Allow"}},
			},
		},
	})
	return out
}

var _ = testutils.E2eDatastoreDescribe("Felix syncer tests", testutils.DatastoreAll, func(config apiconfig.CalicoAPIConfig) {

	var ctx context.Context
	var c clientv3.Interface
	var be api.Client
	var syncTester *testutils.SyncerTester
	var err error
	var datamodelCleanups []func()

	addCleanup := func(cleanup func()) {
		datamodelCleanups = append(datamodelCleanups, cleanup)
	}

	BeforeEach(func() {
		ctx = context.Background()
		// Create a v3 client to drive data changes (luckily because this is the _test module,
		// we don't get circular imports.
		c, err = clientv3.New(config)
		Expect(err).NotTo(HaveOccurred())

		// Create the backend client to obtain a syncer interface.
		be, err = backend.NewClient(config)
		Expect(err).NotTo(HaveOccurred())
		be.Clean()

		// Create a SyncerTester to receive the BGP syncer callback events and to allow us
		// to assert state.
		syncTester = testutils.NewSyncerTester()

		datamodelCleanups = nil
	})

	AfterEach(func() {
		for _, cleanup := range datamodelCleanups {
			cleanup()
		}
	})

	v3Sanitizer := func(v interface{}) interface{} {
		// We expect two kinds of `model.Resource` over the Felix syncer: Nodes and
		// Profiles.  We don't care about anything more than the spec.
		switch val := v.(type) {
		case *apiv3.Node:
			v = &apiv3.Node{Spec: val.Spec}
		case *apiv3.Profile:
			v = &apiv3.Profile{Spec: val.Spec}
		}
		return v
	}

	Describe("Felix syncer functionality", func() {
		It("should receive the synced after return all current data", func() {
			syncer := felixsyncer.New(be, config.Spec, syncTester)
			syncer.Start()
			expectedCacheSize := 0

			By("Checking status is updated to sync'd at start of day")
			syncTester.ExpectStatusUpdate(api.WaitForDatastore)
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectStatusUpdate(api.ResyncInProgress)
			syncTester.ExpectStatusUpdate(api.InSync)

			// Add 2 for the default-allow profile that is always there.
			// However, no profile labels are in the list because the
			// default-allow profile doesn't specify labels.
			expectedProfile := resources.DefaultAllowProfile()
			syncTester.ExpectData(*expectedProfile)
			expectedCacheSize += 1

			syncTester.ExpectData(model.KVPair{
				Key: model.ProfileRulesKey{ProfileKey: model.ProfileKey{Name: "projectcalico-default-allow"}},
				Value: &model.ProfileRules{
					InboundRules:  []model.Rule{{Action: "allow"}},
					OutboundRules: []model.Rule{{Action: "allow"}},
				},
			})
			expectedCacheSize += 1

			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment.
				expectedCacheSize += len(defaultKubernetesResource())

				// Add 1 for the Node resource passed over the felix syncer.
				expectedCacheSize += 1
				syncTester.ExpectPath("/calico/resources/v3/projectcalico.org/nodes/127.0.0.1")

				for _, r := range defaultKubernetesResource() {
					syncTester.ExpectDataSanitized(r, v3Sanitizer)
				}
				syncTester.ExpectCacheSize(expectedCacheSize)
			}
			syncTester.ExpectCacheSize(expectedCacheSize)

			var node *apiv3.Node
			wip := net.MustParseIP("192.168.12.34")
			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				// For Kubernetes, update the existing node config to have some BGP configuration.
				By("Configuring a node with an IP address and tunnel MAC address")
				var (
					oldValuesSaved        bool
					oldBGPSpec            *apiv3.NodeBGPSpec
					oldVXLANTunnelMACAddr string
					oldWireguardSpec      *apiv3.NodeWireguardSpec
					oldWireguardPublicKey string
				)
				for i := 0; i < 5; i++ {
					// This can fail due to an update conflict, so we allow a few retries.
					node, err = c.Nodes().Get(ctx, "127.0.0.1", options.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					if !oldValuesSaved {
						if node.Spec.BGP == nil {
							oldBGPSpec = nil
						} else {
							bgpSpecCopy := *node.Spec.BGP
							oldBGPSpec = &bgpSpecCopy
						}
						oldVXLANTunnelMACAddr = node.Spec.VXLANTunnelMACAddr
						if node.Spec.Wireguard == nil {
							oldWireguardSpec = nil
						} else {
							wireguardSpecCopy := *node.Spec.Wireguard
							oldWireguardSpec = &wireguardSpecCopy
						}
						oldWireguardPublicKey = node.Status.WireguardPublicKey
						oldValuesSaved = true
					}
					node.Spec.BGP = &apiv3.NodeBGPSpec{
						IPv4Address:        "1.2.3.4/24",
						IPv6Address:        "aa:bb::cc/120",
						IPv4IPIPTunnelAddr: "10.10.10.1",
					}
					node.Spec.VXLANTunnelMACAddr = "66:cf:23:df:22:07"
					node.Spec.Wireguard = &apiv3.NodeWireguardSpec{
						InterfaceIPv4Address: "192.168.12.34",
					}
					node.Status = apiv3.NodeStatus{
						WireguardPublicKey: "jlkVyQYooZYzI2wFfNhSZez5eWh44yfq1wKVjLvSXgY=",
					}
					node, err = c.Nodes().Update(ctx, node, options.SetOptions{})
					if err == nil {
						break
					}
				}
				Expect(err).NotTo(HaveOccurred())
				addCleanup(func() {
					for i := 0; i < 5; i++ {
						// This can fail due to an update conflict, so we allow a few retries.
						node, err = c.Nodes().Get(ctx, "127.0.0.1", options.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						node.Spec.BGP = oldBGPSpec
						node.Spec.VXLANTunnelMACAddr = oldVXLANTunnelMACAddr
						node.Spec.Wireguard = oldWireguardSpec
						node.Status.WireguardPublicKey = oldWireguardPublicKey
						node, err = c.Nodes().Update(ctx, node, options.SetOptions{})
						if err == nil {
							break
						}
					}
					Expect(err).NotTo(HaveOccurred())
				})
				syncTester.ExpectData(model.KVPair{
					Key:   model.HostConfigKey{Hostname: "127.0.0.1", Name: "IpInIpTunnelAddr"},
					Value: "10.10.10.1",
				})
				syncTester.ExpectData(model.KVPair{
					Key:   model.HostConfigKey{Hostname: "127.0.0.1", Name: "VXLANTunnelMACAddr"},
					Value: "66:cf:23:df:22:07",
				})
				syncTester.ExpectData(model.KVPair{
					Key:   model.WireguardKey{NodeName: "127.0.0.1"},
					Value: &model.Wireguard{InterfaceIPv4Addr: &wip, PublicKey: "jlkVyQYooZYzI2wFfNhSZez5eWh44yfq1wKVjLvSXgY="},
				})
				expectedCacheSize += 3
			} else {
				// For non-Kubernetes, add a new node with valid BGP configuration.
				By("Creating a node with an IP address and tunnel MAC address")
				node, err = c.Nodes().Create(
					ctx,
					&apiv3.Node{
						ObjectMeta: metav1.ObjectMeta{Name: "127.0.0.1"},
						Spec: apiv3.NodeSpec{
							BGP: &apiv3.NodeBGPSpec{
								IPv4Address:        "1.2.3.4/24",
								IPv6Address:        "aa:bb::cc/120",
								IPv4IPIPTunnelAddr: "10.10.10.1",
							},
							VXLANTunnelMACAddr: "66:cf:23:df:22:07",
							Wireguard: &apiv3.NodeWireguardSpec{
								InterfaceIPv4Address: "192.168.12.34",
							},
						},
						Status: apiv3.NodeStatus{
							WireguardPublicKey: "jlkVyQYooZYzI2wFfNhSZez5eWh44yfq1wKVjLvSXgY=",
						},
					},
					options.SetOptions{},
				)
				Expect(err).NotTo(HaveOccurred())

				// Add 1 for the Node resource passed over the felix syncer.
				expectedCacheSize += 1

				// Creating the node initialises the ClusterInformation as a side effect.
				syncTester.ExpectData(model.KVPair{
					Key:   model.ReadyFlagKey{},
					Value: true,
				})
				syncTester.ExpectValueMatches(
					model.GlobalConfigKey{Name: "ClusterGUID"},
					MatchRegexp("[a-f0-9]{32}"),
				)
				// Creating the node also creates default tier.
				syncTester.ExpectData(model.KVPair{
					Key:   model.TierKey{Name: "default"},
					Value: &model.Tier{},
				})
				syncTester.ExpectData(model.KVPair{
					Key:   model.HostConfigKey{Hostname: "127.0.0.1", Name: "IpInIpTunnelAddr"},
					Value: "10.10.10.1",
				})
				syncTester.ExpectData(model.KVPair{
					Key:   model.HostConfigKey{Hostname: "127.0.0.1", Name: "VXLANTunnelMACAddr"},
					Value: "66:cf:23:df:22:07",
				})
				syncTester.ExpectData(model.KVPair{
					Key:   model.WireguardKey{NodeName: "127.0.0.1"},
					Value: &model.Wireguard{InterfaceIPv4Addr: &wip, PublicKey: "jlkVyQYooZYzI2wFfNhSZez5eWh44yfq1wKVjLvSXgY="},
				})
				//add one for the node resource
				expectedCacheSize += 6
			}

			// The HostIP will be added for the IPv4 address
			expectedCacheSize += 2
			ip := net.MustParseIP("1.2.3.4")
			syncTester.ExpectData(model.KVPair{
				Key:   model.HostIPKey{Hostname: "127.0.0.1"},
				Value: &ip,
			})
			syncTester.ExpectData(model.KVPair{
				Key:   model.HostConfigKey{Hostname: "127.0.0.1", Name: "NodeIP"},
				Value: "1.2.3.4",
			})
			syncTester.ExpectCacheSize(expectedCacheSize)

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
						BlockSize:   30,
					},
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			// The pool will add as single entry ( +1 ), plus will also create the default
			// Felix config with IPIP enabled.
			expectedCacheSize += 2
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectData(model.KVPair{
				Key: model.IPPoolKey{CIDR: net.MustParseCIDR("192.124.0.0/21")},
				Value: &model.IPPool{
					CIDR:          poolCIDRNet,
					IPIPInterface: "tunl0",
					IPIPMode:      encap.CrossSubnet,
					Masquerade:    true,
					IPAM:          true,
					Disabled:      false,
				},
				Revision: pool.ResourceVersion,
			})
			syncTester.ExpectData(model.KVPair{
				Key:   model.GlobalConfigKey{"IpInIpEnabled"},
				Value: "true",
			})

			By("Creating a GlobalNetworkSet")
			gns := apiv3.NewGlobalNetworkSet()
			gns.Name = "anetworkset"
			gns.Labels = map[string]string{
				"a": "b",
			}
			gns.Spec.Nets = []string{
				"11.0.0.0/16",
			}
			gns.Spec.AllowedEgressDomains = []string{
				"direct.gov.uk",
				"cam.ac.uk",
			}
			gns, err = c.GlobalNetworkSets().Create(
				ctx,
				gns,
				options.SetOptions{},
			)
			expectedCacheSize++
			syncTester.ExpectCacheSize(expectedCacheSize)
			_, expGNet, err := net.ParseCIDROrIP("11.0.0.0/16")
			Expect(err).NotTo(HaveOccurred())
			syncTester.ExpectData(model.KVPair{
				Key: model.NetworkSetKey{Name: "anetworkset"},
				Value: &model.NetworkSet{
					Labels: map[string]string{
						"a": "b",
					},
					Nets: []net.IPNet{
						*expGNet,
					},
					AllowedEgressDomains: []string{
						"direct.gov.uk",
						"cam.ac.uk",
					},
				},
				Revision: gns.ResourceVersion,
			})

			By("Creating a NetworkSet")
			ns := apiv3.NewNetworkSet()
			ns.Name = "anetworkset"
			ns.Namespace = "namespace-1"
			ns.Labels = map[string]string{
				"a": "b",
			}
			ns.Spec.Nets = []string{
				"11.0.0.0/16",
			}
			ns, err = c.NetworkSets().Create(
				ctx,
				ns,
				options.SetOptions{},
			)
			expectedCacheSize++
			syncTester.ExpectCacheSize(expectedCacheSize)
			_, expNet, err := net.ParseCIDROrIP("11.0.0.0/16")
			Expect(err).NotTo(HaveOccurred())
			syncTester.ExpectData(model.KVPair{
				Key: model.NetworkSetKey{Name: "namespace-1/anetworkset"},
				Value: &model.NetworkSet{
					Labels: map[string]string{
						"a":                           "b",
						"projectcalico.org/namespace": "namespace-1",
					},
					Nets: []net.IPNet{
						*expNet,
					},
					ProfileIDs: []string{
						"kns.namespace-1",
					},
				},
				Revision: ns.ResourceVersion,
			})

			By("Creating a LicenseKey")
			lk := apiv3.NewLicenseKey()
			lk.Name = "default"
			lk.Spec.Token = "token"
			lk.Spec.Certificate = "certificate"
			lk, err = c.LicenseKey().Create(
				ctx,
				lk,
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			expectedCacheSize++
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectData(model.KVPair{
				Key: model.LicenseKeyKey{Name: "default"},
				Value: &model.LicenseKey{
					Token:       "token",
					Certificate: "certificate",
				},
				Revision: lk.ResourceVersion,
			})

			By("Creating a HostEndpoint")
			hep, err := c.HostEndpoints().Create(
				ctx,
				&apiv3.HostEndpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name: "hosta.eth0-a",
						Labels: map[string]string{
							"label1": "value1",
						},
					},
					Spec: apiv3.HostEndpointSpec{
						Node:          "127.0.0.1",
						InterfaceName: "eth0",
						ExpectedIPs:   []string{"1.2.3.4", "aa:bb::cc:dd"},
						Profiles:      []string{"profile1", "profile2"},
						Ports: []apiv3.EndpointPort{
							{
								Name:     "port1",
								Protocol: numorstring.ProtocolFromString("TCP"),
								Port:     1234,
							},
							{
								Name:     "port2",
								Protocol: numorstring.ProtocolFromString("UDP"),
								Port:     1010,
							},
						},
					},
				},
				options.SetOptions{},
			)

			Expect(err).NotTo(HaveOccurred())
			// The host endpoint will add as single entry ( +1 )
			expectedCacheSize += 1
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectData(model.KVPair{
				Key: model.HostEndpointKey{Hostname: "127.0.0.1", EndpointID: "hosta.eth0-a"},
				Value: &model.HostEndpoint{
					Name:              "eth0",
					ExpectedIPv4Addrs: []net.IP{net.MustParseIP("1.2.3.4")},
					ExpectedIPv6Addrs: []net.IP{net.MustParseIP("aa:bb::cc:dd")},
					Labels: map[string]string{
						"label1": "value1",
					},
					ProfileIDs: []string{"profile1", "profile2"},
					Ports: []model.EndpointPort{
						{
							Name:     "port1",
							Protocol: numorstring.ProtocolFromStringV1("TCP"),
							Port:     1234,
						},
						{
							Name:     "port2",
							Protocol: numorstring.ProtocolFromStringV1("UDP"),
							Port:     1010,
						},
					},
				},
				Revision: hep.ResourceVersion,
			})

			By("Allocating an IP")
			err = c.IPAM().AssignIP(ctx, ipam.AssignIPArgs{
				Hostname: "127.0.0.1",
				IP:       net.MustParseIP("192.124.0.1"),
			})
			Expect(err).NotTo(HaveOccurred())
			expectedCacheSize += 1

			_, cidr, _ := net.ParseCIDR("192.124.0.0/30")
			affinity := "host:127.0.0.1"
			zero := 0
			syncTester.ExpectData(model.KVPair{
				Key: model.BlockKey{CIDR: *cidr},
				Value: &model.AllocationBlock{
					CIDR:        *cidr,
					Affinity:    &affinity,
					Allocations: []*int{nil, &zero, nil, nil},
					Unallocated: []int{0, 2, 3},
					Attributes: []model.AllocationAttribute{
						{},
					},
				},
			})

			By("Creating a Tier")
			tierName := "mytier"
			order := float64(100.00)
			tier, err := c.Tiers().Create(
				ctx,
				&apiv3.Tier{
					ObjectMeta: metav1.ObjectMeta{Name: tierName},
					Spec: apiv3.TierSpec{
						Order: &order,
					},
				},
				options.SetOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			expectedCacheSize += 1
			syncTester.ExpectCacheSize(expectedCacheSize)
			syncTester.ExpectData(model.KVPair{
				Key: model.TierKey{Name: tierName},
				Value: &model.Tier{
					Order: &order,
				},
				Revision: tier.ResourceVersion,
			})

			By("Starting a new syncer and verifying that all current entries are returned before sync status")
			// We need to create a new syncTester and syncer.
			current := syncTester.GetCacheEntries()
			syncTester = testutils.NewSyncerTester()
			syncer = felixsyncer.New(be, config.Spec, syncTester)
			syncer.Start()

			// Verify the data is the same as the data from the previous cache.  We got the cache in the previous
			// step.
			syncTester.ExpectStatusUpdate(api.WaitForDatastore)
			syncTester.ExpectStatusUpdate(api.ResyncInProgress)
			syncTester.ExpectCacheSize(len(current))
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

var _ = testutils.E2eDatastoreDescribe("Felix syncer tests (KDD only)", testutils.DatastoreK8s, func(config apiconfig.CalicoAPIConfig) {
	var be api.Client
	var syncTester *testutils.SyncerTester
	var err error

	BeforeEach(func() {
		// Create the backend client to obtain a syncer interface.
		config.Spec.K8sUsePodCIDR = true
		be, err = backend.NewClient(config)
		Expect(err).NotTo(HaveOccurred())
		be.Clean()

		// Create a SyncerTester to receive the BGP syncer callback events and to allow us
		// to assert state.
		syncTester = testutils.NewSyncerTester()
	})

	It("should handle IPAM blocks properly for host-local IPAM", func() {
		config.Spec.K8sUsePodCIDR = true
		syncer := felixsyncer.New(be, config.Spec, syncTester)
		syncer.Start()

		// Verify we start a resync.
		syncTester.ExpectStatusUpdate(api.WaitForDatastore)
		syncTester.ExpectStatusUpdate(api.ResyncInProgress)

		// Expect a felix config for the IPIP tunnel address, generated from the podCIDR.
		syncTester.ExpectData(model.KVPair{
			Key:   model.HostConfigKey{Hostname: "127.0.0.1", Name: "IpInIpTunnelAddr"},
			Value: "10.10.10.1",
		})

		// Expect to be in-sync.
		syncTester.ExpectStatusUpdate(api.InSync)
	})
})
