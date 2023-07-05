// Copyright (c) 2023 Tigera All rights reserved.

package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func NewBenchmarksBackend(c lmaelastic.Client, cache bapi.Cache, deepPaginationCutOff int64) bapi.BenchmarksBackend {
	return &benchmarksBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
	}
}

type benchmarksBackend struct {
	client               *elastic.Client
	templates            bapi.Cache
	lmaclient            lmaelastic.Client
	deepPaginationCutOff int64
}

func (b *benchmarksBackend) List(ctx context.Context, i bapi.ClusterInfo, opts *v1.BenchmarksParams) (*v1.List[v1.Benchmarks], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(i, opts)
	if err != nil {
		return nil, err
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.Benchmarks{}
	for _, h := range results.Hits.Hits {
		l := v1.Benchmarks{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling log")
			continue
		}
		l.ID = h.Id
		logs = append(logs, l)
	}

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index(i), results, b.deepPaginationCutOff, log)
	if err != nil {
		return nil, err
	}

	return &v1.List[v1.Benchmarks]{
		Items:     logs,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextAfterKey(opts, startFrom, pitID, results, b.deepPaginationCutOff),
	}, nil
}

func (b *benchmarksBackend) Create(ctx context.Context, i bapi.ClusterInfo, l []v1.Benchmarks) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.Benchmarks, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Infof("Writing benchmarks data in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range l {
		// Add this log to the bulk request. Use the given ID, but
		// remove it from the document body.
		id := f.ID
		f.ID = ""
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f).Id(id)
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing benchmarks data: %s", err)
		return nil, fmt.Errorf("failed to write benchmarks data: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("Compliance benchmarks bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

func (b *benchmarksBackend) getSearch(i bapi.ClusterInfo, opts *v1.BenchmarksParams) (*elastic.SearchService, int, error) {
	if err := i.Valid(); err != nil {
		return nil, 0, err
	}

	q, err := b.buildQuery(opts)
	if err != nil {
		return nil, 0, err
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Size(opts.GetMaxPageSize()).
		Query(q)

	// Configure pagination options
	var startFrom int
	query, startFrom, err = logtools.ConfigureCurrentPage(query, opts, b.index(i))
	if err != nil {
		return nil, 0, err
	}

	// Configure sorting.
	if len(opts.Sort) != 0 {
		for _, s := range opts.Sort {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.Sort("timestamp", false)
	}
	return query, startFrom, nil
}

func (b *benchmarksBackend) buildQuery(p *v1.BenchmarksParams) (elastic.Query, error) {
	query := elastic.NewBoolQuery()
	if p.TimeRange != nil {
		unset := time.Time{}
		tr := elastic.NewRangeQuery("timestamp")
		if p.TimeRange.From != unset {
			tr.From(p.TimeRange.From)
		}
		if p.TimeRange.To != unset {
			tr.To(p.TimeRange.To)
		}
		query.Must(tr)
	}
	if p.ID != "" {
		query.Must(elastic.NewTermQuery("_id", p.ID))
	}
	if p.Type != "" {
		query.Must(elastic.NewMatchQuery("type", p.Type))
	}

	if len(p.Filters) > 0 {
		// We combine the filters with a logical OR.
		filterQueries := []elastic.Query{}
		for _, filter := range p.Filters {
			fq := elastic.NewBoolQuery()
			if filter.Version != "" {
				fq.Must(elastic.NewMatchQuery("version", filter.Version))
			}
			if len(filter.NodeNames) != 0 {
				fq.Must(getAnyStringValueQuery("node_name", filter.NodeNames))
			}
			filterQueries = append(filterQueries, fq)
		}
		query.Should(filterQueries...)
	}

	return query, nil
}

func (b *benchmarksBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_benchmark_results.%s.%s.*", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_benchmark_results.%s.*", i.Cluster)
}

func (b *benchmarksBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_benchmark_results.%s.%s.", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_benchmark_results.%s.", i.Cluster)
}

// getAnyStringValueQuery calculates the query for a specific string field to match one of the supplied values.
func getAnyStringValueQuery(field string, vals []string) elastic.Query {
	queries := []elastic.Query{}
	for _, val := range vals {
		queries = append(queries, elastic.NewMatchQuery(field, val))
	}
	return elastic.NewBoolQuery().Should(queries...)
}
