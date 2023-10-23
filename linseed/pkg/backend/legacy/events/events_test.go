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
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	client      lmaelastic.Client
	b           bapi.EventsBackend
	ctx         context.Context
	cache       bapi.IndexInitializer
	cluster     string
	indexGetter bapi.Index
)

// RunAllModes runs the given test function twice, once using the single-index backend, and once using
// the multi-index backend.
func RunAllModes(t *testing.T, name string, testFn func(t *testing.T)) {
	// Run using the multi-index backend.
	t.Run(fmt.Sprintf("%s [legacy]", name), func(t *testing.T) {
		defer setupTest(t, false)()
		testFn(t)
	})

	// Run using the single-index backend.
	t.Run(fmt.Sprintf("%s [singleindex]", name), func(t *testing.T) {
		defer setupTest(t, true)()
		testFn(t)
	})
}

// setupTest runs common logic before each test, and also returns a function to perform teardown
// after each test.
func setupTest(t *testing.T, singleIndex bool) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	client = lmaelastic.NewWithClient(esClient)
	cache = templates.NewCachedInitializer(client, 1, 0)

	// Instantiate a backend.
	if singleIndex {
		b = events.NewSingleIndexBackend(client, cache, 10000)
		indexGetter = index.AlertsIndex
	} else {
		b = events.NewBackend(client, cache, 10000)
		indexGetter = index.EventsMultiIndex
	}

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = backendutils.RandomClusterName()

	// Each test should take less than 5 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	// Function contains teardown logic.
	return func() {
		err = backendutils.CleanupIndices(context.Background(), esClient, singleIndex, indexGetter, bapi.ClusterInfo{Cluster: cluster})
		require.NoError(t, err)

		cancel()
		logCancel()
	}
}

// TestCreateEvent tests running a real elasticsearch query to create an event.
func TestCreateEvent(t *testing.T) {
	// The event to create
	event := v1.Event{
		Time:            v1.NewEventTimestamp(time.Now().Unix()),
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

	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		RunAllModes(t, "Create Event with all valid params", func(t *testing.T) {
			clusterInfo := bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}
			// Create the event in ES.
			resp, err := b.Create(ctx, clusterInfo, []v1.Event{event})
			require.NoError(t, err)
			require.Equal(t, 0, len(resp.Errors))
			require.Equal(t, 1, resp.Total)
			require.Equal(t, 0, resp.Failed)
			require.Equal(t, 1, resp.Succeeded)

			// Refresh the index.
			err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
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
		})
	}

	RunAllModes(t, "Create Event with given event id", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}
		event.ID = "SOMERANDOMID"
		// Create the event in ES.
		resp, err := b.Create(ctx, clusterInfo, []v1.Event{event})
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		// Refresh the index.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
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

		// We expect the ID to be same as the passed event id.
		require.Equal(t, event, results.Items[0])
	})

	RunAllModes(t, "Invalid Cluster Info", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{}
		resp, err := b.Create(ctx, clusterInfo, []v1.Event{event})
		require.Error(t, err)
		require.Equal(t, "no cluster ID provided on request", err.Error())
		require.Nil(t, resp)

		results, err := b.List(ctx, clusterInfo, &v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-1 * time.Minute),
					To:   time.Now().Add(1 * time.Minute),
				},
			},
		})
		require.Error(t, err)
		require.Equal(t, "no cluster ID on request", err.Error())
		require.Nil(t, results)
	})

	RunAllModes(t, "Invalid start from", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		results, err := b.List(ctx, clusterInfo, &v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-1 * time.Minute),
					To:   time.Now().Add(1 * time.Minute),
				},
				AfterKey: map[string]interface{}{"startFrom": "badvalue"},
			},
		})
		require.Error(t, err)
		require.Equal(t, "Could not parse startFrom (badvalue) as an integer", err.Error())
		require.Nil(t, results)
	})

	RunAllModes(t, "Create failure", func(t *testing.T) {
		clusterInfo := bapi.ClusterInfo{}
		resp, err := b.Create(ctx, clusterInfo, []v1.Event{event})
		require.Equal(t, "1", "1")
		require.Error(t, err)
		require.Nil(t, resp)
	})
}

func TestEventSelector(t *testing.T) {
	// This will be used to test various selectors to list events.
	// Selectors can be invalid and return an error.
	// When valid, the number of results we get can change depending
	// on what events the selector matches.
	testSelector := func(t *testing.T, selector string, numResults int, shouldSucceed bool) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		// The event to create
		event := v1.Event{
			Time:            v1.NewEventTimestamp(time.Now().Unix()),
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
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

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
		{"severity=1", 1, true},
		{"origin IN {'**'}", 1, true}, // Matches if non-empty
		{"name NOTIN {'**'}", 1, true},
		{"description=\"Just a city event\"", 1, true},
		{"time>0", 1, true},

		// Valid but do not match any event
		{"host=\"some-other-host\"", 0, true},
		{"severity > 10", 0, true},

		// Those fail for invalid keys.
		// Valid keys are defined in libcalico-go/lib/validator/v3/query/validate_events.go.
		// The validation is performed in linseed/pkg/internal/lma/elastic/index/alerts.go.
		// If we comment out the call to `query.Validate()` in alerts.go, the "invalid key"
		// error won't occur and the resulting ES query will be executed.
		{"Host=\"midnight-train\"", 0, false},
		{"type=\"TODO\"", 0, false},
		{"origin IN {'*'}", 0, false}, // Need to use `**` to match any non-empty value

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
		RunAllModes(t, name, func(t *testing.T) {
			testSelector(t, tt.selector, tt.numResults, tt.shouldSucceed)
		})
	}
}

func TestSecurityEvents(t *testing.T) {
	events := []v1.Event{
		{
			Time:            v1.NewEventTimestamp(time.Now().Unix()),
			Description:     "A sample Security Event",
			Name:            "Proc File Access",
			Origin:          "Proc File Access",
			Severity:        90,
			Type:            "runtime_security",
			Dismissed:       false,
			Host:            "test-host",
			SourceName:      "my-pod-123",
			SourceNameAggr:  "my-pod",
			SourceNamespace: "test-ns",
			AttackVector:    "Process",
			MitreTactic:     "Access",
			MitreIDs:        &[]string{"T1003.007", "T1057", "T1083"},
			Mitigations:     &[]string{"Do not expose proc file system to your containers.", "Do not run containers as root."},
		},
		{
			Time:         v1.NewEventTimestamp(time.Now().Unix()),
			Description:  "A sample WAF Security Event",
			Name:         "WAF Event",
			Origin:       "waf-new-alert-rule-info",
			Severity:     100,
			Type:         "global_alert",
			Dismissed:    false,
			Host:         "test-host",
			AttackVector: "Network",
			MitreTactic:  "Access",
			MitreIDs:     &[]string{"T1190"},
			Mitigations:  &[]string{"Use WAF :)"},
		},
		{
			Time:        v1.NewEventTimestamp(time.Now().Unix()),
			Description: "A hopeless Security Event",
			// No severity, mitigations etc...
		},
	}

	testEventsFiltering := func(t *testing.T, selector string, expectedEvents []v1.Event) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		// Create the event in ES.
		resp, err := b.Create(ctx, clusterInfo, events)
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, len(events), resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, len(events), resp.Succeeded)

		// Refresh the index.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		r, e := b.List(ctx, clusterInfo, &v1.EventParams{
			LogSelectionParams: v1.LogSelectionParams{
				Selector: selector,
			},
		})
		require.NoError(t, e)
		require.Equal(t, len(expectedEvents), len(r.Items))
		for i := range expectedEvents {
			// We expect the ID to be present, but it's a random value so we
			// can't assert on the exact value.
			require.Equal(t, expectedEvents[i], backendutils.AssertEventIDAndReset(t, r.Items[i]))
		}
	}

	tests := []struct {
		selector       string
		expectedEvents []v1.Event
	}{
		{"name=\"Proc File Access\"", []v1.Event{events[0]}},
		{"name IN {\"*file access*\"}", []v1.Event{events[0]}},
		{"origin=\"Proc File Access\"", []v1.Event{events[0]}},
		{"attack_vector=\"Process\"", []v1.Event{events[0]}},
		// Some fields support case-insensitive filtering for Security Events Management UI
		{"attack_vector=\"process\"", []v1.Event{events[0]}},
		{"attack_vector=\"PROCESS\"", []v1.Event{events[0]}},
		{"mitre_tactic=\"Access\"", []v1.Event{events[0], events[1]}},
		{"mitre_tactic=\"access\"", []v1.Event{events[0], events[1]}},
		{"mitre_tactic=\"ACCESS\"", []v1.Event{events[0], events[1]}},
		{"name=\"WAF Event\"", []v1.Event{events[1]}},
		{"name IN {\"*waf*\"}", []v1.Event{events[1]}},
		{"type=global_alert AND origin='waf-new-alert-rule-info'", []v1.Event{events[1]}},
		{"attack_vector=\"Network\"", []v1.Event{events[1]}},
		{"attack_vector=\"netWORK\"", []v1.Event{events[1]}},
		{"host='test-host'", []v1.Event{events[0], events[1]}},
		{"description IN {'*security event*'}", []v1.Event{events[0], events[1], events[2]}},
		{"severity > 95", []v1.Event{events[1]}},
		{"severity = 100", []v1.Event{events[1]}},
		{"severity>=70", []v1.Event{events[0], events[1]}},
		{"severity < 70", []v1.Event{events[2]}},
		{"severity>=70 AND severity < 95", []v1.Event{events[0]}},
		{"mitre_ids IN {'T1190'}", []v1.Event{events[1]}},
		{"mitre_ids IN {'T1057'}", []v1.Event{events[0]}},
		{"mitre_ids IN {'t1057'}", []v1.Event{events[0]}},
		// Getting a bit silly now, but it's nice that it works
		{"mitre_ids IN {'T10*'}", []v1.Event{events[0]}},
		{"mitre_ids NOTIN {'T10*'}", []v1.Event{events[1], events[2]}},
		{"mitigations IN {'Do not*'}", []v1.Event{events[0]}},
		{"mitigations IN {'DO NOT*'}", []v1.Event{events[0]}},
		{"mitigations IN {'Use WAF :)'}", []v1.Event{events[1]}},
		{"mitigations IN {'**'}", []v1.Event{events[0], events[1]}},
		{"mitigations NOTIN {'**'}", []v1.Event{events[2]}},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("TestEventSelector: %s", tt.selector)
		RunAllModes(t, name, func(t *testing.T) {
			testEventsFiltering(t, tt.selector, tt.expectedEvents)
		})
	}
}

func TestPagination(t *testing.T) {
	// The event to create
	event := v1.Event{
		Time:            v1.NewEventTimestamp(time.Now().Unix()),
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

	listSize := 21
	events := make([]v1.Event, 0, listSize)
	for i := 0; i < listSize; i++ {
		events = append(events, event)
	}

	testSelector := func(maxPageSize int, numResults int, afterkey map[string]interface{}, shouldSucceed bool, errmsg string) {
		clusterInfo := bapi.ClusterInfo{Cluster: cluster}

		// Create the events in ES.
		resp, err := b.Create(ctx, clusterInfo, events)
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, listSize, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, listSize, resp.Succeeded)

		// Refresh the index.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		r, e := b.List(ctx, clusterInfo, &v1.EventParams{
			QueryParams: v1.QueryParams{
				MaxPageSize: maxPageSize,
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-1 * time.Minute),
					To:   time.Now().Add(1 * time.Minute),
				},
				AfterKey: afterkey,
			},
		})
		if shouldSucceed {
			require.NoError(t, e)
			require.NotNil(t, 1, r)
			require.Equal(t, numResults, len(r.Items))
			if numResults > 0 {
				// We expect the ID to be present, but it's a random value so we
				// can't assert on the exact value.
				require.Equal(t, event, backendutils.AssertEventIDAndReset(t, r.Items[0]))
			}
		} else {
			require.Error(t, e)
			require.Contains(t, e.Error(), errmsg)
		}
	}

	RunAllModes(t, "check results size is same as max page size", func(t *testing.T) {
		testSelector(10, 10, nil, true, "")
	})

	RunAllModes(t, "check afterkey is used to load the rest of the items", func(t *testing.T) {
		testSelector(0, 11, map[string]interface{}{"startFrom": 10}, true, "")
	})

	RunAllModes(t, "check negative max page size returns error", func(t *testing.T) {
		testSelector(-10, 10, nil, false, "parameter cannot be negative")
	})

	RunAllModes(t, "check afterkey is used to load the rest of the items", func(t *testing.T) {
		testSelector(3, 3, map[string]interface{}{"startFrom": 10}, true, "")
	})
}

func TestSorting(t *testing.T) {
	// Variables used for sorting tests.
	var params v1.EventParams
	var events []v1.Event
	var clusterInfo bapi.ClusterInfo

	// sortingSetup performs additional setup for sorting tests.
	sortingSetup := func() {
		clusterInfo = bapi.ClusterInfo{Cluster: cluster}
		createTime := []time.Time{time.Unix(100, 0), time.Unix(500, 0)}

		// Create array of events.
		listSize := 2
		events = make([]v1.Event, 0, listSize)
		for i := 0; i < listSize; i++ {
			event := v1.Event{
				Time:         v1.NewEventTimestamp(createTime[i].Unix()),
				Description:  "Just a city event",
				Origin:       "South Detroit",
				Severity:     1,
				Type:         "TODO",
				DestIP:       testutils.StringPtr("192.168.1.1"),
				DestName:     "anywhere-1234",
				DestNameAggr: "anywhere",
				DestPort:     testutils.Int64Ptr(53),
				Dismissed:    false,
				// Host:            "midnight-train",
				SourceIP:        testutils.StringPtr("192.168.2.2"),
				SourceName:      "south-detroit-1234",
				SourceNameAggr:  "south-detroit",
				SourceNamespace: "michigan",
				SourcePort:      testutils.Int64Ptr(48127),
			}
			event.Host = fmt.Sprintf("midnight-train-%v", i)
			events = append(events, event)
		}

		// Create the event in ES.
		resp, err := b.Create(ctx, clusterInfo, events)
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, 2, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 2, resp.Succeeded)

		// Refresh the index.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		params = v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Unix(50, 0),
					To:   time.Unix(600, 0),
				},
			},
		}

		// List the events and make sure the events we created are present.
		results, err := b.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.NotNil(t, 1, results)
		require.Equal(t, 2, len(results.Items))

		require.Equal(t, events[0], backendutils.AssertEventIDAndReset(t, results.Items[0]))
		require.Equal(t, events[1], backendutils.AssertEventIDAndReset(t, results.Items[1]))
	}

	RunAllModes(t, "Sort using the time in descending order", func(t *testing.T) {
		sortingSetup()

		// Query again, this time sorting in order to get the logs in reverse order.
		params.Sort = []v1.SearchRequestSortBy{
			{
				Field:      "time",
				Descending: true,
			},
		}

		// List again to check the decending sort works.
		results, err := b.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.NotNil(t, 1, results)
		require.Equal(t, 2, len(results.Items))

		require.Equal(t, events[0], backendutils.AssertEventIDAndReset(t, results.Items[1]))
		require.Equal(t, events[1], backendutils.AssertEventIDAndReset(t, results.Items[0]))
	})

	RunAllModes(t, "Sort using the host in descending order", func(t *testing.T) {
		sortingSetup()

		// Query again, this time sorting in order to get the logs in reverse order.
		params.Sort = []v1.SearchRequestSortBy{
			{
				Field:      "host",
				Descending: true,
			},
		}
		// List again to check the decending sort works.
		results, err := b.List(ctx, clusterInfo, &params)
		require.NoError(t, err)
		require.NotNil(t, 1, results)
		require.Equal(t, 2, len(results.Items))

		require.Equal(t, events[0].Host, results.Items[1].Host)
		require.Equal(t, events[1].Host, results.Items[0].Host)
	})
}

func TestDismissEvent(t *testing.T) {
	// Variables initialized in dissmissSetup but used in the tests.
	var results *v1.List[v1.Event]
	var clusterInfo bapi.ClusterInfo

	dismissSetup := func(t *testing.T) {
		// The event to create
		event := v1.Event{
			Time:            v1.NewEventTimestamp(time.Now().Unix()),
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
		clusterInfo = bapi.ClusterInfo{Cluster: cluster}

		// Create the event in ES.
		resp, err := b.Create(ctx, clusterInfo, []v1.Event{event})
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		// Refresh the index.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		// List the events and make sure the one we created is present.
		results, err = b.List(ctx, clusterInfo, &v1.EventParams{
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

	RunAllModes(t, "Dismiss an Event", func(t *testing.T) {
		dismissSetup(t)

		// Dismiss Event
		resp, err := b.UpdateDismissFlag(ctx, clusterInfo, []v1.Event{
			{
				ID:        results.Items[0].ID,
				Dismissed: true,
			},
		})
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		results, err = b.List(ctx, clusterInfo, &v1.EventParams{
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
		// Check event for the dismiss flag true.
		require.Equal(t, true, results.Items[0].Dismissed)
	})

	RunAllModes(t, "Restore an Event", func(t *testing.T) {
		dismissSetup(t)

		// Dismiss Event (so that it can be restored)
		resp, err := b.UpdateDismissFlag(ctx, clusterInfo, []v1.Event{
			{
				ID:        results.Items[0].ID,
				Dismissed: true,
			},
		})
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		results, err = b.List(ctx, clusterInfo, &v1.EventParams{
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
		// Check event for the dismiss flag true.
		require.Equal(t, true, results.Items[0].Dismissed)

		// Restore dismissed event
		resp, err = b.UpdateDismissFlag(ctx, clusterInfo, []v1.Event{
			{
				ID:        results.Items[0].ID,
				Dismissed: false,
			},
		})
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		results, err = b.List(ctx, clusterInfo, &v1.EventParams{
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
		// Check event for the dismiss flag false.
		require.Equal(t, false, results.Items[0].Dismissed)
	})

	RunAllModes(t, "Try to Dismiss an event that does not exist in Elastic", func(t *testing.T) {
		dismissSetup(t)

		invalidEvent := v1.Event{
			ID: "INVALID",
		}
		resp, err := b.UpdateDismissFlag(ctx, clusterInfo, []v1.Event{invalidEvent})
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Succeeded)
		require.Equal(t, 1, resp.Failed)
		require.NotNil(t, resp.Errors)
		require.Equal(t, "document_missing_exception", resp.Errors[0].Type)
	})

	RunAllModes(t, "Invalid Cluster Info", func(t *testing.T) {
		dismissSetup(t)

		invalidClusterInfo := bapi.ClusterInfo{}
		resp, err := b.UpdateDismissFlag(ctx, invalidClusterInfo, []v1.Event{results.Items[0]})
		require.Error(t, err)
		require.Equal(t, "no cluster ID on request", err.Error())
		require.Nil(t, resp)
	})
}

func TestMultiTenantDismissal(t *testing.T) {
	RunAllModes(t, "tenants should not be able to dismiss other tenant's events", func(t *testing.T) {
		// Create an event for tenantA.
		event := v1.Event{
			Time:            v1.NewEventTimestamp(time.Now().Unix()),
			Description:     "Tenant A event",
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
		clusterInfoA := bapi.ClusterInfo{Cluster: cluster, Tenant: "tenanta"}
		_, err := b.Create(ctx, clusterInfoA, []v1.Event{event})
		require.NoError(t, err)

		// Create an event for tenantB.
		event.Description = "Tenant B event"
		clusterInfoB := bapi.ClusterInfo{Cluster: cluster, Tenant: "tenantb"}
		_, err = b.Create(ctx, clusterInfoB, []v1.Event{event})
		require.NoError(t, err)

		// Refresh
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfoA))
		require.NoError(t, err)
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfoB))
		require.NoError(t, err)

		// Get the ID for each.
		resultsA, err := b.List(ctx, clusterInfoA, &v1.EventParams{
			QueryParams: v1.QueryParams{TimeRange: &lmav1.TimeRange{From: time.Now().Add(-1 * time.Minute), To: time.Now().Add(1 * time.Minute)}},
		})
		require.NoError(t, err)
		require.NotNil(t, 1, resultsA)
		require.Equal(t, 1, len(resultsA.Items))
		eventIDA := resultsA.Items[0].ID
		resultsB, err := b.List(ctx, clusterInfoB, &v1.EventParams{
			QueryParams: v1.QueryParams{TimeRange: &lmav1.TimeRange{From: time.Now().Add(-1 * time.Minute), To: time.Now().Add(1 * time.Minute)}},
		})
		require.NoError(t, err)
		require.NotNil(t, 1, resultsB)
		require.Equal(t, 1, len(resultsB.Items))
		eventIDB := resultsB.Items[0].ID

		// We expect the IDs to be different.
		require.NotEqual(t, eventIDA, eventIDB)

		// Try to dismiss tenant A's event using tenant B's info.
		resp, err := b.UpdateDismissFlag(ctx, clusterInfoB, []v1.Event{
			{
				ID:        resultsA.Items[0].ID,
				Dismissed: true,
			},
		})

		require.NoError(t, err)

		if indexGetter.IsSingleIndex() {
			require.Len(t, resp.Errors, 1)
			require.Equal(t, resp.Failed, 1)
		}

		// Check that neither event is dismissed.
		for _, c := range []bapi.ClusterInfo{clusterInfoA, clusterInfoB} {
			results, err := b.List(ctx, c, &v1.EventParams{
				QueryParams: v1.QueryParams{TimeRange: &lmav1.TimeRange{From: time.Now().Add(-1 * time.Minute), To: time.Now().Add(1 * time.Minute)}},
			})
			require.NoError(t, err)
			require.NotNil(t, 1, results)
			require.Equal(t, 1, len(results.Items))
			require.False(t, results.Items[0].Dismissed, "event should not be dismissed")
		}

		// Dismiss tenant A's event using tenant A's info.
		_, err = b.UpdateDismissFlag(ctx, clusterInfoA, []v1.Event{
			{
				ID:        resultsA.Items[0].ID,
				Dismissed: true,
			},
		})

		require.NoError(t, err)

		// It should be dismissed now.
		results, err := b.List(ctx, clusterInfoA, &v1.EventParams{
			QueryParams: v1.QueryParams{TimeRange: &lmav1.TimeRange{From: time.Now().Add(-1 * time.Minute), To: time.Now().Add(1 * time.Minute)}},
		})
		require.NoError(t, err)
		require.NotNil(t, 1, results)
		require.Equal(t, 1, len(results.Items))
		require.True(t, results.Items[0].Dismissed, "event should be dismissed")
	})
}

func TestMultiTenantDeletion(t *testing.T) {
	RunAllModes(t, "tenants should not be able to delete other tenant's events", func(t *testing.T) {
		// Create an event for tenantA.
		event := v1.Event{
			Time:            v1.NewEventTimestamp(time.Now().Unix()),
			Description:     "Tenant A event",
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
		clusterInfoA := bapi.ClusterInfo{Cluster: cluster, Tenant: "tenanta"}
		_, err := b.Create(ctx, clusterInfoA, []v1.Event{event})
		require.NoError(t, err)

		// Create an event for tenantB.
		event.Description = "Tenant B event"
		clusterInfoB := bapi.ClusterInfo{Cluster: cluster, Tenant: "tenantb"}
		_, err = b.Create(ctx, clusterInfoB, []v1.Event{event})
		require.NoError(t, err)

		// Refresh
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfoA))
		require.NoError(t, err)
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfoB))
		require.NoError(t, err)

		// Get the ID for each.
		resultsA, err := b.List(ctx, clusterInfoA, &v1.EventParams{
			QueryParams: v1.QueryParams{TimeRange: &lmav1.TimeRange{From: time.Now().Add(-1 * time.Minute), To: time.Now().Add(1 * time.Minute)}},
		})
		require.NoError(t, err)
		require.NotNil(t, 1, resultsA)
		require.Equal(t, 1, len(resultsA.Items))
		eventIDA := resultsA.Items[0].ID
		resultsB, err := b.List(ctx, clusterInfoB, &v1.EventParams{
			QueryParams: v1.QueryParams{TimeRange: &lmav1.TimeRange{From: time.Now().Add(-1 * time.Minute), To: time.Now().Add(1 * time.Minute)}},
		})
		require.NoError(t, err)
		require.NotNil(t, 1, resultsB)
		require.Equal(t, 1, len(resultsB.Items))
		eventIDB := resultsB.Items[0].ID

		// We expect the IDs to be different.
		require.NotEqual(t, eventIDA, eventIDB)

		// Try to delete tenant A's event using tenant B's info.
		resp, err := b.Delete(ctx, clusterInfoB, []v1.Event{resultsA.Items[0]})
		require.NoError(t, err)

		if indexGetter.IsSingleIndex() {
			require.Len(t, resp.Errors, 1)
			require.Equal(t, resp.Failed, 1)
		}

		// Check that neither event is deleted.
		for _, c := range []bapi.ClusterInfo{clusterInfoA, clusterInfoB} {
			results, err := b.List(ctx, c, &v1.EventParams{
				QueryParams: v1.QueryParams{TimeRange: &lmav1.TimeRange{From: time.Now().Add(-1 * time.Minute), To: time.Now().Add(1 * time.Minute)}},
			})
			require.NoError(t, err)
			require.NotNil(t, 1, results)
			require.Equal(t, 1, len(results.Items))
			require.False(t, results.Items[0].Dismissed, "event should not be dismissed")
		}

		// Delete tenant A's event using tenant A's info.
		_, err = b.Delete(ctx, clusterInfoA, []v1.Event{resultsA.Items[0]})
		require.NoError(t, err)

		// It should be deleted now.
		results, err := b.List(ctx, clusterInfoA, &v1.EventParams{
			QueryParams: v1.QueryParams{TimeRange: &lmav1.TimeRange{From: time.Now().Add(-1 * time.Minute), To: time.Now().Add(1 * time.Minute)}},
		})
		require.NoError(t, err)
		require.NotNil(t, 1, results)
		require.Equal(t, 0, len(results.Items), "event should be deleted")
	})
}

func TestDeleteEvent(t *testing.T) {
	// Variables created in deleteSetup, but used in the tests.
	var eventID string
	var clusterInfo bapi.ClusterInfo
	var results *v1.List[v1.Event]

	deleteSetup := func(t *testing.T) {
		// The event to create
		event := v1.Event{
			Time:            v1.NewEventTimestamp(time.Now().Unix()),
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

		clusterInfo = bapi.ClusterInfo{Cluster: cluster}
		// Create the event in ES.
		resp, err := b.Create(ctx, clusterInfo, []v1.Event{event})
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		// Refresh the index.
		err = backendutils.RefreshIndex(ctx, client, indexGetter.Index(clusterInfo))
		require.NoError(t, err)

		// List the events and make sure the one we created is present.
		results, err = b.List(ctx, clusterInfo, &v1.EventParams{
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

		// Save the event ID to assert with deleted event.
		eventID = results.Items[0].ID

		// We expect the ID to be present, but it's a random value so we
		// can't assert on the exact value.
		require.Equal(t, event, backendutils.AssertEventIDAndReset(t, results.Items[0]))
	}

	RunAllModes(t, "Delete an Event", func(t *testing.T) {
		deleteSetup(t)

		resp, err := b.Delete(ctx, clusterInfo, []v1.Event{results.Items[0]})
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Errors))
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)

		require.NoError(t, err)
		require.NotNil(t, results)
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Failed)
		require.Equal(t, 1, resp.Succeeded)
		require.Equal(t, eventID, resp.Deleted[0].ID)

		result, err := b.List(ctx, clusterInfo, &v1.EventParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-1 * time.Minute),
					To:   time.Now().Add(1 * time.Minute),
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, 1, result)
		require.Equal(t, 0, len(result.Items))
	})

	RunAllModes(t, "Invalid Cluster Info", func(t *testing.T) {
		deleteSetup(t)

		invalidClusterInfo := bapi.ClusterInfo{}
		resp, err := b.Delete(ctx, invalidClusterInfo, []v1.Event{results.Items[0]})
		require.Error(t, err)
		require.Equal(t, "no cluster ID on request", err.Error())
		require.Nil(t, resp)
	})

	RunAllModes(t, "Try to Delete an event that does not exist in Elastic", func(t *testing.T) {
		deleteSetup(t)

		invalidEvent := v1.Event{
			ID: "INVALIDEVENT",
		}
		resp, err := b.Delete(ctx, clusterInfo, []v1.Event{invalidEvent})
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, resp.Total)
		require.Equal(t, 0, resp.Succeeded)
		require.Equal(t, 1, resp.Failed)

		if !indexGetter.IsSingleIndex() {
			require.Equal(t, 404, resp.Deleted[0].Status)
		} else {
			// Single index mode will not return a deleted section, as the request never
			// actually makes it to ES due to the tenancy check.
			require.Nil(t, resp.Deleted)
		}
	})
}
