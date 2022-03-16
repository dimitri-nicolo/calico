// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

package felixsyncer_test

import (
	"context"
	"os"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	libapiv3 "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/backend"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/k8s"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	. "github.com/projectcalico/calico/libcalico-go/lib/backend/syncersv1/felixsyncer"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/syncersv1/remotecluster"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

var _ = testutils.E2eDatastoreDescribe("Remote cluster syncer tests - connection failures", testutils.DatastoreAll, func(config apiconfig.CalicoAPIConfig) {

	ctx := context.Background()
	var err error
	var c clientv3.Interface
	var be api.Client
	var syncer api.Syncer
	var syncTester *testutils.SyncerTester
	var filteredSyncerTester api.SyncerCallbacks
	var cs kubernetes.Interface

	BeforeEach(func() {
		// Create the v3 client
		c, err = clientv3.New(config)
		Expect(err).NotTo(HaveOccurred())

		// Create the backend client to clean the datastore and obtain a syncer interface.
		be, err = backend.NewClient(config)
		Expect(err).NotTo(HaveOccurred())
		be.Clean()

		// build k8s clientset.
		cfg, err := clientcmd.BuildConfigFromFlags("", "/kubeconfig.yaml")
		Expect(err).NotTo(HaveOccurred())
		cs = kubernetes.NewForConfigOrDie(cfg)
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
		func(name string, spec apiv3.RemoteClusterConfigurationSpec,
			errPrefix string, connTimeout time.Duration) {
			By("Creating the RemoteClusterConfiguration")
			_, outError := c.RemoteClusterConfigurations().Create(ctx, &apiv3.RemoteClusterConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: spec,
			}, options.SetOptions{})
			Expect(outError).NotTo(HaveOccurred())

			By("Creating and starting a syncer")
			syncTester = testutils.NewSyncerTester()
			filteredSyncerTester = NewRemoteClusterConnFailedFilter(NewValidationFilter(syncTester))
			syncer = New(be, config.Spec, filteredSyncerTester, false, true)
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
			}
			defaultCacheEntries := calculateDefaultFelixSyncerEntries(cs, config.Spec.DatastoreType)
			for _, r := range defaultCacheEntries {
				expectedEvents = append(expectedEvents, api.Update{
					KVPair:     r,
					UpdateType: api.UpdateTypeKVNew,
				})
			}

			expectedEvents = append(expectedEvents, api.Update{
				KVPair: model.KVPair{
					Key: model.RemoteClusterStatusKey{Name: name},
					Value: &model.RemoteClusterStatus{
						Status: model.RemoteClusterConnectionFailed,
						Error:  errPrefix,
					},
				},
				UpdateType: api.UpdateTypeKVUpdated,
			})

			err = syncTester.HasUpdates(expectedEvents, false)
			Expect(err).NotTo(HaveOccurred())
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
			".*context deadline exceeded.*", 15*time.Second,
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
			".*could not initialize etcdv3 client: open foo: no such file or directory.*", 3*time.Second,
		),

		Entry("invalid k8s endpoint", "bad-k8s-1",
			apiv3.RemoteClusterConfigurationSpec{
				DatastoreType: "kubernetes",
				KubeConfig: apiv3.KubeConfig{
					K8sAPIEndpoint: "http://foobarbaz:1000",
				},
			},
			".*dial tcp: lookup foobarbaz on.*", 15*time.Second,
		),

		Entry("invalid k8s kubeconfig file - retry for 30s before failing connection", "bad-k8s2",
			apiv3.RemoteClusterConfigurationSpec{
				DatastoreType: "kubernetes",
				KubeConfig: apiv3.KubeConfig{
					Kubeconfig: "foobarbaz",
				},
			},
			".*stat foobarbaz: no such file or directory.*", 15*time.Second,
		),
	)

	Describe("Deleting a RemoteClusterConfiguration before connection fails", func() {
		It("should get a delete event without getting a connection failure event", func() {
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
			filteredSyncerTester = NewRemoteClusterConnFailedFilter(NewValidationFilter(syncTester))
			syncer = New(be, config.Spec, filteredSyncerTester, false, true)
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

			defaultCacheEntries := calculateDefaultFelixSyncerEntries(cs, config.Spec.DatastoreType)
			for _, r := range defaultCacheEntries {
				expectedEvents = append(expectedEvents, api.Update{
					KVPair:     r,
					UpdateType: api.UpdateTypeKVNew,
				})
			}

			syncTester.ExpectUpdates(expectedEvents, false)

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
	var deleteEvents []api.Update
	var cs kubernetes.Interface

	testutils.E2eDatastoreDescribe("Successful connection to cluster", testutils.DatastoreEtcdV3, func(remoteConfig apiconfig.CalicoAPIConfig) {
		BeforeEach(func() {
			// Create the v3 clients for the local and remote clusters.
			localClient, err = clientv3.New(localConfig)
			Expect(err).NotTo(HaveOccurred())
			remoteClient, err = clientv3.New(remoteConfig)
			Expect(err).NotTo(HaveOccurred())

			// build k8s clientset.
			cfg, err := clientcmd.BuildConfigFromFlags("", "/kubeconfig.yaml")
			Expect(err).NotTo(HaveOccurred())
			cs = kubernetes.NewForConfigOrDie(cfg)

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
			deleteEvents = []api.Update{
				{
					KVPair: model.KVPair{
						Key:   wepKey,
						Value: nil,
					},
					UpdateType: api.UpdateTypeKVDeleted,
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
				deleteEvents = append(deleteEvents, api.Update{
					KVPair: model.KVPair{
						Key:   hepKey,
						Value: nil,
					},
					UpdateType: api.UpdateTypeKVDeleted,
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

				// Add the remote profiles to the events - doing this by hand is simpler (although arguably not as
				// maintainable).
				expectedEvents = append(expectedEvents,
					api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileRulesKey{
								ProfileKey: model.ProfileKey{Name: "remote-cluster/projectcalico-default-allow"},
							},
							Value: &model.ProfileRules{},
						},
						UpdateType: api.UpdateTypeKVNew,
					}, api.Update{
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Kind: apiv3.KindProfile,
								Name: "remote-cluster/projectcalico-default-allow",
							},
							Value: &apiv3.Profile{
								TypeMeta: metav1.TypeMeta{
									Kind: apiv3.KindProfile,
								},
								ObjectMeta: metav1.ObjectMeta{
									Name: "projectcalico-default-allow",
								},
							},
						},
						UpdateType: api.UpdateTypeKVNew,
					}, api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileRulesKey{
								ProfileKey: model.ProfileKey{Name: "remote-cluster/profile-1"},
							},
							Value: &model.ProfileRules{},
						},
						UpdateType: api.UpdateTypeKVNew,
					}, api.Update{
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Kind: apiv3.KindProfile,
								Name: "remote-cluster/profile-1",
							},
							Value: &apiv3.Profile{
								TypeMeta: metav1.TypeMeta{
									Kind: apiv3.KindProfile,
								},
								ObjectMeta: metav1.ObjectMeta{
									Name: "profile-1",
								},
								Spec: apiv3.ProfileSpec{
									LabelsToApply: map[string]string{
										"label1": "value1",
										"label2": "value2",
									},
								},
							},
						},
						UpdateType: api.UpdateTypeKVNew,
					},
					api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileLabelsKey{
								ProfileKey: model.ProfileKey{Name: "remote-cluster/profile-1"},
							},
							Value: pro.Spec.LabelsToApply,
						},
						UpdateType: api.UpdateTypeKVNew,
					},
				)
				deleteEvents = append(deleteEvents,
					api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileRulesKey{
								ProfileKey: model.ProfileKey{Name: "remote-cluster/projectcalico-default-allow"},
							},
							Value: nil,
						},
						UpdateType: api.UpdateTypeKVDeleted,
					}, api.Update{
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Kind: apiv3.KindProfile,
								Name: "remote-cluster/projectcalico-default-allow",
							},
							Value: nil,
						},
						UpdateType: api.UpdateTypeKVDeleted,
					}, api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileRulesKey{
								ProfileKey: model.ProfileKey{Name: "remote-cluster/profile-1"},
							},
							Value: nil,
						},
						UpdateType: api.UpdateTypeKVDeleted,
					}, api.Update{
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Kind: apiv3.KindProfile,
								Name: "remote-cluster/profile-1",
							},
							Value: nil,
						},
						UpdateType: api.UpdateTypeKVDeleted,
					},
					api.Update{
						KVPair: model.KVPair{
							Key: model.ProfileLabelsKey{
								ProfileKey: model.ProfileKey{Name: "remote-cluster/profile-1"},
							},
							Value: nil,
						},
						UpdateType: api.UpdateTypeKVDeleted,
					},
				)

				By("Creating and starting a syncer")
				syncTester = testutils.NewSyncerTester()
				filteredSyncerTester = NewRemoteClusterConnFailedFilter(NewValidationFilter(syncTester))
				syncer = New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
				syncer.Start()

				By("Checking status is updated to sync'd at start of day")
				syncTester.ExpectStatusUpdate(api.WaitForDatastore)
				syncTester.ExpectStatusUpdate(api.ResyncInProgress)
				syncTester.ExpectStatusUpdate(api.InSync)

				By("Checking we received the expected events")
				defaultCacheEntries := calculateDefaultFelixSyncerEntries(cs, localConfig.Spec.DatastoreType)
				for _, r := range defaultCacheEntries {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}
			})
			It("Should receive updates for resources created before the syncer is running", func() {
				syncTester.ExpectUpdates(expectedEvents, false)
			})

			It("Should receive delete event when removing the remote cluster after it is synced", func() {
				By("Checking we received the expected events")
				syncTester.ExpectUpdates(expectedEvents, false)

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
				expectedDeleteUpdates = append(expectedDeleteUpdates, deleteEvents...)

				syncTester.ExpectUpdates(expectedDeleteUpdates, false)
			})
		})

		Describe("Should send the correct status updates when the RCC config is modified", func() {
			BeforeEach(func() {
				By("Creating and starting a syncer")
				syncTester = testutils.NewSyncerTester()
				filteredSyncerTester = NewRemoteClusterConnFailedFilter(NewValidationFilter(syncTester))
				syncer = New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
				syncer.Start()

				By("Checking status is updated to sync'd at start of day")
				syncTester.ExpectStatusUpdate(api.WaitForDatastore)
				syncTester.ExpectStatusUpdate(api.ResyncInProgress)
				syncTester.ExpectStatusUpdate(api.InSync)

				By("Checking we received the expected events")
				// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment, so include
				// those events too.
				defaultCacheEntries := calculateDefaultFelixSyncerEntries(cs, apiconfig.Kubernetes)
				for _, r := range defaultCacheEntries {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}

				// Add the remote events (and determine the deletion events associated with the remote cluster).
				defaultCacheEntries = calculateDefaultFelixSyncerEntries(cs, apiconfig.EtcdV3, "remote-cluster")
				for _, r := range defaultCacheEntries {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
					deleteEvents = append(deleteEvents,
						api.Update{
							KVPair: model.KVPair{
								Key:   r.Key,
								Value: nil,
							},
							UpdateType: api.UpdateTypeKVDeleted,
						},
					)
				}

				syncTester.ExpectUpdates(expectedEvents, false)
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

				syncTester.ExpectUpdates(expectedEvents, false)
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
				expectedEvents = append(expectedEvents, deleteEvents...)

				syncTester.ExpectUpdates(expectedEvents, false)
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
				expectedEvents = append(expectedEvents, deleteEvents...)

				By("Checking we receive no events")
				syncTester.ExpectUpdates(expectedEvents, false)
			})
		})

		Describe("should only see restart callbacks when the appropriate config change happens", func() {
			var restartMonitor *remotecluster.RestartMonitor
			var restartCallbackCalled bool
			var restartCallbackMsg string
			BeforeEach(func() {
				By("Creating and starting a syncer")
				syncTester = testutils.NewSyncerTester()
				filteredSyncerTester = NewRemoteClusterConnFailedFilter(NewValidationFilter(syncTester))
				restartCallbackCalled = false
				restartCallbackMsg = ""
				restartMonitor = remotecluster.NewRemoteClusterRestartMonitor(filteredSyncerTester, func(reason string) {
					restartCallbackCalled = true
					restartCallbackMsg = reason
				})
				syncer = New(localBackend, localConfig.Spec, restartMonitor, false, true)
				syncer.Start()

				By("Checking status is updated to sync'd at start of day")
				syncTester.ExpectStatusUpdate(api.WaitForDatastore)
				syncTester.ExpectStatusUpdate(api.ResyncInProgress)
				syncTester.ExpectStatusUpdate(api.InSync)

				By("Checking we received the expected events")
				// Kubernetes will have a bunch of resources that are pre-programmed in our e2e environment, so include
				// those events too.
				defaultCacheEntries := calculateDefaultFelixSyncerEntries(cs, apiconfig.Kubernetes)
				for _, r := range defaultCacheEntries {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
				}

				// Add the remote events (and determine the deletion events associated with the remote cluster).
				defaultCacheEntries = calculateDefaultFelixSyncerEntries(cs, apiconfig.EtcdV3, "remote-cluster")
				for _, r := range defaultCacheEntries {
					expectedEvents = append(expectedEvents, api.Update{
						KVPair:     r,
						UpdateType: api.UpdateTypeKVNew,
					})
					deleteEvents = append(deleteEvents,
						api.Update{
							KVPair: model.KVPair{
								Key:   r.Key,
								Value: nil,
							},
							UpdateType: api.UpdateTypeKVDeleted,
						},
					)
				}

				syncTester.ExpectUpdates(expectedEvents, false)
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

				syncTester.ExpectUpdates(expectedEvents, false)

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
				expectedEvents = append(expectedEvents, deleteEvents...)

				syncTester.ExpectUpdates(expectedEvents, false)
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
		var cs kubernetes.Interface
		var k8sInlineConfig apiconfig.CalicoAPIConfig

		BeforeEach(func() {
			k8sClient, err := clientv3.New(k8sConfig)
			Expect(err).NotTo(HaveOccurred())
			_, _ = k8sClient.HostEndpoints().Delete(context.Background(), "hep1", options.DeleteOptions{})
			etcdClient, err := clientv3.New(etcdConfig)
			Expect(err).NotTo(HaveOccurred())
			_, _ = etcdClient.HostEndpoints().Delete(context.Background(), "hep1", options.DeleteOptions{})

			// build k8s clientset.
			cfg, err := clientcmd.BuildConfigFromFlags("", "/kubeconfig.yaml")
			Expect(err).NotTo(HaveOccurred())
			cs = kubernetes.NewForConfigOrDie(cfg)

			// Get the k8s inline config that we use for testing.
			k8sInlineConfig = testutils.GetK8sInlineConfig()
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

		setup := func(local, remote *apiconfig.CalicoAPIConfig) {
			localConfig = *local
			remoteConfig = *remote
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

			// Get the default cache entries for the local cluster.
			defaultCacheEntries := calculateDefaultFelixSyncerEntries(cs, local.Spec.DatastoreType)
			for _, r := range defaultCacheEntries {
				expectedEvents = append(expectedEvents, api.Update{
					KVPair:     r,
					UpdateType: api.UpdateTypeKVNew,
				})
			}
			// Get the default entries for the remote cluster.
			defaultCacheEntries = calculateDefaultFelixSyncerEntries(cs, remote.Spec.DatastoreType, "remote-cluster")
			for _, r := range defaultCacheEntries {
				expectedEvents = append(expectedEvents, api.Update{
					KVPair:     r,
					UpdateType: api.UpdateTypeKVNew,
				})
			}
			// Add the RCC initial events.
			expectedEvents = append(expectedEvents, rccInitialEvents...)
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
			// We only get the local event if the local config is the same datastore (which we can tell from the
			// datastore type).
			if localConfig.Spec.DatastoreType == remoteConfig.Spec.DatastoreType {
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
			localCfg  *apiconfig.CalicoAPIConfig
			remoteCfg *apiconfig.CalicoAPIConfig
			create    createFunc
			modify    modifyFunc
		}

		for _, tcLoop := range []TestCfg{
			{
				name:      "local K8s with remote etcd direct config",
				localCfg:  &k8sConfig,
				remoteCfg: &etcdConfig,
				create:    createRCCDirect,
				modify:    modifyRCCDirect,
			},
			{
				name:      "local K8s with remote etcd secret config",
				localCfg:  &k8sConfig,
				remoteCfg: &etcdConfig,
				create:    createRCCSecret,
				modify:    modifyRCCSecret,
			},
			{
				name:      "local K8s with remote k8s direct config",
				localCfg:  &k8sConfig,
				remoteCfg: &k8sConfig,
				create:    createRCCDirect,
				modify:    modifyRCCDirect,
			},
			// remoteCfg: k8sConfig,
			// create: createRCCSecret
			// This combination is not possible because there is no
			// way to provide k8s config in a secret that is not inline.
			{
				name:      "local K8s with remote k8s inline direct config",
				localCfg:  &k8sConfig,
				remoteCfg: &k8sInlineConfig,
				create:    createRCCDirect,
				modify:    modifyRCCDirect,
			},
			{
				name:      "local K8s with remote k8s inline secret config",
				localCfg:  &k8sConfig,
				remoteCfg: &k8sInlineConfig,
				create:    createRCCSecret,
				modify:    modifyRCCSecret,
			},
			{
				name:      "local etcd with remote k8s inline secret config",
				localCfg:  &etcdConfig,
				remoteCfg: &k8sInlineConfig,
				create:    createRCCSecret,
				modify:    modifyRCCSecret,
			},
		} {
			// Local variable tc to ensure it's not updated for all tests.
			tc := tcLoop

			Describe("Events are received with "+tc.name, func() {
				It("should get restart message when valid config is changed", func() {
					setup(tc.localCfg, tc.remoteCfg)
					tc.create()

					addHep()

					By("Creating and starting a syncer")
					syncTester = testutils.NewSyncerTester()
					filteredSyncerTester = NewRemoteClusterConnFailedFilter(NewValidationFilter(syncTester))

					os.Setenv("KUBERNETES_MASTER", k8sConfig.Spec.K8sAPIEndpoint)
					syncer = New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
					defer os.Unsetenv("KUBERNETES_MASTER")
					syncer.Start()

					By("Checking status is updated to sync'd at start of day")
					syncTester.ExpectStatusUpdate(api.WaitForDatastore)
					syncTester.ExpectStatusUpdate(api.ResyncInProgress)
					syncTester.ExpectStatusUpdate(api.InSync)

					By("Checking we received the expected events")
					syncTester.ExpectUpdates(expectedEvents, false)

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

					syncTester.ExpectUpdates(expectedEvents, false)
					By("done with It")
				})
				It("Should receive events for resources created before starting syncer", func() {
					//runtime.Breakpoint()
					setup(tc.localCfg, tc.remoteCfg)
					tc.create()

					addHep()

					By("Creating and starting a syncer")
					syncTester = testutils.NewSyncerTester()
					filteredSyncerTester = NewRemoteClusterConnFailedFilter(NewValidationFilter(syncTester))

					os.Setenv("KUBERNETES_MASTER", k8sConfig.Spec.K8sAPIEndpoint)
					syncer = New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
					defer os.Unsetenv("KUBERNETES_MASTER")
					syncer.Start()

					By("Checking status is updated to sync'd at start of day")
					syncTester.ExpectStatusUpdate(api.WaitForDatastore)
					syncTester.ExpectStatusUpdate(api.ResyncInProgress)
					syncTester.ExpectStatusUpdate(api.InSync)

					By("Checking we received the expected events")
					syncTester.ExpectUpdates(expectedEvents, false)
					By("done with It")
				})
				It("Should receive events for resources created after starting syncer", func() {
					setup(tc.localCfg, tc.remoteCfg)
					tc.create()

					By("Creating and starting a syncer")
					syncTester = testutils.NewSyncerTester()
					filteredSyncerTester = NewRemoteClusterConnFailedFilter(NewValidationFilter(syncTester))

					os.Setenv("KUBERNETES_MASTER", k8sConfig.Spec.K8sAPIEndpoint)
					syncer = New(localBackend, localConfig.Spec, filteredSyncerTester, false, true)
					defer os.Unsetenv("KUBERNETES_MASTER")
					syncer.Start()

					By("Checking status is updated to sync'd at start of day")
					syncTester.ExpectStatusUpdate(api.WaitForDatastore)
					syncTester.ExpectStatusUpdate(api.ResyncInProgress)
					syncTester.ExpectStatusUpdate(api.InSync)

					By("Checking we received the expected events")
					syncTester.ExpectUpdates(expectedEvents, false)

					expectedEvents = []api.Update{}
					addHep()
					syncTester.ExpectUpdates(expectedEvents, false)
					By("done with It")
				})
			})
		}
	})
})

func NewRemoteClusterConnFailedFilter(sink api.SyncerCallbacks) *RemoteClusterConnFailedFilter {
	return &RemoteClusterConnFailedFilter{
		sink:    sink,
		handled: make(map[string]bool),
	}
}

type RemoteClusterConnFailedFilter struct {
	sink    api.SyncerCallbacks
	handled map[string]bool
}

func (r *RemoteClusterConnFailedFilter) OnStatusUpdated(status api.SyncStatus) {
	// Pass through.
	r.sink.OnStatusUpdated(status)
}

func (r *RemoteClusterConnFailedFilter) OnUpdates(updates []api.Update) {
	defer GinkgoRecover()

	filteredUpdates := make([]api.Update, 0, len(updates))
	for _, update := range updates {
		if k, ok := update.Key.(model.RemoteClusterStatusKey); ok {
			if v, ok := update.Value.(*model.RemoteClusterStatus); ok && v.Status == model.RemoteClusterConnectionFailed {
				// Only include 1 remote cluster failed update per remote cluster.
				if r.handled[k.Name] {
					continue
				}
				r.handled[k.Name] = true
			}
		}
		filteredUpdates = append(filteredUpdates, update)
	}

	r.sink.OnUpdates(filteredUpdates)
}
