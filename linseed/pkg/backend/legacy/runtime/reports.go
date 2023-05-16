// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type runtimeReportBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	templates bapi.Cache
}

func NewBackend(c lmaelastic.Client, cache bapi.Cache) bapi.RuntimeBackend {
	return &runtimeReportBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
	}
}

// Create the given reports in elasticsearch.
func (b *runtimeReportBackend) Create(ctx context.Context, i bapi.ClusterInfo, reports []v1.Report) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.RuntimeReports, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Debugf("Writing runtime reports in bulk to alias %s", alias)

	// Build a bulk request using the provided reports.
	bulk := b.client.Bulk()

	for _, f := range reports {
		// Reset fields that we do not want to store in Elastic
		// Add this report to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f)
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing report: %s", err)
		return nil, fmt.Errorf("failed to write report: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("RuntimeReports report bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists reports that match the given parameters.
func (b *runtimeReportBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.RuntimeReportParams) (*v1.List[v1.RuntimeReport], error) {
	log := bapi.ContextLogger(i)

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(&opts.QueryParams)
	if err != nil {
		return nil, err
	}

	// Build the query.
	query := b.client.Search().
		Index(b.index(i)).
		Size(opts.GetMaxPageSize()).
		From(startFrom).
		Query(b.buildQuery(i, opts))

	// Configure sorting.
	if len(opts.GetSortBy()) != 0 {
		for _, s := range opts.GetSortBy() {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.Sort("start_time", true)
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	reports := []v1.RuntimeReport{}
	for _, h := range results.Hits.Hits {
		l := v1.Report{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling runtime report")
			continue
		}
		// Populate the runtime report with the ID extracted from Elastic
		id := h.Id

		// Populate the runtime report with tenant and cluster extract from Elastic index
		tenant, cluster := b.extractTenantAndCluster(h.Index)
		reports = append(reports, v1.RuntimeReport{ID: id, Tenant: tenant, Cluster: cluster, Report: l})
	}

	return &v1.List[v1.RuntimeReport]{
		TotalHits: results.TotalHits(),
		Items:     reports,
		AfterKey:  logtools.NextStartFromAfterKey(opts, len(results.Hits.Hits), startFrom, results.TotalHits()),
	}, nil
}

// buildQuery builds an elastic query using the given parameters.
func (b *runtimeReportBackend) buildQuery(i bapi.ClusterInfo, opts *v1.RuntimeReportParams) elastic.Query {
	start, _ := logtools.ExtractTimeRange(opts.GetTimeRange())
	queryTimeRange := elastic.NewBoolQuery().Must(elastic.NewRangeQuery("generated_time").From(start))

	if opts.LegacyTimeRange != nil {
		logrus.Infof("Legacy time range declared")
		legacyStart, _ := logtools.ExtractTimeRange(opts.LegacyTimeRange)

		queryLegacy := elastic.NewBoolQuery().Must(elastic.NewRangeQuery("start_time").From(legacyStart))

		// this query forces presence of the `generated_time` field
		generatedTimeQuery := elastic.NewExistsQuery("generated_time")

		// combining all above queries into one ES query
		return elastic.NewBoolQuery().Should(
			queryTimeRange.Must(generatedTimeQuery),
			queryLegacy.MustNot(generatedTimeQuery)).MinimumShouldMatch("1")
	}

	return queryTimeRange
}

func (b *runtimeReportBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		// Read all the clusters associated with a tenant
		return fmt.Sprintf("tigera_secure_ee_runtime.%s.*", i.Tenant)
	}

	// Otherwise, this is a single-tenant cluster and we read data for all clusters
	return "tigera_secure_ee_runtime.*"
}

func (b *runtimeReportBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_runtime.%s.%s.", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_runtime.%s.", i.Cluster)
}

func (b *runtimeReportBackend) extractTenantAndCluster(index string) (tenant string, cluster string) {
	parts := strings.Split(index, ".")

	// This is an index that contains in its name tenant id & managed cluster
	// Format "tigera_secure_ee_runtime.tenant.cluster.fluentd-timestamp-0001"
	if len(parts) == 4 {
		return parts[1], parts[2]
	} else {
		// This is an index that contains in its name only managed cluster
		// Format "tigera_secure_ee_runtime.cluster.fluentd-timestamp-0001"
		if len(parts) == 3 {
			return "", parts[1]
		}
	}

	return "", ""
}
