// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package elastic

import (
	"context"
	"fmt"
	"time"

	"github.com/tigera/deep-packet-inspection/pkg/config"
	"k8s.io/client-go/util/workqueue"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

const (
	EventIndexPattern = "tigera_secure_ee_events.%s"
)

type newClient func(esCLI *elastic.Client, elasticIndexSuffix string) Client

type Client interface {
	// Upsert adds/updates the document with given document ID to ElasticSearch.
	// Using docID to index ensures there are no duplicate alerts for same document.
	Upsert(ctx context.Context, docID string, doc Doc) error
}

// client is an implementation of Client interface.
type client struct {
	esCLI      *elastic.Client
	eventIndex string
}

func NewClient(esCLI *elastic.Client, elasticIndexSuffix string) Client {
	return &client{
		esCLI:      esCLI,
		eventIndex: fmt.Sprintf(EventIndexPattern, elasticIndexSuffix)}
}

// Upsert adds/updates the document with given document ID to ElasticSearch.
func (es *client) Upsert(ctx context.Context, docID string, doc Doc) error {
	log.Debugf("Sending document to es...")
	_, err := es.esCLI.Index().Index(es.eventIndex).Id(docID).BodyJson(doc).Do(ctx)
	return err
}

type ESForwarder interface {
	Run(ctx context.Context)
	// Forward adds the given event data into worker queue, items in the worker queue are sent to ElasticSearch.
	Forward(item EventData)
}

func NewESForwarder(cfg *config.Config, esClient newClient, elasticRetrySendInterval time.Duration) (ESForwarder, error) {
	c, err := NewElasticClient(cfg)
	if err != nil {
		return nil, err
	}
	fwd := &esForwarder{
		esClient:                 esClient(c, cfg.ElasticIndexSuffix),
		elasticRetrySendInterval: elasticRetrySendInterval,
		queue:                    workqueue.New(),
	}
	return fwd, nil
}

// esForwarder is an implementation of ESForwarder interface.
type esForwarder struct {
	esClient                 Client
	elasticRetrySendInterval time.Duration
	queue                    *workqueue.Type
}

type Doc struct {
	Alert           string `json:"alert"`
	Time            int64  `json:"time"`
	Type            string `json:"type"`
	Host            string `json:"host"`
	SourceIP        string `json:"source_ip"`
	SourceName      string `json:"source_name"`
	SourceNamespace string `json:"source_namespace"`
	DestIP          string `json:"dest_ip"`
	DestName        string `json:"dest_name"`
	DestNamespace   string `json:"dest_namespace"`
	Description     string `json:"description"`
	Severity        int    `json:"severity"`
	Record          Record `json:"record"`
}

type Record struct {
	SnortSignatureID       string `json:"snort_signature_id"`
	SnortSignatureRevision string `json:"snort_signature_revision"`
	SnortAlert             string `json:"snort_alert"`
}

type EventData struct {
	Doc Doc
	ID  string
}

func (s *esForwarder) Run(ctx context.Context) {
	go s.run(ctx)
}

// Forward adds the given event data into worker queue, items in the worker queue are sent to ElasticSearch.
func (s *esForwarder) Forward(item EventData) {
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
		var err error
		err = s.esClient.Upsert(ctx, item.(EventData).ID, item.(EventData).Doc)
		for ; err != nil; err = s.esClient.Upsert(ctx, item.(EventData).ID, item.(EventData).Doc) {
			if elastic.IsConnErr(err) || elastic.IsForbidden(err) || elastic.IsUnauthorized(err) {
				log.WithError(err).Error("Failed to send document to ElasticSearch, will retry after interval.")
				<-time.After(s.elasticRetrySendInterval)
				continue
			}
			log.WithError(err).Error("Failed to send document to ElasticSearch")
			break
		}
		log.Debugf("Removing item from the worker queue after sending to ES %#v", item)
		s.queue.Done(item)
	}
}
