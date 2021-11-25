// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package node

import (
	"fmt"
	"time"

	"k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	fake "sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
)

const (
	dpiName  = "dpiRes-name"
	dpiNs    = "dpiRes-ns"
	pcapName = "pcapRes-name"
	pcapNs   = "pcapRes-ns"
)

var (
	dpiRes = &v3.DeepPacketInspection{ObjectMeta: metav1.ObjectMeta{Name: dpiName, Namespace: dpiNs},
		Status: v3.DeepPacketInspectionStatus{Nodes: []v3.DPINode{
			{Node: "node-0", Active: v3.DPIActive{Success: true}},
		}}}
	pcapRes = &v3.PacketCapture{ObjectMeta: metav1.ObjectMeta{Namespace: pcapNs, Name: pcapName},
		Status: v3.PacketCaptureStatus{Files: []v3.PacketCaptureFile{
			{Node: "node-0", Directory: "/random-dir", FileNames: []string{"file-01"}}}}}
)

func getDPINodes(informer cache.SharedIndexInformer) []interface{} {
	return []interface{}{&v3.DeepPacketInspection{ObjectMeta: metav1.ObjectMeta{Name: dpiName, Namespace: dpiNs},
		Status: v3.DeepPacketInspectionStatus{Nodes: []v3.DPINode{
			{Node: "node-0", Active: v3.DPIActive{Success: true}},
		}}}}
}

func getPCAPNodes(informer cache.SharedIndexInformer) []interface{} {
	return []interface{}{&v3.PacketCapture{ObjectMeta: metav1.ObjectMeta{Namespace: pcapNs, Name: pcapName},
		Status: v3.PacketCaptureStatus{Files: []v3.PacketCaptureFile{
			{Node: "node-0", Directory: "/random-dir", FileNames: []string{"file-01"}}}}}}
}

var _ = Describe("DPI status on node create or delete", func() {

	var cli *FakeCalicoClient
	var tigeraapiCLI tigeraapi.Interface
	var mockDPIClient *MockDeepPacketInspectionInterface
	var mockPCAPClient *MockPacketCaptureInterface
	var expectedDPIUpdateStatusCallCounter, actualDPIUpdateStatusCallCounter int
	var expectedPCAPUpdateStatusCallCounter, actualPCAPUpdateStatusCallCounter int
	var stopCh chan struct{}
	var newCtrl func() *statusUpdateController
	var nodeList []string
	var dpiStatusUpd, pcapStatusUpd ResourceStatusUpdater
	BeforeEach(func() {
		cli = NewFakeCalicoClient()
		stopCh = make(chan struct{})
		nodeList = []string{}
		fn := func() []string { return nodeList }
		dpiStatusUpd = &dpiStatusUpdater{calicoClient: cli, getCachedNodesFn: getDPINodes, informer: &fake.FakeInformer{}}
		pcapStatusUpd = &packetCaptureStatusUpdater{calicoClient: cli, getNodeFn: getPCAPNodes, informer: &fake.FakeInformer{}}
		newCtrl = func() *statusUpdateController {
			return &statusUpdateController{
				rl:               workqueue.DefaultControllerRateLimiter(),
				calicoClient:     cli,
				calicoV3Client:   tigeraapiCLI,
				statusUpdaters:   []ResourceStatusUpdater{dpiStatusUpd, pcapStatusUpd},
				nodeCacheFn:      fn,
				reconcilerPeriod: 5 * time.Second,
			}
		}

		mockDPIClient = &MockDeepPacketInspectionInterface{}
		mockDPIClient.AssertExpectations(GinkgoT())
		cli.On("DeepPacketInspections").Return(mockDPIClient)

		mockPCAPClient = &MockPacketCaptureInterface{}
		mockPCAPClient.AssertExpectations(GinkgoT())
		cli.On("PacketCaptures").Return(mockPCAPClient)

		expectedDPIUpdateStatusCallCounter = 0
		actualDPIUpdateStatusCallCounter = 0
		expectedPCAPUpdateStatusCallCounter = 0
		actualPCAPUpdateStatusCallCounter = 0
	})

	AfterEach(func() {
		close(stopCh)
	})

	It("should update DPI & PCAP resource if there are no nodes cached", func() {
		expectedDPIUpdateStatusCallCounter = 1
		expectedPCAPUpdateStatusCallCounter = 1
		mockDPIClient.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything).Return(dpiRes, nil).Run(
			func(args mock.Arguments) {
				actualDPIUpdateStatusCallCounter++
				dpiRes = args.Get(1).(*v3.DeepPacketInspection)
				Expect(len(dpiRes.Status.Nodes)).Should(Equal(0))
			})
		mockPCAPClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(pcapRes, nil).Run(
			func(args mock.Arguments) {
				actualPCAPUpdateStatusCallCounter++
				pcapRes = args.Get(1).(*v3.PacketCapture)
				Expect(len(pcapRes.Status.Files)).Should(Equal(0))
			})
		ctrl := newCtrl()
		ctrl.Start(stopCh)
		Eventually(func() int { return actualDPIUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedDPIUpdateStatusCallCounter))
		Eventually(func() int { return actualPCAPUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedPCAPUpdateStatusCallCounter))
	})

	It("should update DPI & PCAP resource status when an existing node is deleted", func() {
		nodeList = []string{"node-0"}

		expectedDPIUpdateStatusCallCounter = 1
		expectedPCAPUpdateStatusCallCounter = 1
		mockDPIClient.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything).
			Return(dpiRes, nil).Run(
			func(args mock.Arguments) {
				actualDPIUpdateStatusCallCounter++
				dpiRes = args.Get(1).(*v3.DeepPacketInspection)
				Expect(len(dpiRes.Status.Nodes)).Should(Equal(0))
			})
		mockPCAPClient.On("Update", mock.Anything, mock.Anything, mock.Anything).
			Return(pcapRes, nil).Run(
			func(args mock.Arguments) {
				actualPCAPUpdateStatusCallCounter++
				pcapRes = args.Get(1).(*v3.PacketCapture)
				Expect(len(pcapRes.Status.Files)).Should(Equal(0))
			})

		ctrl := newCtrl()
		ctrl.Start(stopCh)
		nodeList = []string{}
		Eventually(func() int { return actualDPIUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedDPIUpdateStatusCallCounter))
		Eventually(func() int { return actualPCAPUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedPCAPUpdateStatusCallCounter))
	})

	It("should not update DPI or PCAP status for existing nodes", func() {
		nodeList = []string{"node-0"}
		expectedDPIUpdateStatusCallCounter = 0
		expectedPCAPUpdateStatusCallCounter = 0
		ctrl := newCtrl()
		ctrl.Start(stopCh)
		Eventually(func() int { return actualDPIUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedDPIUpdateStatusCallCounter))
		Eventually(func() int { return actualPCAPUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedPCAPUpdateStatusCallCounter))
	})

	It("should not update DPI or PCAP status when node is not available in status", func() {
		nodeList = []string{"node-0", "node-1"}
		expectedDPIUpdateStatusCallCounter = 0
		expectedPCAPUpdateStatusCallCounter = 0
		ctrl := newCtrl()
		ctrl.Start(stopCh)
		// delete a node not in dpi and pcap status
		nodeList = []string{"node-0"}
		Eventually(func() int { return actualDPIUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedDPIUpdateStatusCallCounter))
		Eventually(func() int { return actualPCAPUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedPCAPUpdateStatusCallCounter))
	})

	It("should retry status update on conflict", func() {
		expectedDPIUpdateStatusCallCounter = 3
		expectedPCAPUpdateStatusCallCounter = 3
		// UpdateStatus returns conflict during the first 2 tries
		mockDPIClient.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything).
			Return().Run(
			func(args mock.Arguments) {
				actualDPIUpdateStatusCallCounter++
				actualDPIRes := args.Get(1).(*v3.DeepPacketInspection)
				Expect(len(actualDPIRes.Status.Nodes)).Should(Equal(0))
				for _, c := range mockDPIClient.ExpectedCalls {
					if c.Method == "UpdateStatus" {
						if actualDPIUpdateStatusCallCounter <= 2 {
							c.ReturnArguments = mock.Arguments{nil, errors.NewConflict(schema.GroupResource{
								Group:    "projectcalico.org/v3",
								Resource: v3.KindDeepPacketInspection,
							}, dpiName, fmt.Errorf("randomerr"))}
						} else {
							c.ReturnArguments = mock.Arguments{dpiRes, nil}
						}
					}
				}
			})
		mockPCAPClient.On("Update", mock.Anything, mock.Anything, mock.Anything).
			Return().Run(
			func(args mock.Arguments) {
				actualPCAPUpdateStatusCallCounter++
				actualPCAPRes := args.Get(1).(*v3.PacketCapture)
				Expect(len(actualPCAPRes.Status.Files)).Should(Equal(0))
				for _, c := range mockPCAPClient.ExpectedCalls {
					if c.Method == "Update" {
						if actualPCAPUpdateStatusCallCounter <= 2 {
							c.ReturnArguments = mock.Arguments{nil, errors.NewConflict(schema.GroupResource{
								Group:    "projectcalico.org/v3",
								Resource: v3.KindPacketCapture,
							}, pcapName, fmt.Errorf("randomerr"))}
						} else {
							c.ReturnArguments = mock.Arguments{pcapRes, nil}
						}
					}
				}
			})
		ctrl := newCtrl()
		ctrl.Start(stopCh)
		Eventually(func() int { return actualDPIUpdateStatusCallCounter }, 30*time.Second).Should(Equal(expectedDPIUpdateStatusCallCounter))
		Eventually(func() int { return actualPCAPUpdateStatusCallCounter }, 30*time.Second).Should(Equal(expectedPCAPUpdateStatusCallCounter))
	})
})
