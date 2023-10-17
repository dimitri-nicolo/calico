// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package generic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kelseyhightower/envconfig"
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"

	"github.com/sirupsen/logrus"
)

type GenericProviderConfiguration struct {
	RateLimiterDuration time.Duration `envconfig:"WEBHOOKS_GENERIC_RATE_LIMITER_DURATION" default:"1h"`
	RateLimiterCount    uint          `envconfig:"WEBHOOKS_GENERIC_RATE_LIMITER_COUNT" default:"100"`
	RequestTimeout      time.Duration `envconfig:"WEBHOOKS_GENERIC_REQUEST_TIMEOUT" default:"5s"`
	RetryDuration       time.Duration `envconfig:"WEBHOOKS_GENERIC_RETRY_DURATION" default:"2s"`
	RetryTimes          uint          `envconfig:"WEBHOOKS_GENERIC_RETRY_TIMES" default:"5"`
}

type GenericProvider struct {
	Config *GenericProviderConfiguration
}

func NewProvider() providers.Provider {
	config := new(GenericProviderConfiguration)
	envconfig.MustProcess("webhooks", config)
	return &GenericProvider{
		Config: config,
	}
}

func (p *GenericProvider) Validate(config map[string]string) error {
	if _, urlPresent := config["url"]; !urlPresent {
		return errors.New("url field is not present in webhook configuration")
	}
	return nil
}

func (p *GenericProvider) Process(ctx context.Context, config map[string]string, event *lsApi.Event) (err error) {
	payload, err := json.Marshal(event)
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

		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			return
		}

		return fmt.Errorf("unexpected response [%d]:%s", response.StatusCode, responseText)
	}

	return helpers.RetryWithLinearBackOff(retryFunc, p.RetryConfig(), config["url"])
}

func (p *GenericProvider) RetryConfig() providers.RetryConfig {
	return providers.RetryConfig{
		RequestTimeout: p.Config.RequestTimeout,
		RetryDuration:  p.Config.RetryDuration,
		RetryTimes:     p.Config.RetryTimes,
	}
}

func (p *GenericProvider) RateLimiterConfig() providers.RateLimiterConfig {
	return providers.RateLimiterConfig{
		RateLimiterDuration: p.Config.RateLimiterDuration,
		RateLimiterCount:    p.Config.RateLimiterCount,
	}
}
