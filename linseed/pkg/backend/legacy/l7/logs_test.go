package l7_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"

	elastic "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/l7"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// TestCreateL7Log tests running a real elasticsearch query to create an L7 log.
func TestCreateL7Log(t *testing.T) {
	// Create an elasticsearch client to use for the test. For this test, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"))
	require.NoError(t, err)
	client := lmaelastic.NewWithClient(esClient)
	cache := templates.NewTemplateCache(client, 1, 0)

	// Instantiate a flowlog backend.
	b := l7.NewL7LogBackend(client, cache)

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

	response, err := b.Create(ctx, clusterInfo, []bapi.L7Log{f})
	require.NoError(t, err)
	require.Equal(t, response.Failed, 0)
	require.Equal(t, response.Succeeded, 1)
	require.Len(t, response.Errors, 0)

	// Clean up after ourselves by deleting the index.
	_, err = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_l7.%s.*", clusterInfo.Cluster)).Do(ctx)
	require.NoError(t, err)
}
