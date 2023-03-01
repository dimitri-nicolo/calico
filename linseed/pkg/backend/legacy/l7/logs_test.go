package l7_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/l7"
)

// TestCreateL7Log tests running a real elasticsearch query to create an L7 log.
func TestCreateL7Log(t *testing.T) {
	defer setupTest(t)()

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
}
