// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package alert

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/util/workqueue"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
)

type Forwarder interface {
	Run(ctx context.Context)
	// Forward adds the given event data into worker queue, items in the worker queue are sent to Linseed.
	Forward(item v1.Event)
}

func NewForwarder(lsClient client.Client, retrySendInterval time.Duration, clusterName string) (Forwarder, error) {
	fwd := &forwarder{
		lsClient:          lsClient,
		clusterName:       clusterName,
		retrySendInterval: retrySendInterval,
		queue:             workqueue.New(),
	}
	return fwd, nil
}

// forwarder is an implementation of Forwarder interface.
type forwarder struct {
	lsClient          client.Client
	clusterName       string
	retrySendInterval time.Duration
	queue             *workqueue.Type
}

func (s *forwarder) Run(ctx context.Context) {
	go s.run(ctx)
}

// Forward adds the given event data into worker queue, items in the worker queue are sent to Linseed
func (s *forwarder) Forward(item v1.Event) {
	log.Debugf("Adding item to queue %#v", item)
	s.queue.Add(item)
}

// run gets item from worker queue, sends it to Linseed for processing, if send fails due to connection
// or authorization error, it retries sending the document on interval.
func (s *forwarder) run(ctx context.Context) {
	for {
		// Get blocks until there is item to process in the queue
		item, shutdown := s.queue.Get()
		if shutdown {
			log.Error("Worker queue sending to Linseed has shutdown.")
			return
		}
		if event, ok := item.(v1.Event); ok {
			response, err := s.lsClient.Events(s.clusterName).Create(ctx, []v1.Event{event})
			if err != nil {
				log.WithError(err).Error("Failed to send document to Linseed")
				break
			}

			if response.Failed != 0 {
				log.WithError(err).Error("Failed to send document to Linseed, will retry after interval.")
				<-time.After(s.retrySendInterval)
				continue
			}

		} else {
			log.Error("Failed to parse the security event in worker queue")
		}

		log.Debugf("Removing item from the worker queue after sending to ES %#v", item)
		s.queue.Done(item)
	}
}
