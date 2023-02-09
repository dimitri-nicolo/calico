// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/flows"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// TestCreateFlowLog tests running a real elasticsearch query to create a flow log.
func TestCreateFlowLog(t *testing.T) {
	// Create an elasticsearch client to use for the test. For this test, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"))
	require.NoError(t, err)
	client := lmaelastic.NewWithClient(esClient)
	cache := templates.NewTemplateCache(client, 1, 0)

	// Instantiate a flowlog backend.
	b := flows.NewFlowLogBackend(client, cache)

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := b.Create(ctx, clusterInfo, []v1.FlowLog{f})
	require.NoError(t, err)
	require.Equal(t, 0, response.Failed)

	// Clean up after ourselves by deleting the index.
	_, err = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_flows.%s.*", clusterInfo.Cluster)).Do(ctx)
	require.NoError(t, err)
}
