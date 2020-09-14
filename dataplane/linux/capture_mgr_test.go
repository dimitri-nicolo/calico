// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/felix/ifacemonitor"
	"github.com/projectcalico/felix/proto"

	"github.com/projectcalico/felix/capture"
)

// Mocked Captures
type myMockedCaptures struct {
	mock.Mock
}

func (m *myMockedCaptures) Remove(key capture.Key) (error, string) {
	args := m.Called(key)
	return args.Error(0), args.String(1)

}

func (m *myMockedCaptures) Add(key capture.Key, deviceName string) error {
	args := m.Called(key, deviceName)
	return args.Error(0)
}

var _ = Describe("PacketCapture Manager", func() {
	type output struct {
		key    capture.Key
		device string
		err    error
	}

	DescribeTable("Buffers packet captures until interfaces are up",
		func(updateBatches [][]interface{}, expectedAdditions []output, expectedRemovals []output) {
			var mockedCaptures = myMockedCaptures{}

			// Mock Add to return the expectedAdditions output
			// We expect Add to be called only with these values
			for _, v := range expectedAdditions {
				mockedCaptures.On("Add", v.key, v.device).Return(v.err)
			}
			// Mock Removal to return the expectedRemovals output
			// We expect Removal to be called only with these values
			for _, v := range expectedRemovals {
				mockedCaptures.On("Remove", v.key).Return(v.err, v.device)
			}

			var captureMgr = newCaptureManager(&mockedCaptures, []string{"cali"})

			// Feed updateBatches
			for _, batch := range updateBatches {
				for _, update := range batch {
					captureMgr.OnUpdate(update)
				}
				// Call CompleteDeferredWork to produce an output
				// for each processing batch
				_ = captureMgr.CompleteDeferredWork()
			}

			mockedCaptures.AssertNumberOfCalls(GinkgoT(), "Add", len(expectedAdditions))
			mockedCaptures.AssertNumberOfCalls(GinkgoT(), "Remove", len(expectedRemovals))
			mockedCaptures.AssertExpectations(GinkgoT())
		},
		Entry("1 capture after endpoints and interfaces are up", [][]interface{}{
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
			{
				// capture update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
		}, []output{
			// Expect packet capture to start
			{
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
		}, []output{}),
		Entry("1 capture before interfaces and endpoints are up", [][]interface{}{
			{
				// capture update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
		}, []output{
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
		}, []output{}),
		Entry("1 capture before endpoints and interfaces are up", [][]interface{}{
			{
				// capture update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
		}, []output{
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
			{
				// Expect the second call to start to error out
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    fmt.Errorf("cannot start twice"),
			},
		}, []output{}),
		Entry("multiple captures for different endpoints", [][]interface{}{
			{
				// capture update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod-1",
					},
				},
			},
			{
				// capture update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-2",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod-2",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod-1",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali456",
					State: ifacemonitor.StateUp,
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod-2",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali456",
					},
				},
			},
		}, []output{
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod-1",
				},
				device: "cali123",
				err:    nil,
			},
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-2", WorkloadEndpointId: "default/sample-pod-2",
				},
				device: "cali456",
				err:    nil,
			},
		}, []output{}),
		Entry("overlapping captures for the same endpoint", [][]interface{}{
			{
				// capture update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
			{
				// capture update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-2",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
		}, []output{
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-2", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
		}, []output{}),
		Entry("start/stop for the same endpoint", [][]interface{}{
			{
				// capture update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
			{
				// capture removal will be processed in a single batch
				&proto.PacketCaptureRemove{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
		}, []output{
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
		}, []output{
			{
				// Expect packet capture to stop
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				err: nil,
			},
		}),
		Entry("matches only cali interfaces", [][]interface{}{
			{
				// all updates will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
				&ifaceUpdate{
					Name:  "eth0",
					State: ifacemonitor.StateUp,
				},
				&ifaceUpdate{
					Name:  "lo",
					State: ifacemonitor.StateUp,
				},
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
		}, []output{
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
			{
				// Expect call to start to error out
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    fmt.Errorf("cannot start twice"),
			},
		}, []output{}),
		Entry("interface down stops a capture", [][]interface{}{
			{
				// wep update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateDown,
				},
			},
		}, []output{
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
		}, []output{
			{
				// Expect packet capture to stop
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
		}),
		Entry("start after an interface went down", [][]interface{}{
			{
				// wep update will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
			{
				// wep update will be processed in a single batch
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateDown,
				},
			},
			{
				// interface update will be processed in a single batch
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
			},
		}, []output{
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
			{
				// Expect packet capture to start
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
		}, []output{
			{
				// Expect packet capture to stop
				key: capture.Key{
					Namespace: "default", CaptureName: "packet-capture-1", WorkloadEndpointId: "default/sample-pod",
				},
				device: "cali123",
				err:    nil,
			},
		}),
		Entry("start/stop for the same endpoint in the same batch does not produce output", [][]interface{}{
			{
				// all updates will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
				&proto.PacketCaptureRemove{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
			},
		}, []output{}, []output{}),
		Entry("interface up/down in the same batch does not produce output", [][]interface{}{
			{
				// all updates will be processed in a single batch
				&proto.PacketCaptureUpdate{
					Id: &proto.PacketCaptureID{
						Name:      "packet-capture-1",
						Namespace: "default",
					},
					Endpoint: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
				},
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateUp,
				},
				&proto.WorkloadEndpointUpdate{
					Id: &proto.WorkloadEndpointID{
						WorkloadId: "default/sample-pod",
					},
					Endpoint: &proto.WorkloadEndpoint{
						State: "up",
						Name:  "cali123",
					},
				},
				&ifaceUpdate{
					Name:  "cali123",
					State: ifacemonitor.StateDown,
				},
			},
		}, []output{}, []output{}),
	)
})
