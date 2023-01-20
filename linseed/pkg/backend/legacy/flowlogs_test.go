package legacy_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	elastic "github.com/olivere/elastic/v7"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/stretchr/testify/require"
)

// TestCreateFlowLog tests running a real elasticsearch query to create a flow log.
func TestCreateFlowLog(t *testing.T) {
	// Create an elasticsearch client to use for the test. For this test, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"))
	require.NoError(t, err)
	client := lmaelastic.NewWithClient(esClient)

	// Instantiate a flowlog backend.
	b, err := legacy.NewFlowLogBackend(client)
	require.NoError(t, err)

	clusterInfo := bapi.ClusterInfo{
		Cluster: "testcluster",
	}

	// Create a dummy flow.
	f := legacy.FlowLog{
		StartTime:            fmt.Sprintf("%d", time.Now().Unix()),
		EndTime:              fmt.Sprintf("%d", time.Now().Unix()),
		DestType:             "wep",
		DestNamespace:        "kube-system",
		DestNameAggr:         "kube-dns-*",
		DestServiceNamespace: "default",
		DestServiceName:      "kube-dns",
		DestServicePort:      "53",
		DestServicePortNum:   53,
		DestIP:               net.ParseIP("fe80::0"),
		SourceIP:             net.ParseIP("fe80::1"),
		Protocol:             "udp",
		DestPort:             53,
		SourceType:           "wep",
		SourceNamespace:      "default",
		SourceNameAggr:       "my-deployment",
		ProcessName:          "-",
		Reporter:             "src",
		Action:               "allowed",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = b.Create(ctx, clusterInfo, f)
	require.NoError(t, err)

	// Clean up after ourselves by deleting the index.
	_, err = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_flows.%s", clusterInfo.Cluster)).Do(ctx)
	require.NoError(t, err)
}
