// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package processor_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/deep-packet-inspection/pkg/exec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/deep-packet-inspection/pkg/processor"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var _ = Describe("DPI processor", func() {
	dpiName := "dpi-processor-test"
	dpiNs := "dpi-processor-ns"
	ifaceName := "dpi-wepKey-eth0"
	dpiKey := model.ResourceKey{
		Name:      dpiName,
		Namespace: dpiNs,
		Kind:      "DeepPacketInspection",
	}
	wepKey := model.WorkloadEndpointKey{WorkloadID: "wep1"}

	var mockClient *processor.MockClientInterface
	var mockDPIInterface *processor.MockDeepPacketInspectionInterface
	var mockSnortExec *exec.MockExec
	var ctx context.Context
	var dpiRes *v3.DeepPacketInspection
	var p processor.Processor

	snortExecFn := func(podName string, iface string, namespace string, dpiName string, alertPath string, alertFileSize int, communityRulesFile string) (exec.Exec, error) {
		return mockSnortExec, nil
	}

	BeforeEach(func() {
		mockClient = &processor.MockClientInterface{}
		mockDPIInterface = &processor.MockDeepPacketInspectionInterface{}
		mockSnortExec = &exec.MockExec{}
		ctx = context.Background()

		dpiRes = &v3.DeepPacketInspection{
			ObjectMeta: metav1.ObjectMeta{Name: dpiName, Namespace: dpiNs},
			Spec:       v3.DeepPacketInspectionSpec{Selector: "k8s-app=='dpi'"},
		}
		mockClient.On("DeepPacketInspections").Return(mockDPIInterface)
	})

	It("Starts snort when new WEP interface is added", func() {
		// Status is updated after initializing the processor, once after WEP is added and once when the processor is closed.
		mockDPIInterface.On("Get", ctx, dpiKey.Namespace, dpiKey.Name, mock.Anything).Return(dpiRes, nil).Times(3)
		mockDPIInterface.On("UpdateStatus", mock.Anything, dpiRes, mock.Anything).Return(dpiRes, nil).Times(3)

		p = processor.NewProcessor(ctx, mockClient, dpiKey, "node0", snortExecFn, "", 0, "")

		Expect(p.WEPInterfaceCount()).Should(Equal(0))
		mockSnortExec.On("Start").Return(nil).Times(1)
		mockSnortExec.On("Wait").Return(nil).Times(1)
		p.Add(ctx, wepKey, ifaceName)

		mockSnortExec.On("Stop").Return(nil).Times(1)
		p.Close()
	})

	It("Stop snort when WEP interface is removed", func() {
		// Status is updated after initializing the processor, once after WEP is added and once when the WEP is removed.
		mockDPIInterface.On("Get", mock.Anything, dpiNs, dpiName, mock.Anything).Return(dpiRes, nil).Times(3)
		mockDPIInterface.On("UpdateStatus", mock.Anything, dpiRes, mock.Anything).Return(dpiRes, nil).Times(3)

		p = processor.NewProcessor(ctx, mockClient, dpiKey, "node0", snortExecFn, "", 0, "")
		mockSnortExec.On("Start").Return(nil)
		mockSnortExec.On("Wait").Return(nil)
		p.Add(ctx, wepKey, ifaceName)

		mockSnortExec.On("Stop").Return(nil).Times(1)
		p.Remove(wepKey)

		// Close shouldn't call Stop on already stopped snort process.
		p.Close()
	})

	It("Restart snort if snort process fails", func() {
		numberOfCallsToWait := 0

		// Status is updated after initializing the processor, once after WEP is added, once when snort fails and once when the WEP is removed.
		mockDPIInterface.On("Get", mock.Anything, dpiNs, dpiName, mock.Anything).Return(dpiRes, nil).Times(4)
		mockDPIInterface.On("UpdateStatus", mock.Anything, dpiRes, mock.Anything).Return(dpiRes, nil).Times(4)

		p = processor.NewProcessor(ctx, mockClient, dpiKey, "node0", snortExecFn, "", 0, "")
		mockSnortExec.On("Start").Return(nil).Times(2)
		mockSnortExec.On("Stop").Return(nil).Times(1)

		// Return error on first call to wait (this will restart snort) and nil on second call
		mockSnortExec.On("Wait").Return().Run(func(args mock.Arguments) {
			for _, c := range mockSnortExec.ExpectedCalls {
				if c.Method == "Wait" {
					if numberOfCallsToWait == 1 {
						c.ReturnArguments = mock.Arguments{errors.New("snort error")}
					} else if numberOfCallsToWait == 2 {
						c.ReturnArguments = mock.Arguments{nil}
					}
				}
			}
		}).Times(2)

		numberOfCallsToWait++
		p.Add(ctx, wepKey, ifaceName)
		numberOfCallsToWait++

		p.Close()

	})

	It("Closes all snort process", func() {
		// Status is updated after initializing the processor, twice when WEP is added, and once when process is closed.
		mockDPIInterface.On("Get", mock.Anything, dpiNs, dpiName, mock.Anything).Return(dpiRes, nil).Times(4)
		mockDPIInterface.On("UpdateStatus", mock.Anything, dpiRes, mock.Anything).Return(dpiRes, nil).Times(4)

		p = processor.NewProcessor(ctx, mockClient, dpiKey, "node0", snortExecFn, "", 0, "")

		mockSnortExec.On("Start").Return(nil).Times(2)
		mockSnortExec.On("Wait").Return(nil).Times(2)
		mockSnortExec.On("Stop").Return(nil).Times(1)
		p.Add(ctx, wepKey, ifaceName)
		p.Add(ctx, model.WorkloadEndpointKey{WorkloadID: "wep2"}, "iface2")

		mockSnortExec.On("Stop").Return(nil).Times(2)
		p.Close()
	})
})
