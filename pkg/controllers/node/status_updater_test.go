// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package node

import (
	"fmt"
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
)

const (
	dpiName = "dpiRes-name"
	dpiNs   = "dpiRes-ns"
)

type mockDPIWatcher struct {
}

func (w *mockDPIWatcher) Run(stopCh chan struct{}) {}

func (w *mockDPIWatcher) GetExistingResources() []interface{} {
	return []interface{}{&v3.DeepPacketInspection{ObjectMeta: metav1.ObjectMeta{Name: dpiName, Namespace: dpiNs},
		Status: v3.DeepPacketInspectionStatus{Nodes: []v3.DPINode{
			{Node: "node-0", Active: v3.DPIActive{Success: true}},
		}}}}
}

var _ = Describe("DPI status on node create or delete", func() {

	var cli *FakeCalicoClient
	var tigeraapiCLI tigeraapi.Interface
	var mockDPIClient *MockDeepPacketInspectionInterface
	var dpiRes *v3.DeepPacketInspection
	var expectedUpdateStatusCallCounter, actualUpdateStatusCallCounter int
	var stopCh chan struct{}
	var newCtrl func() *statusUpdateController
	var nodeList []string
	var mockWatcher Watcher
	BeforeEach(func() {
		cli = NewFakeCalicoClient()
		stopCh = make(chan struct{})
		nodeList = []string{}
		fn := func() []string { return nodeList }
		mockWatcher = &mockDPIWatcher{}

		newCtrl = func() *statusUpdateController {
			return &statusUpdateController{
				rl:               workqueue.DefaultControllerRateLimiter(),
				calicoClient:     cli,
				calicoV3Client:   tigeraapiCLI,
				dpiWch:           mockWatcher,
				nodeCacheFn:      fn,
				reconcilerPeriod: 5 * time.Second,
			}
		}

		mockDPIClient = &MockDeepPacketInspectionInterface{}
		mockDPIClient.AssertExpectations(GinkgoT())
		cli.On("DeepPacketInspections").Return(mockDPIClient)
		dpiRes = &v3.DeepPacketInspection{ObjectMeta: metav1.ObjectMeta{Name: dpiName, Namespace: dpiNs},
			Status: v3.DeepPacketInspectionStatus{Nodes: []v3.DPINode{
				{Node: "node-0", Active: v3.DPIActive{Success: true}},
			}}}

		expectedUpdateStatusCallCounter = 0
		actualUpdateStatusCallCounter = 0
	})

	AfterEach(func() {
		close(stopCh)
	})

	It("should update DPI resource if there are no nodes cached", func() {
		expectedUpdateStatusCallCounter = 1
		mockDPIClient.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything).Return(dpiRes, nil).Run(
			func(args mock.Arguments) {
				actualUpdateStatusCallCounter++
				dpiRes = args.Get(1).(*v3.DeepPacketInspection)
				Expect(len(dpiRes.Status.Nodes)).Should(Equal(0))
			})

		ctrl := newCtrl()
		ctrl.Start(stopCh)
		Eventually(func() int { return actualUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedUpdateStatusCallCounter))
	})

	It("should update DPI resource status when an existing node is deleted", func() {
		nodeList = []string{"node-0"}

		expectedUpdateStatusCallCounter = 1
		mockDPIClient.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything).
			Return(dpiRes, nil).Run(
			func(args mock.Arguments) {
				actualUpdateStatusCallCounter++
				dpiRes = args.Get(1).(*v3.DeepPacketInspection)
				Expect(len(dpiRes.Status.Nodes)).Should(Equal(0))
			})

		ctrl := newCtrl()
		ctrl.Start(stopCh)
		nodeList = []string{}
		Eventually(func() int { return actualUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedUpdateStatusCallCounter))
	})

	It("should not update DPI resource status for existing nodes", func() {
		nodeList = []string{"node-0"}
		expectedUpdateStatusCallCounter = 0
		ctrl := newCtrl()
		ctrl.Start(stopCh)
		Eventually(func() int { return actualUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedUpdateStatusCallCounter))
	})

	It("should not update DPI resource status when node is not available in status", func() {
		nodeList = []string{"node-0", "node-1"}
		expectedUpdateStatusCallCounter = 0
		ctrl := newCtrl()
		ctrl.Start(stopCh)
		// delete a node not in dpi status
		nodeList = []string{"node-0"}
		Eventually(func() int { return actualUpdateStatusCallCounter }, 10*time.Second).Should(Equal(expectedUpdateStatusCallCounter))
	})

	It("should retry status update on conflict", func() {
		expectedUpdateStatusCallCounter = 3
		// UpdateStatus returns conflict during the first 2 tries
		mockDPIClient.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything).
			Return().Run(
			func(args mock.Arguments) {
				actualUpdateStatusCallCounter++
				actualDPIRes := args.Get(1).(*v3.DeepPacketInspection)
				Expect(len(actualDPIRes.Status.Nodes)).Should(Equal(0))
				for _, c := range mockDPIClient.ExpectedCalls {
					if c.Method == "UpdateStatus" {
						if actualUpdateStatusCallCounter <= 2 {
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
		ctrl := newCtrl()
		ctrl.Start(stopCh)
		Eventually(func() int { return actualUpdateStatusCallCounter }, 30*time.Second).Should(Equal(expectedUpdateStatusCallCounter))
	})
})
