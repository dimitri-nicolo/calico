// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates

import (
	"github.com/stretchr/testify/require"
	"testing"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	utils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/testutils"
)

func TestCompareEventStructAndTemplate(t *testing.T) {

	t.Run("Check for event api and template matches", func(t *testing.T) {
		eventsMap := testutils.MustUnmarshalToMap(t, EventsMappings)
		require.Equal(t, true, utils.AssertStructAndMap(t, v1.Event{}, eventsMap))
	})
	t.Run("Check for event api and template not matches", func(t *testing.T) {
		eventsMap := testutils.MustUnmarshalToMap(t, EventsMappings)
		properties := eventsMap["properties"].(map[string]interface{})
		properties["random"] = map[string]interface{}{
			"unknown": "element",
		}
		require.Equal(t, false, utils.AssertStructAndMap(t, v1.Event{}, eventsMap))
	})

	t.Run("Check for event struct with same count and diff element", func(t *testing.T) {
		type fakeEvent struct {
			ID              string             `json:"id"`
			Time            v1.TimestampOrDate `json:"time" validate:"required"`
			Description     string             `json:"description" validate:"required"`
			Origin          string             `json:"origin" validate:"required"`
			Severity        int                `json:"severity" validate:"required"`
			Type            string             `json:"type" validate:"required"`
			Alert           string             `json:"alert,omitempty"`
			DestIP          *string            `json:"dest_ip,omitempty"`
			DestName        string             `json:"dest_name,omitempty"`
			DestNameAggr    string             `json:"dest_name_aggr,omitempty"`
			DestNamespace   string             `json:"dest_namespace,omitempty"`
			DestPort        *int64             `json:"dest_port,omitempty"`
			Protocol        string             `json:"protocol,omitempty"`
			Dismissed       bool               `json:"dismissed,omitempty"`
			Host            string             `json:"host,omitempty"`
			SourceIP        *string            `json:"source_ip,omitempty"`
			SourceName      string             `json:"source_name,omitempty"`
			SourceNameAggr  string             `json:"source_name_aggr,omitempty"`
			SourceNamespace string             `json:"source_namespace,omitempty"`
			SourcePort      *int64             `json:"source_port,omitempty"`
			FakeRecord      interface{}        `json:"fake_record,omitempty"`
		}
		eventsMap := testutils.MustUnmarshalToMap(t, EventsMappings)
		require.Equal(t, false, utils.AssertStructAndMap(t, fakeEvent{}, eventsMap))
	})

}
