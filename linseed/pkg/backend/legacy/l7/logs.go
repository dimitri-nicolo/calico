// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l7

import (
	"context"
	"fmt"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type l7LogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
	helper    lmaindex.Helper
	templates bapi.Cache
}

func NewL7LogBackend(c lmaelastic.Client, cache bapi.Cache) bapi.L7LogBackend {
	b := &l7LogBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
		helper:    lmaindex.L7Logs(),
	}
	return b
}

// Create the given log in elasticsearch.
func (b *l7LogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.L7Log) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.L7Logs, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Debugf("Writing L7 logs in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f)
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing L7 log: %s", err)
		return nil, fmt.Errorf("failed to write L7 log: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("L7 log bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

func (b *l7LogBackend) Aggregations(ctx context.Context, i api.ClusterInfo, opts *v1.L7AggregationParams) (*elastic.Aggregations, error) {
	// Get the base query.
	search, _, err := b.getSearch(ctx, i, &opts.L7LogParams)
	if err != nil {
		return nil, err
	}

	// Add in any aggregations provided by the client. We need to handle two cases - one where this is a
	// time-series request, and another when it's just an aggregation request.
	if opts.NumBuckets > 0 {
		// Time-series.
		hist := elastic.NewAutoDateHistogramAggregation().
			Field(b.helper.GetTimeField()).
			Buckets(opts.NumBuckets)
		for name, agg := range opts.Aggregations {
			hist = hist.SubAggregation(name, logtools.RawAggregation{RawMessage: agg})
		}
		search.Aggregation(v1.TimeSeriesBucketName, hist)
	} else {
		// Not time-series. Just add the aggs as they are.
		for name, agg := range opts.Aggregations {
			search = search.Aggregation(name, logtools.RawAggregation{RawMessage: agg})
		}
	}

	// Do the search.
	results, err := search.Do(ctx)
	if err != nil {
		return nil, err
	}

	return &results.Aggregations, nil
}

// List lists logs that match the given parameters.
func (b *l7LogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.L7LogParams) (*v1.List[v1.L7Log], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(ctx, i, opts)
	if err != nil {
		return nil, err
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.L7Log{}
	for _, h := range results.Hits.Hits {
		l := v1.L7Log{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling log")
			continue
		}
		logs = append(logs, l)
	}

	return &v1.List[v1.L7Log]{
		Items:     logs,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextStartFromAfterKey(opts, len(results.Hits.Hits), startFrom),
	}, nil
}

func (b *l7LogBackend) getSearch(ctx context.Context, i api.ClusterInfo, opts *v1.L7LogParams) (*elastic.SearchService, int, error) {
	if i.Cluster == "" {
		return nil, 0, fmt.Errorf("no cluster ID on request")
	}

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(opts)
	if err != nil {
		return nil, 0, err
	}

	start, end := logtools.ExtractTimeRange(opts.QueryParams.TimeRange)
	q, err := logtools.BuildQuery(b.helper, i, opts.LogSelectionParams, start, end)
	if err != nil {
		return nil, 0, err
	}

	// Build the query.
	query := b.client.Search().
		Index(b.index(i)).
		Size(opts.QueryParams.GetMaxPageSize()).
		From(startFrom).
		Query(q)

	// Configure sorting.
	if len(opts.GetSortBy()) != 0 {
		for _, s := range opts.GetSortBy() {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.Sort(b.helper.GetTimeField(), true)
	}
	return query, startFrom, nil
}

func (b *l7LogBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_l7.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_l7.%s.*", i.Cluster)
}

func (b *l7LogBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_l7.%s.%s.", i.Tenant, i.Cluster)
	}

	return fmt.Sprintf("tigera_secure_ee_l7.%s.", i.Cluster)
}
