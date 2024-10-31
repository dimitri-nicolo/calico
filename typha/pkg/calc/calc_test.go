// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package calc_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	. "github.com/projectcalico/calico/typha/pkg/calc"
)

type testSink struct {
	countUpdates int
	values       []interface{}
}

func (s *testSink) OnStatusUpdated(status api.SyncStatus) {}

func (s *testSink) OnUpdates(updates []api.Update) {
	s.countUpdates += len(updates)
	for _, u := range updates {
		s.values = append(s.values, u.Value)
	}
}

var _ = Describe("ValidationFilter", func() {

	var (
		s *testSink
		v *ValidationFilter
	)

	BeforeEach(func() {
		s = &testSink{}
		v = NewValidationFilter(s)
	})

	It("it should reject an invalid v3 Profile", func() {
		v.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Name: "prof1", Kind: v3.KindProfile},
				Value: &v3.Profile{
					ObjectMeta: v1.ObjectMeta{
						Name: "prof1",
					},
					Spec: v3.ProfileSpec{
						LabelsToApply: map[string]string{
							"a//b": "c//d",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		}})
		Expect(s.countUpdates).To(Equal(1))
		Expect(s.values[0]).To(BeNil()) // failed validation -> value nil
	})

	It("it should allow a valid v3 Profile", func() {
		v.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Name: "prof1", Kind: v3.KindProfile},
				Value: &v3.Profile{
					ObjectMeta: v1.ObjectMeta{
						Name: "prof1",
					},
					Spec: v3.ProfileSpec{
						LabelsToApply: map[string]string{
							"a--b": "c--d",
						},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		}})
		Expect(s.countUpdates).To(Equal(1))
		Expect(s.values[0]).NotTo(BeNil())
	})

	DescribeTable("it should filter out some RemoteClusterStatus types", func(statusType model.RemoteClusterStatusType, shouldFilter bool) {
		v.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: "rccs1"},
				Value: &model.RemoteClusterStatus{
					Status: statusType,
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		}})
		expectedCount := 0
		if !shouldFilter {
			expectedCount = 1
		}
		Expect(s.countUpdates).To(Equal(expectedCount))
	}, []TableEntry{
		remoteClusterStatusCase(model.RemoteClusterConnecting, true),
		remoteClusterStatusCase(model.RemoteClusterConnectionFailed, true),
		remoteClusterStatusCase(model.RemoteClusterResyncInProgress, true),
		remoteClusterStatusCase(model.RemoteClusterInSync, true),
		remoteClusterStatusCase(model.RemoteClusterConfigIncomplete, true),
		remoteClusterStatusCase(model.RemoteClusterConfigChangeRestartRequired, false),
	}...)
})

func remoteClusterStatusCase(statusType model.RemoteClusterStatusType, shouldFilter bool) TableEntry {
	return TableEntry{Description: statusType.String(), Parameters: []interface{}{statusType, shouldFilter}}
}
