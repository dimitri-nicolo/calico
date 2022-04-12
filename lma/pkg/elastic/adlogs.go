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

	api "github.com/projectcalico/calico/lma/pkg/api"
)

// Anomaly detection results indices.
// NOTE that inbound/outbound-service-bytes jobs share the same index.
const (
	PortScanJobIndex               = ".ml-anomalies-custom-port_scan_pods"
	InboundConnectionSpikeJobIndex = ".ml-anomalies-custom-inbound_connection_spike"
	IPSweepJobIndex                = ".ml-anomalies-custom-ip_sweep_pods"
	InboundServiceBytesJobIndex    = ".ml-anomalies-custom-service_bytes_anomaly"
	OutboundServiceBytesJobIndex   = ".ml-anomalies-custom-service_bytes_anomaly"
	DefaultADPageSize              = 100
)

func (c *client) GetADLogs(ctx context.Context, start, end *time.Time) <-chan *api.ADResult {
	return c.SearchADLogs(ctx, nil, start, end)
}

// Issue an Elasticsearch query that matches anomaly detection logs.
func (c *client) SearchADLogs(ctx context.Context, filter *api.ADLogsSelection, start, end *time.Time) <-chan *api.ADResult {
	resultChan := make(chan *api.ADResult, DefaultADPageSize)
	adIndices := getADJobIndices(c)

	// Create ES queries using given filters and time interval.
	queries := constructADLogsQuery(filter, start, end)

	go func() {
		defer close(resultChan)

		scroll := c.Scroll(adIndices...).
			Size(DefaultADPageSize).
			Query(queries).
			Sort(api.ADLogTimestamp, true)

		// Issue the query to Elasticsearch and send results out through the resultsChan.
		// We only terminate the search if when there are no more results to scroll through.
		for {
			log.Debug("Issuing anomaly detection search query")
			res, err := scroll.Do(ctx)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.WithError(err).Error("Failed to search anomaly detection logs")
				resultChan <- &api.ADResult{Err: err}
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
				log.WithError(err).Warn("Unexpected results from anomaly detection logs search")
				resultChan <- &api.ADResult{Err: err}
				return
			}
			log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

			// Push the search results into the channel.
			for _, hit := range res.Hits.Hits {
				var m map[string]interface{}
				if err := json.Unmarshal(hit.Source, &m); err != nil {
					log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("Failed to unmarshal event json")
					resultChan <- &api.ADResult{Err: fmt.Errorf("Failed to unmarshal event json")}
					continue
				}

				// Unmarshal the data into the correct log type.
				resultType, ok := m[api.ADLogResultType].(string)
				if !ok {
					log.Warn("Error getting anomaly detection result type field")
					resultChan <- &api.ADResult{Err: fmt.Errorf("Error getting anomaly detection result type field")}
					continue
				}
				switch resultType {
				case api.ADRecordResultType:
					var d api.ADRecordLog
					if err := json.Unmarshal(hit.Source, &d); err != nil {
						log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("Failed to unmarshal anomaly detection record json")
						resultChan <- &api.ADResult{Err: fmt.Errorf("Failed to unmarshal anomaly detection record json")}
						continue
					}
					resultChan <- &api.ADResult{ADRecordLog: &d}
				default:
					log.WithField("result_type", resultType).Debug("Cannot unmarshal anomaly detection json, unsupported type")
				}
			}
		}

		if err := scroll.Clear(ctx); err != nil {
			log.WithError(err).Info("Failed to clear scroll context")
		}
	}()

	return resultChan
}

func constructADLogsQuery(filter *api.ADLogsSelection, start, end *time.Time) elastic.Query {
	queries := []elastic.Query{}

	// Query by filter if specified.
	if filter != nil {
		queries = append(queries, adLogQueryFromADLogsSelection(filter))
	}

	// Query by from/to if specified.
	if start != nil || end != nil {
		rangeQuery := elastic.NewRangeQuery(api.ADLogTimestamp)
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

func adLogQueryFromADLogsSelection(filter *api.ADLogsSelection) elastic.Query {
	if len(filter.Resources) == 0 {
		return nil
	}
	queries := []elastic.Query{}
	for _, res := range filter.Resources {
		queries = append(queries, adLogQueryFromADResource(res))
	}
	return elastic.NewBoolQuery().Should(queries...)
}

func adLogQueryFromADResource(res api.ADResource) elastic.Query {
	queries := []elastic.Query{}
	if res.ResultType != "" {
		queries = append(queries, elastic.NewMatchQuery(api.ADLogResultType, res.ResultType))
	}
	if res.MinRecordScore != nil || res.MaxRecordScore != nil {
		q := elastic.NewRangeQuery(api.ADLogRecordScore)
		if res.MinRecordScore != nil {
			q = q.From(res.MinRecordScore)
		}
		if res.MaxRecordScore != nil {
			q = q.To(res.MaxRecordScore)
		}
		queries = append(queries, q)
	}
	if res.MinEventCount != nil {
		queries = append(queries, elastic.NewRangeQuery(api.ADLogEventCount).From(res.MinEventCount))
	}
	return elastic.NewBoolQuery().Must(queries...)
}

func getADJobIndices(c *client) []string {
	return []string{
		c.ClusterIndex(PortScanJobIndex, ""),
		c.ClusterIndex(InboundConnectionSpikeJobIndex, ""),
		c.ClusterIndex(IPSweepJobIndex, ""),
		c.ClusterIndex(InboundServiceBytesJobIndex, ""),
		c.ClusterIndex(OutboundServiceBytesJobIndex, ""),
	}
}
