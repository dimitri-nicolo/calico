// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package testutils

import (
	"reflect"
	"strings"
	"testing"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/stretchr/testify/require"
)

func AssertLogIDAndCopyFlowLogsWithoutID(t *testing.T, r *v1.List[v1.FlowLog]) []v1.FlowLog {
	require.NotNil(t, r)

	// Asert that we have an ID assigned from Elastic
	var copyOfLogs []v1.FlowLog
	for _, item := range r.Items {
		item = AssertFlowLogIDAndReset(t, item)
		copyOfLogs = append(copyOfLogs, item)
	}
	return copyOfLogs
}

func AssertFlowLogIDAndReset(t *testing.T, item v1.FlowLog) v1.FlowLog {
	require.NotEmpty(t, item.ID)
	item.ID = ""

	return item
}

func AssertEventIDAndReset(t *testing.T, item v1.Event) v1.Event {
	require.NotEmpty(t, item.ID)
	item.ID = ""

	return item
}

func AssertLogIDAndCopyDNSLogsWithoutID(t *testing.T, r *v1.List[v1.DNSLog]) []v1.DNSLog {
	require.NotNil(t, r)

	// Asert that we have an ID assigned from Elastic
	var copyOfLogs []v1.DNSLog
	for _, item := range r.Items {
		item = AssertDNSLogIDAndReset(t, item)
		copyOfLogs = append(copyOfLogs, item)
	}
	return copyOfLogs
}

func AssertDNSLogIDAndReset(t *testing.T, item v1.DNSLog) v1.DNSLog {
	require.NotEmpty(t, item.ID)
	item.ID = ""

	return item
}

func AssertLogIDAndCopyEventsWithoutID(t *testing.T, r *v1.List[v1.Event]) []v1.Event {
	require.NotNil(t, r)

	// Asert that we have an ID assigned from Elastic
	var copyOfEvents []v1.Event
	for _, item := range r.Items {
		item = AssertEventIDAndReset(t, item)
		copyOfEvents = append(copyOfEvents, item)
	}
	return copyOfEvents
}

func AssertRuntimeReportIDAndReset(t *testing.T, item v1.RuntimeReport) v1.RuntimeReport {
	require.NotEmpty(t, item.ID)
	item.ID = ""

	return item
}

func AssertLogIDAndCopyRuntimeReportsWithoutThem(t *testing.T, r *v1.List[v1.RuntimeReport]) []v1.RuntimeReport {
	require.NotNil(t, r)

	// Asert that we have an ID assigned from Elastic
	var copyOfReports []v1.RuntimeReport
	for _, item := range r.Items {
		item = AssertRuntimeReportIDAndReset(t, item)
		copyOfReports = append(copyOfReports, item)
	}
	return copyOfReports
}

func AssertStructAndMap(t *testing.T, logType interface{}, mappings map[string]interface{}) bool {
	require.NotNil(t, mappings)
	if len(mappings) != 2 {
		return false
	}
	// Check Dynamic is false
	require.NotNil(t, mappings["dynamic"])
	require.Equal(t, "false", mappings["dynamic"])

	//Fetch Properties from the json template
	require.NotNil(t, mappings["properties"])
	properties := mappings["properties"].(map[string]interface{})

	obj := reflect.ValueOf(logType).Type()
	if obj.NumField() != len(properties) {
		return false
	}

	// Check each field in the struct exist in json template mapping
	for i := 0; i < obj.NumField(); i++ {
		field := obj.Field(i)
		tags := strings.Split(field.Tag.Get("json"), ",")
		val := ""
		if len(tags) > 0 {
			val = tags[0]
		}
		require.NotEmpty(t, val)
		if _, ok := properties[val]; !ok {
			return false
		}
	}
	return true
}
