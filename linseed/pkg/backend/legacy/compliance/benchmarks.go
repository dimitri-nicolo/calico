// Copyright (c) 2023 Tigera All rights reserved.

package compliance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func NewBenchmarksBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64) bapi.BenchmarksBackend {
	return &benchmarksBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		singleIndex:          false,
		index:                index.ComplianceBenchmarkMultiIndex,
		queryHelper:          lmaindex.MultiIndexBenchmarks(),
	}
}

func NewSingleIndexBenchmarksBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64, options ...index.Option) bapi.BenchmarksBackend {
	return &benchmarksBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		singleIndex:          true,
		index:                index.ComplianceBenchmarksIndex(options...),
		queryHelper:          lmaindex.SingleIndexBenchmarks(),
	}
}

type benchmarksBackend struct {
	client               *elastic.Client
	templates            bapi.IndexInitializer
	lmaclient            lmaelastic.Client
	deepPaginationCutOff int64
	queryHelper          lmaindex.Helper
	singleIndex          bool
	index                api.Index
}

type benchmarkWithExtras struct {
	v1.Benchmarks `json:",inline"`
	Cluster       string `json:"cluster"`
	Tenant        string `json:"tenant,omitempty"`
}

// prepareForWrite wraps a log in a document that includes the cluster and tenant if
// the backend is configured to write to a single index.
func (b *benchmarksBackend) prepareForWrite(i bapi.ClusterInfo, l v1.Benchmarks) interface{} {
	if b.singleIndex {
		return &benchmarkWithExtras{
			Benchmarks: l,
			Cluster:    i.Cluster,
			Tenant:     i.Tenant,
		}
	}
	return l
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
		l.ID = backend.ToApplicationID(b.singleIndex, h.Id, i)
		logs = append(logs, l)
	}

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index.Index(i), results, b.deepPaginationCutOff, log)
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

	err := b.templates.Initialize(ctx, b.index, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.index.Alias(i)
	log.Infof("Writing benchmarks data in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range l {
		// Add this log to the bulk request. Use the given ID, but remove it from the document body.
		id := backend.ToElasticID(b.singleIndex, f.ID, i)
		f.ID = ""
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(b.prepareForWrite(i, f)).Id(id)
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

	q, err := b.buildQuery(i, opts)
	if err != nil {
		return nil, 0, err
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Size(opts.GetMaxPageSize()).
		Query(q)

	// Configure pagination options
	var startFrom int
	query, startFrom, err = logtools.ConfigureCurrentPage(query, opts, b.index.Index(i))
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

func (b *benchmarksBackend) buildQuery(i bapi.ClusterInfo, p *v1.BenchmarksParams) (elastic.Query, error) {
	query := b.queryHelper.BaseQuery(i)

	if p.TimeRange != nil {
		query.Must(b.queryHelper.NewTimeRangeQuery(p.TimeRange))
	}
	if p.ID != "" {
		query.Must(elastic.NewTermQuery("_id", backend.ToElasticID(b.singleIndex, p.ID, i)))
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
		query.Should(filterQueries...).MinimumNumberShouldMatch(1)
	}
	return query, nil
}

// getAnyStringValueQuery calculates the query for a specific string field to match one of the supplied values.
func getAnyStringValueQuery(field string, vals []string) elastic.Query {
	queries := []elastic.Query{}
	for _, val := range vals {
		queries = append(queries, elastic.NewMatchQuery(field, val))
	}
	return elastic.NewBoolQuery().Should(queries...)
}
