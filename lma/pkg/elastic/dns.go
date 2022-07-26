// Copyright (c) 2020-2022 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	api "github.com/projectcalico/calico/lma/pkg/api"
)

const DNSIndex = "tigera_secure_ee_dns"

func (c *client) GetDNSLogs(ctx context.Context, start, end *time.Time) <-chan *api.DNSResult {
	return c.SearchDNSLogs(ctx, nil, start, end)
}

// Issue an Elasticsearch query that matches alert logs.
func (c *client) SearchDNSLogs(ctx context.Context, filter *api.DNSLogsSelection, start, end *time.Time) <-chan *api.DNSResult {
	resultChan := make(chan *api.DNSResult, DefaultPageSize)
	dnsSearchIndex := c.ClusterIndex(DNSIndex, "*")

	// Create ES queries using given filters and time interval.
	queries := constructDNSLogsQuery(filter, start, end)

	go func() {
		defer close(resultChan)

		scroll := c.Scroll(dnsSearchIndex).
			Size(DefaultPageSize).
			Query(queries).
			Sort(api.DNSLogStartTime, true)

		// Issue the query to Elasticsearch and send results out through the resultsChan.
		// We only terminate the search if when there are no more results to scroll through.
		for {
			log.Debug("Issuing DNS search query")
			res, err := scroll.Do(ctx)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.WithError(err).Error("Failed to search DNS logs")
				resultChan <- &api.DNSResult{Err: err}
				return
			}

			if res == nil {
				err = fmt.Errorf("search expected results != nil; got nil")
			} else if res.Hits == nil {
				err = fmt.Errorf("search expected results.Hits != nil; got nil")
			} else if len(res.Hits.Hits) == 0 {
				err = fmt.Errorf("search expected results.Hits.Hits > 0; got 0")
			}
			if err != nil {
				log.WithError(err).Warn("Unexpected results from DNS logs search")
				resultChan <- &api.DNSResult{Err: err}
				return
			}
			log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

			// Pushes the search results into the channel.
			for _, hit := range res.Hits.Hits {
				var d api.DNSLog
				if err := json.Unmarshal(hit.Source, &d); err != nil {
					log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
					continue
				}
				resultChan <- &api.DNSResult{DNSLog: &d}
			}
		}

		if err := scroll.Clear(ctx); err != nil {
			log.WithError(err).Info("Failed to clear scroll context")
		}
	}()

	return resultChan
}

func constructDNSLogsQuery(filter *api.DNSLogsSelection, start, end *time.Time) elastic.Query {
	queries := []elastic.Query{}

	// Query by filter if specified.
	if filter != nil {
		queries = append(queries, dnsLogQueryFromDNSLogsSelection(filter))
	}

	// Query by from/to if specified.
	if start != nil || end != nil {
		rangeQuery := elastic.NewRangeQuery(api.DNSLogStartTime)
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

func dnsLogQueryFromDNSLogsSelection(filter *api.DNSLogsSelection) elastic.Query {
	if len(filter.Resources) == 0 {
		return nil
	}
	queries := []elastic.Query{}
	for _, res := range filter.Resources {
		queries = append(queries, dnsLogQueryFromDNSResource(res))
	}
	return elastic.NewBoolQuery().Should(queries...)
}

func dnsLogQueryFromDNSResource(res api.DNSResource) elastic.Query {
	queries := []elastic.Query{}
	if res.ClientNamespace != "" {
		queries = append(queries, elastic.NewMatchQuery(api.DNSLogClientNamespace, res.ClientNamespace))
	}
	if res.Qtype != "" {
		queries = append(queries, elastic.NewMatchQuery(api.DNSLogQtype, res.Qtype))
	}
	return elastic.NewBoolQuery().Must(queries...)
}
