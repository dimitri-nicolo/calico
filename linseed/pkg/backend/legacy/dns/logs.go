// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns

import (
	"context"
	"fmt"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/olivere/elastic/v7"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type dnsLogBackend struct {
	client               *elastic.Client
	lmaclient            lmaelastic.Client
	helper               lmaindex.Helper
	templates            bapi.Cache
	deepPaginationCutOff int64
}

func NewDNSLogBackend(c lmaelastic.Client, cache bapi.Cache, deepPaginationCutOff int64) bapi.DNSLogBackend {
	return &dnsLogBackend{
		client:               c.Backend(),
		lmaclient:            c,
		helper:               lmaindex.DnsLogs(),
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
	}
}

func (b *dnsLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.DNSLog) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.DNSLogs, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Debugf("Writing DNS logs in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		// Add this log to the bulk request.
		dnsLog, err := json.Marshal(f)
		if err != nil {
			log.WithError(err).Warningf("Failed to marshal dns log and add it to the request %+v", f)
			continue
		}
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(string(dnsLog))
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing DNS log: %s", err)
		return nil, fmt.Errorf("failed to write DNS log: %s", err)
	}
	log.WithField("count", len(logs)).Debugf("Wrote DNS log to index: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

func (b *dnsLogBackend) Aggregations(ctx context.Context, i api.ClusterInfo, opts *v1.DNSAggregationParams) (*elastic.Aggregations, error) {
	// Get the base query.
	search, _, err := b.getSearch(i, &opts.DNSLogParams)
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
func (b *dnsLogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.DNSLogParams) (*v1.List[v1.DNSLog], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(i, opts)
	if err != nil {
		return nil, err
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.DNSLog{}
	for _, h := range results.Hits.Hits {
		l := v1.DNSLog{}
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

	return &v1.List[v1.DNSLog]{
		Items:     logs,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextAfterKey(opts, startFrom, pitID, results, b.deepPaginationCutOff),
	}, nil
}

func (b *dnsLogBackend) getSearch(i bapi.ClusterInfo, opts *v1.DNSLogParams) (*elastic.SearchService, int, error) {
	if i.Cluster == "" {
		return nil, 0, fmt.Errorf("no cluster ID on request")
	}

	q, err := b.buildQuery(i, opts)
	if err != nil {
		return nil, 0, err
	}

	// Build the query.
	query := b.client.Search().
		Size(opts.QueryParams.GetMaxPageSize()).
		Query(q)

	// Configure pagination options
	var startFrom int
	query, startFrom, err = logtools.ConfigureCurrentPage(query, opts, b.index(i))
	if err != nil {
		return nil, 0, err
	}

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

// buildQuery builds an elastic query using the given parameters.
func (b *dnsLogBackend) buildQuery(i bapi.ClusterInfo, opts *v1.DNSLogParams) (elastic.Query, error) {
	// Start with the base dns log query using common fields.
	start, end := logtools.ExtractTimeRange(opts.GetTimeRange())
	query, err := logtools.BuildQuery(b.helper, i, opts, start, end)
	if err != nil {
		return nil, err
	}

	if len(opts.DomainMatches) > 0 {
		for _, match := range opts.DomainMatches {
			// Get the list of values as an interface{}, as needed for a terms query.
			values := []interface{}{}
			for _, t := range match.Domains {
				values = append(values, t)
			}

			switch match.Type {
			case v1.DomainMatchQname:
				query.Filter(elastic.NewTermsQuery("qname", values...))
			case v1.DomainMatchRRSet:
				query.Filter(elastic.NewNestedQuery("rrsets", elastic.NewTermsQuery("rrsets.name", values...)))
			case v1.DomainMatchRRData:
				query.Filter(elastic.NewNestedQuery("rrsets", elastic.NewTermsQuery("rrsets.rdata", values...)))
			default:
				query.Filter(elastic.NewBoolQuery().Should(
					elastic.NewTermsQuery("qname", values...),
					elastic.NewNestedQuery("rrsets", elastic.NewTermsQuery("rrsets.name", values...)),
					elastic.NewNestedQuery("rrsets", elastic.NewTermsQuery("rrsets.rdata", values...)),
				).MinimumNumberShouldMatch(1))
			}
		}
	}

	return query, nil
}

func (b *dnsLogBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_dns.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_dns.%s.*", i.Cluster)
}

func (b *dnsLogBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_dns.%s.%s.", i.Tenant, i.Cluster)
	}

	return fmt.Sprintf("tigera_secure_ee_dns.%s.", i.Cluster)
}
