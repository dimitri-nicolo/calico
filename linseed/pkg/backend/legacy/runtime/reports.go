// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type runtimeReportBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	templates            bapi.IndexInitializer
	deepPaginationCutOff int64

	queryHelper lmaindex.Helper
	singleIndex bool
	index       bapi.Index
}

func NewBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64) bapi.RuntimeBackend {
	return &runtimeReportBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		index:                index.RuntimeReportMultiIndex,
		singleIndex:          false,
		queryHelper:          lmaindex.MultiIndexRuntimeReports(),
	}
}

func NewSingleIndexBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64, options ...index.Option) bapi.RuntimeBackend {
	return &runtimeReportBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		index:                index.RuntimeReportsIndex(options...),
		singleIndex:          true,
		queryHelper:          lmaindex.SingleIndexRuntimeReports(),
	}
}

type logWithExtras struct {
	v1.Report `json:",inline"`
	Tenant    string `json:"tenant,omitempty"`
}

// prepareForWrite wraps a log in a document that includes the cluster and tenant if
// the backend is configured to write to a single index.
func (b *runtimeReportBackend) prepareForWrite(i bapi.ClusterInfo, l v1.Report) interface{} {
	l.Cluster = i.Cluster

	if b.singleIndex {
		return &logWithExtras{
			Report: l,
			Tenant: i.Tenant,
		}
	}
	return l
}

// Create the given reports in elasticsearch.
func (b *runtimeReportBackend) Create(ctx context.Context, i bapi.ClusterInfo, reports []v1.Report) (*v1.BulkResponse, error) {
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
	log.Debugf("Writing runtime reports in bulk to alias %s", alias)

	// Build a bulk request using the provided reports.
	bulk := b.client.Bulk()

	for _, f := range reports {
		// Populate the report's GeneratedTime field.  This field exists purely to enable a
		// way for clients to efficiently query newly generated reports, and having Linseed
		// fill it in - instead of Skimble as previously, or overwriting the Skimble value -
		// is just the last step in making that really robust.  (Because with Skimble
		// populating it we are still vulnerable to time skew between the nodes, and/or
		// differing latencies (2), (3) from Skimble to Linseed; Linseed population
		// eliminates those problems.)  Please note that if someone wants to know when a
		// report really happened, we have the StartTime and EndTime fields for that
		// purpose.
		//
		// Why not compute `time.Now().UTC()` before this loop and then store the same value
		// in each report?  Because if we can advance GeneratedTime as much as possible
		// (i.e. to the real current time), a client will compute a later value for
		// `LatestSeenGeneratedTime`, and then a subsequent query with `GeneratedTime >=
		// LatestSeenGeneratedTime` will return fewer previously seen results.
		generatedTime := time.Now().UTC()
		f.GeneratedTime = &generatedTime

		// If there were any fields that we did not want to store in Elastic, we would reset
		// them here.  But currently there are not.

		// Add this report to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(b.prepareForWrite(i, f))
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

	// Build the query.
	q, err := b.buildQuery(i, opts)
	if err != nil {
		return nil, err
	}
	query := b.client.Search().
		Size(opts.GetMaxPageSize()).
		Query(q)

	// Configure pagination options
	var startFrom int
	query, startFrom, err = logtools.ConfigureCurrentPage(query, opts, b.index.Index(i))
	if err != nil {
		return nil, err
	}

	// Configure sorting.
	if len(opts.GetSortBy()) != 0 {
		for _, s := range opts.GetSortBy() {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.SortBy(elastic.NewFieldSort("start_time").Order(true), elastic.SortByDoc{})
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	reports := []v1.RuntimeReport{}
	for _, h := range results.Hits.Hits {
		l := logWithExtras{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling runtime report")
			continue
		}

		// Populate the runtime report with tenant and cluster information. In single-index mode, this is stored
		// directly in the document. In multi-index mode, we need to extract it from the index name.
		cluster := l.Cluster
		tenant := l.Tenant
		if !b.singleIndex {
			tenant, cluster = b.extractTenantAndCluster(h.Index)
		}
		reports = append(reports, v1.RuntimeReport{ID: h.Id, Tenant: tenant, Cluster: cluster, Report: l.Report})
	}

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index.Index(i), results, b.deepPaginationCutOff, log)
	if err != nil {
		return nil, err
	}

	return &v1.List[v1.RuntimeReport]{
		TotalHits: results.TotalHits(),
		Items:     reports,
		AfterKey:  logtools.NextAfterKey(opts, startFrom, pitID, results, b.deepPaginationCutOff),
	}, nil
}

// buildQuery builds an elastic query using the given parameters.
func (b *runtimeReportBackend) buildQuery(i bapi.ClusterInfo, opts *v1.RuntimeReportParams) (elastic.Query, error) {
	query, err := b.queryHelper.BaseQuery(i, opts)
	if err != nil {
		return nil, err
	}

	tr := logtools.WithDefaultUntilNow(opts.GetTimeRange())
	queryTimeRange := elastic.NewBoolQuery().Must(elastic.NewRangeQuery("generated_time").From(tr.From))
	query.Must(queryTimeRange)

	// If a selector was provided, parse it and add it in.
	if sel := opts.Selector; len(sel) > 0 {
		selQuery, err := b.queryHelper.NewSelectorQuery(sel)
		if err != nil {
			return nil, err
		}
		if selQuery != nil {
			query.Must(selQuery)
		}
	}

	return query, nil
}

// extractTenantAndCluster extracts tenant and cluster from the given index name. This is needed in multi-index mode
// where the documents themselves do not contain this information. For single index mode, the tenant and cluster are
// already populated in the documents at write-time.
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
