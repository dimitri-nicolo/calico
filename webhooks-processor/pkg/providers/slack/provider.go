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

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
)

const (
	RequestTimeout = 5 * time.Second
	RetryDuration  = 2 * time.Second
	RetryTimes     = 5
)

type Slack struct {
}

func NewProvider() *Slack {
	return &Slack{}
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

	retryFunc := func() (err error) {
		requestCtx, requestCtxCancel := context.WithTimeout(ctx, RequestTimeout)
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

	return helpers.RetryWithLinearBackOff(retryFunc, RetryDuration, RetryTimes, config["url"])
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
