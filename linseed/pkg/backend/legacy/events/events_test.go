// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package events_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/projectcalico/calico/linseed/pkg/testutils"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/events"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client  lmaelastic.Client
	b       bapi.EventsBackend
	ctx     context.Context
	cache   bapi.Cache
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
	client = lmaelastic.NewWithClient(esClient)
	cache = templates.NewTemplateCache(client, 1, 0)

	// Instantiate a backend.
	b = events.NewBackend(client, cache)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = backendutils.RandomClusterName()

	// Each test should take less than 5 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	// Function contains teardown logic.
	return func() {
		err = backendutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_events.%s", cluster))
		require.NoError(t, err)

		cancel()
		logCancel()
	}
}

// TestCreateEvent tests running a real elasticsearch query to create an event.
func TestCreateEvent(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	// The event to create
	event := v1.Event{
		Time:            time.Now().Unix(),
		Description:     "Just a city event",
		Origin:          "South Detroit",
		Severity:        1,
		Type:            "TODO",
		DestIP:          testutils.StringPtr("192.168.1.1"),
		DestName:        "anywhere-1234",
		DestNameAggr:    "anywhere",
		DestPort:        testutils.Int64Ptr(53),
		Dismissed:       false,
		Host:            "midnight-train",
		SourceIP:        testutils.StringPtr("192.168.2.2"),
		SourceName:      "south-detroit-1234",
		SourceNameAggr:  "south-detroit",
		SourceNamespace: "michigan",
		SourcePort:      testutils.Int64Ptr(48127),
	}

	// Create the event in ES.
	resp, err := b.Create(ctx, clusterInfo, []v1.Event{event})
	require.NoError(t, err)
	require.Equal(t, 0, len(resp.Errors))
	require.Equal(t, 1, resp.Total)
	require.Equal(t, 0, resp.Failed)
	require.Equal(t, 1, resp.Succeeded)

	// Refresh the index.
	err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_events.*")
	require.NoError(t, err)

	// List the events and make sure the one we created is present.
	results, err := b.List(ctx, clusterInfo, &v1.EventParams{
		QueryParams: v1.QueryParams{
			TimeRange: &lmav1.TimeRange{
				From: time.Now().Add(-1 * time.Minute),
				To:   time.Now().Add(1 * time.Minute),
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, 1, results)
	require.Equal(t, 1, len(results.Items))

	// We expect the ID to be present, but it's a random value so we
	// can't assert on the exact value.
	require.Equal(t, event, backendutils.AssertEventIDAndReset(t, results.Items[0]))
}

func TestEventSelector(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	// The event to create
	event := v1.Event{
		Time:            time.Now().Unix(),
		Description:     "Just a city event",
		Origin:          "South Detroit",
		Severity:        1,
		Type:            "TODO",
		DestIP:          testutils.StringPtr("192.168.1.1"),
		DestName:        "anywhere-1234",
		DestNameAggr:    "anywhere",
		DestPort:        testutils.Int64Ptr(53),
		Dismissed:       false,
		Host:            "midnight-train",
		SourceIP:        testutils.StringPtr("192.168.2.2"),
		SourceName:      "south-detroit-1234",
		SourceNameAggr:  "south-detroit",
		SourceNamespace: "michigan",
		SourcePort:      testutils.Int64Ptr(48127),
	}

	// Create the event in ES.
	resp, err := b.Create(ctx, clusterInfo, []v1.Event{event})
	require.NoError(t, err)
	require.Equal(t, 0, len(resp.Errors))
	require.Equal(t, 1, resp.Total)
	require.Equal(t, 0, resp.Failed)
	require.Equal(t, 1, resp.Succeeded)

	// Refresh the index.
	err = backendutils.RefreshIndex(ctx, client, "tigera_secure_ee_events.*")
	require.NoError(t, err)

	// This will be used to test various selectors to list events.
	// Selectors can be invalid and return an error.
	// When valid, the number of results we get can change depending
	// on what events the selector matches.
	testSelector := func(selector string, numResults int, shouldSucceed bool) {
		r, e := b.List(ctx, clusterInfo, &v1.EventParams{
			LogSelectionParams: v1.LogSelectionParams{
				Selector: selector,
			},
		})
		if shouldSucceed {
			require.NoError(t, e)
			require.Equal(t, numResults, len(r.Items))
			if numResults > 0 {
				// We expect the ID to be present, but it's a random value so we
				// can't assert on the exact value.
				require.Equal(t, event, backendutils.AssertEventIDAndReset(t, r.Items[0]))
			}
		} else {
			require.Error(t, e)
		}
	}

	tests := []struct {
		selector      string
		numResults    int
		shouldSucceed bool
	}{
		// These ones match events as expected
		{"host=\"midnight-train\"", 1, true},
		{"source_name=\"south-detroit-1234\"", 1, true},
		{"origin=\"South Detroit\"", 1, true},
		{"dest_port=53", 1, true},
		{"source_port=48127", 1, true},
		{"source_port > 1024", 1, true},

		// Valid but do not match any event
		{"host=\"some-other-host\"", 0, true},

		// Those fail for invalid keys.
		// Valid keys are defined in libcalico-go/lib/validator/v3/query/validate_events.go.
		// The validation is performed in linseed/pkg/internal/lma/elastic/index/alerts.go.
		// If we comment out the call to `query.Validate()` in alerts.go, the "invalid key"
		// error won't occur and the resulting ES query will be executed.
		{"Host=\"midnight-train\"", 0, false},
		{"description=\"Just a city event\"", 0, false},
		{"type=\"TODO\"", 0, false},
		{"severity=1", 0, false},
		{"time>0", 0, false},

		// The dismissed key is a bit odd (probably like all boolean values).
		// There is validation for the value, but it does not return
		// the event with a seemingly valid selector (dismissed=false).
		// Instead we need to use something like "dismissed != true".
		// The UI uses "NOT dismissed = true"
		{"dismissed=f", 0, false},
		{"dismissed=t", 0, false},
		{"dismissed=False", 0, false},
		{"dismissed=True", 0, false},
		{"dismissed=0", 0, false},
		{"dismissed=1", 0, false},
		{"dismissed=false", 0, true},
		{"dismissed=true", 0, true},
		{"dismissed=\"false\"", 0, true},
		{"dismissed=\"true\"", 0, true},
		{"dismissed!=\"true\"", 1, true},
		{"dismissed!=true", 1, true},
		{"dismissed != true", 1, true},
		{"NOT dismissed = true", 1, true},
		{"NOT dismissed", 0, false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("TestEventSelector: %s", tt.selector)
		t.Run(name, func(t *testing.T) {
			testSelector(tt.selector, tt.numResults, tt.shouldSucceed)
		})
	}
}
