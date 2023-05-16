// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/google/gopacket/layers"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func dnslogSetupAndTeardown(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an ES client.
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	lmaClient = lmaelastic.NewWithClient(esClient)

	// Instantiate a client.
	cli, err = NewLinseedClient()
	require.NoError(t, err)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		// Cleanup indices created by the test.
		testutils.CleanupIndices(context.Background(), esClient, cluster)
		logCancel()
		cancel()
	}
}

func TestDNS_FlowLogs(t *testing.T) {
	t.Run("should return an empty list if there are no dns logs", func(t *testing.T) {
		defer dnslogSetupAndTeardown(t)()

		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		logs, err := cli.DNSLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.DNSLog{}, logs.Items)
	})

	t.Run("should create and list dns logs", func(t *testing.T) {
		defer dnslogSetupAndTeardown(t)()

		// Create a basic flow log.
		logs := []v1.DNSLog{
			{
				EndTime: time.Now().UTC(), // TODO: Add more fields
				QName:   "service.namespace.svc.cluster.local",
				QClass:  v1.DNSClass(layers.DNSClassIN),
				QType:   v1.DNSType(layers.DNSTypeAAAA),
				RCode:   v1.DNSResponseCode(layers.DNSResponseCodeNXDomain),
				RRSets:  v1.DNSRRSets{},
			},
		}
		bulk, err := cli.DNSLogs(cluster).Create(ctx, logs)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create dns log did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_dns*")

		// Read it back.
		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.DNSLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		actualLogs := testutils.AssertLogIDAndCopyDNSLogsWithoutID(t, resp)
		require.Equal(t, logs, actualLogs)
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer dnslogSetupAndTeardown(t)()

		totalItems := 5

		// Create 5 dns logs.
		logTime := time.Unix(0, 0).UTC()
		for i := 0; i < totalItems; i++ {
			logs := []v1.DNSLog{
				{
					StartTime: logTime,
					EndTime:   logTime.Add(time.Duration(i) * time.Second), // Make sure logs are ordered.
					Host:      fmt.Sprintf("%d", i),
				},
			}
			bulk, err := cli.DNSLogs(cluster).Create(ctx, logs)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create dns log did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_dns*")

		// Iterate through the first 4 pages and check they are correct.
		var afterKey map[string]interface{}
		for i := 0; i < totalItems-1; i++ {
			params := v1.DNSLogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: logTime.Add(-5 * time.Second),
						To:   logTime.Add(5 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.DNSLogs(cluster).List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			require.Equal(t, []v1.DNSLog{
				{
					StartTime: logTime,
					EndTime:   logTime.Add(time.Duration(i) * time.Second),
					Host:      fmt.Sprintf("%d", i),
					RCode:     v1.DNSResponseCode(0),
					RRSets:    v1.DNSRRSets{},
				},
			}, testutils.AssertLogIDAndCopyDNSLogsWithoutID(t, resp), fmt.Sprintf("DNS #%d did not match", i))
			require.NotNil(t, resp.AfterKey)
			require.Contains(t, resp.AfterKey, "startFrom")
			require.Equal(t, resp.AfterKey["startFrom"], float64(i+1))
			require.Equal(t, resp.TotalHits, int64(totalItems))

			// Use the afterKey for the next query.
			afterKey = resp.AfterKey
		}

		// If we query once more, we should get the last page, and no afterkey, since
		// we have paged through all the items.
		lastItem := totalItems - 1
		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: logTime.Add(-5 * time.Second),
					To:   logTime.Add(5 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.DNSLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Items))
		require.Equal(t, []v1.DNSLog{
			{
				StartTime: logTime,
				EndTime:   logTime.Add(time.Duration(lastItem) * time.Second),
				Host:      fmt.Sprintf("%d", lastItem),
				RCode:     v1.DNSResponseCode(0),
				RRSets:    v1.DNSRRSets{},
			},
		}, testutils.AssertLogIDAndCopyDNSLogsWithoutID(t, resp), fmt.Sprintf("DNS #%d did not match", lastItem))
		require.Equal(t, resp.TotalHits, int64(totalItems))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})
}

func TestFV_DNSLogTenancy(t *testing.T) {
	t.Run("should support tenancy restriction", func(t *testing.T) {
		defer dnslogSetupAndTeardown(t)()

		// Instantiate a client for an unexpected tenant.
		tenantCLI, err := NewLinseedClientForTenant("bad-tenant")
		require.NoError(t, err)

		// Create a basic log. We expect this to fail, since we're using
		// an unexpected tenant ID on the request.
		logs := []v1.DNSLog{
			{
				EndTime: time.Now().UTC(),
				QName:   "service.namespace.svc.cluster.local",
				QClass:  v1.DNSClass(layers.DNSClassIN),
				QType:   v1.DNSType(layers.DNSTypeAAAA),
				RCode:   v1.DNSResponseCode(layers.DNSResponseCodeNXDomain),
				RRSets:  v1.DNSRRSets{},
			},
		}
		bulk, err := tenantCLI.DNSLogs(cluster).Create(ctx, logs)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, bulk)

		// Try a read as well.
		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := tenantCLI.DNSLogs(cluster).List(ctx, &params)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, resp)
	})
}
