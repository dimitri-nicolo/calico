package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"

	api "github.com/tigera/lma/pkg/api"
)

func (c *client) GetBenchmarks(cxt context.Context, id string) (*api.Benchmarks, error) {
	clog := log.WithField("id", id)

	searchIndex := c.ClusterIndex(BenchmarksIndex, "*")

	// Execute query.
	res, err := c.Search().
		Index(searchIndex).
		Query(elastic.NewTermQuery("_id", id)).
		Size(1). // Only retrieve the first document found.
		Do(cxt)
	if err != nil {
		clog.WithError(err).Error("failed to execute query")
		return nil, err
	}
	clog.WithField("latency (ms)", res.TookInMillis).Debug("query success")

	// Should only return one document.
	switch len(res.Hits.Hits) {
	case 0:
		clog.Error("no hits found")
		return nil, cerrors.ErrorResourceDoesNotExist{
			Identifier: id,
			Err:        errors.New("no benchmarks exist with the requested ID"),
		}
	case 1:
		break
	default:
		clog.WithField("hits", len(res.Hits.Hits)).
			Warn("expected to receive only one hit")
	}

	// Extract list from result.
	hit := res.Hits.Hits[0]
	b := new(api.Benchmarks)
	if err = json.Unmarshal(hit.Source, b); err != nil {
		clog.WithError(err).Error("failed to extract benchmarks from result")
		return nil, err
	}

	return b, nil

}

// StoreBenchmarks stores the supplied benchmarks.
func (c *client) StoreBenchmarks(ctx context.Context, b *api.Benchmarks) error {
	index := c.ClusterAlias(BenchmarksIndex)
	benchmarksTemplate, err := c.IndexTemplate(index, BenchmarksIndex, benchmarksMapping, true)
	if err != nil {
		log.WithError(err).Error("failed to build index template")
		return err
	}
	if err := c.ensureIndexExistsWithRetry(BenchmarksIndex, benchmarksTemplate, true); err != nil {
		return err
	}
	res, err := c.Index().
		Index(index).
		Id(b.UID()).
		BodyJson(b).
		Do(ctx)
	if err != nil {
		log.WithError(err).Error("failed to store benchmarks")
		return err
	}
	log.WithFields(log.Fields{"id": res.Id, "index": res.Index, "type": res.Type}).
		Info("successfully stored benchmarks")
	return nil
}

// RetrieveLatestBenchmarks returns the set of BenchmarkSetIDs within the time interval.
//
// Filters are OR'd together. Options within the filter are ANDed.
func (c *client) RetrieveLatestBenchmarks(ctx context.Context, ct api.BenchmarkType, filters []api.BenchmarkFilter, start, end time.Time) <-chan api.BenchmarksResult {
	ch := make(chan api.BenchmarksResult, DefaultPageSize)
	searchIndex := c.ClusterIndex(BenchmarksIndex, "*")

	// Keep track of the latest results set from each node.
	go func() {
		log.Debug("Starting benchmarks query")
		defer func() {
			log.Debug("Completed benchmarks query")
			close(ch)
		}()

		// Keep track of benchmarks we've already seen for the node.
		seen := make(map[string]*api.Benchmarks)

		// Make search query with scroll. We do a reverse timestamp sorted query so that we can send an update as soon
		// we get a non-errored set of benchmarks for a particular node.
		scroll := c.Scroll(searchIndex).
			Query(getBenchmarksQuery(ct, filters, start, end)).
			Sort("timestamp", false).
			Size(DefaultPageSize)
		for {
			log.Debug("Running scroll to return next set of Benchmarks")
			res, err := scroll.Do(context.Background())
			log.Debug("Scroll complete")
			if err == io.EOF {
				log.Debug("No more results from scroll")
				break
			}
			if err != nil {
				log.WithError(err).Warn("failed to search for audit events")
				ch <- api.BenchmarksResult{Err: err}
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
				log.WithError(err).Warn("Unexpected results from audit events search")
				ch <- api.BenchmarksResult{Err: err}
				return
			}
			log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

			// Push results into the channel as needed.  Note that we want the most recent successful result for
			// each node. If the benchmarks indicate an unsuccessful query then don't return it immediately because an
			// earlier run within the time frame may have been successful.
			for _, hit := range res.Hits.Hits {
				benchmarks := new(api.Benchmarks)
				if err := json.Unmarshal(hit.Source, benchmarks); err != nil {
					log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal benchmark result json")
					continue
				}
				if prev, ok := seen[benchmarks.NodeName]; ok {
					log.WithFields(
						log.Fields{
							"node":         benchmarks.NodeName,
							"previousTime": prev.Timestamp,
							"thisTime":     benchmarks.Timestamp,
						}).Debug("Found an earlier benchmark set for this node")

					if prev.Error == "" || benchmarks.Error != "" {
						// Either the previous entry did not indicate error, or this entry does indicate an error
						// in either case continue processing entries.
						continue
					}
				}

				// Either this is a new node, or this is the first non-errored entry for that node. Update our seen map
				// and if not errored send the update now.
				seen[benchmarks.NodeName] = benchmarks
				if benchmarks.Error == "" {
					log.WithFields(
						log.Fields{
							"node": benchmarks.NodeName,
							"time": benchmarks.Timestamp,
						}).Debug("Found latest successful benchmark set for this node")
					ch <- api.BenchmarksResult{Benchmarks: benchmarks}
				}
			}
		}

		// We have iterated through all sets. Any that contain an error, send those now since we were previously holding
		// off in case we found a non-errored set.
		for _, benchmarks := range seen {
			if benchmarks.Error != "" {
				log.WithFields(
					log.Fields{
						"node": benchmarks.NodeName,
						"time": benchmarks.Timestamp,
					}).Debug("Sending errored benchmark set for this node")
				ch <- api.BenchmarksResult{Benchmarks: benchmarks}
			}
		}

		if err := scroll.Clear(context.Background()); err != nil {
			log.WithError(err).Info("Failed to clear scroll context")
		}
	}()

	return ch
}

// getBenchmarksQuery calculates the query for a set of benchmarks.
func getBenchmarksQuery(ct api.BenchmarkType, filters []api.BenchmarkFilter, start, end time.Time) elastic.Query {
	// Limit query to include ResponseComplete stage only since that has that has the most information, and only
	// to the configuration event verb types.
	queries := []elastic.Query{
		elastic.NewMatchQuery("type", ct),
		elastic.NewRangeQuery("timestamp").From(start).To(end),
	}

	// Query by filter if specified.
	if len(filters) != 0 {
		queries = append(queries, getBenchmarksFiltersQuery(filters))
	}

	return elastic.NewBoolQuery().Must(queries...)
}

// getBenchmarksFiltersQuery calculates the query for the set of benchmark filters.
func getBenchmarksFiltersQuery(filters []api.BenchmarkFilter) elastic.Query {
	queries := []elastic.Query{}
	for _, filter := range filters {
		queries = append(queries, getBenchmarksFilterQuery(filter))
	}
	return elastic.NewBoolQuery().Should(queries...)
}

// getBenchmarksFilterQuery calculates the query for a single benchmark filters.
func getBenchmarksFilterQuery(filter api.BenchmarkFilter) elastic.Query {
	queries := []elastic.Query{}
	if filter.Version != "" {
		queries = append(queries, elastic.NewMatchQuery("version", filter.Version))
	}
	if len(filter.NodeNames) != 0 {
		queries = append(queries, getAnyStringValueQuery("node_name", filter.NodeNames))
	}
	return elastic.NewBoolQuery().Must(queries...)
}
