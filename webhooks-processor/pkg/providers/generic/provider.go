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

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"

	"github.com/sirupsen/logrus"
)

type GenericProvider struct {
	config providers.Config
}

func NewProvider(config providers.Config) providers.Provider {
	return &GenericProvider{
		config: config,
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

	retryFunc := func() (err error) {
		requestCtx, requestCtxCancel := context.WithTimeout(ctx, p.Config().RequestTimeout)
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

	c := p.Config()
	return helpers.RetryWithLinearBackOff(retryFunc, c.RetryDuration, c.RetryTimes, config["url"])
}

func (p *GenericProvider) Config() providers.Config {
	return p.config
}
