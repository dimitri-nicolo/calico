// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package threatfeeds

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/projectcalico/go-json/json"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func NewIPSetBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64) bapi.IPSetBackend {
	return &ipSetThreatFeedBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		singleIndex:          false,
		index:                index.ThreatfeedsIPSetMultiIndex,
		queryHelper:          lmaindex.MultiIndexThreatfeedsIPSet(),
	}
}

func NewSingleIndexIPSetBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64) bapi.IPSetBackend {
	return &ipSetThreatFeedBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		singleIndex:          true,
		index:                index.ThreatfeedsIPSetIndex,
		queryHelper:          lmaindex.SingleIndexThreatfeedsIPSet(),
	}
}

type ipSetThreatFeedBackend struct {
	client               *elastic.Client
	templates            bapi.IndexInitializer
	lmaclient            lmaelastic.Client
	deepPaginationCutOff int64
	queryHelper          lmaindex.Helper
	singleIndex          bool
	index                bapi.Index
}

type ipsetWithExtras struct {
	v1.IPSetThreatFeedData `json:",inline"`
	Cluster                string `json:"cluster"`
	Tenant                 string `json:"tenant,omitempty"`
}

// prepareForWrite wraps a log in a document that includes the cluster and tenant if
// the backend is configured to write to a single index.
func (b *ipSetThreatFeedBackend) prepareForWrite(i bapi.ClusterInfo, l *v1.IPSetThreatFeedData) interface{} {
	if b.singleIndex {
		return &ipsetWithExtras{
			IPSetThreatFeedData: *l,
			Cluster:             i.Cluster,
			Tenant:              i.Tenant,
		}
	}
	return l
}

func (b *ipSetThreatFeedBackend) Create(ctx context.Context, i bapi.ClusterInfo, feeds []v1.IPSetThreatFeed) (*v1.BulkResponse, error) {
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
	log.Infof("Writing ip set threat feeds data in bulk to alias %s", alias)

	// Build a bulk request using the provided threat feeds.
	bulk := b.client.Bulk()

	for _, f := range feeds {
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(b.prepareForWrite(i, f.Data)).Id(f.ID)
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

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index.Index(i), results, b.deepPaginationCutOff, log)
	if err != nil {
		return nil, err
	}

	return &v1.List[v1.IPSetThreatFeed]{
		Items:     feeds,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextAfterKey(params, startFrom, pitID, results, b.deepPaginationCutOff),
	}, nil
}

func (b *ipSetThreatFeedBackend) getSearch(i bapi.ClusterInfo, p *v1.IPSetThreatFeedParams) (*elastic.SearchService, int, error) {
	if err := i.Valid(); err != nil {
		return nil, 0, err
	}

	q, err := b.buildQuery(i, p)
	if err != nil {
		return nil, 0, err
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Size(p.GetMaxPageSize()).
		Query(q)

	// Configure pagination options
	var startFrom int
	query, startFrom, err = logtools.ConfigureCurrentPage(query, p, b.index.Index(i))
	if err != nil {
		return nil, 0, err
	}

	query.Sort(b.queryHelper.GetTimeField(), true)

	return query, startFrom, nil
}

func (b *ipSetThreatFeedBackend) buildQuery(i bapi.ClusterInfo, p *v1.IPSetThreatFeedParams) (elastic.Query, error) {
	query := b.queryHelper.BaseQuery(i)

	if p.TimeRange != nil {
		query.Must(b.queryHelper.NewTimeRangeQuery(p.TimeRange.From, p.TimeRange.To))
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

	alias := b.index.Alias(i)

	// Build a bulk request using the provided feeds.
	bulk := b.client.Bulk()
	numToDelete := 0
	bulkErrs := []v1.BulkError{}
	for _, feed := range feeds {
		if err := b.checkTenancy(ctx, i, &feed); err != nil {
			logrus.WithError(err).WithField("id", feed.ID).Warn("Error checking tenancy for feed")
			bulkErrs = append(bulkErrs, v1.BulkError{Resource: feed.ID, Type: "document_missing_exception", Reason: err.Error()})
			continue
		}

		req := elastic.NewBulkDeleteRequest().Index(alias).Id(feed.ID)
		bulk.Add(req)
		numToDelete++
	}

	if numToDelete == 0 {
		// If there are no feeds to delete, short-circuit and return an empty response.
		return &v1.BulkResponse{
			Total:     len(bulkErrs),
			Succeeded: 0,
			Failed:    len(bulkErrs),
			Errors:    bulkErrs,
		}, nil
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

func (b *ipSetThreatFeedBackend) checkTenancy(ctx context.Context, i bapi.ClusterInfo, feed *v1.IPSetThreatFeed) error {
	// If we're in single index mode, we need to check tenancy. Otherwise, we can skip this because
	// the index name already contains the cluster and tenant ID.
	if b.singleIndex {
		// We need to protect against tenancy here. In single index mode without this check, any tenant could send a request which
		// dismisses or deletes a feed for any other tenant if they guess the right ID.
		// Query the feed to compare the tenant and cluster to the request. If they don't match, skip.
		// This is not a perfect solution, but it's better than nothing.
		// By Listing with the given cluster info and ID, we can ensure that the feed is visible to that tenant.
		items, err := b.List(ctx, i, &v1.IPSetThreatFeedParams{ID: feed.ID, QueryParams: v1.QueryParams{MaxPageSize: 1}})
		if err != nil {
			return err
		}
		if len(items.Items) == 0 {
			return fmt.Errorf("event not found during tenancy check")
		}
	}
	return nil
}
