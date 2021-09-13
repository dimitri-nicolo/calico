// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package processor_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"github.com/tigera/deep-packet-inspection/pkg/dpiupdater"
	"github.com/tigera/deep-packet-inspection/pkg/exec"
	"github.com/tigera/deep-packet-inspection/pkg/processor"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var _ = Describe("DPI processor", func() {
	dpiName := "dpi-name"
	dpiNs := "dpi-ns"
	ifaceName := "dpi-wepKey-eth0"
	dpiKey := model.ResourceKey{
		Name:      dpiName,
		Namespace: dpiNs,
		Kind:      "DeepPacketInspection",
	}
	wepKey := model.WorkloadEndpointKey{WorkloadID: "podname"}

	var mockDPIUpdater *dpiupdater.MockDPIStatusUpdater
	var mockSnortExec *exec.MockExec
	var ctx context.Context
	var p processor.Processor

	snortExecFn := func(podName string, iface string, namespace string, dpiName string, alertPath string, alertFileSize int, communityRulesFile string) (exec.Exec, error) {
		return mockSnortExec, nil
	}

	BeforeEach(func() {
		mockSnortExec = &exec.MockExec{}
		mockDPIUpdater = &dpiupdater.MockDPIStatusUpdater{}
		ctx = context.Background()
	})

	It("Starts snort when new WEP interface is added", func() {
		// Status is updated after initializing the processor, once after WEP is added and once when the processor is closed.
		mockDPIUpdater.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(3)

		p = processor.NewProcessor(ctx, dpiKey, "node0", snortExecFn, "", 0, "", mockDPIUpdater)

		Expect(p.WEPInterfaceCount()).Should(Equal(0))
		mockSnortExec.On("Start").Return(nil).Times(1)
		mockSnortExec.On("Wait").Return(nil).Times(1)
		p.Add(ctx, wepKey, ifaceName)

		mockSnortExec.On("Stop").Return(nil).Times(1)
		p.Close()
	})

	It("StopGeneratingEventsForWEP snort when WEP interface is removed", func() {
		// Status is updated after initializing the processor, once after WEP is added and once when the WEP is removed.
		mockDPIUpdater.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(3)

		p = processor.NewProcessor(ctx, dpiKey, "node0", snortExecFn, "", 0, "", mockDPIUpdater)
		mockSnortExec.On("Start").Return(nil)
		mockSnortExec.On("Wait").Return(nil)
		p.Add(ctx, wepKey, ifaceName)

		mockSnortExec.On("Stop").Return(nil).Times(1)
		p.Remove(wepKey)

		// Close shouldn't call StopGeneratingEventsForWEP on already stopped snort process.
		p.Close()
	})

	It("Restart snort if snort process fails", func() {
		numberOfCallsToWait := 0

		// Status is updated after initializing the processor, once after WEP is added, once when snort fails and once when the WEP is removed.
		mockDPIUpdater.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(4)

		p = processor.NewProcessor(ctx, dpiKey, "node0", snortExecFn, "", 0, "", mockDPIUpdater)
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
		mockDPIUpdater.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(4)

		p = processor.NewProcessor(ctx, dpiKey, "node0", snortExecFn, "", 0, "", mockDPIUpdater)

		mockSnortExec.On("Start").Return(nil).Times(2)
		mockSnortExec.On("Wait").Return(nil).Times(2)
		mockSnortExec.On("Stop").Return(nil).Times(1)
		p.Add(ctx, wepKey, ifaceName)
		p.Add(ctx, model.WorkloadEndpointKey{WorkloadID: "wep2"}, "iface2")

		mockSnortExec.On("Stop").Return(nil).Times(2)
		p.Close()
	})
})
