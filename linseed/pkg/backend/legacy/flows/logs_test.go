// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows_test

import (
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

// TestCreateFlowLog tests running a real elasticsearch query to create a flow log.
func TestCreateFlowLog(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{
		Cluster: "testcluster",
	}

	// Create a dummy flow.
	f := v1.FlowLog{
		StartTime:            time.Now().Unix(),
		EndTime:              time.Now().Unix(),
		DestType:             "wep",
		DestNamespace:        "kube-system",
		DestNameAggr:         "kube-dns-*",
		DestServiceNamespace: "default",
		DestServiceName:      "kube-dns",
		DestServicePortNum:   testutils.Int64Ptr(53),
		DestIP:               testutils.StringPtr("fe80::0"),
		SourceIP:             testutils.StringPtr("fe80::1"),
		Protocol:             "udp",
		DestPort:             testutils.Int64Ptr(53),
		SourceType:           "wep",
		SourceNamespace:      "default",
		SourceNameAggr:       "my-deployment",
		ProcessName:          "-",
		Reporter:             "src",
		Action:               "allowed",
	}

	response, err := flb.Create(ctx, clusterInfo, []v1.FlowLog{f})
	require.NoError(t, err)
	require.Equal(t, []v1.BulkError(nil), response.Errors)
	require.Equal(t, 0, response.Failed)
}
