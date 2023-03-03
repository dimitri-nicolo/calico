// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l7_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	elastic "github.com/olivere/elastic/v7"

	gojson "encoding/json"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

// TestCreateL7Log tests running a real elasticsearch query to create an L7 log.
func TestCreateL7Log(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{
		Cluster: cluster,
	}

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

	response, err := lb.Create(ctx, clusterInfo, []v1.L7Log{f})
	require.NoError(t, err)
	require.Equal(t, response.Failed, 0)
	require.Equal(t, response.Succeeded, 1)
	require.Len(t, response.Errors, 0)
}

// TestAggregations tests running a real elasticsearch query to get aggregations.
func TestAggregations(t *testing.T) {
	t.Run("should return time-series L7 log aggregation results", func(t *testing.T) {
		defer setupTest(t)()
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		// Start the test numLogs minutes in the past.
		numLogs := 5
		timeBetweenLogs := 10 * time.Second
		testStart := time.Unix(0, 0)
		now := testStart.Add(time.Duration(numLogs) * time.Minute)

		// Several dummy logs.
		logs := []v1.L7Log{}
		for i := 1; i < numLogs; i++ {
			start := testStart.Add(time.Duration(i) * time.Second)
			end := start.Add(timeBetweenLogs)
			l := v1.L7Log{
				StartTime:            start.Unix(),
				EndTime:              end.Unix(),
				DestType:             "wep",
				SourceNamespace:      "default",
				DestNamespace:        "kube-system",
				DestNameAggr:         "kube-dns-*",
				DestServiceNamespace: "default",
				DestServiceName:      "kube-dns",
				ResponseCode:         "200",
				DurationMean:         300,
			}
			logs = append(logs, l)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := lb.Create(ctx, clusterInfo, logs)
		require.NoError(t, err)
		require.Empty(t, resp.Errors)

		// Refresh.
		index := fmt.Sprintf("tigera_secure_ee_l7.%s.", cluster)
		err = backendutils.RefreshIndex(ctx, client, index)
		require.NoError(t, err)

		params := v1.L7AggregationParams{}
		params.TimeRange = &lmav1.TimeRange{}
		params.TimeRange.From = testStart
		params.TimeRange.To = now
		params.NumBuckets = 4

		// Add a simple aggregation to add up the total duration from the logs.
		sumAgg := elastic.NewSumAggregation().Field("duration_mean")
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

		times := []string{"11", "12", "13", "14"}

		for i, b := range ts.Buckets {
			require.Equal(t, int64(1), b.DocCount, fmt.Sprintf("Bucket %d", i))

			// We asked for a count agg, which should include a single log
			// in each bucket.
			count, ok := b.Sum("count")
			require.True(t, ok, "Bucket missing count agg")
			require.NotNil(t, count.Value)
			require.Equal(t, float64(300), *count.Value)

			// The key should be the timestamp for the bucket.
			require.NotNil(t, b.KeyAsString)
			require.Equal(t, times[i], *b.KeyAsString)
		}
	})

	t.Run("should return aggregate stats", func(t *testing.T) {
		defer setupTest(t)()
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		// Start the test numLogs minutes in the past.
		numLogs := 5
		timeBetweenLogs := 10 * time.Second
		testStart := time.Unix(0, 0)
		now := testStart.Add(time.Duration(numLogs) * time.Minute)

		// Several dummy logs.
		logs := []v1.L7Log{}
		for i := 1; i < numLogs; i++ {
			start := testStart.Add(time.Duration(i) * time.Second)
			end := start.Add(timeBetweenLogs)
			l := v1.L7Log{
				StartTime:            start.Unix(),
				EndTime:              end.Unix(),
				DestType:             "wep",
				SourceNamespace:      "default",
				DestNamespace:        "kube-system",
				DestNameAggr:         "kube-dns-*",
				DestServiceNamespace: "default",
				DestServiceName:      "kube-dns",
				ResponseCode:         "200",
				DurationMean:         300,
			}
			logs = append(logs, l)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := lb.Create(ctx, clusterInfo, logs)
		require.NoError(t, err)
		require.Empty(t, resp.Errors)

		// Refresh.
		index := fmt.Sprintf("tigera_secure_ee_l7.%s.", cluster)
		err = backendutils.RefreshIndex(ctx, client, index)
		require.NoError(t, err)

		params := v1.L7AggregationParams{}
		params.TimeRange = &lmav1.TimeRange{}
		params.TimeRange.From = testStart
		params.TimeRange.To = now
		params.NumBuckets = 0 // Return aggregated stats over the whole time range.

		// Add a simple aggregation to add up the total duration_mean from the logs.
		sumAgg := elastic.NewSumAggregation().Field("duration_mean")
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
		require.Equal(t, float64(1200), *count.Value)
	})
}
