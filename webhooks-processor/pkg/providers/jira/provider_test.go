// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func sampleValidConfig() map[string]string {
	return map[string]string{
		"url":       "some-url",
		"project":   "test",
		"issueType": "test",
		"username":  "test",
		"apiToken":  "test",
	}
}
func TestJiraProviderValidation(t *testing.T) {
	p := Jira{}
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

	t.Run("no url", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "url")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("no project", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "project")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("no issueType", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "issueType")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("no username", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "username")

		err := p.Validate(c)
		require.Error(t, err)
	})

	t.Run("no apiToken", func(t *testing.T) {
		c := sampleValidConfig()
		delete(c, "apiToken")

		err := p.Validate(c)
		require.Error(t, err)
	})
}

func TestSTringReplace(t *testing.T) {
	str := `2023-10-23 14:57:40 &#43;0100 IST`
	str2 := strings.Replace(str, "&#43;", "+", -1)
	require.Contains(t, str2, "+")
}

func TestJiraProviderProcessing(t *testing.T) {
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
	t.Run("jira success", func(t *testing.T) {
		setup(t)
		c := sampleValidConfig()
		c["url"] = fmt.Sprintf("%s/test", ts.URL)

		err := p.Process(ctx, c, event)
		require.NoError(t, err)

		require.Eventually(t, func() bool { return len(requests) == 1 }, 15*time.Second, 10*time.Millisecond)

		var jiraPayload jiraPayload
		err = json.Unmarshal(requests[0].Body, &jiraPayload)
		require.NoError(t, err)

		// Check that some Jira-specific headers are set
		require.NotEmpty(t, requests[0].Header.Get("Authorization"))
		require.Equal(t, "application/json", requests[0].Header.Get("Content-Type"))

		// Not sure how much we want to test the content of each block...
		require.Equal(t, jiraPayload.Fields.Project.Key, c["project"])
		require.Equal(t, jiraPayload.Fields.IssueType.Name, c["issueType"])
		require.Equal(t, jiraPayload.Fields.Summary, "Calico Security Alert")
		require.Contains(t, jiraPayload.Fields.Description, event.Description)
		require.Contains(t, jiraPayload.Fields.Description, event.Type)
		require.Contains(t, jiraPayload.Fields.Description, event.Origin)
		// The '+' is templated to '&#43;'
		require.Contains(t, jiraPayload.Fields.Description, strings.Replace(event.Time.GetTime().String(), "+", "&#43;", -1))
		require.Contains(t, jiraPayload.Fields.Description, fmt.Sprint(event.Severity))
	})

	t.Run("jira failure", func(t *testing.T) {
		setup(t)
		c := sampleValidConfig()
		c["url"] = fmt.Sprintf("%s/test", ts.URL)

		// Override default parameters to make sure we retry quickly and get to a failure state quicker
		slackProvider := p.(*Jira)
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
