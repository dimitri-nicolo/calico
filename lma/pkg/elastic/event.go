// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"encoding/json"
	"fmt"

	_ "embed"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/lma/pkg/api"
)

var eventDismissDoc map[string]bool = map[string]bool{"dismissed": true}

func (c *client) EventsIndexExists(ctx context.Context) (bool, error) {
	alias := c.ClusterAlias(EventsIndex)
	return c.IndexExists(alias).Do(ctx)
}

var (
	EventsIndex = "tigera_secure_ee_events"

	// TODO: Remove this once intrusion detection have been migrated to Linseed.
	//go:embed events_mappings.json
	eventsMapping string
)

// CreateEventsIndex creates events index with mapping if it doesn't exist.
// It marks the new index as write index for events index alias and marks the old index (prior to CEv3.12)
// as read index for the alias.
// TODO CASEY: Delete this, and update anyyone using it to just wait for the index to exist. Linseed will make it.
func (c *client) CreateEventsIndex(ctx context.Context) error {
	alias := c.ClusterAlias(EventsIndex)

	// The index pattern used in index template should only map to the new index created by CE >= v3.12, so
	// pass the write index name to create index template.
	eventsIndexTemplate, err := c.IndexTemplate(alias, EventsIndex, eventsMapping, false)
	if err != nil {
		log.WithError(err).Error("failed to build index template")
		return err
	}
	if err := c.ensureIndexExistsWithRetry(EventsIndex, eventsIndexTemplate, false); err != nil {
		return err
	}

	return nil
}

// PutSecurityEventWithID adds the given data into events index for the given id.
// If id is empty, Elasticsearch generates one.
// This function can be used to send same events multiple time without creating duplicate
// entries in Elasticsearch as long as the id remains the same.
func (c *client) PutSecurityEventWithID(ctx context.Context, data api.EventsData, id string) (*elastic.IndexResponse, error) {
	alias := c.ClusterAlias(EventsIndex)

	// Marshall the api.EventsData to ignore empty values
	b, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("failed to marshall")
		return nil, err
	}
	return c.Index().Index(alias).Id(id).BodyString(string(b)).Do(ctx)
}

// PutSecurityEvent adds the given data into events index.
func (c *client) PutSecurityEvent(ctx context.Context, data api.EventsData) (*elastic.IndexResponse, error) {
	alias := c.ClusterAlias(EventsIndex)

	// Marshall the api.EventsData to ignore empty values
	b, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("failed to marshall")
		return nil, err
	}
	return c.Index().Index(alias).BodyString(string(b)).Do(ctx)
}

// DismissSecurityEvent sets the dismissed field to true for an event by id.
func (c *client) DismissSecurityEvent(ctx context.Context, index, id string) (*elastic.UpdateResponse, error) {
	return c.Update().Index(index).Id(id).Doc(eventDismissDoc).Do(ctx)
}

// DeleteSecurityEvent deletes the event by id.
func (c *client) DeleteSecurityEvent(ctx context.Context, index, id string) (*elastic.DeleteResponse, error) {
	return c.Delete().Index(index).Id(id).Do(ctx)
}

// BulkProcessorInitialize creates a bulk processor service and sets default flush size and BulkAfterFunc that
// needs to be executed after bulk request is committed.
func (c *client) BulkProcessorInitialize(ctx context.Context, afterFn elastic.BulkAfterFunc) error {
	return c.bulkProcessorInit(ctx, afterFn, api.AutoBulkFlush)
}

// BulkProcessorInitialize creates a bulk processor service and sets given flush size and BulkAfterFunc that
// needs to be executed after bulk request is committed.
func (c *client) BulkProcessorInitializeWithFlush(ctx context.Context, afterFn elastic.BulkAfterFunc, bulkActions int) error {
	return c.bulkProcessorInit(ctx, afterFn, bulkActions)
}

func (c *client) bulkProcessorInit(ctx context.Context, afterFn elastic.BulkAfterFunc, bulkActions int) error {
	var err error
	c.bulkProcessor, err = c.BulkProcessor().
		BulkActions(bulkActions).
		After(afterFn).
		Do(ctx)
	return err
}

// PutBulkSecurityEvent adds the given data to bulk processor service,
// the data is flushed either automatically to Elasticsearch when the document count reaches BulkActions, or
// when bulk processor service is closed.
func (c *client) PutBulkSecurityEvent(data api.EventsData) error {
	if c.bulkProcessor == nil {
		return fmt.Errorf("BulkProcessor not initalized")
	}
	alias := c.ClusterAlias(EventsIndex)

	// Marshall the api.EventsData to ignore empty values
	b, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("failed to marshall")
		return err
	}
	r := elastic.NewBulkIndexRequest().Index(alias).Doc(string(b))
	c.bulkProcessor.Add(r)
	return nil
}

// DismissBulkSecurityEvent adds the event dismissal request to bulk processor service,
// the data is flushed either automatically to Elasticsearch when the document count reaches BulkActions, or
// when bulk processor service is closed.
func (c *client) DismissBulkSecurityEvent(index, id string) error {
	if c.bulkProcessor == nil {
		return fmt.Errorf("BulkProcessor not initialized")
	}

	r := elastic.NewBulkUpdateRequest().Index(index).Id(id).Doc(eventDismissDoc)
	c.bulkProcessor.Add(r)
	return nil
}

// DeleteBulkSecurityEvent adds the event deleting request to bulk processor service,
// the data is flushed either automatically to Elasticsearch when the document count reaches BulkActions, or
// when bulk processor service is closed.
func (c *client) DeleteBulkSecurityEvent(index, id string) error {
	if c.bulkProcessor == nil {
		return fmt.Errorf("BulkProcessor not initalized")
	}

	r := elastic.NewBulkDeleteRequest().Index(index).Id(id)
	c.bulkProcessor.Add(r)
	return nil
}

// BulkProcessorFlush is called to manually flush the pending requests in bulk processor service
func (c *client) BulkProcessorFlush() error {
	return c.bulkProcessor.Flush()
}

// BulkProcessorClose flushes the pending requests in bulk processor service and closes it.
func (c *client) BulkProcessorClose() error {
	if err := c.bulkProcessor.Flush(); err != nil {
		return err
	}
	return c.bulkProcessor.Close()
}
