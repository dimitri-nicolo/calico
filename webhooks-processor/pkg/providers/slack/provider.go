// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package slack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
	"github.com/sirupsen/logrus"
)

const (
	RequestTimeout = 5 * time.Second
	RetryDuration  = 2 * time.Second
	RetryTimes     = 5
)

type SlackProviderConfiguration struct {
	RateLimiterDuration time.Duration `envconfig:"WEBHOOKS_SLACK_RATE_LIMITER_DURATION" default:"5m"`
	RateLimiterCount    uint          `envconfig:"WEBHOOKS_SLACK_RATE_LIMITER_COUNT" default:"3"`
	RequestTimeout      time.Duration `envconfig:"WEBHOOKS_SLACK_REQUEST_TIMEOUT" default:"5s"`
	RetryDuration       time.Duration `envconfig:"WEBHOOKS_SLACK_RETRY_DURATION" default:"2s"`
	RetryTimes          uint          `envconfig:"WEBHOOKS_SLACK_RETRY_TIMES" default:"5"`
}

type Slack struct {
	ProviderConfig providers.Config
}

func NewProvider() providers.Provider {
	slackConfig := new(SlackProviderConfiguration)
	envconfig.MustProcess("webhooks", slackConfig)
	return &Slack{
		ProviderConfig: providers.Config{
			RequestTimeout:      slackConfig.RequestTimeout,
			RetryDuration:       slackConfig.RetryDuration,
			RetryTimes:          slackConfig.RetryTimes,
			RateLimiterDuration: slackConfig.RateLimiterDuration,
			RateLimiterCount:    slackConfig.RateLimiterCount,
		},
	}
}

func (p *Slack) Validate(config map[string]string) error {
	if url, urlPresent := config["url"]; !urlPresent {
		return errors.New("url field is not present in webhook configuration")
	} else if !strings.HasPrefix(url, "https://hooks.slack.com/") {
		return errors.New("url field does not start with 'https://hooks.slack.com/'")
	}
	return nil
}

func (p *Slack) Process(ctx context.Context, config map[string]string, event *lsApi.Event) (err error) {
	payload, err := p.message(event).JSON()
	if err != nil {
		return
	}

	retryFunc := func(requestTimeout time.Duration) (err error) {
		requestCtx, requestCtxCancel := context.WithTimeout(ctx, requestTimeout)
		defer requestCtxCancel()

		request, err := http.NewRequestWithContext(requestCtx, "POST", config["url"], bytes.NewReader(payload))
		if err != nil {
			return
		}
		request.Header.Set("Content-Type", "application/json")

		response, err := new(http.Client).Do(request)
		if err != nil {
			return
		}
		defer response.Body.Close()
		responseBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return
		}
		responseText := string(responseBytes)

		logrus.WithField("url", config["url"]).
			WithField("statusCode", response.StatusCode).
			WithField("response", responseText).
			Info("HTTP request processed")

		_, slackError := SlackErrors[responseText]

		switch {
		case response.StatusCode == 200:
			return
		case slackError:
			return helpers.NewNoRetryError(fmt.Errorf("known Slack error: %s", responseText))
		default:
			return fmt.Errorf("unexpected Slack response [%d]:%s", response.StatusCode, responseText)
		}
	}

	c := p.Config()
	return helpers.RetryWithLinearBackOff(retryFunc, c.RetryDuration, c.RetryTimes, c.RequestTimeout, config["url"])
}

func (p *Slack) message(event *lsApi.Event) *SlackMessage {
	message := NewMessage().AddBlocks(
		NewBlock("header", NewField("plain_text", "Calico Alert")),
		NewDivider(),
		NewBlock(
			"section", nil,
			NewMrkdwnField("⚠️ Alert Type:", event.Type),
			NewMrkdwnField("📟 Origin:", event.Origin),
			NewMrkdwnField("⏱️ Time:", event.Time.GetTime().String()),
			NewMrkdwnField("🔥 Severity:", fmt.Sprint(event.Severity)),
		),
		NewBlock("section", NewMrkdwnField("🗎 Description:", event.Description)),
	)

	return message
}

func (p *Slack) Config() providers.Config {
	return p.ProviderConfig
}
