// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
)

type Jira struct {
	ProviderConfig providers.Config
}

func NewProvider(config providers.Config) providers.Provider {
	return &Jira{
		ProviderConfig: config,
	}
}

func (p *Jira) Validate(config map[string]string) error {
	if _, ok := config["url"]; !ok {
		return errors.New("url field is not present in webhook configuration")
	}
	if _, ok := config["project"]; !ok {
		return errors.New("project field not present in webhook configuration")
	}
	if _, ok := config["issueType"]; !ok {
		return errors.New("issueType field not present in webhook configuration")
	}
	if _, ok := config["username"]; !ok {
		return errors.New("username field not present in webhook configuration")
	}
	if _, ok := config["apiToken"]; !ok {
		return errors.New("apiToken field not present in webhook configuration")
	}
	return nil
}

func (p *Jira) Process(ctx context.Context, config map[string]string, event *lsApi.Event) (err error) {
	payload := new(jiraPayload)
	payload.Fields.Project.Key = config["project"]
	payload.Fields.IssueType.Name = config["issueType"]
	if payload.Fields.Summary, err = buildSummary(event); err != nil {
		return
	}
	if payload.Fields.Description, err = buildDescription(event); err != nil {
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	retryFunc := func() (err error) {
		requestCtx, requestCtxCancel := context.WithTimeout(ctx, p.Config().RequestTimeout)
		defer requestCtxCancel()

		request, err := http.NewRequestWithContext(requestCtx, "POST", config["url"], bytes.NewReader(payloadBytes))
		if err != nil {
			return
		}
		request.SetBasicAuth(config["username"], config["apiToken"])
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

		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			return
		}
		return fmt.Errorf("unexpected Jira response [%d]:%s", response.StatusCode, responseText)
	}

	c := p.Config()
	return helpers.RetryWithLinearBackOff(retryFunc, c.RetryDuration, c.RetryTimes, config["url"])
}

func (p *Jira) Config() providers.Config {
	return p.ProviderConfig
}
