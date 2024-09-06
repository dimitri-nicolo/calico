// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package threatfeeds

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/projectcalico/go-json/json"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func NewDomainNameSetBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64) bapi.DomainNameSetBackend {
	return &domainNameSetThreatFeedBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		singleIndex:          false,
		index:                index.ThreatfeedsDomainMultiIndex,
		queryHelper:          lmaindex.MultiIndexThreatfeedsDomainNameSet(),
	}
}

func NewSingleIndexDomainNameSetBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64, options ...index.Option) bapi.DomainNameSetBackend {
	return &domainNameSetThreatFeedBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		singleIndex:          true,
		index:                index.ThreatFeedsDomainSetIndex(options...),
		queryHelper:          lmaindex.SingleIndexThreatfeedsDomainNameSet(),
	}
}

type domainNameSetThreatFeedBackend struct {
	client               *elastic.Client
	templates            bapi.IndexInitializer
	lmaclient            lmaelastic.Client
	deepPaginationCutOff int64
	queryHelper          lmaindex.Helper
	singleIndex          bool
	index                bapi.Index
}

type domainsetWithExtras struct {
	v1.DomainNameSetThreatFeedData `json:",inline"`
	Cluster                        string `json:"cluster"`
	Tenant                         string `json:"tenant,omitempty"`
}

// prepareForWrite wraps a log in a document that includes the cluster and tenant if
// the backend is configured to write to a single index.
func (b *domainNameSetThreatFeedBackend) prepareForWrite(i bapi.ClusterInfo, l *v1.DomainNameSetThreatFeedData) interface{} {
	if b.singleIndex {
		return &domainsetWithExtras{
			DomainNameSetThreatFeedData: *l,
			Cluster:                     i.Cluster,
			Tenant:                      i.Tenant,
		}
	}
	return l
}

func (b *domainNameSetThreatFeedBackend) Create(ctx context.Context, i bapi.ClusterInfo, feeds []v1.DomainNameSetThreatFeed) (*v1.BulkResponse, error) {
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
	log.Infof("Writing domain name set threat feeds data in bulk to alias %s", alias)

	// Build a bulk request using the provided threat feeds.
	bulk := b.client.Bulk()

	for _, f := range feeds {
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(b.prepareForWrite(i, f.Data)).Id(backend.ToElasticID(b.singleIndex, f.ID, i))
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
			ID:          backend.ToApplicationID(b.singleIndex, h.Id, i),
			Data:        &feed,
			SeqNumber:   h.SeqNo,
			PrimaryTerm: h.PrimaryTerm,
		}

		feeds = append(feeds, domainNameSetFeed)
	}

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index.Index(i), results, b.deepPaginationCutOff, log)
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

func (b *domainNameSetThreatFeedBackend) buildQuery(i bapi.ClusterInfo, p *v1.DomainNameSetThreatFeedParams) (elastic.Query, error) {
	query := b.queryHelper.BaseQuery(i)

	if p.TimeRange != nil {
		query.Must(b.queryHelper.NewTimeRangeQuery(p.TimeRange))
	}

	if p.ID != "" {
		query.Must(elastic.NewTermQuery("_id", backend.ToElasticID(b.singleIndex, p.ID, i)))
	}

	return query, nil
}

func (b *domainNameSetThreatFeedBackend) Delete(ctx context.Context, i bapi.ClusterInfo, feeds []v1.DomainNameSetThreatFeed) (*v1.BulkResponse, error) {
	if err := i.Valid(); err != nil {
		return nil, err
	}

	if err := b.checkTenancy(ctx, i, feeds); err != nil {
		logrus.WithError(err).Warn("Error checking tenancy")
		return &v1.BulkResponse{
			Total:     len(feeds),
			Succeeded: 0,
			Failed:    len(feeds),
			Errors:    []v1.BulkError{{Resource: "", Type: "document_missing_exception", Reason: err.Error()}},
			Deleted:   nil,
		}, nil
	}

	alias := b.index.Alias(i)

	// Build a bulk request using the provided feeds.
	bulk := b.client.Bulk()
	numToDelete := 0
	bulkErrs := []v1.BulkError{}
	for _, feed := range feeds {
		req := elastic.NewBulkDeleteRequest().Index(alias).Id(backend.ToElasticID(b.singleIndex, feed.ID, i))
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

func (b *domainNameSetThreatFeedBackend) checkTenancy(ctx context.Context, i bapi.ClusterInfo, feeds []v1.DomainNameSetThreatFeed) error {
	// If we're in single index mode, we need to check tenancy. Otherwise, we can skip this because
	// the index name already contains the cluster and tenant ID.
	if !b.singleIndex {
		return nil
	}

	// This is a shared index.
	// We need to protect against tenancy here. In single index mode without this check, any tenant could send a request which
	// deletes a feed for any other tenant if they guess the right ID.
	// Query the given feed IDs using the tenant and cluster from the request to ensure that each feed is visible to that tenant.
	ids := []string{}
	for _, feed := range feeds {
		ids = append(ids, backend.ToElasticID(b.singleIndex, feed.ID, i))
	}

	// Build a query which matches on:
	// - The given cluster and tenant (from BaseQuery)
	// - An OR combintation of the given IDs
	q := b.queryHelper.BaseQuery(i)
	q = q.Must(elastic.NewIdsQuery().Ids(ids...))
	idsQuery := b.client.Search().
		Size(len(ids)).
		Index(b.index.Index(i)).
		Query(q)
	idsResult, err := idsQuery.Do(ctx)
	if err != nil {
		return err
	}

	// Build a lookup map of the found feeds.
	foundIDs := map[string]struct{}{}
	for _, hit := range idsResult.Hits.Hits {
		foundIDs[backend.ToApplicationID(b.singleIndex, hit.Id, i)] = struct{}{}
	}

	// Now make sure that all of the given feeds were found.
	for _, feed := range feeds {
		if _, found := foundIDs[feed.ID]; !found {
			return fmt.Errorf("feed %s not found", feed.ID)
		}
	}
	return nil
}
