// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func eventsSetupAndTeardown(t *testing.T) func() {
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

func TestFV_Events(t *testing.T) {
	t.Run("should return an empty list if there are no events", func(t *testing.T) {
		defer eventsSetupAndTeardown(t)()

		params := v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		events, err := cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.Event{}, events.Items)
	})

	t.Run("should create and list events", func(t *testing.T) {
		defer eventsSetupAndTeardown(t)()

		// Create a basic event.
		events := []v1.Event{
			{
				Time:        v1.NewEventTimestamp(time.Now().Unix()),
				Description: "A rather uneventful evening",
				Origin:      "TODO",
				Severity:    1,
				Type:        "TODO",
			},
		}
		bulk, err := cli.Events(cluster).Create(ctx, events)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create event did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_events*")

		// Read it back.
		params := v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set, but random, so we can't assert on its value.
		require.Equal(t, events, testutils.AssertLogIDAndCopyEventsWithoutID(t, resp))
	})

	t.Run("should dismiss and delete events", func(t *testing.T) {
		defer eventsSetupAndTeardown(t)()

		// Create a basic event.
		events := []v1.Event{
			{
				Time:        v1.NewEventTimestamp(time.Now().Unix()),
				Description: "A rather uneventful evening",
				Origin:      "TODO",
				Severity:    1,
				Type:        "TODO",
			},
		}
		bulk, err := cli.Events(cluster).Create(ctx, events)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create event did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_events*")

		// Read it back.
		params := v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)

		// The ID should be set, it should not be dismissed.
		require.NotEqual(t, "", resp.Items[0].ID)
		require.False(t, resp.Items[0].Dismissed)

		// We should be able to dismiss the event.
		bulk, err = cli.Events(cluster).Dismiss(ctx, resp.Items)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "dismiss event did not succeed")

		// Reading it back should show the event as dismissed.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_events*")
		resp, err = cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)
		require.True(t, resp.Items[0].Dismissed)

		// Now, delete the event.
		bulk, err = cli.Events(cluster).Delete(ctx, resp.Items)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "delete event did not succeed")

		// Reading it back should show the no events.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_events*")
		resp, err = cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Len(t, resp.Items, 0)
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer eventsSetupAndTeardown(t)()

		totalItems := 5

		// Create 5 events.
		logTime := time.Unix(100, 0).UTC()
		for i := 0; i < totalItems; i++ {
			events := []v1.Event{
				{
					Time: v1.NewEventTimestamp(logTime.Unix() + int64(i)), // Make sure events are ordered.
					Host: fmt.Sprintf("%d", i),
				},
			}
			bulk, err := cli.Events(cluster).Create(ctx, events)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create events did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_event*")

		// Read them back one at a time.
		var afterKey map[string]interface{}
		for i := 0; i < totalItems-1; i++ {
			params := v1.EventParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: logTime.Add(-5 * time.Second),
						To:   logTime.Add(5 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.Events(cluster).List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			require.Equal(t, []v1.Event{
				{
					Time: v1.NewEventTimestamp(logTime.Unix() + int64(i)),
					Host: fmt.Sprintf("%d", i),
				},
			}, testutils.AssertLogIDAndCopyEventsWithoutID(t, resp), fmt.Sprintf("Event #%d did not match", i))
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
		params := v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: logTime.Add(-5 * time.Second),
					To:   logTime.Add(5 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.Events(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Items))
		require.Equal(t, []v1.Event{
			{
				Time: v1.NewEventTimestamp(logTime.Unix() + int64(lastItem)),
				Host: fmt.Sprintf("%d", lastItem),
			},
		}, testutils.AssertLogIDAndCopyEventsWithoutID(t, resp), fmt.Sprintf("Event #%d did not match", lastItem))
		require.Equal(t, resp.TotalHits, int64(totalItems))

		// Once we reach the end of the data, we should not receive
		// an afterKey
		require.Nil(t, resp.AfterKey)
	})
}

func TestFV_EventsTenancy(t *testing.T) {
	t.Run("should support tenancy restriction", func(t *testing.T) {
		defer eventsSetupAndTeardown(t)()

		// Instantiate a client for an unexpected tenant.
		tenantCLI, err := NewLinseedClientForTenant("bad-tenant")
		require.NoError(t, err)

		// Create a basic log. We expect this to fail, since we're using
		// an unexpected tenant ID on the request.
		events := []v1.Event{
			{
				Time:        v1.NewEventTimestamp(time.Now().Unix()),
				Description: "A rather uneventful evening",
				Origin:      "TODO",
				Severity:    1,
				Type:        "TODO",
			},
		}
		bulk, err := tenantCLI.Events(cluster).Create(ctx, events)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, bulk)

		// Try a read as well.
		params := v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := tenantCLI.Events(cluster).List(ctx, &params)
		require.ErrorContains(t, err, "Bad tenant identifier")
		require.Nil(t, resp)
	})
}
