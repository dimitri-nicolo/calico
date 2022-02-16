// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/lma/pkg/api"
)

const (
	DefaultEventPageSize = 100
)

var (
	eventDismissDoc map[string]bool = map[string]bool{"dismissed": true}
)

func (c *client) EventsIndexExists(ctx context.Context) (bool, error) {
	alias := c.ClusterAlias(EventsIndex)
	return c.IndexExists(alias).Do(ctx)
}

// CreateEventsIndex creates events index with mapping if it doesn't exist.
// It marks the new index as write index for events index alias and marks the old index (prior to CEv3.12)
// as read index for the alias.
func (c *client) CreateEventsIndex(ctx context.Context) error {
	alias := c.ClusterAlias(EventsIndex)
	oldIndex := c.ClusterIndex(EventsIndex, "")

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

	// If there is an old events index created by CE <v3.12, add it to the alias as read index.
	exists, err := c.IndexExists(oldIndex).Do(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to check if index exists")
		return err
	}
	if exists {
		_, err := c.Alias().Action(elastic.NewAliasAddAction(alias).Index(oldIndex).IsWriteIndex(false)).Do(ctx)
		if err != nil {
			log.WithError(err).Error("Failed to mark old events index as read index")
			return err
		}
	}

	return nil
}

// PutSecurityEventWithID adds the given data into events index for the given ID.
// If ID is empty, Elasticsearch generates one.
// This function can be used to send same events multiple time without creating duplicate
// entries in Elasticsearch as long as the ID remains the same.
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

func (c *client) DismissSecurityEvent(ctx context.Context, id string) (*elastic.UpdateResponse, error) {
	alias := c.ClusterAlias(EventsIndex)

	return c.Update().Index(alias).Id(id).Doc(eventDismissDoc).Do(ctx)
}

func (c *client) DeleteSecurityEvent(ctx context.Context, id string) (*elastic.DeleteResponse, error) {
	alias := c.ClusterAlias(EventsIndex)

	return c.Delete().Index(alias).Id(id).Do(ctx)
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

func (c *client) DismissBulkSecurityEvent(id string) error {
	if c.bulkProcessor == nil {
		return fmt.Errorf("BulkProcessor not initialized")
	}
	alias := c.ClusterAlias(EventsIndex)

	r := elastic.NewBulkUpdateRequest().Index(alias).Id(id).Doc(eventDismissDoc)
	c.bulkProcessor.Add(r)
	return nil
}

func (c *client) DeleteBulkSecurityEvent(id string) error {
	if c.bulkProcessor == nil {
		return fmt.Errorf("BulkProcessor not initalized")
	}
	alias := c.ClusterAlias(EventsIndex)

	r := elastic.NewBulkDeleteRequest().Index(alias).Id(id)
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

// This is used by Honeypod + Alert forwader
// if start/end is passed it will be used insted of EventsSearchFields.Time
func (c *client) SearchSecurityEvents(ctx context.Context, start, end *time.Time, filterData []api.EventsSearchFields, allClusters bool) <-chan *api.EventResult {
	resultChan := make(chan *api.EventResult, resultBucketSize)
	var index string
	if allClusters {
		// When allClusters is true use wildcard to query all events index instead of alias to
		// cover older managed clusters that do not use alias for events index.
		index = api.EventIndexWildCardPattern
	} else {
		index = c.ClusterAlias(EventsIndex)
	}
	queries := constructEventLogsQuery(start, end, filterData)
	go func() {
		defer close(resultChan)
		scroll := c.Scroll(index).
			Size(DefaultEventPageSize).
			Query(queries).
			Sort(api.EventTime, true)

		// Issue the query to Elasticsearch and send results out through the resultsChan.
		// We only terminate the search if when there are no more results to scroll through.
		for {
			log.Debug("Issuing alerts search query")
			res, err := scroll.Do(ctx)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.WithError(err).Error("Failed to search alert logs")

				resultChan <- &api.EventResult{Err: err}
				return
			}
			if res == nil {
				err = fmt.Errorf("Search expected results != nil; got nil")
			} else if res.Hits == nil {
				err = fmt.Errorf("Search expected results.Hits != nil; got nil")
			} else if len(res.Hits.Hits) == 0 {
				err = fmt.Errorf("Search expected results.Hits.Hits > 0; got 0")
			}
			if err != nil {
				log.WithError(err).Warn("Unexpected results from alert logs search")
				resultChan <- &api.EventResult{Err: err}
				return
			}
			log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

			// Pushes the search results into the channel.
			for _, hit := range res.Hits.Hits {
				var a api.EventsData
				if err := json.Unmarshal(hit.Source, &a); err != nil {
					log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
					continue
				}
				resultChan <- &api.EventResult{EventsData: &a, ID: hit.Id}
			}
		}
	}()

	return resultChan
}

func constructEventLogsQuery(start *time.Time, end *time.Time, filterData []api.EventsSearchFields) elastic.Query {
	queries := []elastic.Query{}
	for _, data := range filterData {
		innerQ := []elastic.Query{}
		v := reflect.ValueOf(data)
		values := make([]interface{}, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			innerQ = append(innerQ, elastic.NewMatchQuery(v.Field(i).String(), values[i]))
		}
		queries = append(queries, elastic.NewBoolQuery().Must(innerQ...))
	}

	if start != nil || end != nil {
		rangeQuery := elastic.NewRangeQuery(api.EventTime)
		if start != nil {
			rangeQuery = rangeQuery.From(*start)
		}
		if end != nil {
			rangeQuery = rangeQuery.To(*end)
		}
		queries = append(queries, rangeQuery)
	}

	return elastic.NewBoolQuery().Must(queries...)
}
