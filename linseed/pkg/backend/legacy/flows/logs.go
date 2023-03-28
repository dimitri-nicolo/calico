// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows

import (
	"context"
	"fmt"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
)

type flowLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
	helper    lmaindex.Helper
	templates bapi.Cache
}

func NewFlowLogBackend(c lmaelastic.Client, cache bapi.Cache) bapi.FlowLogBackend {
	return &flowLogBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
		helper:    lmaindex.FlowLogs(),
	}
}

// Create the given flow log in elasticsearch.
func (b *flowLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.FlowLog) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("No cluster ID on request")
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.FlowLogs, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Infof("Writing flow logs in bulk to alias %s", alias)

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
		log.Errorf("Error writing flow log: %s", err)
		return nil, fmt.Errorf("failed to write flow log: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("Flow log bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists logs that match the given parameters.
func (b *flowLogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.FlowLogParams) (*v1.List[v1.FlowLog], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(ctx, i, opts)
	if err != nil {
		return nil, err
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.FlowLog{}
	for _, h := range results.Hits.Hits {
		l := v1.FlowLog{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling log")
			continue
		}
		l.ID = h.Id
		logs = append(logs, l)
	}

	return &v1.List[v1.FlowLog]{
		Items:     logs,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextStartFromAfterKey(opts, len(results.Hits.Hits), startFrom),
	}, nil
}

func (b *flowLogBackend) Aggregations(ctx context.Context, i api.ClusterInfo, opts *v1.FlowLogAggregationParams) (*elastic.Aggregations, error) {
	// Get the base query.
	search, _, err := b.getSearch(ctx, i, &opts.FlowLogParams)
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

func (b *flowLogBackend) getSearch(ctx context.Context, i api.ClusterInfo, opts *v1.FlowLogParams) (*elastic.SearchService, int, error) {
	if i.Cluster == "" {
		return nil, 0, fmt.Errorf("no cluster ID on request")
	}

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(opts)
	if err != nil {
		return nil, 0, err
	}

	q, err := b.buildQuery(i, opts)
	if err != nil {
		return nil, 0, err
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Index(b.index(i)).
		Size(opts.QueryParams.GetMaxPageSize()).
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
func (b *flowLogBackend) buildQuery(i bapi.ClusterInfo, opts *v1.FlowLogParams) (elastic.Query, error) {
	// Start with the base flow log query using common fields.
	query, err := logtools.BuildQuery(b.helper, i, opts)
	if err != nil {
		return nil, err
	}

	if len(opts.IPMatches) > 0 {
		for _, match := range opts.IPMatches {
			// Get the list of values as an interface{}, as needed for a terms query.
			values := []interface{}{}
			for _, t := range match.IPs {
				values = append(values, t)
			}

			switch match.Type {
			case v1.MatchTypeSource:
				query.Filter(elastic.NewTermsQuery("source_ip", values...))
			case v1.MatchTypeDest:
				query.Filter(elastic.NewTermsQuery("dest_ip", values...))
			case v1.MatchTypeAny:
				fallthrough
			default:
				// By default, treat as an "any" match. Return any flows that have a source
				// or destination name that matches.
				query.Filter(elastic.NewBoolQuery().Should(
					elastic.NewTermsQuery("source_ip", values...),
					elastic.NewTermsQuery("dest_ip", values...),
				).MinimumNumberShouldMatch(1))
			}
		}
	}

	return query, nil
}

func (b *flowLogBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_flows.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_flows.%s.*", i.Cluster)
}

func (b *flowLogBackend) writeAlias(i bapi.ClusterInfo) string {
	return fmt.Sprintf("tigera_secure_ee_flows.%s.", i.Cluster)
}
