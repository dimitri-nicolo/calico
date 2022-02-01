// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

// Query encapsulates parameters that define the hits requested from an ES query.
type Query struct {
	// The elastic-formatted query.
	EsQuery elastic.Query

	// The name of the index to use for the search.
	Index string

	// The page size per query.
	PageSize int

	// The from value.
	From int

	// Query max duration.
	Timeout time.Duration

	// Sort by information.
	SortBy []SortBy
}

type SortBy struct {
	Field     string
	Ascending bool
}

// ESResults encapsulates a set elastic search response items.
type ESResults struct {
	// True if ElasticSearch timed out.
	TimedOut bool `json:"timed_out"`

	// Search time in milliseconds.
	TookInMillis int64 `json:"took_in_millis"`

	// Value of the total hit count.
	TotalHits int64 `json:"total_hits"`

	// The actual hits returned, as a raw json.
	RawHits []json.RawMessage `json:"raw_hits"`
}

// TimedOutError is returned when the response indicates a timeout.
type TimedOutError string

func (e TimedOutError) Error() string {
	return string(e)
}

// Hits returns a collection of elastic search hits.
//
// Will return a timeout, along with the total search time. The response is JSON
// serializable. If an error occurs will return that error to the caller.
func Hits(ctx context.Context, query *Query, client *elastic.Client) (*ESResults, error) {
	// Query the hits at the document index.
	// Return a page of results, starting at a given offset.
	searchQuery := client.Search().
		Index(query.Index).
		Size(query.PageSize).
		Query(query.EsQuery).
		From(query.From)
	for _, s := range query.SortBy {
		searchQuery.Sort(s.Field, s.Ascending)
	}

	var rawHits []json.RawMessage
	if result, err := searchQuery.Do(ctx); err != nil {
		// We hit an error, exit.
		log.WithError(err).Debugf("Error searching %s", query.Index)
		return nil, err
	} else {
		// Return specific error type that can be recognized by the consumer - this is useful in
		// propagating the timeout up the stack when we are doing server side aggregation.
		if result.TimedOut {
			log.Errorf("Elastic query timed out: %s", query.Index)
			err = TimedOutError(fmt.Sprintf("timed out querying %s", query.Index))
		}
		if result.Hits != nil && result.Hits.TotalHits.Value > 0 {
			for _, hit := range result.Hits.Hits {
				rawHit, err := json.Marshal(map[string]interface{}{
					"id":     hit.Id,
					"index":  hit.Index,
					"source": hit.Source,
				})
				if err != nil {
					continue
				}
				rawHits = append(rawHits, rawHit)
			}
		} else {
			// Log when no hits are returned.
			log.Infof("No results for query of %s", query.Index)
		}

		return &ESResults{
			TimedOut:     result.TimedOut,
			TookInMillis: result.TookInMillis,
			TotalHits:    result.Hits.TotalHits.Value,
			RawHits:      rawHits,
		}, err
	}
}
