// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/testutils"
)

const (
	testUrl = "https://test-hook"
)

// This file contains tests that combine the webhooks building blocks as a cohesive unit
// and assert that all work together as expected to implement the desired behavior.

func TestWebhooksProcessorExitsOnCancel(t *testing.T) {
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event {
		return []lsApi.Event{}
	})

	// Just making sure everything can run without crashing
	require.True(t, testState.Running)

	// And that we can stop it on demand
	testState.Stop()
	require.Eventually(t, func() bool { return !testState.Running }, 3*time.Second, 10*time.Millisecond)

	// Flag that we already stopped it and don't need to clean it up further during test teardown..
	testState.Stop = nil
}

func TestWebhookHealthy(t *testing.T) {
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event {
		return []lsApi.Event{}
	})

	startTime := time.Now()

	// New webhook has no status
	wh := newTestWebhook("test-wh")
	require.Nil(t, wh.Status)

	testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: wh}

	// Check that webhook status is eventually updated to healthy
	require.Eventually(t, isHealthy(wh), time.Second, 10*time.Millisecond)
	require.True(t, wh.Status[0].LastTransitionTime.After(startTime))

	// And make sure we're still running
	require.Eventually(t, func() bool { return testState.Running }, time.Second, 10*time.Millisecond)
}

func TestFailedWebhookProviderValidationReportedNotHealthy(t *testing.T) {
	testState := &TestState{}
	testState.WebHooksAPI = &testutils.FakeSecurityEventWebhook{}
	testState.GetEvents = func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event {
		return []lsApi.Event{}
	}
	testState.Running = false

	testState.Providers = DefaultProviders()

	SetupWithTestState(t, testState)

	wh := newTestWebhook("test-invalid-generic-webhook")
	wh.Spec.Consumer = api.SecurityEventWebhookConsumerGeneric
	wh.Spec.Config = []api.SecurityEventWebhookConfigVar{}

	startTime := time.Now()

	// Sanity check: webHook has no status initially
	require.Nil(t, wh.Status)

	testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: wh}

	// Check that webhook status is eventually updated to NOT healthy
	require.Eventually(t, func() bool {
		isHealthyFn := isHealthy(wh)
		return !isHealthyFn()
	}, time.Second, 10*time.Millisecond)
	require.True(t, wh.Status[0].LastTransitionTime.After(startTime))

	require.Equal(t, "url field is not present in webhook configuration", wh.Status[0].Message)
}

func TestWebhookSent(t *testing.T) {
	testEvent := lsApi.Event{
		ID:          "testid",
		Description: "This is an event",
		Severity:    42,
		Time:        lsApi.NewEventTimestamp(time.Now().Unix()),
		Type:        "runtime_security",
	}
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event {
		return []lsApi.Event{testEvent}
	})

	wh := newTestWebhook("test-wh")
	testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: wh}

	// Make sure the webhook eventually hits the test provider
	require.Eventually(t, hasOneRequest(testState.TestSlackProvider()), testState.FetchingInterval*4, 10*time.Millisecond)
	require.Equal(t, testUrl, testState.TestSlackProvider().Requests[0].Config["url"])
	require.Equal(t, testEvent, testState.TestSlackProvider().Requests[0].Event)
}

func TestSendsOneWebhookPerEvent(t *testing.T) {
	// Making sure that if we test multiple events at once
	// we still get the expected number of webhooks triggered.
	testEvent1 := lsApi.Event{
		ID:          "testid1",
		Description: "This is an event",
		Severity:    41,
		Time:        lsApi.NewEventTimestamp(time.Now().Unix()),
		Type:        "runtime_security",
	}
	testEvent2 := lsApi.Event{
		ID:          "testid2",
		Description: "This is an event",
		Severity:    42,
		Time:        lsApi.NewEventTimestamp(time.Now().Unix()),
		Type:        "runtime_security",
	}
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event {
		return []lsApi.Event{testEvent1, testEvent2}
	})

	wh := newTestWebhook("test-wh")
	testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: wh}

	// Make sure the webhook eventually hits the test provider
	testProvider := testState.TestSlackProvider()
	require.Eventually(t, hasNRequest(testProvider, 2), 15*time.Second, 10*time.Millisecond)

	eventsFromLinseed := []lsApi.Event{testEvent1, testEvent2}
	eventsSentToProvider := []lsApi.Event{testProvider.Requests[0].Event, testProvider.Requests[1].Event}
	require.ElementsMatch(t, eventsFromLinseed, eventsSentToProvider)
}

func TestEventsFetchedUsingNonOverlappingIntervals(t *testing.T) {
	// Making sure that a webhook goroutine does not fetch/process
	// the same event twice by looking at the queried timestamps
	// and make sure they don't overlap.
	testStartTime := time.Now()
	requestedTimes := [][]time.Time{}
	testState := Setup(t, func(ctx context.Context, query *query.Query, from time.Time, to time.Time) []lsApi.Event {
		logrus.Infof("Reading events (from: %s, to: %s)", from, to)
		requestedTimes = append(requestedTimes, []time.Time{from, to})
		return []lsApi.Event{}
	})

	wh := newTestWebhook("test-wh")
	testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: wh}

	// Wait that we get a few fetch requests
	require.Eventually(t, func() bool {
		return len(requestedTimes) == 3
	}, 35*time.Second, 10*time.Millisecond)

	testEndTime := time.Now()
	require.Less(t, testStartTime, testEndTime)

	// No time overlap within queries
	require.Less(t, requestedTimes[0][0], requestedTimes[0][1])
	require.Less(t, requestedTimes[1][0], requestedTimes[1][1])
	require.Less(t, requestedTimes[2][0], requestedTimes[2][1])

	// Next time range picks up exactly where the previous one stopped
	require.Equal(t, requestedTimes[0][1], requestedTimes[1][0])
	require.Equal(t, requestedTimes[1][1], requestedTimes[2][0])
}

func TestTooManyEventsAreRateLimited(t *testing.T) {
	// Testing what happens when we get a burst of events that's larger than the rate limiter allows...
	// In this case we simply ignore the additional events. That doesn't feel right...
	fetchedEvents := []lsApi.Event{newEvent(1), newEvent(2), newEvent(3), newEvent(4), newEvent(5), newEvent(6)}
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event {
		return fetchedEvents
	})

	// TODO: Add a check to test that the rate limiter is set to less than len(fetchedEvents)
	// Right now it's hardcoded to 5 in the test setup (but that could and likely will change)
	wh := newTestWebhook("test-wh")
	testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: wh}

	// Make sure the webhook eventually hits the test server
	testProvider := testState.TestSlackProvider()
	// Make sure the test is valid (we're providing more events than allowed)
	numEventsAllowed := int(testState.Providers[api.SecurityEventWebhookConsumerSlack].RateLimiterCount)
	require.Less(t, numEventsAllowed, len(fetchedEvents))
	require.Eventually(t, hasNRequest(testProvider, numEventsAllowed), 15*time.Second, 10*time.Millisecond)

	// Even if we wait, we're not getting the missing event, it's gone forever.
	// Is this good enough?
	time.Sleep(testState.FetchingInterval * 2)
	require.Eventually(t, hasNRequest(testProvider, numEventsAllowed), 15*time.Second, 10*time.Millisecond)
}

type HttpRequest struct {
	Method string
	URL    string
	Header http.Header
	Body   []byte
}

func TestGenericProvider(t *testing.T) {
	requests := []HttpRequest{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Does anyone read this?")
		request := HttpRequest{
			Method: r.Method,
			URL:    r.URL.String(),
			Header: r.Header,
		}
		var err error
		request.Body, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, request)
	}))
	defer ts.Close()

	fetchedEvents := []lsApi.Event{newEvent(1)}
	testState := NewTestState(func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event {
		return fetchedEvents
	}, DefaultProviders())

	SetupWithTestState(t, testState)

	whUrl := fmt.Sprintf("%s/test-hook", ts.URL)
	wh := newTestWebhook("test-generic-webhook")
	wh.Spec.Consumer = api.SecurityEventWebhookConsumerGeneric
	// Making sure we'll update the right config...
	require.Equal(t, wh.Spec.Config[0].Name, "url")
	// Updating URL to point to the test server
	wh.Spec.Config[0].Value = whUrl
	testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: wh}

	// Make sure the webhook eventually hits the test provider
	require.Eventually(t, func() bool { return len(requests) == 1 }, 15*time.Second, 10*time.Millisecond)

	// We got the webhook as expected
	require.Equal(t, "POST", requests[0].Method)
	require.Equal(t, "/test-hook", requests[0].URL)
	// And check that we get a JSON of the original event
	var whEvent lsApi.Event
	err := json.Unmarshal(requests[0].Body, &whEvent)
	require.NoError(t, err)
	require.Equal(t, fetchedEvents[0], whEvent)
}

func TestBackoffOnInitialFailure(t *testing.T) {
	requests := []HttpRequest{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let's make the first request fail
		if len(requests) == 0 {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintln(w, "Does anyone read this?")
		request := HttpRequest{
			Method: r.Method,
			URL:    r.URL.String(),
			Header: r.Header,
		}
		var err error
		request.Body, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, request)
	}))
	defer ts.Close()

	fetchedEvents := []lsApi.Event{newEvent(1)}
	testState := NewTestState(func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event {
		return fetchedEvents
	}, DefaultProviders())

	SetupWithTestState(t, testState)

	whUrl := fmt.Sprintf("%s/test-hook", ts.URL)
	wh := newTestWebhook("test-generic-webhook")
	wh.Spec.Consumer = api.SecurityEventWebhookConsumerGeneric
	// Making sure we'll update the right config...
	require.Equal(t, wh.Spec.Config[0].Name, "url")
	// Updating URL to point to the test server
	wh.Spec.Config[0].Value = whUrl
	testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: wh}

	// We will get the first request that will be denied
	require.Eventually(t, func() bool { return len(requests) == 1 }, 15*time.Second, 10*time.Millisecond)

	// I thought we would update the health of the webhook to something that shows the error
	// or the fact that the request is not going through but seems like we don't.
	// TODO: Do we want to consider doing this?
	// // and update the health of the webhook
	// require.Eventually(t, func() bool { return !isHealthy(wh)() }, 15*time.Second, 10*time.Millisecond)
	// require.Equal(t, "server doesn't like me :sob:", wh.Status[0].Message)

	// Wait for the second request following the initial failure
	require.Eventually(t, func() bool { return len(requests) == 2 }, 15*time.Second, 10*time.Millisecond)

	// And check that the data is as expected
	require.Equal(t, "POST", requests[0].Method)
	require.Equal(t, "/test-hook", requests[0].URL)
	// And check that we get a JSON of the original event
	var whEvent lsApi.Event
	err := json.Unmarshal(requests[0].Body, &whEvent)
	require.NoError(t, err)
	require.Equal(t, fetchedEvents[0], whEvent)

	// We could also check that the webhook remains healthy as per comment above...
}

func newEvent(n int) lsApi.Event {
	return lsApi.Event{
		ID:          fmt.Sprintf("testid%d", n),
		Description: "This is an event",
		Severity:    n,
		Time:        lsApi.NewEventTimestamp(time.Now().Unix()),
		Type:        "runtime_security",
	}
}

func newTestWebhook(name string) *api.SecurityEventWebhook {
	wh := api.NewSecurityEventWebhook()
	wh.Name = name
	wh.Spec.Consumer = api.SecurityEventWebhookConsumerSlack
	wh.Spec.State = api.SecurityEventWebhookStateEnabled
	wh.Spec.Query = "type = runtime_security"
	wh.Spec.Config = []api.SecurityEventWebhookConfigVar{{
		Name:  "url",
		Value: testUrl,
	}}
	wh.UID = types.UID(fmt.Sprintf("%s-uid", name))
	return wh
}

func isHealthy(webhook *api.SecurityEventWebhook) func() bool {
	return func() bool {
		return webhook != nil &&
			webhook.Status != nil &&
			len(webhook.Status) == 1 &&
			webhook.Status[0].Type == "Healthy" &&
			webhook.Status[0].Status == metav1.ConditionStatus("True")
	}
}

func hasOneRequest(provider *testutils.TestProvider) func() bool {
	return hasNRequest(provider, 1)
}

func hasNRequest(provider *testutils.TestProvider, n int) func() bool {
	return func() bool {
		return len(provider.Requests) == n
	}
}

type TestState struct {
	Running          bool
	Stop             func()
	WebHooksAPI      *testutils.FakeSecurityEventWebhook
	GetEvents        func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event
	Providers        map[api.SecurityEventWebhookConsumer]*ProviderConfiguration
	FetchingInterval time.Duration
}

func NewTestState(getEvents func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event, providers map[api.SecurityEventWebhookConsumer]*ProviderConfiguration) *TestState {
	testState := &TestState{}
	testState.WebHooksAPI = &testutils.FakeSecurityEventWebhook{}
	testState.GetEvents = getEvents
	testState.Running = false
	testState.FetchingInterval = 2 * time.Second
	testState.Providers = providers

	return testState
}

func (t *TestState) TestSlackProvider() *testutils.TestProvider {
	return t.Providers[api.SecurityEventWebhookConsumerSlack].Provider.(*testutils.TestProvider)
}

func Setup(t *testing.T, getEvents func(context.Context, *query.Query, time.Time, time.Time) []lsApi.Event) *TestState {
	providers := make(map[api.SecurityEventWebhookConsumer]*ProviderConfiguration)
	providers[api.SecurityEventWebhookConsumerSlack] = &ProviderConfiguration{
		Provider:            &testutils.TestProvider{},
		RateLimiterDuration: 5 * time.Second,
		RateLimiterCount:    3,
	}
	testState := NewTestState(getEvents, providers)

	return SetupWithTestState(t, testState)
}

func SetupWithTestState(t *testing.T, testState *TestState) *TestState {
	logrus.SetLevel(logrus.DebugLevel)

	config := &ControllerConfig{
		ClientV3:            testState.WebHooksAPI,
		EventsFetchFunction: testState.GetEvents,
		Providers:           testState.Providers,
		FetchingInterval:    testState.FetchingInterval,
	}

	var ctx context.Context
	wg := sync.WaitGroup{}
	ctx, testState.Stop = context.WithCancel(context.Background())
	go func() {
		testState.Running = true
		webhookWatcherUpdater := NewWebhookWatcherUpdater().WithClient(config.ClientV3)
		controllerState := NewControllerState().WithConfig(config)
		webhookController := NewWebhookController().WithState(controllerState)

		wg.Add(2)
		go webhookController.WithUpdater(webhookWatcherUpdater).Run(ctx, testState.Stop, &wg)
		go webhookWatcherUpdater.WithController(webhookController).Run(ctx, testState.Stop, &wg)
		wg.Wait()
		testState.Running = false
	}()

	require.Eventually(t, func() bool { return testState.Running }, time.Second, 10*time.Millisecond)

	// Sanity test
	require.NotNil(t, testState.WebHooksAPI.Watcher)

	t.Cleanup(func() {
		if testState.Stop != nil {
			// Making sure it's still running before we turn it off
			require.Eventually(t, func() bool { return testState.Running }, time.Second, 10*time.Millisecond)
			testState.Stop()
			require.Eventually(t, func() bool { return !testState.Running }, 3*time.Second, 10*time.Millisecond)
		}
	})

	return testState
}
