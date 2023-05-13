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
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func NewReportsBackend(c lmaelastic.Client, cache bapi.Cache) bapi.ReportsBackend {
	return &reportsBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
	}
}

type reportsBackend struct {
	client    *elastic.Client
	templates bapi.Cache
	lmaclient lmaelastic.Client
}

func (b *reportsBackend) List(ctx context.Context, i bapi.ClusterInfo, p *v1.ReportDataParams) (*v1.List[v1.ReportData], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(ctx, i, p)
	if err != nil {
		return nil, err
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.ReportData{}
	for _, h := range results.Hits.Hits {
		l := v1.ReportData{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling log")
			continue
		}
		l.ID = h.Id
		logs = append(logs, l)
	}

	return &v1.List[v1.ReportData]{
		Items:     logs,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextStartFromAfterKey(p, len(results.Hits.Hits), startFrom, results.TotalHits()),
	}, nil
}

func (b *reportsBackend) Create(ctx context.Context, i bapi.ClusterInfo, l []v1.ReportData) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.ReportData, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Infof("Writing report data in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range l {
		// Add this log to the bulk request. Set the ID, and remove it from the
		// body of the document.
		id := f.ID
		f.ID = ""
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f).Id(id)
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing report data: %s", err)
		return nil, fmt.Errorf("failed to write report data: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("Compliance report bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

func (b *reportsBackend) getSearch(ctx context.Context, i api.ClusterInfo, p *v1.ReportDataParams) (*elastic.SearchService, int, error) {
	if err := i.Valid(); err != nil {
		return nil, 0, err
	}

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(p)
	if err != nil {
		return nil, 0, err
	}

	q := b.buildQuery(p)

	// Build the query, sorting by time.
	query := b.client.Search().
		Index(b.index(i)).
		Size(p.GetMaxPageSize()).
		From(startFrom).
		Query(q)

	// Configure sorting.
	if len(p.Sort) != 0 {
		for _, s := range p.Sort {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.Sort("endTime", true)
	}
	return query, startFrom, nil
}

func (b *reportsBackend) buildQuery(p *v1.ReportDataParams) elastic.Query {
	query := elastic.NewBoolQuery()
	if p.TimeRange != nil {
		unset := time.Time{}
		if p.TimeRange.From != unset && p.TimeRange.To != unset {
			query.Must(elastic.NewBoolQuery().Should(
				elastic.NewRangeQuery("startTime").From(p.TimeRange.From).To(p.TimeRange.To),
				elastic.NewRangeQuery("endTime").From(p.TimeRange.From).To(p.TimeRange.To),
			))
		} else if p.TimeRange.From != unset && p.TimeRange.To == unset {
			query.Must(elastic.NewRangeQuery("endTime").From(p.TimeRange.From))
		} else if p.TimeRange.From == unset && p.TimeRange.To != unset {
			query.Must(elastic.NewRangeQuery("startTime").To(p.TimeRange.To))
		}
	}
	if p.ID != "" {
		query.Must(elastic.NewTermQuery("_id", p.ID))
	}
	if len(p.ReportMatches) > 0 {
		rqueries := []elastic.Query{}
		for _, r := range p.ReportMatches {
			if r.ReportName != "" && r.ReportTypeName != "" {
				rqueries = append(rqueries, elastic.NewBoolQuery().Must(
					elastic.NewMatchQuery("reportTypeName", r.ReportTypeName),
					elastic.NewMatchQuery("reportName", r.ReportName),
				))
			} else if r.ReportName == "" && r.ReportTypeName != "" {
				rqueries = append(rqueries, elastic.NewMatchQuery("reportTypeName", r.ReportTypeName))
			} else if r.ReportName != "" && r.ReportTypeName == "" {
				rqueries = append(rqueries, elastic.NewMatchQuery("reportName", r.ReportName))
			}
		}
		if len(rqueries) > 0 {
			// Must match at least one of the given report matches.
			query.Must(elastic.NewBoolQuery().Should(rqueries...))
		}
	}

	return query
}

func (b *reportsBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_compliance_reports.%s.%s.*", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_compliance_reports.%s.*", i.Cluster)
}

func (b *reportsBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_compliance_reports.%s.%s.", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_compliance_reports.%s.", i.Cluster)
}
