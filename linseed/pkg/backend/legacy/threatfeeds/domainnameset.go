// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package threatfeeds

import (
	"context"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/projectcalico/go-json/json"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func NewDomainNameSetBackend(c lmaelastic.Client, cache bapi.Cache, deepPaginationCutOff int64) bapi.DomainNameSetBackend {
	return &domainNameSetThreatFeedBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
	}
}

type domainNameSetThreatFeedBackend struct {
	client               *elastic.Client
	templates            bapi.Cache
	lmaclient            lmaelastic.Client
	deepPaginationCutOff int64
}

func (b *domainNameSetThreatFeedBackend) Create(ctx context.Context, i bapi.ClusterInfo, feeds []v1.DomainNameSetThreatFeed) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.DomainNameSet, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Infof("Writing domain name set threat feeds data in bulk to alias %s", alias)

	// Build a bulk request using the provided threat feeds.
	bulk := b.client.Bulk()

	for _, f := range feeds {
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f.Data).Id(f.ID)
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing domain name sets threat feeds data: %s", err)
		return nil, fmt.Errorf("failed to write domain name sets threat feeds data: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("Threat feeds domain name sets bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

func (b *domainNameSetThreatFeedBackend) List(ctx context.Context, i bapi.ClusterInfo, params *v1.DomainNameSetThreatFeedParams) (*v1.List[v1.DomainNameSetThreatFeed], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(i, params)
	if err != nil {
		return nil, err
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	feeds := []v1.DomainNameSetThreatFeed{}
	for _, h := range results.Hits.Hits {
		feed := v1.DomainNameSetThreatFeedData{}
		err = json.Unmarshal(h.Source, &feed)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling threat feed")
			continue
		}
		domainNameSetFeed := v1.DomainNameSetThreatFeed{
			ID:          h.Id,
			Data:        &feed,
			SeqNumber:   h.SeqNo,
			PrimaryTerm: h.PrimaryTerm,
		}

		feeds = append(feeds, domainNameSetFeed)
	}

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index(i), results, b.deepPaginationCutOff, log)
	if err != nil {
		return nil, err
	}

	return &v1.List[v1.DomainNameSetThreatFeed]{
		Items:     feeds,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextAfterKey(params, startFrom, pitID, results, b.deepPaginationCutOff),
	}, nil
}

func (b *domainNameSetThreatFeedBackend) getSearch(i bapi.ClusterInfo, p *v1.DomainNameSetThreatFeedParams) (*elastic.SearchService, int, error) {
	if err := i.Valid(); err != nil {
		return nil, 0, err
	}

	// Get the startFrom param, if any.
	q, err := b.buildQuery(p)
	if err != nil {
		return nil, 0, err
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Size(p.GetMaxPageSize()).
		Query(q)

	// Configure pagination options
	var startFrom int
	query, startFrom, err = logtools.ConfigureCurrentPage(query, p, b.index(i))
	if err != nil {
		return nil, 0, err
	}

	query.Sort("created_at", true)

	return query, startFrom, nil
}

func (b *domainNameSetThreatFeedBackend) buildQuery(p *v1.DomainNameSetThreatFeedParams) (elastic.Query, error) {
	query := elastic.NewBoolQuery()
	if p.TimeRange != nil {
		unset := time.Time{}
		tr := elastic.NewRangeQuery("created_at")
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

	return query, nil
}

func (b *domainNameSetThreatFeedBackend) Delete(ctx context.Context, i bapi.ClusterInfo, feeds []v1.DomainNameSetThreatFeed) (*v1.BulkResponse, error) {
	if err := i.Valid(); err != nil {
		return nil, err
	}

	alias := b.writeAlias(i)

	// Build a bulk request using the provided feeds.
	bulk := b.client.Bulk()
	for _, feed := range feeds {
		req := elastic.NewBulkDeleteRequest().Index(alias).Id(feed.ID)
		bulk.Add(req)
	}

	// Send the bulk request. Wait for results to be refreshed before replying,
	// so that subsequent reads show consistent data.
	resp, err := bulk.Refresh("wait_for").Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete feeds: %s", err)
	}

	// Convert individual success / failure responses.
	del := []v1.BulkItem{}
	for _, i := range resp.Deleted() {
		bi := v1.BulkItem{ID: i.Id, Status: i.Status}
		del = append(del, bi)
	}

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
		Deleted:   del,
	}, nil
}

func (b *domainNameSetThreatFeedBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_threatfeeds_domainnameset.%s.%s.*", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_threatfeeds_domainnameset.%s.*", i.Cluster)
}

func (b *domainNameSetThreatFeedBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_threatfeeds_domainnameset.%s.%s.", i.Tenant, i.Cluster)
	}

	return fmt.Sprintf("tigera_secure_ee_threatfeeds_domainnameset.%s.", i.Cluster)
}
