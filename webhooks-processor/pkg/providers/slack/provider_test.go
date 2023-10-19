// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func sampleValidConfig() map[string]string {
	return map[string]string{
		"url": "https://hooks.slack.com/test-hook",
	}
}

func TestSlackProviderValidation(t *testing.T) {
	p := Slack{}
	t.Run("valid config", func(t *testing.T) {
		err := p.Validate(sampleValidConfig())
		require.NoError(t, err)
	})

	t.Run("no url", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "url")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("not a slack url", func(t *testing.T) {
		c := sampleValidConfig()
		c["url"] = "https://somethinng-thats-not-slack.com/test"

		err := p.Validate(c)
		require.Error(t, err)
	})
}

func TestSlackProviderProcessing(t *testing.T) {
	var requests []testutils.HttpRequest
	var shouldFail bool
	var ts *httptest.Server
	var ctx context.Context
	var p providers.Provider
	var event *lsApi.Event
	setup := func(t *testing.T) {
		requests = []testutils.HttpRequest{}
		shouldFail = false
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Let's make requests fail on demand
			if shouldFail {
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
		t.Cleanup(func() {
			ts.Close()
		})

		ctx = context.Background()
		p = NewProvider()
		event = &lsApi.Event{
			ID:          "testid",
			Description: "This is an event",
			Severity:    3,
			Time:        lsApi.NewEventTimestamp(time.Now().Unix()),
			Type:        "runtime_security",
		}
	}
	t.Run("slack success", func(t *testing.T) {
		setup(t)
		c := sampleValidConfig()
		c["url"] = fmt.Sprintf("%s/test", ts.URL)

		err := p.Process(ctx, c, event)
		require.NoError(t, err)

		require.Eventually(t, func() bool { return len(requests) == 1 }, 15*time.Second, 10*time.Millisecond)

		var slackMessage SlackMessage
		err = json.Unmarshal(requests[0].Body, &slackMessage)
		require.NoError(t, err)

		// Not sure how much we want to test the content of each block...
		// Probably want to check that we can find important fields in the event
		require.NotNil(t, slackMessage.Blocks)
		require.Len(t, slackMessage.Blocks, 4)
		require.Equal(t, slackMessage.Blocks[0].Type, "header")
		require.Equal(t, slackMessage.Blocks[1].Type, "divider")
		require.Equal(t, slackMessage.Blocks[2].Type, "section")
		require.Equal(t, slackMessage.Blocks[3].Type, "section")

		messageJsonBytes, err := slackMessage.JSON()
		require.NoError(t, err)
		messageJson := string(messageJsonBytes)
		require.Contains(t, messageJson, event.Type)
		require.Contains(t, messageJson, event.Origin)
		require.Contains(t, messageJson, event.Time.GetTime().String())
		require.Contains(t, messageJson, fmt.Sprint(event.Severity))
		require.Contains(t, messageJson, event.Description)
	})

	t.Run("slack failure", func(t *testing.T) {
		setup(t)
		c := sampleValidConfig()
		c["url"] = fmt.Sprintf("%s/test", ts.URL)

		// Override default parameters to make sure we retry quickly and get to a failure state quicker
		slackProvider := p.(*Slack)
		slackProvider.Config.RetryTimes = 2
		slackProvider.Config.RetryDuration = 1 * time.Millisecond
		slackProvider.Config.RequestTimeout = 1 * time.Millisecond

		shouldFail = true
		// This will take a while and only return once finished
		err := p.Process(ctx, c, event)
		require.Error(t, err)

		// At this stage all the retries and errors have gone through, no need to wait further...
		require.GreaterOrEqual(t, len(requests), 2)

		// This works as designed, and the error eventually coming up from p.Process()
		// will be logged but there will be no other traces of the failure.
		// Should we consider handling this differently?
		// Should we update the health of the webhook while retrying?
		// Ultimately we may never recover from failed webhook and a user ought to know somehow...
	})
}
