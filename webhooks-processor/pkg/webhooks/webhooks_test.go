// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/generic"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/slack"
	"github.com/projectcalico/calico/webhooks-processor/pkg/testutils"
)

// This file contains tests that combine the webhooks building blocks as a cohesive unit
// and assert that all work together as expected to implement the desired behavior.

func TestWebhooksProcessorExitsOnCancel(t *testing.T) {
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
		return []lsApi.Event{}, nil
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
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
		return []lsApi.Event{}, nil
	})

	startTime := time.Now()

	// New webhook has no status
	wh := testutils.NewTestWebhook("test-wh")
	require.Nil(t, wh.Status)

	_, err := testState.WebHooksAPI.Update(context.Background(), wh, options.SetOptions{})
	require.NoError(t, err)

	// Check that webhook status is eventually updated to healthy
	require.Eventually(t, isHealthy(wh), time.Second, 10*time.Millisecond)
	require.True(t, wh.Status[0].LastTransitionTime.After(startTime))
}

func TestWebhookNonHealthyStates(t *testing.T) {
	testNonHealthyState := func(webhook *api.SecurityEventWebhook, reason string, message string) {
		testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
			return []lsApi.Event{}, nil
		})

		startTime := time.Now()

		// New webhook has no status
		require.Nil(t, webhook.Status)

		testState.WebHooksAPI.Watcher.Results <- watch.Event{Type: watch.Added, Object: webhook}

		// Check that webhook status is eventually updated to healthy
		require.Eventually(t, func() bool {
			return webhook != nil &&
				webhook.Status != nil &&
				len(webhook.Status) == 1 &&
				webhook.Status[0].Type == "Healthy" &&
				webhook.Status[0].Status == metav1.ConditionStatus("False")
		}, time.Second, 10*time.Millisecond)
		require.True(t, webhook.Status[0].LastTransitionTime.After(startTime))

		// Check details
		require.Contains(t, webhook.Status[0].Reason, reason)
		require.Contains(t, webhook.Status[0].Message, message)
	}

	t.Run("disabled", func(t *testing.T) {
		wh := testutils.NewTestWebhook("test-wh")
		wh.Spec.State = "Disabled"
		testNonHealthyState(wh, "WebhookState", "the webhook has been disabled")
	})

	t.Run("malformed query", func(t *testing.T) {
		wh := testutils.NewTestWebhook("test-wh")
		wh.Spec.Query = "= runtime_security"
		testNonHealthyState(wh, "QueryParsing", "unexpected token")
	})

	t.Run("invalid query", func(t *testing.T) {
		wh := testutils.NewTestWebhook("test-wh")
		wh.Spec.Query = "type = runtime_securit"
		testNonHealthyState(wh, "QueryValidation", "invalid value for type: runtime_securit")
	})

	// The following test will fail to initialise a CoreV1 API client.
	// TODO: Add test for config parsing failure, with `configItem.ValueFrom`,
	// that test the "key not found in secret/config map" code paths.
	t.Run("config parsing - no cli client", func(t *testing.T) {
		wh := testutils.NewTestWebhook("test-wh")

		wh.Spec.Config = append(wh.Spec.Config, api.SecurityEventWebhookConfigVar{
			Name:      "some-secret",
			ValueFrom: &api.SecurityEventWebhookConfigVarSource{},
		})
		testNonHealthyState(wh, "ConfigurationParsing", "invalid configuration: no configuration has been provided, try setting KUBERNETES_MASTER environment variable")
	})

	t.Run("unknown consumer", func(t *testing.T) {
		wh := testutils.NewTestWebhook("test-wh")
		wh.Spec.Consumer = "Unknown"
		testNonHealthyState(wh, "ConsumerDiscovery", "unknown consumer: Unknown")
	})

	t.Run("invalid provider config", func(t *testing.T) {
		wh := testutils.NewTestWebhook("test-wh")
		wh.Spec.Config = []api.SecurityEventWebhookConfigVar{}
		testNonHealthyState(wh, "ConsumerConfigurationValidation", "url field is not present in webhook configuration")
	})
}

func TestWebhookSent(t *testing.T) {
	testEvent := lsApi.Event{
		ID:          "testid",
		Description: "This is an event",
		Severity:    42,
		Time:        lsApi.NewEventTimestamp(time.Now().Unix()),
		Type:        "runtime_security",
	}
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
		return []lsApi.Event{testEvent}, nil
	})

	wh := testutils.NewTestWebhook("test-wh")
	_, err := testState.WebHooksAPI.Update(context.Background(), wh, options.SetOptions{})
	require.NoError(t, err)

	// Make sure the webhook eventually hits the test provider
	require.Eventually(t, hasOneRequest(testState.TestSlackProvider()), testState.FetchingInterval*4, 10*time.Millisecond)
	require.Equal(t, wh.Spec.Config[0].Name, "url")
	require.Equal(t, wh.Spec.Config[0].Value, testState.TestSlackProvider().Requests[0].Config["url"])
	require.Equal(t, testEvent, testState.TestSlackProvider().Requests[0].Event)

	// Make sure labels annotation is correctly processed
	require.Eventually(t,
		func() bool {
			return assert.Equal(t,
				map[string]string{
					"hips dont lie": "true",
					"anything":      "goes",
					"also-this":     "",
				},
				testState.TestSlackProvider().Requests[0].Labels,
			)
		}, testState.FetchingInterval*4, 10*time.Millisecond,
	)
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
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
		return []lsApi.Event{testEvent1, testEvent2}, nil
	})

	wh := testutils.NewTestWebhook("test-wh")
	_, err := testState.WebHooksAPI.Update(context.Background(), wh, options.SetOptions{})
	require.NoError(t, err)

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
	testState := Setup(t, func(ctx context.Context, query *query.Query, from time.Time, to time.Time) ([]lsApi.Event, error) {
		logrus.Infof("Reading events (from: %s, to: %s)", from, to)
		requestedTimes = append(requestedTimes, []time.Time{from, to})
		return []lsApi.Event{}, nil
	})

	wh := testutils.NewTestWebhook("test-wh")
	_, err := testState.WebHooksAPI.Update(context.Background(), wh, options.SetOptions{})
	require.NoError(t, err)

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
	testState := Setup(t, func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
		return fetchedEvents, nil
	})

	// TODO: Add a check to test that the rate limiter is set to less than len(fetchedEvents)
	// Right now it's hardcoded to 5 in the test setup (but that could and likely will change)
	wh := testutils.NewTestWebhook("test-wh")
	_, err := testState.WebHooksAPI.Update(context.Background(), wh, options.SetOptions{})
	require.NoError(t, err)

	// Make sure the webhook eventually hits the test server
	testProvider := testState.TestSlackProvider()
	// Make sure the test is valid (we're providing more events than allowed)
	numEventsAllowed := int(testState.Providers[api.SecurityEventWebhookConsumerSlack].Config().RateLimiterCount)
	require.Less(t, numEventsAllowed, len(fetchedEvents))
	require.Eventually(t, hasNRequest(testProvider, numEventsAllowed), 15*time.Second, 10*time.Millisecond)

	// Even if we wait, we're not getting the missing event, it's gone forever.
	// Is this good enough?
	time.Sleep(testState.FetchingInterval * 2)
	require.Eventually(t, hasNRequest(testProvider, numEventsAllowed), 15*time.Second, 10*time.Millisecond)
}

func TestGenericProvider(t *testing.T) {
	requests := []testutils.HttpRequest{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Does anyone read this?")
		request := testutils.HttpRequest{
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
	testState := NewTestState(func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
		return fetchedEvents, nil
	}, DefaultProviders())

	SetupWithTestState(t, testState)

	whUrl := fmt.Sprintf("%s/test-hook", ts.URL)
	wh := testutils.NewTestWebhook("test-generic-webhook")
	wh.Spec.Consumer = api.SecurityEventWebhookConsumerGeneric
	// Making sure we'll update the right config...
	require.Equal(t, wh.Spec.Config[0].Name, "url")
	// Updating URL to point to the test server
	wh.Spec.Config[0].Value = whUrl
	_, err := testState.WebHooksAPI.Update(context.Background(), wh, options.SetOptions{})
	require.NoError(t, err)

	// Make sure the webhook eventually hits the test provider
	require.Eventually(t, func() bool { return len(requests) == 1 }, 5*time.Second, 10*time.Millisecond)

	// We got the webhook as expected
	require.Equal(t, "POST", requests[0].Method)
	require.Equal(t, "/test-hook", requests[0].URL)
	// And check that we get a JSON of the original event
	var whEvent lsApi.Event
	err = json.Unmarshal(requests[0].Body, &whEvent)
	require.NoError(t, err)
	require.Equal(t, fetchedEvents[0], whEvent)
}

func TestBackoffOnInitialFailure(t *testing.T) {
	const retryTimes int = 3
	requests := []testutils.HttpRequest{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let's make the first requests fail
		if len(requests) < retryTimes {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintln(w, "Does anyone read this?")
		request := testutils.HttpRequest{
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
	providers := DefaultProviders()
	genericProviderConfig := providers[api.SecurityEventWebhookConsumerGeneric].Config()
	genericProviderConfig.RetryTimes = uint(retryTimes)
	genericProviderConfig.RetryDuration = 1 * time.Second
	providers[api.SecurityEventWebhookConsumerGeneric] = generic.NewProvider(genericProviderConfig)
	testState := NewTestState(func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
		return fetchedEvents, nil
	}, providers)

	SetupWithTestState(t, testState)

	whUrl := fmt.Sprintf("%s/test-hook", ts.URL)
	wh := testutils.NewTestWebhook("test-generic-webhook")
	wh.Spec.Consumer = api.SecurityEventWebhookConsumerGeneric
	// Making sure we'll update the right config...
	require.Equal(t, wh.Spec.Config[0].Name, "url")
	// Updating URL to point to the test server
	wh.Spec.Config[0].Value = whUrl
	_, err := testState.WebHooksAPI.Update(context.Background(), wh, options.SetOptions{})
	require.NoError(t, err)

	// The health of the webhook is only updated to an error status after the maximum number of retries
	// has been reached and we give up on that event.
	require.Eventually(t, func() bool { return !isHealthy(wh)() }, 20*time.Second, 10*time.Millisecond)
	require.Contains(t, wh.Status[0].Message, "unexpected response [500]:Does anyone read this?")

	// And check that the data is as expected
	require.Equal(t, "POST", requests[0].Method)
	require.Equal(t, "/test-hook", requests[0].URL)
	// And check that we get a JSON of the original event
	var whEvent lsApi.Event
	err = json.Unmarshal(requests[0].Body, &whEvent)
	require.NoError(t, err)
	require.Equal(t, fetchedEvents[0], whEvent)

	// Eventually, the webhook gets back to healthy status as requests start to go through
	require.Eventually(t, func() bool { return isHealthy(wh)() }, 5*time.Second, 10*time.Millisecond)
}

func TestWebhookErrorsDontDisappear(t *testing.T) {
	providers := DefaultProviders()
	genericProviderConfig := providers[api.SecurityEventWebhookConsumerGeneric].Config()
	genericProviderConfig.RequestTimeout = 100 * time.Millisecond
	genericProviderConfig.RetryTimes = 3
	genericProviderConfig.RetryDuration = 1 * time.Second
	providers[api.SecurityEventWebhookConsumerGeneric] = generic.NewProvider(genericProviderConfig)
	shouldSendEvents := false
	testState := NewTestState(func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error) {
		if shouldSendEvents {
			// Set to false so that we only do it once
			return []lsApi.Event{{ID: "test", Description: "test", Severity: 3, Type: "runtime_security", Origin: "test"}}, nil
		}
		return []lsApi.Event{}, nil
	}, providers)

	SetupWithTestState(t, testState)

	previousTime := time.Now()

	// New webhook has no status
	wh := testutils.NewTestWebhook("test-wh")
	wh.Spec.Consumer = api.SecurityEventWebhookConsumerGeneric
	// Use an invalid URL to generate an error
	require.Equal(t, "url", wh.Spec.Config[0].Name)
	wh.Spec.Config[0].Value = "http://test.com/does-not-exists"
	require.Nil(t, wh.Status)

	_, err := testState.WebHooksAPI.Update(context.Background(), wh, options.SetOptions{})
	require.NoError(t, err)

	// Check that webhook status is eventually updated to healthy
	require.Eventually(t, hasHealthStatus(wh, true), time.Second, 10*time.Millisecond)
	require.True(t, wh.Status[0].LastTransitionTime.After(previousTime))
	previousTime = wh.Status[0].LastTransitionTime.Time

	// Let's return test events
	shouldSendEvents = true
	logrus.Debug("Setting shouldSendEvents to true")
	// Wait for the status to go bad
	require.Eventually(t, hasHealthStatus(wh, false), 20*time.Second, time.Second)
	require.True(t, wh.Status[0].LastTransitionTime.After(previousTime))
	previousError := wh.Status[0].Message
	previousTime = wh.Status[0].LastTransitionTime.Time
	require.Contains(t, previousError, "context deadline exceeded")

	shouldSendEvents = false

	// Wait for another round of processing (with no new events)
	time.Sleep(testState.FetchingInterval * 2)

	// And make sure that the previous error is still visible
	require.Eventually(t, hasHealthStatus(wh, false), time.Second, 10*time.Millisecond)
	require.Greater(t, wh.Status[0].LastTransitionTime.Time, previousTime)
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

func hasHealthStatus(webhook *api.SecurityEventWebhook, status bool) func() bool {
	value := "False"
	if status {
		value = "True"
	}
	return func() bool {
		return webhook != nil &&
			webhook.Status != nil &&
			len(webhook.Status) == 1 &&
			webhook.Status[0].Type == "Healthy" &&
			webhook.Status[0].Status == metav1.ConditionStatus(value)
	}
}

func isHealthy(webhook *api.SecurityEventWebhook) func() bool {
	return hasHealthStatus(webhook, true)
}

func hasOneRequest(provider *TestProvider) func() bool {
	return hasNRequest(provider, 1)
}

func hasNRequest(provider *TestProvider, n int) func() bool {
	return func() bool {
		return len(provider.Requests) == n
	}
}

type TestState struct {
	Running          bool
	Stop             func()
	WebHooksAPI      *testutils.FakeSecurityEventWebhook
	GetEvents        func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error)
	Providers        map[api.SecurityEventWebhookConsumer]providers.Provider
	FetchingInterval time.Duration
}

func NewTestState(getEvents func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error), providers map[api.SecurityEventWebhookConsumer]providers.Provider) *TestState {
	testState := &TestState{}
	testState.WebHooksAPI = &testutils.FakeSecurityEventWebhook{}
	testState.GetEvents = getEvents
	testState.Running = false
	testState.FetchingInterval = 2 * time.Second
	testState.Providers = providers

	return testState
}

func (t *TestState) TestSlackProvider() *TestProvider {
	return t.Providers[api.SecurityEventWebhookConsumerSlack].(*TestProvider)
}

func Setup(t *testing.T, getEvents func(context.Context, *query.Query, time.Time, time.Time) ([]lsApi.Event, error)) *TestState {
	testProviders := make(map[api.SecurityEventWebhookConsumer]providers.Provider)
	testProviders[api.SecurityEventWebhookConsumerSlack] = NewTestProvider()

	require.NotZero(t, testProviders[api.SecurityEventWebhookConsumerSlack].Config().RateLimiterCount)
	testState := NewTestState(getEvents, testProviders)

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

type Request struct {
	Config map[string]string
	Labels map[string]string
	Event  lsApi.Event
}

type SlackProvider = slack.Slack
type TestProvider struct {
	SlackProvider
	Requests []Request
}

func NewTestProvider() providers.Provider {
	return &TestProvider{
		SlackProvider: *slack.NewProvider(GetTestProviderConfig()).(*slack.Slack),
	}
}
func (p *TestProvider) Validate(config map[string]string) error {
	if _, urlPresent := config["url"]; !urlPresent {
		return errors.New("url field is not present in webhook configuration")
	}
	return nil
}

func (p *TestProvider) Process(ctx context.Context, config map[string]string, labels map[string]string, event *lsApi.Event) (err error) {
	logrus.Infof("Processing event %s", event.ID)
	p.Requests = append(p.Requests, Request{
		Config: config,
		Labels: labels,
		Event:  *event,
	})

	return nil
}

func (p *TestProvider) Config() providers.Config {
	return p.SlackProvider.Config()
}

func GetTestProviderConfig() providers.Config {
	return providers.Config{
		RateLimiterDuration: time.Hour,
		RateLimiterCount:    5,
		RequestTimeout:      time.Second,
		RetryDuration:       time.Millisecond,
		RetryTimes:          2,
	}
}
