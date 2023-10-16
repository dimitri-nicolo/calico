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
	requests := []testutils.HttpRequest{}
	shouldFail := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer ts.Close()

	ctx := context.Background()
	p := Slack{}
	event := &lsApi.Event{
		ID:          "testid",
		Description: "This is an event",
		Severity:    3,
		Time:        lsApi.NewEventTimestamp(time.Now().Unix()),
		Type:        "runtime_security",
	}
	t.Run("success", func(t *testing.T) {
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
}
