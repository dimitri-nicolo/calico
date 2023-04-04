// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/google/gopacket/layers"
	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/dns"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client  lmaelastic.Client
	cache   bapi.Cache
	b       bapi.DNSFlowBackend
	lb      bapi.DNSLogBackend
	ctx     context.Context
	cluster string
)

// setupTest runs common logic before each test, and also returns a function to perform teardown
// after each test.
func setupTest(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	client = lmaelastic.NewWithClient(esClient)
	cache = templates.NewTemplateCache(client, 1, 0)

	// Instantiate backends.
	b = dns.NewDNSFlowBackend(client)
	lb = dns.NewDNSLogBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Each test should take less than 60 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)

	// Function contains teardown logic.
	return func() {
		// Cancel the context.
		cancel()

		// Cleanup any data that might left over from a previous failed run.
		err = testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_dns.%s", cluster))
		require.NoError(t, err)

		// Cancel logging
		logCancel()
	}
}

// TestListDNSFlows tests running a real elasticsearch query to list DNS flows.
func TestListDNSFlows(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	// Put some data into ES so we can query it.
	expected, reqTime := populateDNSLogData(t, ctx, client, clusterInfo.Cluster)

	// Set time range so that we capture all of the populated flow logs.
	opts := v1.DNSFlowParams{}
	opts.TimeRange = &lmav1.TimeRange{}
	opts.TimeRange.From = reqTime.Add(-5 * time.Second)
	opts.TimeRange.To = reqTime.Add(5 * time.Second)

	// Query for flows. There should be a single flow from the populated data.
	r, err := b.List(ctx, clusterInfo, &opts)
	require.NoError(t, err)
	require.Len(t, r.Items, 1)

	// Assert that the flow data is populated correctly.
	require.Equal(t, expected, r.Items[0])
}

// populateDNSLogData writes a series of DNS logs to elasticsearch, and returns the DNSFlow that we
// should expect to exist as a result. This can be used to assert round-tripping and aggregation against ES is working correctly.
func populateDNSLogData(t *testing.T, ctx context.Context, client lmaelastic.Client, cluster string) (v1.DNSFlow, time.Time) {
	// The expected flow log - we'll populate fields as we go.
	reqTime := time.Now()
	expected := v1.DNSFlow{}
	expected.Key = v1.DNSFlowKey{
		Source: v1.Endpoint{
			Namespace:      "default",
			Type:           "wep",
			AggregatedName: "my-deployment",
		},
		ResponseCode: "NOERROR",
	}
	expected.Count = 10
	expected.LatencyStats = &v1.DNSLatencyStats{
		LatencyCount:       0, // To be filled in below.
		MinRequestLatency:  100,
		MaxRequestLatency:  100,
		MeanRequestLatency: 100,
	}

	batch := []v1.DNSLog{}

	for i := 0; i < 10; i++ {
		ip := net.ParseIP("10.0.1.1")
		f := v1.DNSLog{
			StartTime:       reqTime,
			EndTime:         reqTime,
			Type:            v1.DNSLogTypeLog,
			Count:           1,
			ClientName:      "my-deployment-1",
			ClientNameAggr:  "my-deployment",
			ClientNamespace: "default",
			ClientIP:        &ip,
			ClientLabels:    map[string]string{"pickles": "good"},
			QName:           "qname",
			QType:           v1.DNSType(layers.DNSTypeA),
			QClass:          v1.DNSClass(layers.DNSClassIN),
			RCode:           v1.DNSResponseCode(layers.DNSResponseCodeNoErr),
			Servers: []v1.DNSServer{
				{
					Endpoint: v1.Endpoint{
						Name:           "kube-dns-one",
						AggregatedName: "kube-dns",
						Namespace:      "kube-system",
						Type:           v1.WEP,
					},
					IP:     net.ParseIP("10.0.0.10"),
					Labels: map[string]string{"app": "dns"},
				},
			},
			RRSets: v1.DNSRRSets{},
			Latency: v1.DNSLatency{
				Count: 15,
				Mean:  5 * time.Second,
				Max:   10 * time.Second,
			},
			LatencyCount: 100,
			LatencyMean:  100,
			LatencyMax:   100,
		}

		// Add it to the batch
		batch = append(batch, f)

		expected.LatencyStats.LatencyCount += f.LatencyCount
	}

	resp, err := lb.Create(ctx, bapi.ClusterInfo{Cluster: cluster}, batch)
	require.NoError(t, err)
	require.Empty(t, resp.Errors)

	// Refresh the index so that data is readily available for the test. Otherwise, we need to wait
	// for the refresh interval to occur.
	index := fmt.Sprintf("tigera_secure_ee_dns.%s.", cluster)
	err = testutils.RefreshIndex(ctx, client, index)
	require.NoError(t, err)

	return expected, reqTime
}
