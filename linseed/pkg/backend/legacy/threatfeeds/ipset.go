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

func NewIPSetBackend(c lmaelastic.Client, cache bapi.Cache) bapi.IPSetBackend {
	return &ipSetThreatFeedBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
	}
}

type ipSetThreatFeedBackend struct {
	client    *elastic.Client
	templates bapi.Cache
	lmaclient lmaelastic.Client
}

func (b *ipSetThreatFeedBackend) Create(ctx context.Context, i bapi.ClusterInfo, feeds []v1.IPSetThreatFeed) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.IPSet, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Infof("Writing ip set threat feeds data in bulk to alias %s", alias)

	// Build a bulk request using the provided threat feeds.
	bulk := b.client.Bulk()

	for _, f := range feeds {
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f.Data).Id(f.ID)
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing ip sets threat feeds data: %s", err)
		return nil, fmt.Errorf("failed to write ip sets threat feeds data: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("Threat feeds ip sets bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

func (b *ipSetThreatFeedBackend) List(ctx context.Context, i bapi.ClusterInfo, params *v1.IPSetThreatFeedParams) (*v1.List[v1.IPSetThreatFeed], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(i, params)
	if err != nil {
		return nil, err
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	feeds := []v1.IPSetThreatFeed{}
	for _, h := range results.Hits.Hits {
		feed := v1.IPSetThreatFeedData{}
		err = json.Unmarshal(h.Source, &feed)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling threat feed")
			continue
		}
		ipSetFeed := v1.IPSetThreatFeed{
			ID:          h.Id,
			Data:        &feed,
			SeqNumber:   h.SeqNo,
			PrimaryTerm: h.PrimaryTerm,
		}

		feeds = append(feeds, ipSetFeed)
	}

	return &v1.List[v1.IPSetThreatFeed]{
		Items:     feeds,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextStartFromAfterKey(params, len(results.Hits.Hits), startFrom, results.TotalHits()),
	}, nil
}

func (b *ipSetThreatFeedBackend) getSearch(i bapi.ClusterInfo, p *v1.IPSetThreatFeedParams) (*elastic.SearchService, int, error) {
	if err := i.Valid(); err != nil {
		return nil, 0, err
	}

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(p)
	if err != nil {
		return nil, 0, err
	}

	q, err := b.buildQuery(p)
	if err != nil {
		return nil, 0, err
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Index(b.index(i)).
		Size(p.GetMaxPageSize()).
		From(startFrom).
		Query(q)

	return query, startFrom, nil
}

func (b *ipSetThreatFeedBackend) buildQuery(p *v1.IPSetThreatFeedParams) (elastic.Query, error) {
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

func (b *ipSetThreatFeedBackend) Delete(ctx context.Context, i bapi.ClusterInfo, feeds []v1.IPSetThreatFeed) (*v1.BulkResponse, error) {
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

func (b *ipSetThreatFeedBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_threatfeeds_ipset.%s.%s.*", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_threatfeeds_ipset.%s.*", i.Cluster)
}

func (b *ipSetThreatFeedBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_threatfeeds_ipset.%s.%s.", i.Tenant, i.Cluster)
	}

	return fmt.Sprintf("tigera_secure_ee_threatfeeds_ipset.%s.", i.Cluster)
}
