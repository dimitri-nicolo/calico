// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	api "github.com/tigera/lma/pkg/api"
)

const (
	AlertIndex           = "tigera_secure_ee_events"
	DefaultAlertPageSize = 100
)

func (c *client) GetAlertLogs(ctx context.Context, start, end *time.Time) <-chan *api.AlertResult {
	return c.SearchAlertLogs(ctx, nil, start, end)
}

// Issue an Elasticsearch query that matches alert logs.
func (c *client) SearchAlertLogs(ctx context.Context, filter *api.AlertLogsSelection, start, end *time.Time) <-chan *api.AlertResult {
	resultChan := make(chan *api.AlertResult, resultBucketSize)
	alertSearchIndex := c.ClusterIndex(AlertIndex, "")

	// Create ES queries using given filters and time interval.
	queries := constructAlertLogsQuery(filter, start, end)

	go func() {
		defer close(resultChan)

		scroll := c.Scroll(alertSearchIndex).
			Size(DefaultAlertPageSize).
			Query(queries).
			Sort(api.AlertLogTime, true)

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
				resultChan <- &api.AlertResult{Err: err}
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
				resultChan <- &api.AlertResult{Err: err}
				return
			}
			log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

			// define function that pushes the search results into the channel.
			for _, hit := range res.Hits.Hits {
				var a api.Alert
				if err := json.Unmarshal(hit.Source, &a); err != nil {
					log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
					continue
				}
				resultChan <- &api.AlertResult{Alert: &a}
			}
		}

		if err := scroll.Clear(ctx); err != nil {
			log.WithError(err).Info("Failed to clear scroll context")
		}
	}()

	return resultChan
}

func constructAlertLogsQuery(filter *api.AlertLogsSelection, start, end *time.Time) elastic.Query {
	queries := []elastic.Query{}

	// Query by filter if specified.
	if filter != nil {
		queries = append(queries, alertLogQueryFromAlertLogsSelection(filter))
	}

	// Query by from/to if specified.
	if start != nil || end != nil {
		rangeQuery := elastic.NewRangeQuery(api.AlertLogTime)
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

func alertLogQueryFromAlertLogsSelection(filter *api.AlertLogsSelection) elastic.Query {
	if len(filter.Resources) == 0 {
		return nil
	}
	queries := []elastic.Query{}
	for _, res := range filter.Resources {
		queries = append(queries, alertLogQueryFromAlertResource(res))
	}
	return elastic.NewBoolQuery().Should(queries...)
}

func alertLogQueryFromAlertResource(res api.AlertResource) elastic.Query {
	queries := []elastic.Query{}
	if res.Type != "" {
		queries = append(queries, elastic.NewMatchQuery(api.AlertLogType, res.Type))
	}
	if res.SourceNamespace != "" {
		queries = append(queries, elastic.NewMatchQuery(api.AlertLogSourceNamespace, res.SourceNamespace))
	}
	if res.DestNamespace != "" {
		queries = append(queries, elastic.NewMatchQuery(api.AlertLogDestNamespace, res.DestNamespace))
	}
	if res.Alert != "" {
		queries = append(queries, elastic.NewMatchQuery(api.AlertLogAlert, res.Alert))
	}
	return elastic.NewBoolQuery().Must(queries...)
}
