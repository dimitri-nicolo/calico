// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package syncer

import (
	"context"
	"fmt"
	"sync"

	k8sapi "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/libcalico-go/lib/backend"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s"
	"github.com/projectcalico/libcalico-go/lib/backend/model"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/dpisyncer"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

var _ = Describe("[FV] Syncer", func() {
	var nodename = "127.0.0.1"
	var k8sAPIEndpoint = "http://localhost:8080"
	var namespace = "test-dpi"
	var calicoClient clientv3.Interface
	var k8sClientset *kubernetes.Clientset
	var err error
	var ctx context.Context
	var cfg apiconfig.CalicoAPIConfig
	var healthy = func(live bool) {}

	BeforeSuite(func() {
		ctx = context.Background()
		cfg = apiconfig.CalicoAPIConfig{
			Spec: apiconfig.CalicoAPIConfigSpec{
				DatastoreType: apiconfig.Kubernetes,
				KubeConfig: apiconfig.KubeConfig{
					K8sAPIEndpoint: k8sAPIEndpoint,
				},
			},
		}

		// Create the backend client to obtain a syncer interface.
		k8sBackend, err := backend.NewClient(cfg)
		Expect(err).NotTo(HaveOccurred())
		k8sClientset = k8sBackend.(*k8s.KubeClient).ClientSet
		_ = k8sBackend.Clean()

		// Create a client.
		calicoClient, err = clientv3.New(cfg)
		Expect(err).ShouldNot(HaveOccurred())

		// Remove the test namespace
		_ = k8sClientset.CoreV1().Pods(namespace).Delete(ctx, "pod1", metav1.DeleteOptions{})
		_ = k8sClientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})

		// setup
		ns := k8sapi.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = k8sClientset.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		_, err = k8sClientset.CoreV1().Pods(namespace).Create(ctx, &k8sapi.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod1"},
			Spec: k8sapi.PodSpec{
				NodeName: nodename,
				Containers: []k8sapi.Container{
					{
						Name:  "container1",
						Image: "test",
					},
				},
			},
		},
			metav1.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())
	})
	AfterSuite(func() {
		// Remove the test namespace
		_ = k8sClientset.CoreV1().Pods(namespace).Delete(ctx, "pod1", metav1.DeleteOptions{})
		_ = k8sClientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	})

	It("handles updates to resource and sync status", func(done Done) {
		defer close(done)
		var name = "test-dpi-1"
		wg := sync.WaitGroup{}
		r := &reconciler{
			nodeName: nodename,
			cfg:      &cfg,
			client:   calicoClient,
			ch:       make(chan cacheRequest, bufferQueueSize),
			healthy:  healthy,
		}

		ExpectedCacheReq := []cacheRequest{
			// Expect status update with value of bapi.SyncStatus to be WaitForDatastore during sync start.
			{
				requestType: updateSyncStatus,
				inSync:      false,
			},
			// Expect status update with value of bapi.SyncStatus to be ResyncInProgress when sync has started.
			{
				requestType: updateSyncStatus,
				inSync:      false,
			},
			// Expect status update with value of bapi.SyncStatus to be InSync when sync is complete.
			{
				requestType: updateSyncStatus,
				inSync:      true,
			},
			// Expect update when WEP resource is created
			{
				requestType: updateResource,
				updateType:  bapi.UpdateTypeKVNew,
				kvPair: model.KVPair{
					Key: model.WorkloadEndpointKey{
						Hostname:       "127.0.0.1",
						OrchestratorID: "k8s",
						WorkloadID:     "test-dpi/pod1",
						EndpointID:     "eth0",
					},
				},
			},
			// Expect update when DPI resource is created
			{
				requestType: updateResource,
				updateType:  bapi.UpdateTypeKVNew,
				kvPair: model.KVPair{
					Key: model.KeyFromDefaultPath(fmt.Sprintf("/calico/resources/v3/projectcalico.org/deeppacketinspections/%s/%s", namespace, name)),
				},
			},
			// Expect update when DPI resource is updated
			{
				requestType: updateResource,
				updateType:  bapi.UpdateTypeKVUpdated,
				kvPair: model.KVPair{
					Key: model.KeyFromDefaultPath(fmt.Sprintf("/calico/resources/v3/projectcalico.org/deeppacketinspections/%s/%s", namespace, name)),
				},
			},
			// Expect update when WEP resource is deleted
			{
				requestType: updateResource,
				updateType:  bapi.UpdateTypeKVDeleted,
				kvPair: model.KVPair{
					Key: model.WorkloadEndpointKey{
						Hostname:       "127.0.0.1",
						OrchestratorID: "k8s",
						WorkloadID:     "test-dpi/pod1",
						EndpointID:     "eth0",
					},
				},
			},
			// Expect update when DPI resource is deleted
			{
				requestType: updateResource,
				updateType:  bapi.UpdateTypeKVDeleted,
				kvPair: model.KVPair{
					Key: model.KeyFromDefaultPath(fmt.Sprintf("/calico/resources/v3/projectcalico.org/deeppacketinspections/%s/%s", namespace, name)),
				},
			},
		}

		// Start a routine that watches for update event from syncer
		go func() {
			i := 0
			defer GinkgoRecover()
			for {
				req := <-r.ch
				wg.Done()
				if req.requestType == updateResource {
					Expect(req.updateType).Should(BeEquivalentTo(ExpectedCacheReq[i].updateType))
					Expect(req.kvPair.Key).Should(BeEquivalentTo(ExpectedCacheReq[i].kvPair.Key))

				} else {
					Expect(req).Should(BeEquivalentTo(ExpectedCacheReq[i]))
				}
				i += 1
				// exit routine
				if i == len(ExpectedCacheReq) {
					close(r.ch)
					return
				}
			}
		}()

		syncer := dpisyncer.New(calicoClient.(backendClientAccessor).Backend(), r)
		// Expect 3 sync events : WaitForDatastore -> ResyncInProgress -> InSync
		wg.Add(3)
		syncer.Start()
		defer syncer.Stop()
		wg.Wait()

		wg.Add(1)
		By("creating WEP and checking updates are received by reconciler")
		_, err = calicoClient.WorkloadEndpoints().Create(ctx, &apiv3.WorkloadEndpoint{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: fmt.Sprintf("%s-k8s-pod1-eth0", nodename)},
			Spec: apiv3.WorkloadEndpointSpec{
				Orchestrator:  "k8s",
				Node:          nodename,
				ContainerID:   "container1",
				Pod:           "pod1",
				Endpoint:      "eth0",
				IPNetworks:    []string{"10.100.10.1"},
				Profiles:      []string{"this-profile", "that-profile"},
				InterfaceName: "cali01234",
			},
		}, options.SetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		wg.Wait()

		wg.Add(1)
		By("creating DPI and checking updates are received by reconciler")
		dpi, err := calicoClient.DeepPacketInspections().Create(
			ctx,
			&apiv3.DeepPacketInspection{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec:       apiv3.DeepPacketInspectionSpec{Selector: "k8s-app=='dpi'"},
			},
			options.SetOptions{},
		)
		Expect(err).ShouldNot(HaveOccurred())
		wg.Wait()

		By("creating WEP for non-local node and checking updates are not sent to reconciler")
		tempNode := "tempnode"
		_, err = calicoClient.WorkloadEndpoints().Create(ctx, &apiv3.WorkloadEndpoint{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: fmt.Sprintf("%s-k8s-pod1-eth0", tempNode)},
			Spec: apiv3.WorkloadEndpointSpec{
				Orchestrator:  "k8s",
				Node:          tempNode,
				ContainerID:   "container1",
				Pod:           "pod1",
				Endpoint:      "eth0",
				IPNetworks:    []string{"10.100.10.1"},
				Profiles:      []string{"this-profile", "that-profile"},
				InterfaceName: "cali01234",
			},
		}, options.SetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		wg.Add(1)
		By("updating DPI and checking updates are received by reconciler")
		_, err = calicoClient.DeepPacketInspections().Update(
			ctx,
			&apiv3.DeepPacketInspection{
				ObjectMeta: dpi.ObjectMeta,
				Spec:       apiv3.DeepPacketInspectionSpec{Selector: "k8s=='dpi'"},
			},
			options.SetOptions{},
		)
		Expect(err).ShouldNot(HaveOccurred())
		wg.Wait()

		wg.Add(1)
		By("deleting WEP and checking updates are received by reconciler")
		_, err = calicoClient.WorkloadEndpoints().Delete(ctx, namespace, fmt.Sprintf("%s-k8s-pod1-eth0", nodename), options.DeleteOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		wg.Wait()

		wg.Add(1)
		By("deleting DPI and checking updates are received by reconciler")
		_, err = calicoClient.DeepPacketInspections().Delete(ctx, namespace, name, options.DeleteOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		wg.Wait()
	}, 15)

	It("syncs all resources created before the syncer is started", func(done Done) {
		defer close(done)
		var name = "test-dpi-2"
		By("creating WEP before starting syncer")
		_, err = calicoClient.WorkloadEndpoints().Create(ctx, &apiv3.WorkloadEndpoint{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: fmt.Sprintf("%s-k8s-pod1-eth1", nodename)},
			Spec: apiv3.WorkloadEndpointSpec{
				Orchestrator:  "k8s",
				Node:          nodename,
				ContainerID:   "container1",
				Pod:           "pod1",
				Endpoint:      "eth1",
				IPNetworks:    []string{"10.100.10.1"},
				Profiles:      []string{"this-profile", "that-profile"},
				InterfaceName: "cali01235",
			},
		}, options.SetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		By("creating DPI resource before starting syncer")
		_, err := calicoClient.DeepPacketInspections().Create(
			ctx,
			&apiv3.DeepPacketInspection{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec:       apiv3.DeepPacketInspectionSpec{Selector: "k8s-app=='dpi'"},
			},
			options.SetOptions{},
		)
		Expect(err).ShouldNot(HaveOccurred())

		By("starting syncer after WEP and DPI are created")
		wg := sync.WaitGroup{}
		r := &reconciler{
			nodeName: nodename,
			cfg:      &cfg,
			client:   calicoClient,
			ch:       make(chan cacheRequest, bufferQueueSize),
			healthy:  healthy,
		}

		expectedCacheReq := []cacheRequest{
			// Expect status update with value of bapi.SyncStatus to be WaitForDatastore during sync start.
			{
				requestType: updateSyncStatus,
				inSync:      false,
			},
			// Expect status update with value of bapi.SyncStatus to be ResyncInProgress when sync has started.
			{
				requestType: updateSyncStatus,
				inSync:      false,
			},
			// Expect update for WEP resource created before syncer started
			{
				requestType: updateResource,
				updateType:  bapi.UpdateTypeKVNew,
				kvPair: model.KVPair{
					Key: model.WorkloadEndpointKey{
						Hostname:       "127.0.0.1",
						OrchestratorID: "k8s",
						WorkloadID:     "test-dpi/pod1",
						EndpointID:     "eth1",
					},
				},
			},
			// Expect update for DPI resource created before syncer started
			{
				requestType: updateResource,
				updateType:  bapi.UpdateTypeKVNew,
				kvPair: model.KVPair{
					Key: model.KeyFromDefaultPath(fmt.Sprintf("/calico/resources/v3/projectcalico.org/deeppacketinspections/%s/%s", namespace, name)),
				},
			},
			// Expect status update with value of bapi.SyncStatus to be InSync when sync is complete.
			{
				requestType: updateSyncStatus,
				inSync:      true,
			},
		}
		wg.Add(len(expectedCacheReq))

		var actualCacheReq []cacheRequest

		// Start a routine that watches for update event from syncer
		go func() {
			i := 0
			defer GinkgoRecover()
			for {
				req := <-r.ch
				actualCacheReq = append(actualCacheReq, req)
				wg.Done()
				i += 1
				// exit routine
				if i == len(expectedCacheReq) {
					return
				}
			}
		}()

		syncer := dpisyncer.New(calicoClient.(backendClientAccessor).Backend(), r)
		syncer.Start()
		defer syncer.Stop()
		wg.Wait()

		// Order of update events received can be random,
		// verify if expected number of update requests are received
		Expect(len(actualCacheReq)).To(Equal(len(expectedCacheReq)))

		// Last request should be for in-sync syncer status
		Expect(actualCacheReq[len(actualCacheReq)-1]).Should(BeEquivalentTo(expectedCacheReq[len(expectedCacheReq)-1]))

	}, 15)
})
