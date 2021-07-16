// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.

package felixsyncer_test

import (
	"context"
	"os"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	libapiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/felixsyncer"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/remotecluster"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

// Set the list interval and watch interval in the WatcherSyncer.  We do this to reduce
// the test time.
func setWatchIntervals(listRetryInterval, watchPollInterval time.Duration) {
	watchersyncer.ListRetryInterval = listRetryInterval
	watchersyncer.WatchPollInterval = watchPollInterval
}

func commonSanitizer(callersUpdate *api.Update) *api.Update {
	u := *callersUpdate
	u.Revision = ""
	u.TTL = 0

	// We expect two kinds of `model.Resource` over the Felix syncer:
	// Nodes and Profiles.  We don't care about anything more than the
	// spec.  We could also get k8s services and endpoints, but tests that use this commonSanitizer
	// will either need filter out these resources, or disclude them in the syncer configuration.
	switch key := u.KVPair.Key.(type) {
	case model.ResourceKey:
		log.Infof("model.ResourceKey = %v", key)
		switch val := u.KVPair.Value.(type) {
		case *apiv3.Node:
			u.KVPair.Value = &apiv3.Node{Spec: val.Spec}

			// In KDD mode, we receive periodic updates of the
			// node resource. We can't guess when these will
			// happen, so just ignore them. It's a race
			// condition and not one that we want to cause a
			// failure.
			if u.UpdateType == api.UpdateTypeKVUpdated {
				return nil
			}
		case *apiv3.Profile:
			if key.Name == "projectcalico-default-allow" {
				// Suppress, because the test code hasn't been updated to expect these.
				return nil
			}
			u.KVPair.Value = &apiv3.Profile{Spec: val.Spec}
		default:
			// Unhandled v3 resource type.
			Expect(false).To(BeTrue())
		}
	case model.ProfileRulesKey:
		log.Infof("model.ProfileRulesKey = %v", key)
		if strings.Contains(key.Name, "projectcalico-default-allow") {
			// Suppress, because the test code hasn't been updated to expect these.
			return nil
		}
	default:
		log.Infof("default = %v: type %v", key, u.UpdateType)
	}

	return &u
}

var _ = testutils.E2eDatastoreDescribe("Remote cluster syncer tests - connection failures", testutils.DatastoreAll, func(config apiconfig.CalicoAPIConfig) {

	ctx := context.Background()
	var err error
	var c clientv3.Interface
	var be api.Client
	var syncer api.Syncer
	var syncTester *testutils.SyncerTester
	var filteredSyncerTester api.SyncerCallbacks

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
			filteredSyncerTester = NewValidationFilter(syncTester)
			syncer = felixsyncer.New(be, config.Spec, filteredSyncerTester, false, true)
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
				for _, r := range defaultKubernetesResource() {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}

				// Add an expected event for the node object.
				expectedEvents = append(expectedEvents, api.Update{
					UpdateType: api.UpdateTypeKVNew,
					KVPair: model.KVPair{
						Key: model.ResourceKey{Name: "127.0.0.1", Kind: "Node"},
						Value: &apiv3.Node{
							Spec: apiv3.NodeSpec{
								OrchRefs: []apiv3.OrchRef{
									{
										NodeName:     "127.0.0.1",
										Orchestrator: "k8s",
									},
								},
							},
						},
					},
				})
			}

			// Sanitize the actual events received to remove revision info and to handle prefix matching of the
			// RemoteClusterStatus error. Compare with the expected events.
			syncTester.ExpectUpdatesSanitized(expectedEvents, false, func(callersUpdate *api.Update) *api.Update {
				u := commonSanitizer(callersUpdate)
				if u != nil {
					if r, ok := u.Value.(*model.RemoteClusterStatus); ok {
						if r.Error != "" && strings.HasPrefix(r.Error, errPrefix) {
							// The error has the expected prefix. Substitute the actual error for the prefix so that the
							// exact comparison can be made.
							r.Error = errPrefix
						}
					}
				}
				return u
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
			"context deadline exceeded", 15*time.Second,
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

		//TODO: This test is pending the fix for CNX-3031
		// WatcherCaches each send a connection error which triggers the remote connection failed handling.
		// This should be refactored so that only the first watcher cache to receive an error sends an error in the updates.
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

	// TODO: This test should be removed and replaced with the one above when the watcher
	// syncer is refactored so that connection errors from the watcher cache are all consolidated
	// into one error update.
	DescribeTable("Configuring another RemoteClusterConfiguration resource with",
		func(name string, spec apiv3.RemoteClusterConfigurationSpec, errPrefix string, connTimeout time.Duration) {
			By("Creating the RemoteClusterConfiguration")
			_, outError := c.RemoteClusterConfigurations().Create(ctx, &apiv3.RemoteClusterConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       spec,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())

			By("Setting longer list intervals")
			defer setWatchIntervals(watchersyncer.ListRetryInterval, watchersyncer.WatchPollInterval)
			setWatchIntervals(12*time.Second, watchersyncer.WatchPollInterval)

			By("Creating and starting a syncer")
			syncTester = testutils.NewSyncerTester()
			filteredSyncerTester = NewValidationFilter(syncTester)
			syncer = felixsyncer.New(be, config.Spec, filteredSyncerTester, false, true)
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
				for _, r := range defaultKubernetesResource() {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}

				// Add an expected event for the node object.
				expectedEvents = append(expectedEvents, api.Update{
					UpdateType: api.UpdateTypeKVNew,
					KVPair: model.KVPair{
						Key: model.ResourceKey{Name: "127.0.0.1", Kind: "Node"},
						Value: &libapiv3.Node{
							Spec: libapiv3.NodeSpec{
								OrchRefs: []libapiv3.OrchRef{
									{
										NodeName:     "127.0.0.1",
										Orchestrator: "k8s",
									},
								},
							},
						},
					},
				})
			}

			// Sanitize the actual events received to remove revision info and to handle prefix matching of the
			// RemoteClusterStatus error. Compare with the expected events.
			syncTester.ExpectUpdatesSanitized(expectedEvents, false, func(callersUpdate *api.Update) *api.Update {
				u := commonSanitizer(callersUpdate)
				if u != nil {
					if r, ok := u.Value.(*model.RemoteClusterStatus); ok {
						// Need to clip off the exact client error
						// ex: "Get http://foobarbaz:1000/api/v1/namespaces: ..."
						splitErr := strings.SplitN(r.Error, ": ", 2)
						if len(splitErr) == 2 {
							r.Error = splitErr[1]
						}
						if r.Error != "" && strings.HasPrefix(r.Error, errPrefix) {
							// The error has the expected prefix. Substitute the actual error for the prefix so that the
							// exact comparison can be made.
							r.Error = errPrefix
						}
					}
				}
				return u
			})
		},

		Entry("invalid k8s endpoint", "bad-k8s-1",
			apiv3.RemoteClusterConfigurationSpec{
				DatastoreType: "kubernetes",
				KubeConfig: apiv3.KubeConfig{
					K8sAPIEndpoint: "http://foobarbaz:1000",
				},
			},
			"dial tcp: lookup foobarbaz on", 15*time.Second,
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
			filteredSyncerTester = NewValidationFilter(syncTester)
			syncer = felixsyncer.New(be, config.Spec, filteredSyncerTester, false, true)
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
				for _, r := range defaultKubernetesResource() {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}

				// Add an expected event for the node object.
				expectedEvents = append(expectedEvents, api.Update{
					UpdateType: api.UpdateTypeKVNew,
					KVPair: model.KVPair{
						Key: model.ResourceKey{Name: "127.0.0.1", Kind: "Node"},
						Value: &libapiv3.Node{
							Spec: libapiv3.NodeSpec{
								OrchRefs: []libapiv3.OrchRef{
									{
										NodeName:     "127.0.0.1",
										Orchestrator: "k8s",
									},
								},
							},
						},
					},
				})

			}

			// Sanitize the actual events received to remove revision info and compare against those expected.
			syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)

			By("Deleting the RemoteClusterConfiguration")
			_, err = c.RemoteClusterConfigurations().Delete(ctx, "etcd-timeout", options.DeleteOptions{ResourceVersion: rcc.ResourceVersion})
			Expect(err).NotTo(HaveOccurred())

			By("Expecting the syncer to move quickly to in-sync")
			syncTester.ExpectStatusUpdate(api.InSync)

			By("Checking we received the delete event messages for the remote cluster")
			expectedDeleteEvents := []api.Update{
				{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: "etcd-timeout"},
					},
					UpdateType: api.UpdateTypeKVDeleted,
				},
			}
			syncTester.ExpectUpdates(expectedDeleteEvents, false)
		})
	})
})

// To test successful Remote Cluster connections, we use a local k8s cluster with a remote etcd cluster.
var _ = testutils.E2eDatastoreDescribe("Remote cluster syncer config manipulation tests", testutils.DatastoreK8s, func(localConfig apiconfig.CalicoAPIConfig) {
	ctx := context.Background()
	var err error
	var localClient clientv3.Interface
	var localBackend api.Client
	var syncer api.Syncer
	var syncTester *testutils.SyncerTester
	var filteredSyncerTester api.SyncerCallbacks
	var remoteClient clientv3.Interface
	var rcc *apiv3.RemoteClusterConfiguration
	var expectedEvents []api.Update
	var kvResources []model.KVPair

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

			By("Configuring the RemoteClusterConfiguration for the remote")
			rcc = &apiv3.RemoteClusterConfiguration{ObjectMeta: metav1.ObjectMeta{Name: "remote-cluster"}}
			rcc.Spec.DatastoreType = string(remoteConfig.Spec.DatastoreType)
			rcc.Spec.EtcdEndpoints = remoteConfig.Spec.EtcdEndpoints
			rcc.Spec.EtcdUsername = remoteConfig.Spec.EtcdUsername
			rcc.Spec.EtcdPassword = remoteConfig.Spec.EtcdPassword
			rcc.Spec.EtcdKeyFile = remoteConfig.Spec.EtcdKeyFile
			rcc.Spec.EtcdCertFile = remoteConfig.Spec.EtcdCertFile
			rcc.Spec.EtcdCACertFile = remoteConfig.Spec.EtcdCACertFile
			rcc, err = localClient.RemoteClusterConfigurations().Create(ctx, rcc, options.SetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Keep track of the set of events we will expect from the Felix syncer. Start with the remote
			// cluster status updates as the connection succeeds.
			expectedEvents = []api.Update{
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
			wep := libapiv3.NewWorkloadEndpoint()
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
					Kind:      libapiv3.KindWorkloadEndpoint,
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
			kvResources = []model.KVPair{
				{
					Key: wepKey,
				},
			}
		})

		AfterEach(func() {
			if syncer != nil {
				syncer.Stop()
				syncer = nil
			}
			if localBackend != nil {
				localBackend.Clean()
				localBackend.Close()
				localBackend = nil
			}
		})

		Describe("Should connect to the remote cluster and sync the remote data", func() {
			BeforeEach(func() {
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
				up := updateprocessors.NewHostEndpointUpdateProcessor()
				kvps, err := up.Process(&model.KVPair{
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
				kvResources = append(kvResources, model.KVPair{
					Key: hepKey,
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
				kvResources = append(kvResources, []model.KVPair{
					{
						Key: model.ProfileLabelsKey{
							ProfileKey: model.ProfileKey{Name: "remote-cluster/profile-1"},
						},
					},
					{
						Key: model.ProfileRulesKey{
							ProfileKey: model.ProfileKey{Name: "remote-cluster/profile-1"},
						},
					},
				}...)

				By("Creating and starting a syncer")
				syncTester = testutils.NewSyncerTester()
				filteredSyncerTester = NewValidationFilter(syncTester)
				syncer = felixsyncer.New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
				syncer.Start()

				By("Checking status is updated to sync'd at start of day")
				syncTester.ExpectStatusUpdate(api.WaitForDatastore)
				syncTester.ExpectStatusUpdate(api.ResyncInProgress)
				syncTester.ExpectStatusUpdate(api.InSync)

				By("Checking we received the expected events")
				if localConfig.Spec.DatastoreType == apiconfig.Kubernetes {
					// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment, so include
					// those events too.
					for _, r := range defaultKubernetesResource() {
						expectedEvents = append(expectedEvents, api.Update{
							KVPair:     r,
							UpdateType: api.UpdateTypeKVNew,
						})
					}

					// Add an expected event for the node object.
					expectedEvents = append(expectedEvents, api.Update{
						UpdateType: api.UpdateTypeKVNew,
						KVPair: model.KVPair{
							Key: model.ResourceKey{Name: "127.0.0.1", Kind: "Node"},
							Value: &libapiv3.Node{
								Spec: libapiv3.NodeSpec{
									OrchRefs: []libapiv3.OrchRef{
										{
											NodeName:     "127.0.0.1",
											Orchestrator: "k8s",
										},
									},
								},
							},
						},
					})
				}
			})
			It("Should receive updates for resources created before the syncer is running", func() {
				// Sanitize the actual events received to remove revision info and compare against those expected.
				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)
			})

			It("Should receive delete event when removing the remote cluster after it is synced", func() {
				By("Checking we received the expected events")
				// Sanitize the actual events received to remove revision info and compare against those expected.
				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)

				By("Deleting the remote cluster configuration")
				_, outError := localClient.RemoteClusterConfigurations().Delete(
					ctx, rcc.Name, options.DeleteOptions{ResourceVersion: rcc.ResourceVersion},
				)
				Expect(outError).NotTo(HaveOccurred())

				By("Checking we received the expected events")

				expectedDeleteUpdates := []api.Update{{
					KVPair:     model.KVPair{Key: model.RemoteClusterStatusKey{Name: "remote-cluster"}},
					UpdateType: api.UpdateTypeKVDeleted,
				}}
				for _, kv := range kvResources {
					expectedDeleteUpdates = append(expectedDeleteUpdates, api.Update{
						KVPair:     kv,
						UpdateType: api.UpdateTypeKVDeleted,
					})
				}

				// Sanitize the actual events received to remove revision info and compare against those expected.
				syncTester.ExpectUpdatesSanitized(expectedDeleteUpdates, false, commonSanitizer)
			})
		})

		Describe("Should send the correct status updates when the RCC config is modified", func() {
			BeforeEach(func() {
				By("Creating and starting a syncer")
				syncTester = testutils.NewSyncerTester()
				filteredSyncerTester = NewValidationFilter(syncTester)
				syncer = felixsyncer.New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
				syncer.Start()

				By("Checking status is updated to sync'd at start of day")
				syncTester.ExpectStatusUpdate(api.WaitForDatastore)
				syncTester.ExpectStatusUpdate(api.ResyncInProgress)
				syncTester.ExpectStatusUpdate(api.InSync)

				By("Checking we received the expected events")
				// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment, so include
				// those events too.
				for _, r := range defaultKubernetesResource() {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}

				// Add an expected event for the node object.
				expectedEvents = append(expectedEvents, api.Update{
					UpdateType: api.UpdateTypeKVNew,
					KVPair: model.KVPair{
						Key: model.ResourceKey{Name: "127.0.0.1", Kind: "Node"},
						Value: &libapiv3.Node{
							Spec: libapiv3.NodeSpec{
								OrchRefs: []libapiv3.OrchRef{
									{
										NodeName:     "127.0.0.1",
										Orchestrator: "k8s",
									},
								},
							},
						},
					},
				})

				// Sanitize the actual events received to remove revision info and compare against those expected.
				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)
			})
			It("should send no updates when reapplying the RCC", func() {

				By("Updating the remote cluster config (but without changing the syncer connection config)")
				// Just re-apply the last settings.
				var outError error
				rcc, outError = localClient.RemoteClusterConfigurations().Update(ctx, rcc, options.SetOptions{})
				Expect(outError).NotTo(HaveOccurred())

				By("Checking we received no events")
				syncTester.ExpectUpdates([]api.Update{}, false)

			})
			It("should send restart required when RCC is changed", func() {
				By("Updating the remote cluster config (changing the syncer connection config)")
				rcc.Spec.EtcdUsername = "fakeusername"
				var err error
				rcc, err = localClient.RemoteClusterConfigurations().Update(ctx, rcc, options.SetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Checking we received a restart required event")
				expectedEvents = []api.Update{
					{
						KVPair: model.KVPair{
							Key: model.RemoteClusterStatusKey{Name: rcc.Name},
							Value: &model.RemoteClusterStatus{
								Status: model.RemoteClusterConfigChangeRestartRequired,
							},
						},
						UpdateType: api.UpdateTypeKVUpdated,
					},
				}

				// Sanitize the actual events received to remove revision info and compare against those expected.
				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)
			})
			It("should send invalid status and delete events when config is changed to an invalid one", func() {
				By("Updating the remote cluster config so that it is no longer valid for the syncer")
				// To update the RCC so that it is no longer valid, we access the backend client and set a rogue datastore
				// type. The felix syncer will not recognize this and return the RCC as invalid. The remote cluster
				// processing will treat this as a delete.
				rcc.Spec.DatastoreType = "this-is-not-valid"
				_, outError := localBackend.Update(ctx, &model.KVPair{
					Key: model.ResourceKey{
						Name: rcc.Name,
						Kind: apiv3.KindRemoteClusterConfiguration,
					},
					Value:    rcc,
					Revision: rcc.ResourceVersion,
				})
				Expect(outError).NotTo(HaveOccurred())

				By("Checking we received an update event")
				expectedEvents = []api.Update{
					{
						KVPair: model.KVPair{
							Key: model.RemoteClusterStatusKey{Name: rcc.Name},
							Value: &model.RemoteClusterStatus{
								Status: model.RemoteClusterConfigIncomplete,
								Error:  "Config is incomplete, stopping watch remote",
							},
						},
						UpdateType: api.UpdateTypeKVUpdated,
					},
				}
				for _, kv := range kvResources {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     kv,
						UpdateType: api.UpdateTypeKVDeleted,
					})
				}

				// Sanitize the actual events received to remove revision info and compare against those expected.
				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)

			})
			It("should send invalid status and delete events when config is changed to an invalid one", func() {
				By("Deleting the remote cluster")
				_, err := localClient.RemoteClusterConfigurations().Delete(
					ctx, rcc.Name, options.DeleteOptions{ResourceVersion: rcc.ResourceVersion},
				)
				Expect(err).NotTo(HaveOccurred())

				By("Checking we received an update event")
				expectedEvents = []api.Update{
					{
						KVPair: model.KVPair{
							Key: model.RemoteClusterStatusKey{Name: rcc.Name},
						},
						UpdateType: api.UpdateTypeKVDeleted,
					},
				}
				for _, kv := range kvResources {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     kv,
						UpdateType: api.UpdateTypeKVDeleted,
					})
				}

				By("Checking we receive no events")
				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)
			})
		})
		Describe("should only see restart callbacks when the appropriate config change happens", func() {
			var restartMonitor *remotecluster.RestartMonitor
			var restartCallbackCalled bool
			var restartCallbackMsg string
			BeforeEach(func() {
				By("Creating and starting a syncer")
				syncTester = testutils.NewSyncerTester()
				filteredSyncerTester = NewValidationFilter(syncTester)
				restartCallbackCalled = false
				restartCallbackMsg = ""
				restartMonitor = remotecluster.NewRemoteClusterRestartMonitor(filteredSyncerTester, func(reason string) {
					restartCallbackCalled = true
					restartCallbackMsg = reason
				})
				syncer = felixsyncer.New(localBackend, localConfig.Spec, restartMonitor, false, true)
				syncer.Start()

				By("Checking status is updated to sync'd at start of day")
				syncTester.ExpectStatusUpdate(api.WaitForDatastore)
				syncTester.ExpectStatusUpdate(api.ResyncInProgress)
				syncTester.ExpectStatusUpdate(api.InSync)

				By("Checking we received the expected events")
				// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment, so include
				// those events too.
				for _, r := range defaultKubernetesResource() {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}

				// Add an expected event for the node object.
				expectedEvents = append(expectedEvents, api.Update{
					UpdateType: api.UpdateTypeKVNew,
					KVPair: model.KVPair{
						Key: model.ResourceKey{Name: "127.0.0.1", Kind: "Node"},
						Value: &libapiv3.Node{
							Spec: libapiv3.NodeSpec{
								OrchRefs: []libapiv3.OrchRef{
									{
										NodeName:     "127.0.0.1",
										Orchestrator: "k8s",
									},
								},
							},
						},
					},
				})

				// Sanitize the actual events received to remove revision info and compare against those expected.
				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)
			})
			It("should see no callback when reapplying the RCC", func() {

				By("Updating the remote cluster config (but without changing the syncer connection config)")
				// Just re-apply the last settings.
				var outError error
				rcc, outError = localClient.RemoteClusterConfigurations().Update(ctx, rcc, options.SetOptions{})
				Expect(outError).NotTo(HaveOccurred())

				By("Checking we received no events")
				syncTester.ExpectUpdates([]api.Update{}, false)

				Expect(restartCallbackCalled).To(BeFalse())
			})
			It("should see restart callback when RCC is changed", func() {
				By("Updating the remote cluster config (changing the syncer connection config)")
				rcc.Spec.EtcdUsername = "fakeusername"
				var err error
				rcc, err = localClient.RemoteClusterConfigurations().Update(ctx, rcc, options.SetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Checking we received a restart required event")
				expectedEvents = []api.Update{
					{
						KVPair: model.KVPair{
							Key: model.RemoteClusterStatusKey{Name: rcc.Name},
							Value: &model.RemoteClusterStatus{
								Status: model.RemoteClusterConfigChangeRestartRequired,
							},
						},
						UpdateType: api.UpdateTypeKVUpdated,
					},
				}

				// Sanitize the actual events received to remove revision info and compare against those expected.
				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)

				By("Expecting that the RestartMonitor is appropriately calling back")
				Expect(restartCallbackCalled).To(BeTrue())
				Expect(restartCallbackMsg).NotTo(BeEmpty())
			})
			It("should see no callback when config is changed to an invalid one", func() {
				By("Deleting the remote cluster")
				_, err := localClient.RemoteClusterConfigurations().Delete(
					ctx, rcc.Name, options.DeleteOptions{ResourceVersion: rcc.ResourceVersion},
				)
				Expect(err).NotTo(HaveOccurred())

				By("Checking we received an update event")
				expectedEvents = []api.Update{
					{
						KVPair: model.KVPair{
							Key: model.RemoteClusterStatusKey{Name: rcc.Name},
						},
						UpdateType: api.UpdateTypeKVDeleted,
					},
				}
				for _, kv := range kvResources {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     kv,
						UpdateType: api.UpdateTypeKVDeleted,
					})
				}

				syncTester.ExpectUpdatesSanitized(expectedEvents, false, commonSanitizer)
				Expect(restartCallbackCalled).To(BeFalse())
			})
		})
	})
})

var _ = testutils.E2eDatastoreDescribe("Remote cluster syncer datastore config tests", testutils.DatastoreEtcdV3, func(etcdConfig apiconfig.CalicoAPIConfig) {
	testutils.E2eDatastoreDescribe("", testutils.DatastoreK8s, func(k8sConfig apiconfig.CalicoAPIConfig) {
		rccName := "remote-cluster"
		rccSecretName := "remote-cluster-config"

		var ctx context.Context
		var err error
		var localClient clientv3.Interface
		var localBackend api.Client
		var remoteBackend api.Client
		var syncer api.Syncer
		var syncTester *testutils.SyncerTester
		var filteredSyncerTester api.SyncerCallbacks
		var remoteClient clientv3.Interface
		var k8sClientset *kubernetes.Clientset
		var remoteConfig apiconfig.CalicoAPIConfig
		var localConfig apiconfig.CalicoAPIConfig
		var expectedEvents []api.Update

		BeforeEach(func() {
			k8sClient, err := clientv3.New(k8sConfig)
			Expect(err).NotTo(HaveOccurred())
			_, _ = k8sClient.HostEndpoints().Delete(context.Background(), "hep1", options.DeleteOptions{})
			etcdClient, err := clientv3.New(etcdConfig)
			Expect(err).NotTo(HaveOccurred())
			_, _ = etcdClient.HostEndpoints().Delete(context.Background(), "hep1", options.DeleteOptions{})
		})
		AfterEach(func() {
			By("doing aftereach")
			if syncer != nil {
				syncer.Stop()
				syncer = nil
			}
			if localBackend != nil {
				localBackend.Clean()
				localBackend.Close()
				localBackend = nil
			}
			if remoteBackend != nil {
				remoteBackend.Clean()
				remoteBackend.Close()
				remoteBackend = nil
			}
			if k8sClientset != nil {
				_ = k8sClientset.CoreV1().Secrets("namespace-1").Delete(ctx, rccSecretName, metav1.DeleteOptions{})
			}
			if remoteClient != nil {
				_, _ = remoteClient.HostEndpoints().Delete(ctx, "hep1", options.DeleteOptions{})
			}
			By("done with aftereach")
		})

		setup := func(local, remote apiconfig.CalicoAPIConfig) {
			localConfig = local
			remoteConfig = remote
			ctx = context.Background()
			log.SetLevel(log.DebugLevel)
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
			remoteBackend, err = backend.NewClient(remoteConfig)
			Expect(err).NotTo(HaveOccurred())
			remoteBackend.Clean()

			k8sBackend, err := backend.NewClient(k8sConfig)
			Expect(err).NotTo(HaveOccurred())
			k8sClientset = k8sBackend.(*k8s.KubeClient).ClientSet
			expectedEvents = []api.Update{}
		}

		sanitizer := func(callersUpdate *api.Update) *api.Update {
			u := *callersUpdate
			u.Revision = ""
			u.TTL = 0

			switch key := u.KVPair.Key.(type) {
			case model.RemoteClusterStatusKey:
				log.Infof("RemoteClusterStatusKey = %v, type = %v", key, u.UpdateType)
				return &u
			case model.HostEndpointKey:
				log.Infof("HostEndpointKey = %v", key)
				return &u
			default:
				log.Infof("default = %v", key)
			}

			return nil
		}

		rccInitialEvents := []api.Update{
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

		type createFunc func()
		type modifyFunc func()

		// createRCCDirect creates an RCC with the configuration in the RCC from the remoteConfig
		createRCCDirect := func() {
			By("Creating direct RemoteClusterConfiguration")
			rcc := &apiv3.RemoteClusterConfiguration{ObjectMeta: metav1.ObjectMeta{Name: rccName}}
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
			rcc.Spec.KubeconfigInline = remoteConfig.Spec.KubeconfigInline
			_, outError := localClient.RemoteClusterConfigurations().Create(ctx, rcc, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
		}

		// modifyRCCDirect updates the already created RCC with a change, not creating a valid config
		// but that should be fine for using it to test what happens when a valid RCC is updated
		modifyRCCDirect := func() {
			By("Modifying direct RemoteClusterConfiguration")
			rcc, err := localClient.RemoteClusterConfigurations().Get(ctx, rccName, options.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			if remoteConfig.Spec.DatastoreType == apiconfig.Kubernetes {
				rcc.Spec.Kubeconfig = "notreal"
			} else {
				rcc.Spec.EtcdUsername = "fakeusername"
			}
			_, err = localClient.RemoteClusterConfigurations().Update(ctx, rcc, options.SetOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

		// createRCCSecret creates a secret and an RCC that references the secret, both based on the remoteConfig
		createRCCSecret := func() {
			By("Creating secret for the RemoteClusterConfiguration")
			_, err = k8sClientset.CoreV1().Secrets("namespace-1").Create(ctx,
				&kapiv1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: rccSecretName, Namespace: "namespace-1"},
					StringData: map[string]string{
						"datastoreType": string(remoteConfig.Spec.DatastoreType),
						"kubeconfig":    remoteConfig.Spec.KubeconfigInline,
						"etcdEndpoints": remoteConfig.Spec.EtcdEndpoints,
						"etcdUsername":  remoteConfig.Spec.EtcdUsername,
						"etcdPassword":  remoteConfig.Spec.EtcdPassword,
						"etcdKey":       remoteConfig.Spec.EtcdKey,
						"etcdCert":      remoteConfig.Spec.EtcdCert,
						"etcdCACert":    remoteConfig.Spec.EtcdCACert,
					},
				},
				metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Configuring the RemoteClusterConfiguration referencing secret")
			rcc := &apiv3.RemoteClusterConfiguration{ObjectMeta: metav1.ObjectMeta{Name: rccName}}
			rcc.Spec.ClusterAccessSecret = &kapiv1.ObjectReference{
				Kind:      reflect.TypeOf(kapiv1.Secret{}).String(),
				Namespace: "namespace-1",
				Name:      rccSecretName,
			}
			_, outError := localClient.RemoteClusterConfigurations().Create(ctx, rcc, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())
		}

		// modifyRCCSecret modifies an already created Secret that is referenced by an RCC
		modifyRCCSecret := func() {
			By("Modifying RCC Secret RemoteClusterConfiguration")
			s, err := k8sClientset.CoreV1().Secrets("namespace-1").Get(ctx, rccSecretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			if remoteConfig.Spec.DatastoreType == apiconfig.Kubernetes {
				s.StringData = map[string]string{"kubeconfig": "notreal"}
			} else {
				s.StringData = map[string]string{"etcdPassword": "fakeusername"}
			}
			_, err = k8sClientset.CoreV1().Secrets("namespace-1").Update(ctx, s, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

		// addHep creates a hep on the remoteClient and adds the expected events to expectedEvents
		// for later checking
		addHep := func() {
			// Keep track of the set of events we will expect from the Felix syncer. Start with the remote
			// cluster status updates as the connection succeeds.
			By("Creating a HEP")
			_, err = remoteClient.HostEndpoints().Create(ctx, &apiv3.HostEndpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "hep1"},
				Spec: apiv3.HostEndpointSpec{
					Node:          "node-hep",
					InterfaceName: "eth1",
				},
			}, options.SetOptions{})
			Expect(err).NotTo(HaveOccurred())
			expectedEvents = append(expectedEvents, api.Update{
				KVPair: model.KVPair{
					Key: model.HostEndpointKey{
						Hostname:   "remote-cluster/node-hep",
						EndpointID: "hep1",
					},
					Value: &model.HostEndpoint{
						Name:              "eth1",
						ExpectedIPv4Addrs: nil,
						ExpectedIPv6Addrs: nil,
						Labels:            nil,
						ProfileIDs:        nil,
						Ports:             []model.EndpointPort{},
					},
				},
				UpdateType: api.UpdateTypeKVNew,
			})
			// We only get the local event if the local config is also Kubernetes
			if localConfig.Spec.DatastoreType == apiconfig.Kubernetes {
				expectedEvents = append(expectedEvents, api.Update{
					KVPair: model.KVPair{
						Key: model.HostEndpointKey{
							Hostname:   "node-hep",
							EndpointID: "hep1",
						},
						Value: &model.HostEndpoint{
							Name:              "eth1",
							ExpectedIPv4Addrs: nil,
							ExpectedIPv6Addrs: nil,
							Labels:            nil,
							ProfileIDs:        nil,
							Ports:             []model.EndpointPort{},
						},
					},
					UpdateType: api.UpdateTypeKVNew,
				})
			}
		}

		type TestCfg struct {
			name      string
			localCfg  apiconfig.CalicoAPIConfig
			remoteCfg apiconfig.CalicoAPIConfig
			create    createFunc
			modify    modifyFunc
		}

		for _, tc := range []TestCfg{
			{
				name:      "local K8s with remote etcd direct config",
				localCfg:  k8sConfig,
				remoteCfg: etcdConfig,
				create:    createRCCDirect,
				modify:    modifyRCCDirect,
			},
			{
				name:      "local K8s with remote etcd secret config",
				localCfg:  k8sConfig,
				remoteCfg: etcdConfig,
				create:    createRCCSecret,
				modify:    modifyRCCSecret,
			},
			{
				name:      "local K8s with remote k8s direct config",
				localCfg:  k8sConfig,
				remoteCfg: k8sConfig,
				create:    createRCCDirect,
				modify:    modifyRCCDirect,
			},

			/* TODO: Add OSS test cases.
				   https://tigera.slack.com/archives/CC08CBB43/p1626427401360200
			// remoteCfg: k8sConfig,
			// create: createRCCSecret
			// This combination is not possible because there is no
			// way to provide k8s config in a secret that is not inline.
			{
				name:      "local K8s with remote k8s inline direct config",
				localCfg:  k8sConfig,
				remoteCfg: k8sInline,
				create:    createRCCDirect,
				modify:    modifyRCCDirect,
			},
			{
				name:      "local K8s with remote k8s inline secret config",
				localCfg:  k8sConfig,
				remoteCfg: k8sInline,
				create:    createRCCSecret,
				modify:    modifyRCCSecret,
			},
			{
				name:      "local etcd with remote k8s inline secret config",
				localCfg:  etcdConfig,
				remoteCfg: k8sInline,
				create:    createRCCSecret,
				modify:    modifyRCCSecret,
			},
			*/
		} {
			Describe("Events are received with "+tc.name, func() {
				It("should get restart message when valid config is changed", func() {
					setup(tc.localCfg, tc.remoteCfg)
					tc.create()

					expectedEvents = append(expectedEvents, rccInitialEvents...)
					addHep()

					By("Creating and starting a syncer")
					syncTester = testutils.NewSyncerTester()
					filteredSyncerTester = NewValidationFilter(syncTester)

					os.Setenv("KUBERNETES_MASTER", k8sConfig.Spec.K8sAPIEndpoint)
					syncer = felixsyncer.New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
					defer os.Unsetenv("KUBERNETES_MASTER")
					syncer.Start()

					By("Checking status is updated to sync'd at start of day")
					syncTester.ExpectStatusUpdate(api.WaitForDatastore)
					syncTester.ExpectStatusUpdate(api.ResyncInProgress)
					syncTester.ExpectStatusUpdate(api.InSync)

					By("Checking we received the expected events")
					// Sanitize the actual events received to remove revision info and compare against those expected.
					syncTester.ExpectUpdatesSanitized(expectedEvents, false, sanitizer)

					By("Modifying the RCC/Secret config")
					tc.modify()
					By("Checking we received a restart required event")
					expectedEvents = []api.Update{
						{
							KVPair: model.KVPair{
								Key: model.RemoteClusterStatusKey{Name: rccName},
								Value: &model.RemoteClusterStatus{
									Status: model.RemoteClusterConfigChangeRestartRequired,
								},
							},
							UpdateType: api.UpdateTypeKVUpdated,
						},
					}

					// Sanitize the actual events received to remove revision info and compare against those expected.
					syncTester.ExpectUpdatesSanitized(expectedEvents, false, sanitizer)
					By("done with It")
				})
				It("Should receive events for resources created before starting syncer", func() {
					//runtime.Breakpoint()
					setup(tc.localCfg, tc.remoteCfg)
					tc.create()

					expectedEvents = append(expectedEvents, rccInitialEvents...)
					addHep()

					By("Creating and starting a syncer")
					syncTester = testutils.NewSyncerTester()
					filteredSyncerTester = NewValidationFilter(syncTester)

					os.Setenv("KUBERNETES_MASTER", k8sConfig.Spec.K8sAPIEndpoint)
					syncer = felixsyncer.New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
					defer os.Unsetenv("KUBERNETES_MASTER")
					syncer.Start()

					By("Checking status is updated to sync'd at start of day")
					syncTester.ExpectStatusUpdate(api.WaitForDatastore)
					syncTester.ExpectStatusUpdate(api.ResyncInProgress)
					syncTester.ExpectStatusUpdate(api.InSync)

					By("Checking we received the expected events")
					// Sanitize the actual events received to remove revision info and compare against those expected.
					syncTester.ExpectUpdatesSanitized(expectedEvents, false, sanitizer)
					By("done with It")
				})
				It("Should receive events for resources created after starting syncer", func() {
					setup(tc.localCfg, tc.remoteCfg)
					tc.create()

					By("Creating and starting a syncer")
					syncTester = testutils.NewSyncerTester()
					filteredSyncerTester = NewValidationFilter(syncTester)

					os.Setenv("KUBERNETES_MASTER", k8sConfig.Spec.K8sAPIEndpoint)
					syncer = felixsyncer.New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
					defer os.Unsetenv("KUBERNETES_MASTER")
					syncer.Start()

					By("Checking status is updated to sync'd at start of day")
					syncTester.ExpectStatusUpdate(api.WaitForDatastore)
					syncTester.ExpectStatusUpdate(api.ResyncInProgress)
					syncTester.ExpectStatusUpdate(api.InSync)

					By("Checking we received the expected events")
					// Sanitize the actual events received to remove revision info and compare against those expected.
					syncTester.ExpectUpdatesSanitized(rccInitialEvents, false, sanitizer)

					expectedEvents = []api.Update{}
					addHep()
					syncTester.ExpectUpdatesSanitized(expectedEvents, false, sanitizer)
					By("done with It")
				})
			})
		}
	})
})
