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
	Query elastic.Query

	// The name of the index to use for the search.
	Index string

	// The page size per query.
	PageSize int

	// The sort values that indicates which docs this request should "search after".
	SearchAfter interface{}

	// Query max duration.
	Timeout time.Duration
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
	RawHits []json.RawMessage `json:"raw_messages"`
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
func Hits(
	ctxIn context.Context, client *elastic.Client, query *Query,
) (*ESResults, error) {
	// Create a context with timeout to ensure we don't block for too long with this query.
	ctx, cancelWithTimeout := context.WithTimeout(ctxIn, query.Timeout)
	// Releases timer resources if the operation completes before the timeout.
	defer cancelWithTimeout()

	// Retrieve hits. Time request and decoding.
	result, err := searchElastic(
		ctx, client, query.Query, query.Index, query.PageSize, query.SearchAfter)

	// If there was an error, check for a timeout. If it timed out just flag this in the response,
	// but return whatever data we already have. Otherwise return the error.
	// For timeouts we have a couple of mechanisms for hitting this:
	// - The elastic search query returns a timeout.
	if err != nil {
		if _, ok := err.(TimedOutError); !ok { //nolint:golint,gosimple
			// Just pass the received error up the stack.
			log.WithError(err).Warning("Error response from elasticsearch query")
		} else {
			// Response from ES indicates a handled timeout.
			log.WithError(err).
				Warning("Response from ES indicates time out - return as timedout error")
		}
		return nil, err
	}
	return result, nil
}

// searchElastic queries ES for the requested hits.
func searchElastic(
	ctx context.Context,
	c *elastic.Client,
	esq elastic.Query,
	index string,
	pageSize int,
	searchAfter interface{},
) (*ESResults, error) {
	// Query the hits at the document index.
	// Return a page of results, starting at a given offset.
	searchQuery := c.Search().Index(index).Size(pageSize).Query(esq)

	if searchAfter != nil {
		searchQuery.SearchAfter(searchAfter)
	}

	var rawHits []json.RawMessage
	if result, err := searchQuery.Do(ctx); err != nil {
		// We hit an error, exit.
		log.WithError(err).Debugf("Error searching %s", index)
		return nil, err
	} else {
		// Return specific error type that can be recognized by the consumer - this is useful in
		// propagating the timeout up the stack when we are doing server side aggregation.
		if result.TimedOut {
			log.Errorf("Elastic query timed out: %s", index)
			err = TimedOutError(fmt.Sprintf("timed out querying %s", index))
		}
		if result.Hits != nil && result.Hits.TotalHits.Value > 0 {
			for _, hit := range result.Hits.Hits {
				rawHits = append(rawHits, hit.Source)
			}
		} else {
			// Log when no hits are returned.
			log.Infof("No results for query of %s", index)
		}

		return &ESResults{
			TimedOut:     result.TimedOut,
			TookInMillis: result.TookInMillis,
			TotalHits:    result.Hits.TotalHits.Value,
			RawHits:      rawHits,
		}, err
	}
}
