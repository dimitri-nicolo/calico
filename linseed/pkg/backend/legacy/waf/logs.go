// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package waf

import (
	"context"
	"fmt"

	"github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type wafLogBackend struct {
	client    *elastic.Client
	helper    index.Helper
	lmaclient lmaelastic.Client
	templates bapi.Cache
}

func NewBackend(c lmaelastic.Client, cache bapi.Cache) bapi.WAFBackend {
	return &wafLogBackend{
		client:    c.Backend(),
		helper:    index.WAFLogs(),
		lmaclient: c,
		templates: cache,
	}
}

// Create the given logs in elasticsearch.
func (b *wafLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.WAFLog) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.WAFLogs, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Debugf("Writing WAF logs in bulk to alias %s", alias)

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
		log.Errorf("Error writing log: %s", err)
		return nil, fmt.Errorf("failed to write log: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("WAF log bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists logs that match the given parameters.
func (b *wafLogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.WAFLogParams) (*v1.List[v1.WAFLog], error) {
	log := bapi.ContextLogger(i)

	// Get the base query.
	search, startFrom, err := b.getSearch(ctx, i, opts)
	if err != nil {
		return nil, err
	}

	results, err := search.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.WAFLog{}
	for _, h := range results.Hits.Hits {
		l := v1.WAFLog{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling WAF log")
			continue
		}
		logs = append(logs, l)
	}

	return &v1.List[v1.WAFLog]{
		TotalHits: results.TotalHits(),
		Items:     logs,
		AfterKey:  logtools.NextStartFromAfterKey(opts, len(results.Hits.Hits), startFrom),
	}, nil
}

func (b *wafLogBackend) Aggregations(ctx context.Context, i api.ClusterInfo, opts *v1.WAFLogAggregationParams) (*elastic.Aggregations, error) {
	// Get the base query.
	search, _, err := b.getSearch(ctx, i, &opts.WAFLogParams)
	if err != nil {
		return nil, err
	}

	// Add in any aggregations provided by the client. We need to handle two cases - one where this is a
	// time-series request, and another when it's just an aggregation request.
	if opts.NumBuckets > 0 {
		// Time-series.
		hist := elastic.NewAutoDateHistogramAggregation().
			Field("@timestamp").
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

func (b *wafLogBackend) getSearch(ctx context.Context, i api.ClusterInfo, opts *v1.WAFLogParams) (*elastic.SearchService, int, error) {
	if err := i.Valid(); err != nil {
		return nil, 0, err
	}

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(opts)
	if err != nil {
		return nil, 0, err
	}

	// Build the query.
	q, err := b.buildQuery(i, opts)
	if err != nil {
		return nil, 0, err
	}
	query := b.client.Search().
		Index(b.index(i)).
		Size(opts.GetMaxPageSize()).
		From(startFrom).
		Query(q)

	// Configure sorting.
	if len(opts.Sort) != 0 {
		for _, s := range opts.Sort {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.Sort(b.helper.GetTimeField(), true)
	}
	return query, startFrom, nil
}

// buildQuery builds an elastic query using the given parameters.
func (b *wafLogBackend) buildQuery(i bapi.ClusterInfo, opts *v1.WAFLogParams) (elastic.Query, error) {
	// Start with the base flow log query using common fields.
	start, end := logtools.ExtractTimeRange(opts.GetTimeRange())
	query, err := logtools.BuildQuery(b.helper, i, opts, start, end)
	if err != nil {
		return nil, err
	}

	return query, nil
}

func (b *wafLogBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_waf.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_waf.%s.*", i.Cluster)
}

func (b *wafLogBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_waf.%s.%s.", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_waf.%s.", i.Cluster)
}
