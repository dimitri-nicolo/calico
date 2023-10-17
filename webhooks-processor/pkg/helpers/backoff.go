// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package helpers

import (
	"time"

	"github.com/projectcalico/calico/webhooks-processor/pkg/providers"
)

type NoRetryError struct {
	err error
}

func NewNoRetryError(err error) *NoRetryError {
	return &NoRetryError{err: err}
}

func (e *NoRetryError) Error() string {
	return e.err.Error()
}

type RetryFunction func(requestTimeout time.Duration) error

type BackOffFunction func(time.Duration, uint) <-chan time.Time

func RetryWithBackOff(retryFunc RetryFunction, backOffFunc BackOffFunction, config providers.RetryConfig, info string) (err error) {
	for iteration := uint(0); iteration < config.RetryTimes; iteration++ {
		if iteration > 0 {
			<-backOffFunc(config.RetryDuration, iteration)
		}
		err = retryFunc(config.RequestTimeout)
		if err == nil {
			break
		}
		switch err.(type) {
		case *NoRetryError:
			break
		default:
			continue
		}
	}
	return
}

func RetryWithConstantBackOff(retry RetryFunction, config providers.RetryConfig, info string) (err error) {
	backOffFunc := func(duration time.Duration, iteration uint) <-chan time.Time {
		return time.NewTimer(duration).C
	}
	return RetryWithBackOff(retry, backOffFunc, config, info)
}

func RetryWithLinearBackOff(retry RetryFunction, config providers.RetryConfig, info string) (err error) {
	backOffFunc := func(duration time.Duration, iteration uint) <-chan time.Time {
		return time.NewTimer(duration * time.Duration(iteration)).C
	}
	return RetryWithBackOff(retry, backOffFunc, config, info)
}

func RetryWithExponentialBackOff(retry RetryFunction, config providers.RetryConfig, info string) (err error) {
	backOffFunc := func(duration time.Duration, iteration uint) <-chan time.Time {
		return time.NewTimer(duration * time.Duration(0x01<<iteration-1)).C
	}
	return RetryWithBackOff(retry, backOffFunc, config, info)
}
