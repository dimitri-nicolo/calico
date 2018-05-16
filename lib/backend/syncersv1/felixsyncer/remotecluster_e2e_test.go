// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package felixsyncer_test

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/felixsyncer"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

var _ = testutils.E2eDatastoreDescribe("Remote cluster syncer tests - connection failures", testutils.DatastoreAll, func(config apiconfig.CalicoAPIConfig) {

	ctx := context.Background()
	var err error
	var c clientv3.Interface
	var be api.Client
	var syncer api.Syncer
	var syncTester *testutils.SyncerTester

	BeforeEach(func() {
		// Create the v3 client
		c, err = clientv3.New(config)
		Expect(err).NotTo(HaveOccurred())

		// Create the backend client to clean the datastore and obtain a syncer interface.
		be, err = backend.NewClient(config)
		Expect(err).NotTo(HaveOccurred())
		be.Clean()
	})

	AfterEach(func() {
		if syncer != nil {
			syncer.Stop()
			syncer = nil
		}
		if be != nil {
			be.Close()
			be = nil
		}
	})

	DescribeTable("Configuring a RemoteClusterConfiguration resource with",
		func(name string, spec apiv3.RemoteClusterConfigurationSpec, errPrefix string, connTimeout time.Duration) {
			By("Creating the RemoteClusterConfiguration")
			_, outError := c.RemoteClusterConfigurations().Create(ctx, &apiv3.RemoteClusterConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       spec,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())

			By("Creating and starting a syncer")
			syncTester = testutils.NewSyncerTester()
			syncer = felixsyncer.New(be, syncTester)
			syncer.Start()

			By("Checking status is updated to sync'd at start of day")
			syncTester.ExpectStatusUpdate(api.WaitForDatastore)
			syncTester.ExpectStatusUpdate(api.ResyncInProgress)
			syncTester.ExpectStatusUpdate(api.InSync, connTimeout)

			By("Checking we received the event messages for the remote cluster")
			// We should receive Connecting and ConnectionFailed
			expectedEvents := []api.Update{
				{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: name},
						Value: &model.RemoteClusterStatus{
							Status: model.RemoteClusterConnecting,
						},
					},
					UpdateType: api.UpdateTypeKVNew,
				},
				{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: name},
						Value: &model.RemoteClusterStatus{
							Status: model.RemoteClusterConnectionFailed,
							Error:  errPrefix,
						},
					},
					UpdateType: api.UpdateTypeKVUpdated,
				},
			}
			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment, so include
				// those events too.
				for _, r := range defaultKubernetesResource {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}
				for _, n := range []string{"default", "kube-public", "kube-system", "namespace-1", "namespace-2"} {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileRulesKey{ProfileKey: model.ProfileKey{Name: "ksa." + n + ".default"}},
							Value: &model.ProfileRules{
								InboundRules:  nil,
								OutboundRules: nil,
							},
						},
						UpdateType: api.UpdateTypeKVNew,
					})
				}
			}

			// Sanitize the actual events received to remove revision info and to handle prefix matching of the
			// RemoteClusterStatus error. Compare with the expected events.
			syncTester.ExpectUpdatesSanitized(expectedEvents, false, func(u *api.Update) {
				u.Revision = ""
				u.TTL = 0

				if r, ok := u.Value.(*model.RemoteClusterStatus); ok {
					if r.Error != "" && strings.HasPrefix(r.Error, errPrefix) {
						// The error has the expected prefix. Substitute the actual error for the prefix so that the
						// exact comparison can be made.
						r.Error = errPrefix
					}
				}
			})
		},

		// An invalid etcd endpoint takes the configured dial timeout to fail (10s), so let's wait for at least 15s
		// for the test.
		Entry("invalid etcd endpoint", "bad-etcdv3-1",
			apiv3.RemoteClusterConfigurationSpec{
				DatastoreType: "etcdv3",
				EtcdConfig: apiv3.EtcdConfig{
					EtcdEndpoints: "http://foobarbaz:1000",
				},
			},
			"dial tcp: lookup foobarbaz on", 15*time.Second,
		),
		Entry("invalid etcd cert files", "bad-etcdv3-2",
			apiv3.RemoteClusterConfigurationSpec{
				DatastoreType: "etcdv3",
				EtcdConfig: apiv3.EtcdConfig{
					EtcdEndpoints:  "https://127.0.0.1:2379",
					EtcdCertFile:   "foo",
					EtcdCACertFile: "bar",
					EtcdKeyFile:    "baz",
				},
			},
			"could not initialize etcdv3 client: open foo: no such file or directory", 3*time.Second,
		),
		//TODO: This test is pending a fix for CNX-2562
		// An invalid kubernetes endpoint will successfully create the client, but will fail in the watchersyncer
		// when attempting to List and Watch the remote.
		PEntry("invalid k8s endpoint", "bad-k8s-1",
			apiv3.RemoteClusterConfigurationSpec{
				DatastoreType: "kubernetes",
				KubeConfig: apiv3.KubeConfig{
					K8sAPIEndpoint: "http://foobarbaz:1000",
				},
			},
			"dial tcp: lookup foobarbaz on", 15*time.Second,
		),
		Entry("invalid k8s kubeconfig file", "bad-k8s2",
			apiv3.RemoteClusterConfigurationSpec{
				DatastoreType: "kubernetes",
				KubeConfig: apiv3.KubeConfig{
					Kubeconfig: "foobarbaz",
				},
			},
			"stat foobarbaz: no such file or directory", 3*time.Second,
		),
	)

	Describe("Deleting a RemoteClusterConfiguration before connection fails", func() {
		It("should get delete event without getting a connection failure event", func() {
			// Create a RemoteClusterConfiguration with a bad etcd endpoint - this takes a little while to timeout
			// which gives us time to delete the RemoteClusterConfiguration before we actually get the connection
			// failure notification.
			By("Creating the RemoteClusterConfiguration")
			rcc, outError := c.RemoteClusterConfigurations().Create(ctx, &apiv3.RemoteClusterConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: "etcd-timeout"},
				Spec: apiv3.RemoteClusterConfigurationSpec{
					DatastoreType: "etcdv3",
					EtcdConfig: apiv3.EtcdConfig{
						EtcdEndpoints: "http://foobarbaz:1000",
					},
				},
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())

			By("Creating and starting a syncer")
			syncTester = testutils.NewSyncerTester()
			syncer = felixsyncer.New(be, syncTester)
			syncer.Start()

			By("Checking status is updated to resync in progress")
			syncTester.ExpectStatusUpdate(api.WaitForDatastore)
			syncTester.ExpectStatusUpdate(api.ResyncInProgress)

			By("Checking we received the event messages for the remote cluster")
			// We should receive Connecting event first.
			expectedEvents := []api.Update{
				{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: "etcd-timeout"},
						Value: &model.RemoteClusterStatus{
							Status: model.RemoteClusterConnecting,
						},
					},
					UpdateType: api.UpdateTypeKVNew,
				},
			}
			if config.Spec.DatastoreType == apiconfig.Kubernetes {
				// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment, so include
				// those events too.
				for _, r := range defaultKubernetesResource {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}
				for _, n := range []string{"default", "kube-public", "kube-system", "namespace-1", "namespace-2"} {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileRulesKey{ProfileKey: model.ProfileKey{Name: "ksa." + n + ".default"}},
							Value: &model.ProfileRules{
								InboundRules:  nil,
								OutboundRules: nil,
							},
						},
						UpdateType: api.UpdateTypeKVNew,
					})
				}
			}

			// Sanitize the actual events received to remove revision info and compare against those expected.
			syncTester.ExpectUpdatesSanitized(expectedEvents, false, func(u *api.Update) {
				u.Revision = ""
				u.TTL = 0
			})

			By("Deleting the RemoteClusterConfiguration")
			_, err = c.RemoteClusterConfigurations().Delete(ctx, "etcd-timeout", options.DeleteOptions{ResourceVersion: rcc.ResourceVersion})
			Expect(err).NotTo(HaveOccurred())

			By("Expecting the syncer to move quickly to in-sync")
			syncTester.ExpectStatusUpdate(api.InSync)

			By("Checking we received the delete event messages for the remote cluster")
			expectedEvents = []api.Update{
				{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: "etcd-timeout"},
					},
					UpdateType: api.UpdateTypeKVDeleted,
				},
			}
			syncTester.ExpectUpdates(expectedEvents, false)
		})
	})
})

// To test successful Remote Cluster connections, we use a local k8s cluster with a remote etcd cluster.
var _ = testutils.E2eDatastoreDescribe("Remote cluster syncer tests", testutils.DatastoreK8s, func(localConfig apiconfig.CalicoAPIConfig) {
	ctx := context.Background()
	var err error
	var localClient clientv3.Interface
	var localBackend api.Client
	var syncer api.Syncer
	var syncTester *testutils.SyncerTester
	var remoteClient clientv3.Interface

	testutils.E2eDatastoreDescribe("Successful connection to cluster", testutils.DatastoreEtcdV3, func(remoteConfig apiconfig.CalicoAPIConfig) {
		BeforeEach(func() {
			// Create the v3 clients for the local and remote clusters.
			localClient, err = clientv3.New(localConfig)
			Expect(err).NotTo(HaveOccurred())
			remoteClient, err = clientv3.New(remoteConfig)
			Expect(err).NotTo(HaveOccurred())

			// Create the local backend client to clean the datastore and obtain a syncer interface.
			localBackend, err = backend.NewClient(localConfig)
			Expect(err).NotTo(HaveOccurred())
			localBackend.Clean()

			// Create the remote backend client to clean the datastore.
			remoteBackend, err := backend.NewClient(remoteConfig)
			Expect(err).NotTo(HaveOccurred())
			remoteBackend.Clean()
			err = remoteBackend.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if syncer != nil {
				syncer.Stop()
				syncer = nil
			}
			if localBackend != nil {
				localBackend.Close()
				localBackend = nil
			}
		})

		It("Should connect to the remote cluster and sync the remote data", func() {
			By("Configuring the RemoteClusterConfiguration for the remote")
			rcc := &apiv3.RemoteClusterConfiguration{ObjectMeta: metav1.ObjectMeta{Name: "remote-cluster"}}
			rcc.Spec.DatastoreType = string(remoteConfig.Spec.DatastoreType)
			rcc.Spec.EtcdEndpoints = remoteConfig.Spec.EtcdEndpoints
			rcc.Spec.EtcdUsername = remoteConfig.Spec.EtcdUsername
			rcc.Spec.EtcdPassword = remoteConfig.Spec.EtcdPassword
			rcc.Spec.EtcdKeyFile = remoteConfig.Spec.EtcdKeyFile
			rcc.Spec.EtcdCertFile = remoteConfig.Spec.EtcdCertFile
			rcc.Spec.EtcdCACertFile = remoteConfig.Spec.EtcdCACertFile
			rcc.Spec.Kubeconfig = remoteConfig.Spec.Kubeconfig
			rcc.Spec.K8sAPIEndpoint = remoteConfig.Spec.K8sAPIEndpoint
			rcc.Spec.K8sKeyFile = remoteConfig.Spec.K8sKeyFile
			rcc.Spec.K8sCertFile = remoteConfig.Spec.K8sCertFile
			rcc.Spec.K8sCAFile = remoteConfig.Spec.K8sCAFile
			rcc.Spec.K8sAPIToken = remoteConfig.Spec.K8sAPIToken
			rcc.Spec.K8sInsecureSkipTLSVerify = remoteConfig.Spec.K8sInsecureSkipTLSVerify
			_, outError := localClient.RemoteClusterConfigurations().Create(ctx, rcc, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())

			// Keep track of the set of events we will expect from the Felix syncer. Start with the remote
			// cluster staus updates as the connection succeeds.
			expectedEvents := []api.Update{
				{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: "remote-cluster"},
						Value: &model.RemoteClusterStatus{
							Status: model.RemoteClusterConnecting,
						},
					},
					UpdateType: api.UpdateTypeKVNew,
				},
				{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: "remote-cluster"},
						Value: &model.RemoteClusterStatus{
							Status: model.RemoteClusterResyncInProgress,
						},
					},
					UpdateType: api.UpdateTypeKVUpdated,
				},
				{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: "remote-cluster"},
						Value: &model.RemoteClusterStatus{
							Status: model.RemoteClusterInSync,
						},
					},
					UpdateType: api.UpdateTypeKVUpdated,
				},
			}

			By("Creating a remote WorkloadEndpoint")
			wep := apiv3.NewWorkloadEndpoint()
			wep.Namespace = "ns1"
			wep.Spec.Node = "node1"
			wep.Spec.Orchestrator = "k8s"
			wep.Spec.Pod = "pod-1"
			wep.Spec.ContainerID = "container-1"
			wep.Spec.Endpoint = "eth0"
			wep.Spec.InterfaceName = "cali01234"
			wep.Spec.IPNetworks = []string{"10.100.10.1"}
			wep.Spec.Profiles = []string{"this-profile", "that-profile"}
			wep, err = remoteClient.WorkloadEndpoints().Create(ctx, wep, options.SetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Pass the resource through the update processor to get the expected syncer update (we'll need to
			// modify it to include the remote cluster details).
			up := updateprocessors.NewWorkloadEndpointUpdateProcessor()
			kvps, err := up.Process(&model.KVPair{
				Key: model.ResourceKey{
					Kind:      apiv3.KindWorkloadEndpoint,
					Name:      "node1-k8s-pod--1-eth0",
					Namespace: "ns1",
				},
				Value: wep,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(kvps).To(HaveLen(1))

			// Modify the values as expected for the remote cluster
			wepKey := kvps[0].Key.(model.WorkloadEndpointKey)
			wepValue := kvps[0].Value.(*model.WorkloadEndpoint)
			wepKey.Hostname = "remote-cluster/node1"
			wepValue.ProfileIDs = []string{"remote-cluster/this-profile", "remote-cluster/that-profile"}

			// Add this WEP to the set of expected events that we'll get from the syncer.
			expectedEvents = append(expectedEvents, api.Update{
				KVPair: model.KVPair{
					Key:   wepKey,
					Value: wepValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			})

			By("Creating a remote HostEndpoint")
			hep := apiv3.NewHostEndpoint()
			hep.Name = "hep-1"
			hep.Spec.Node = "node2"
			hep.Spec.InterfaceName = "eth1"
			hep.Spec.Profiles = []string{"foo", "bar"}
			hep, err = remoteClient.HostEndpoints().Create(ctx, hep, options.SetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Pass the resource through the update processor to get the expected syncer update (we'll need to
			// modify it to include the remote cluster details).
			up = updateprocessors.NewHostEndpointUpdateProcessor()
			kvps, err = up.Process(&model.KVPair{
				Key: model.ResourceKey{
					Kind:      apiv3.KindHostEndpoint,
					Name:      "hep-1",
					Namespace: "ns1",
				},
				Value: hep,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(kvps).To(HaveLen(1))

			// Modify the values as expected for the remote cluster
			hepKey := kvps[0].Key.(model.HostEndpointKey)
			hepValue := kvps[0].Value.(*model.HostEndpoint)
			hepKey.Hostname = "remote-cluster/node2"
			hepValue.ProfileIDs = []string{"remote-cluster/foo", "remote-cluster/bar"}

			// Add this HEP to the set of expected events that we'll get from the syncer.
			expectedEvents = append(expectedEvents, api.Update{
				KVPair: model.KVPair{
					Key:   hepKey,
					Value: hepValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			})

			By("Creating a remote Profile")
			pro := apiv3.NewProfile()
			pro.Name = "profile-1"
			pro.Spec.LabelsToApply = map[string]string{
				"label1": "value1",
				"label2": "value2",
			}
			pro.Spec.Ingress = []apiv3.Rule{
				{
					Action: "Allow",
				},
			}
			pro, err = remoteClient.Profiles().Create(ctx, pro, options.SetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Add this Profile to the set of expected events that we'll get from the syncer (no need to
			// run through a processor since the conversion is much simpler and we only require the labels
			// to apply, none of the policy is transferred to the local cluster).
			expectedEvents = append(expectedEvents, []api.Update{
				{
					KVPair: model.KVPair{
						Key: model.ProfileRulesKey{
							ProfileKey: model.ProfileKey{Name: "remote-cluster/profile-1"},
						},
						Value: &model.ProfileRules{
							InboundRules:  []model.Rule{},
							OutboundRules: []model.Rule{},
						},
					},
					UpdateType: api.UpdateTypeKVNew,
				},
				{
					KVPair: model.KVPair{
						Key: model.ProfileLabelsKey{
							ProfileKey: model.ProfileKey{Name: "remote-cluster/profile-1"},
						},
						Value: pro.Spec.LabelsToApply,
					},
					UpdateType: api.UpdateTypeKVNew,
				},
			}...)

			By("Creating and starting a syncer")
			syncTester = testutils.NewSyncerTester()
			syncer = felixsyncer.New(localBackend, syncTester)
			syncer.Start()

			By("Checking status is updated to sync'd at start of day")
			syncTester.ExpectStatusUpdate(api.WaitForDatastore)
			syncTester.ExpectStatusUpdate(api.ResyncInProgress)
			syncTester.ExpectStatusUpdate(api.InSync)

			By("Checking we received the expected events.")
			if localConfig.Spec.DatastoreType == apiconfig.Kubernetes {
				// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment, so include
				// those events too.
				for _, r := range defaultKubernetesResource {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}
				for _, n := range []string{"default", "kube-public", "kube-system", "namespace-1", "namespace-2"} {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileRulesKey{ProfileKey: model.ProfileKey{Name: "ksa." + n + ".default"}},
							Value: &model.ProfileRules{
								InboundRules:  nil,
								OutboundRules: nil,
							},
						},
						UpdateType: api.UpdateTypeKVNew,
					})
				}
			}

			// Sanitize the actual events received to remove revision info and compare against those expected.
			syncTester.ExpectUpdatesSanitized(expectedEvents, false, func(u *api.Update) {
				u.Revision = ""
				u.TTL = 0
			})
		})
	})
})
