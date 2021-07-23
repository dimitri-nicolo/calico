// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calc_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/felix/calc"

	v3 "github.com/projectcalico/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

// Mocked callbacks for PacketCaptureCallbacks
type myMockedCallbacks struct {
	mock.Mock
}

func (m *myMockedCallbacks) OnPacketCaptureActive(key model.ResourceKey, endpoint model.WorkloadEndpointKey) {
	_ = m.Called(key, endpoint)
}

func (m *myMockedCallbacks) OnPacketCaptureInactive(key model.ResourceKey, endpoint model.WorkloadEndpointKey) {
	_ = m.Called(key, endpoint)
}

var _ = Describe("PacketCaptureCalculator", func() {
	// output defines how PacketCaptureCallbacks calls will be matched
	type output struct {
		captureKey model.ResourceKey
		endpoint   model.WorkloadEndpointKey
	}

	DescribeTable("Starts matching workload endpoints against packet captures",
		func(updates []api.Update, expectedActive []output, expectedInactive []output) {
			var mockCallbacks = &myMockedCallbacks{}

			// Mock OnPacketCaptureActive to return the expectedActive output
			// We expect OnPacketCaptureActive to be called only with these values
			for _, value := range expectedActive {
				mockCallbacks.On("OnPacketCaptureActive", value.captureKey, value.endpoint)
			}

			// Mock OnPacketCaptureInactive to return the expectedInactive output
			// We expect OnPacketCaptureInactive to be called only with these values
			for _, value := range expectedInactive {
				mockCallbacks.On("OnPacketCaptureInactive", value.captureKey, value.endpoint)
			}

			var pcc = calc.NewPacketCaptureCalculator(mockCallbacks)

			// Sending the updates for all the event that the packet calculator is registered for
			for _, update := range updates {
				pcc.OnUpdate(update)
			}

			mockCallbacks.AssertNumberOfCalls(GinkgoT(), "OnPacketCaptureInactive", len(expectedInactive))
			mockCallbacks.AssertNumberOfCalls(GinkgoT(), "OnPacketCaptureActive", len(expectedActive))
			mockCallbacks.AssertExpectations(GinkgoT())
		},
		Entry("1 workload endpoint sent after capture with selector all()", []api.Update{
			// update for packet capture for packet-capture-all
			{
				KVPair: model.KVPair{
					Key:   CaptureAllKey,
					Value: CaptureAllValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect a single update for packet-capture-all -> wep1
			{
				captureKey: CaptureAllKey,
				endpoint:   Wep1Key,
			},
		}, []output{}),
		Entry("2 workload endpoints sent before capture with selector all()", []api.Update{
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep2
			{
				KVPair: model.KVPair{
					Key:   Wep2Key,
					Value: Wep2Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for packet capture for packet-capture-all
			{
				KVPair: model.KVPair{
					Key:   CaptureAllKey,
					Value: CaptureAllValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect two updates for packet-capture-all -> wep1, wep2
			{
				captureKey: CaptureAllKey,
				endpoint:   Wep1Key,
			},
			{
				captureKey: CaptureAllKey,
				endpoint:   Wep2Key,
			},
		}, []output{}),
		Entry("2 workload endpoints sent before and after capture with selector all()", []api.Update{
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for packet capture for packet-capture-all
			{
				KVPair: model.KVPair{
					Key:   CaptureAllKey,
					Value: CaptureAllValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep2
			{
				KVPair: model.KVPair{
					Key:   Wep2Key,
					Value: Wep2Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect two updates for packet-capture-all -> wep1, wep2
			{
				captureKey: CaptureAllKey,
				endpoint:   Wep1Key,
			},
			{
				captureKey: CaptureAllKey,
				endpoint:   Wep2Key,
			},
		}, []output{}),
		Entry("2 workload endpoints sent after capture with selector all()", []api.Update{
			// update for packet capture for packet-capture-all
			{
				KVPair: model.KVPair{
					Key:   CaptureAllKey,
					Value: CaptureAllValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep2
			{
				KVPair: model.KVPair{
					Key:   Wep2Key,
					Value: Wep2Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect two updates for packet-capture-all -> wep1, wep2
			{
				captureKey: CaptureAllKey,
				endpoint:   Wep1Key,
			},
			{
				captureKey: CaptureAllKey,
				endpoint:   Wep2Key,
			},
		}, []output{}),
		Entry("2 workload endpoints sent after capture with selector label == 'a'", []api.Update{
			// update for packet capture for packet-capture-selection
			{
				KVPair: model.KVPair{
					Key:   CaptureSelectionKey,
					Value: CaptureSelectAValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep2
			{
				KVPair: model.KVPair{
					Key:   Wep2Key,
					Value: Wep2Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect one update for packet-capture-selection -> wep1
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep1Key,
			},
		}, []output{}),
		Entry("2 workload endpoints sent after capture with selector label != 'b'", []api.Update{
			// update for packet capture for packet-capture-selection
			{
				KVPair: model.KVPair{
					Key:   CaptureSelectionKey,
					Value: CaptureSelectAValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep2
			{
				KVPair: model.KVPair{
					Key:   Wep2Key,
					Value: Wep2Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect one update for packet-capture-selection -> wep1
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep1Key,
			},
		}, []output{}),
		Entry("update capture with selector label == 'b'", []api.Update{
			// update for packet capture for packet-capture-selection with label == 'a'
			{
				KVPair: model.KVPair{
					Key:   CaptureSelectionKey,
					Value: CaptureSelectAValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep2
			{
				KVPair: model.KVPair{
					Key:   Wep2Key,
					Value: Wep2Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for packet capture for packet-capture-selection with label == 'b'
			{
				KVPair: model.KVPair{
					Key:   CaptureSelectionKey,
					Value: CaptureSelectBValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect one update for packet-capture-selection -> wep1
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep1Key,
			},
			// Expect one update for packet-capture-selection -> wep2
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep2Key,
			},
		}, []output{
			// Expect one removal for packet-capture-selection -> wep1
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep1Key,
			},
		}),
		Entry("delete capture", []api.Update{
			// update for packet capture for packet-capture-selection with label == 'a'
			{
				KVPair: model.KVPair{
					Key:   CaptureSelectionKey,
					Value: CaptureSelectAValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep2
			{
				KVPair: model.KVPair{
					Key:   Wep2Key,
					Value: Wep2Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// delete packet capture
			{
				KVPair: model.KVPair{
					Key:   CaptureSelectionKey,
					Value: nil,
				},
				UpdateType: api.UpdateTypeKVDeleted,
			},
		}, []output{
			// Expect one update for packet-capture-selection -> wep1
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep1Key,
			},
		}, []output{
			// Expect one removal for packet-capture-selection -> wep1
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep1Key,
			},
		}),
		Entry("update workload endpoint with selector label == 'c'", []api.Update{
			// update for packet capture for packet-capture-selection for label == 'a'
			{
				KVPair: model.KVPair{
					Key:   CaptureSelectionKey,
					Value: CaptureSelectAValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1 with label == 'c'
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1UpdatedValue,
				},
				UpdateType: api.UpdateTypeKVUpdated,
			},
		}, []output{
			// Expect one update for packet-capture-selection -> wep1
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep1Key,
			},
		}, []output{
			// Expect one removal for packet-capture-selection -> wep1
			{
				captureKey: CaptureSelectionKey,
				endpoint:   Wep1Key,
			},
		}),
		Entry("delete update without an new update", []api.Update{
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep2
			{
				KVPair: model.KVPair{
					Key:   Wep2Key,
					Value: Wep2Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// delete packet capture
			{
				KVPair: model.KVPair{
					Key:   CaptureSelectionKey,
					Value: nil,
				},
				UpdateType: api.UpdateTypeKVDeleted,
			},
		}, []output{}, []output{}),

		Entry("1 profile and 1 workload endpoint sent before capture with selector profile == 'dev'", []api.Update{
			// update for profile-dev
			{
				KVPair: model.KVPair{
					Key:   ProfileDevKey,
					Value: ProfileDevValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep-profile that matched profile-dev
			{
				KVPair: model.KVPair{
					Key:   WepWithProfileKey,
					Value: WepWithProfileValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for packet capture for packet-capture-dev
			{
				KVPair: model.KVPair{
					Key:   CaptureDevKey,
					Value: CaptureSelectDevValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect a single update for packet-capture-dev -> wep-profile
			{
				captureKey: CaptureDevKey,
				endpoint:   WepWithProfileKey,
			},
		}, []output{}),
		Entry("1 workload endpoint 1 profile sent before capture with selector profile == 'dev'", []api.Update{
			// update for workload endpoint wep-profile that matched profile-dev
			{
				KVPair: model.KVPair{
					Key:   WepWithProfileKey,
					Value: WepWithProfileValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for profile-dev
			{
				KVPair: model.KVPair{
					Key:   ProfileDevKey,
					Value: ProfileDevValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for packet capture for packet-capture-dev
			{
				KVPair: model.KVPair{
					Key:   CaptureDevKey,
					Value: CaptureSelectDevValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect a single update for packet-capture-dev -> wep-profile
			{
				captureKey: CaptureDevKey,
				endpoint:   WepWithProfileKey,
			},
		}, []output{}),
		Entry("1 workload endpoint 1 profile sent after capture with selector profile == 'dev'", []api.Update{
			// update for packet capture for packet-capture-dev
			{
				KVPair: model.KVPair{
					Key:   CaptureDevKey,
					Value: CaptureSelectDevValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep-profile that matched profile-dev
			{
				KVPair: model.KVPair{
					Key:   WepWithProfileKey,
					Value: WepWithProfileValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for profile-dev
			{
				KVPair: model.KVPair{
					Key:   ProfileDevKey,
					Value: ProfileDevValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect a single update for packet-capture-dev -> wep-profile
			{
				captureKey: CaptureDevKey,
				endpoint:   WepWithProfileKey,
			},
		}, []output{}),
		Entry("capture a different namespace", []api.Update{
			// update for packet capture for packet-capture-different-namespace
			{
				KVPair: model.KVPair{
					Key:   CaptureDifferentNamespaceKey,
					Value: CaptureDifferentNamespaceValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1 and namespace default
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{}, []output{}),
		Entry("same update is ignored", []api.Update{
			// update for packet capture for packet-capture-all
			{
				KVPair: model.KVPair{
					Key:   CaptureAllKey,
					Value: CaptureAllValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			// update for workload endpoint wep1
			{
				KVPair: model.KVPair{
					Key:   Wep1Key,
					Value: Wep1Value,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
			{
				KVPair: model.KVPair{
					Key:   CaptureAllKey,
					Value: CaptureAllValue,
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{
			// Expect a single update for packet-capture-all -> wep1
			{
				captureKey: CaptureAllKey,
				endpoint:   Wep1Key,
			},
		}, []output{}),
		Entry("unknown type is ignored", []api.Update{
			// update for a resource type that is not tracked
			{
				KVPair: model.KVPair{
					Key:   model.ResourceKey{Kind: v3.KindNode},
					Value: v3.Node{},
				},
				UpdateType: api.UpdateTypeKVNew,
			},
		}, []output{}, []output{}),
	)
})
