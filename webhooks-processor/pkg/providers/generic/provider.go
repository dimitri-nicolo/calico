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

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
)

const (
	RequestTimeout = 5 * time.Second
	RetryDuration  = 2 * time.Second
	RetryTimes     = 5
)

type GenericProvider struct {
}

func NewProvider() *GenericProvider {
	return &GenericProvider{}
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

		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			return
		}

		return fmt.Errorf("unexpected response [%d]:%s", response.StatusCode, responseText)
	}

	return helpers.RetryWithLinearBackOff(retryFunc, RetryDuration, RetryTimes, config["url"])
}
