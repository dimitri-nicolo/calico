// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/testutils"
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

func TestJiraProviderProcessing(t *testing.T) {
	var fc *testutils.FakeConsumer
	var ctx context.Context
	var p providers.Provider
	var event *lsApi.Event
	setup := func(t *testing.T) {
		fc = testutils.NewFakeConsumer(t)
		ctx = context.Background()
		p = NewProvider(providers.Config{
			RateLimiterDuration: time.Hour,
			RateLimiterCount:    100,
			RequestTimeout:      time.Second,
			RetryDuration:       time.Millisecond,
			RetryTimes:          2,
		})
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
		c["url"] = fmt.Sprintf("%s/test", fc.Url())

		err := p.Process(ctx, c, event)
		require.NoError(t, err)

		require.Eventually(t, func() bool { return len(fc.Requests) == 1 }, 15*time.Second, 10*time.Millisecond)

		var jiraPayload jiraPayload
		err = json.Unmarshal(fc.Requests[0].Body, &jiraPayload)
		require.NoError(t, err)

		// Check that some Jira-specific headers are set
		require.NotEmpty(t, fc.Requests[0].Header.Get("Authorization"))
		require.Equal(t, "application/json", fc.Requests[0].Header.Get("Content-Type"))

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
		c["url"] = fmt.Sprintf("%s/test", fc.Url())

		// Override default parameters to make sure we retry quickly and get to a failure state quicker
		slackProvider := p.(*Jira)
		slackProvider.ProviderConfig.RetryTimes = 2
		slackProvider.ProviderConfig.RetryDuration = 1 * time.Millisecond
		slackProvider.ProviderConfig.RequestTimeout = 1 * time.Millisecond

		fc.ShouldFail = true
		// This will take a while and only return once finished
		err := p.Process(ctx, c, event)
		require.Error(t, err)

		require.Eventually(t, func() bool { return len(fc.Requests) >= 2 }, time.Second, 10*time.Millisecond)

		// This works as designed, and the error eventually coming up from p.Process()
		// will be logged but there will be no other traces of the failure.
		// Should we consider handling this differently?
		// Should we update the health of the webhook while retrying?
		// Ultimately we may never recover from failed webhook and a user ought to know somehow...
	})
}
