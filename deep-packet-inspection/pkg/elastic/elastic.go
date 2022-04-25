// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package elastic

import (
	"context"
	"time"

	lmaAPI "github.com/tigera/lma/pkg/api"
	lma "github.com/tigera/lma/pkg/elastic"

	"k8s.io/client-go/util/workqueue"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

type ESForwarder interface {
	Run(ctx context.Context)
	// Forward adds the given event data into worker queue, items in the worker queue are sent to ElasticSearch.
	Forward(item SecurityEvent)
}

func NewESForwarder(lmaClient lma.Client, elasticRetrySendInterval time.Duration) (ESForwarder, error) {
	fwd := &esForwarder{
		lmaESClient:              lmaClient,
		elasticRetrySendInterval: elasticRetrySendInterval,
		queue:                    workqueue.New(),
	}
	return fwd, nil
}

// esForwarder is an implementation of ESForwarder interface.
type esForwarder struct {
	lmaESClient              lma.Client
	elasticRetrySendInterval time.Duration
	queue                    *workqueue.Type
}

type SecurityEvent struct {
	lmaAPI.EventsData
	DocID string
}

type Record struct {
	SnortSignatureID       string `json:"snort_signature_id"`
	SnortSignatureRevision string `json:"snort_signature_revision"`
	SnortAlert             string `json:"snort_alert"`
}

func (e SecurityEvent) EventData() lmaAPI.EventsData {
	return e.EventsData
}

func (e SecurityEvent) ID() string {
	return e.DocID
}

func (s *esForwarder) Run(ctx context.Context) {
	go s.run(ctx)
}

// Forward adds the given event data into worker queue, items in the worker queue are sent to ElasticSearch.
func (s *esForwarder) Forward(item SecurityEvent) {
	log.Debugf("Adding item to queue %#v", item)
	s.queue.Add(item)
}

// run gets item from worker queue, sends it to ElasticSearch for processing, if send fails due to connection
// or authorization error, it retries sending the document on interval.
func (s *esForwarder) run(ctx context.Context) {
	for {
		// Get blocks until there is item to process in the queue
		item, shutdown := s.queue.Get()
		if shutdown {
			log.Error("Worker queue sending to ElasticSearch has shutdown.")
			return
		}
		if event, ok := item.(SecurityEvent); ok {
			_, err := s.lmaESClient.PutSecurityEventWithID(ctx, event.EventData(), event.DocID)
			for ; err != nil; _, err = s.lmaESClient.PutSecurityEventWithID(ctx, event.EventData(), event.DocID) {
				if elastic.IsConnErr(err) || elastic.IsForbidden(err) || elastic.IsUnauthorized(err) {
					log.WithError(err).Error("Failed to send document to ElasticSearch, will retry after interval.")
					<-time.After(s.elasticRetrySendInterval)
					continue
				}
				log.WithError(err).Error("Failed to send document to ElasticSearch")
				break
			}
		} else {
			log.Error("Failed to parse the security event in worker queue")
		}

		log.Debugf("Removing item from the worker queue after sending to ES %#v", item)
		s.queue.Done(item)
	}
}
