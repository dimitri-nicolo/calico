// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
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
