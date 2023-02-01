package legacy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	elastic "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// TestCreateL7Log tests running a real elasticsearch query to create an L7 log.
func TestCreateL7Log(t *testing.T) {
	// Create an elasticsearch client to use for the test. For this test, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"))
	require.NoError(t, err)
	client := lmaelastic.NewWithClient(esClient)

	// Instantiate a flowlog backend.
	b := legacy.NewL7LogBackend(client)

	clusterInfo := bapi.ClusterInfo{
		Cluster: "testcluster",
	}

	// Create a dummy flow.
	f := bapi.L7Log{
		StartTime:            fmt.Sprintf("%d", time.Now().Unix()),
		EndTime:              fmt.Sprintf("%d", time.Now().Unix()),
		DestType:             "wep",
		DestNamespace:        "kube-system",
		DestNameAggr:         "kube-dns-*",
		DestServiceNamespace: "default",
		DestServiceName:      "kube-dns",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = b.Create(ctx, clusterInfo, []bapi.L7Log{f})
	require.NoError(t, err)

	// Clean up after ourselves by deleting the index.
	_, err = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_l7.%s.*", clusterInfo.Cluster)).Do(ctx)
	require.NoError(t, err)
}
