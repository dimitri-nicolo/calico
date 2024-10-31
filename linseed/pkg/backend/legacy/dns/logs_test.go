// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns_test

import (
	"context"
	gojson "encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

// TestCreateDNSLog tests running a real elasticsearch query to create a DNS log.
func TestCreateDNSLog(t *testing.T) {
	RunAllModes(t, "TestCreateDNSLog", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		ip := net.ParseIP("10.0.1.1")

		reqTime := time.Unix(0, 0)
		// Create a dummy log.
		f := v1.DNSLog{
			StartTime:       reqTime,
			EndTime:         reqTime,
			Type:            v1.DNSLogTypeLog,
			Count:           1,
			ClientName:      "client-name",
			ClientNameAggr:  "client-",
			ClientNamespace: "default",
			ClientIP:        &ip,
			ClientLabels:    map[string]string{"pickles": "good"},
			QName:           "qname",
			QType:           v1.DNSType(layers.DNSTypeA),
			QClass:          v1.DNSClass(layers.DNSClassIN),
			RCode:           v1.DNSResponseCode(layers.DNSResponseCodeNoErr),
			RRSets:          v1.DNSRRSets{},
			Servers: []v1.DNSServer{
				{
					Endpoint: v1.Endpoint{
						Name:           "kube-dns-one",
						AggregatedName: "kube-dns",
						Namespace:      "kube-system",
					},
					IP:     net.ParseIP("10.0.0.10"),
					Labels: map[string]string{"app": "dns"},
				},
			},
			Latency: v1.DNSLatency{
				Count: 15,
				Mean:  5 * time.Second,
				Max:   10 * time.Second,
			},
			LatencyCount: 100,
			LatencyMean:  100,
			LatencyMax:   100,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		resp, err := lb.Create(ctx, clusterInfo, []v1.DNSLog{f})
		require.NoError(t, err)
		require.Empty(t, resp.Errors)

		// Refresh.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		// List out the log we just created.
		params := v1.DNSLogParams{}
		params.TimeRange = &lmav1.TimeRange{}
		params.TimeRange.From = reqTime.Add(-20 * time.Minute)
		params.TimeRange.To = reqTime.Add(1 * time.Minute)

		listResp, err := lb.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.Len(t, listResp.Items, 1)

		// Compare the result. Timestamps don't serialize well,
		// so ignore them in the comparison.
		actual := listResp.Items[0]
		require.NotEqual(t, time.Time{}, actual.StartTime)
		require.NotEqual(t, time.Time{}, actual.EndTime)
		actual.StartTime = f.StartTime
		actual.EndTime = f.EndTime
		require.Equal(t, f, backendutils.AssertDNSLogIDAndReset(t, actual))

		// If we update the query params to specify matching against the "generated_time"
		// field, we should get no results, because the time right now (>=2023) is years
		// later than reqTime (1970).
		params.TimeRange.Field = lmav1.FieldGeneratedTime
		listResp, err = lb.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.Len(t, listResp.Items, 0)

		// Now if we keep using "generated_time" and change the time range to cover the time
		// period when this test has been running, we should get back that log again.
		params.TimeRange.To = time.Now().Add(10 * time.Second)
		params.TimeRange.From = params.TimeRange.To.Add(-5 * time.Minute)
		listResp, err = lb.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.Len(t, listResp.Items, 1)
	})
}

// TestAggregations tests running a real elasticsearch query to get aggregations.
func TestAggregations(t *testing.T) {
	RunAllModes(t, "should return time-series DNS aggregation results", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}
		ip := net.ParseIP("10.0.1.1")

		// Start the test numLogs minutes in the past.
		numLogs := 5
		timeBetweenLogs := 10 * time.Second
		testStart := time.Unix(0, 0)
		now := testStart.Add(time.Duration(numLogs) * time.Minute)

		// Several dummy logs.
		logs := []v1.DNSLog{}
		for i := 1; i < numLogs; i++ {
			start := testStart.Add(time.Duration(i) * time.Second)
			end := start.Add(timeBetweenLogs)
			log := v1.DNSLog{
				StartTime:       start,
				EndTime:         end,
				Type:            v1.DNSLogTypeLog,
				Count:           1,
				ClientName:      "client-name",
				ClientNameAggr:  "client-",
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
				Latency: v1.DNSLatency{
					Count: 1,
					Mean:  time.Duration(i) * time.Millisecond,
					Max:   time.Duration(2*i) * time.Millisecond,
				},
				LatencyCount: 1,
				LatencyMean:  time.Duration(i) * time.Millisecond,
				LatencyMax:   time.Duration(2*i) * time.Millisecond,
			}
			logs = append(logs, log)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := lb.Create(ctx, clusterInfo, logs)
		require.NoError(t, err)
		require.Empty(t, resp.Errors)

		// Refresh.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		params := v1.DNSAggregationParams{}
		params.TimeRange = &lmav1.TimeRange{}
		params.TimeRange.From = testStart
		params.TimeRange.To = now
		params.NumBuckets = 4

		// Add a simple aggregation to add up the count of logs.
		sumAgg := elastic.NewSumAggregation().Field("count")
		src, err := sumAgg.Source()
		require.NoError(t, err)
		bytes, err := json.Marshal(src)
		require.NoError(t, err)
		params.Aggregations = map[string]gojson.RawMessage{"count": bytes}

		// Use the backend to perform a query.
		aggs, err := lb.Aggregations(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.NotNil(t, aggs)

		ts, ok := aggs.AutoDateHistogram("tb")
		require.True(t, ok)

		// We asked for 4 buckets.
		require.Len(t, ts.Buckets, 4)

		times := []string{
			"1970-01-01T00:00:11.000Z",
			"1970-01-01T00:00:12.000Z",
			"1970-01-01T00:00:13.000Z",
			"1970-01-01T00:00:14.000Z",
		}

		for i, b := range ts.Buckets {
			require.Equal(t, int64(1), b.DocCount, fmt.Sprintf("Bucket %d", i))

			// We asked for a count agg, which should include a single log
			// in each bucket.
			count, ok := b.Sum("count")
			require.True(t, ok, "Bucket missing count agg")
			require.NotNil(t, count.Value)
			require.Equal(t, float64(1), *count.Value)

			// The key should be the timestamp for the bucket.
			require.NotNil(t, b.KeyAsString)
			require.Equal(t, times[i], *b.KeyAsString)
		}
	})

	RunAllModes(t, "should return aggregate DNS stats", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}
		ip := net.ParseIP("10.0.1.1")

		// Start the test numLogs minutes in the past.
		numLogs := 5
		timeBetweenLogs := 10 * time.Second
		testStart := time.Unix(0, 0)
		now := testStart.Add(time.Duration(numLogs) * time.Minute)

		// Several dummy logs.
		logs := []v1.DNSLog{}
		for i := 1; i < numLogs; i++ {
			start := testStart.Add(time.Duration(i) * time.Second)
			end := start.Add(timeBetweenLogs)
			log := v1.DNSLog{
				StartTime:       start,
				EndTime:         end,
				Type:            v1.DNSLogTypeLog,
				Count:           1,
				ClientName:      "client-name",
				ClientNameAggr:  "client-",
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
				Latency: v1.DNSLatency{
					Count: 1,
					Mean:  time.Duration(i) * time.Millisecond,
					Max:   time.Duration(2*i) * time.Millisecond,
				},
				LatencyCount: 1,
				LatencyMean:  time.Duration(i) * time.Millisecond,
				LatencyMax:   time.Duration(2*i) * time.Millisecond,
			}
			logs = append(logs, log)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := lb.Create(ctx, clusterInfo, logs)
		require.NoError(t, err)
		require.Empty(t, resp.Errors)

		// Refresh.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		params := v1.DNSAggregationParams{}
		params.TimeRange = &lmav1.TimeRange{}
		params.TimeRange.From = testStart
		params.TimeRange.To = now
		params.NumBuckets = 0 // Return aggregated stats over the whole time range.

		// Add a simple aggregation to add up the count of logs.
		sumAgg := elastic.NewSumAggregation().Field("count")
		src, err := sumAgg.Source()
		require.NoError(t, err)
		bytes, err := json.Marshal(src)
		require.NoError(t, err)
		params.Aggregations = map[string]gojson.RawMessage{"count": bytes}

		// Use the backend to perform a stats query.
		result, err := lb.Aggregations(ctx, clusterInfo, &params)
		require.NoError(t, err)

		// We should get a sum aggregation with all 4 logs.
		count, ok := result.ValueCount("count")
		require.True(t, ok)
		require.NotNil(t, count.Value)
		require.Equal(t, float64(4), *count.Value)
	})
}
