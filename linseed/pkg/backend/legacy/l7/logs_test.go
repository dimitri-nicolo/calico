package l7_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

	elastic "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/l7"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// TestCreateL7Log tests running a real elasticsearch query to create an L7 log.
func TestCreateL7Log(t *testing.T) {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)
	defer logCancel()

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	client := lmaelastic.NewWithClient(esClient)
	cache := templates.NewTemplateCache(client, 1, 0)

	// Instantiate a flowlog backend.
	b := l7.NewL7LogBackend(client, cache)

	clusterInfo := bapi.ClusterInfo{
		Cluster: cluster,
	}

	// Create a dummy flow.
	f := v1.L7Log{
		StartTime:            time.Now().Unix(),
		EndTime:              time.Now().Unix(),
		DestType:             "wep",
		DestNamespace:        "kube-system",
		DestNameAggr:         "kube-dns-*",
		DestServiceNamespace: "default",
		DestServiceName:      "kube-dns",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := b.Create(ctx, clusterInfo, []v1.L7Log{f})
	require.NoError(t, err)
	require.Equal(t, response.Failed, 0)
	require.Equal(t, response.Succeeded, 1)
	require.Len(t, response.Errors, 0)

	// Clean up after ourselves by deleting the index.
	_, err = esClient.DeleteIndex(fmt.Sprintf("tigera_secure_ee_l7.%s.*", clusterInfo.Cluster)).Do(ctx)
	require.NoError(t, err)
}
