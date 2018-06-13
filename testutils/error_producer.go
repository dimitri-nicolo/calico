// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package testutils

import (
	"fmt"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

// ErrorProducer produces a sequence of previously-queued errors from its NextError method.
type ErrorProducer map[string][]error

func NewErrorQueue() ErrorProducer {
	return ErrorProducer{}
}

// QueueError adds an error to the sequence of errors with the given name.
func (e ErrorProducer) QueueError(queueName string) {
	e[queueName] = append(e[queueName], fmt.Errorf("dummy ErrorQueue %s error", queueName))
}

// QueueError adds an error to the sequence of errors with the given name.
func (e ErrorProducer) QueueNErrors(queueName string, n int) {
	for i := 0; i < n; i++ {
		e[queueName] = append(e[queueName], fmt.Errorf("dummy ErrorQueue %s error", queueName))
	}
}

// NextError returns the next error in the sequence with the given name.  It returns nil if there is no such error.
func (e ErrorProducer) NextError(queueName string) error {
	errs := e[queueName]
	if len(errs) > 0 {
		err := errs[0]
		if len(errs) == 1 {
			delete(e, queueName)
		} else {
			e[queueName] = errs[1:]
		}
		if err != nil {
			logrus.WithError(err).WithField("type", queueName).Warn("Simulating error")
			return err
		}
	}
	return nil
}

func (e ErrorProducer) ExpectAllErrorsConsumed() {
	gomega.ExpectWithOffset(1, e).To(gomega.BeEmpty())
}
