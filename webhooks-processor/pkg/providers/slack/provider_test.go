// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/testutils"
)

func sampleValidConfig() map[string]string {
	return map[string]string{
		"url": "https://hooks.slack.com/test-hook",
	}
}

func sampleLabels() map[string]string {
	return map[string]string{
		"Cluster": "unit-test-cluster",
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
			Description:  "$DEADB33F is now under attack",
			Time:         lsApi.NewEventTimestamp(0),
			Origin:       "test",
			AttackVector: "unit test",
			Severity:     10,
			MitreIDs:     &[]string{"1234", "5678"},
			MitreTactic:  "cork boi",
			Mitigations:  &[]string{"do this", "do that too"},
		}
	}
	t.Run("slack success", func(t *testing.T) {
		setup(t)
		c := sampleValidConfig()
		c["url"] = fmt.Sprintf("%s/test", fc.Url())

		_, err := p.Process(ctx, c, sampleLabels(), event)
		require.NoError(t, err)

		require.Eventually(t, func() bool { return len(fc.Requests) == 1 }, 5*time.Second, 10*time.Millisecond)

		var slackMessage SlackMessage
		err = json.Unmarshal(fc.Requests[0].Body, &slackMessage)
		require.NoError(t, err)

		messageJsonBytes, err := slackMessage.JSON()
		require.NoError(t, err)
		messageJson := string(messageJsonBytes)
		expectedJson := `{"blocks":[` +
			`{"type":"header","text":{"type":"plain_text","text":"⚠ Calico Security Alert"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*$DEADB33F is now under attack*"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*‣ Mitigations:*\n\ndo this\n\ndo that too"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*‣ Event source:* test"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*‣ Attack vector:* unit test"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*‣ Severity:* 10/100"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*‣ Mitre IDs:* 1234, 5678"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*‣ Mitre tactic:* cork boi"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*‣ Cluster:* unit-test-cluster"}},` +
			`{"type":"section","text":{"type":"mrkdwn","text":"*‣ Detailed record information:* ` + "```n/a```" + `"}}]}`
		require.Equal(t, messageJson, expectedJson)
	})
	t.Run("slack failure", func(t *testing.T) {
		setup(t)
		c := sampleValidConfig()
		c["url"] = fmt.Sprintf("%s/test", fc.Url())

		fc.ShouldFail = true
		// This will take a while and only return once finished
		_, err := p.Process(ctx, c, sampleLabels(), event)
		require.Error(t, err)

		require.Eventually(t, func() bool { return len(fc.Requests) >= 2 }, time.Second, 10*time.Millisecond)

		// This works as designed, and the error eventually coming up from p.Process()
		// will be logged but there will be no other traces of the failure.
		// Should we consider handling this differently?
		// Should we update the health of the webhook while retrying?
		// Ultimately we may never recover from failed webhook and a user ought to know somehow...
	})
}
