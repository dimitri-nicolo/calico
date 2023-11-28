// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package events

import (
	"context"
	"fmt"

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

type eventsBackend struct {
	client               *elastic.Client
	lmaclient            lmaelastic.Client
	templates            bapi.IndexInitializer
	deepPaginationCutOff int64
	queryHelper          lmaindex.Helper
	singleIndex          bool
	index                bapi.Index
}

func NewBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64) bapi.EventsBackend {
	return &eventsBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		queryHelper:          lmaindex.MultiIndexAlerts(),
		deepPaginationCutOff: deepPaginationCutOff,
		index:                index.EventsMultiIndex,
	}
}

func NewSingleIndexBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64, options ...index.Option) bapi.EventsBackend {
	return &eventsBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		queryHelper:          lmaindex.SingleIndexAlerts(),
		deepPaginationCutOff: deepPaginationCutOff,
		index:                index.AlertsIndex(options...),
		singleIndex:          true,
	}
}

type withExtras struct {
	v1.Event `json:",inline"`
	Cluster  string `json:"cluster"`
	Tenant   string `json:"tenant,omitempty"`
}

// prepareForWrite wraps a log in a document that includes the cluster and tenant if
// the backend is configured to write to a single index.
func (b *eventsBackend) prepareForWrite(i bapi.ClusterInfo, l v1.Event) interface{} {
	// We don't want to include the ID in the document ever.
	l.ID = ""

	if b.singleIndex {
		return &withExtras{
			Event:   l,
			Cluster: i.Cluster,
			Tenant:  i.Tenant,
		}
	}
	return l
}

// Create the given events in elasticsearch.
func (b *eventsBackend) Create(ctx context.Context, i bapi.ClusterInfo, events []v1.Event) (*v1.BulkResponse, error) {
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
	log.Debugf("Writing events in bulk to index %s", alias)

	// Build a bulk request using the provided events.
	bulk := b.client.Bulk()

	for _, event := range events {
		id := event.ID
		eventJSON, err := json.Marshal(b.prepareForWrite(i, event))
		if err != nil {
			log.WithError(err).Warningf("Failed to marshal event and add it to the request %+v", event)
			continue
		}

		req := elastic.NewBulkIndexRequest().Index(alias).Doc(string(eventJSON)).Id(id)
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to write events: %s", err)
	}
	log.WithField("count", len(events)).Debugf("Wrote events to index: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists events that match the given parameters.
func (b *eventsBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.EventParams) (*v1.List[v1.Event], error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	q, err := logtools.BuildQuery(b.queryHelper, i, opts)
	if err != nil {
		return nil, err
	}

	// If an ID was given on the request, limit to just that ID.
	if opts.ID != "" {
		q.Must(elastic.NewTermQuery("_id", opts.ID))
	}

	// Build the query.
	query := b.client.Search().
		Size(opts.QueryParams.GetMaxPageSize()).
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
		query.SortBy(elastic.NewFieldSort(b.queryHelper.GetTimeField()).Order(true))
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	events := []v1.Event{}
	for _, h := range results.Hits.Hits {
		event := v1.Event{}
		err = json.Unmarshal(h.Source, &event)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling event")
			continue
		}
		event.ID = h.Id
		events = append(events, event)
	}

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index.Index(i), results, b.deepPaginationCutOff, log)
	if err != nil {
		return nil, err
	}

	return &v1.List[v1.Event]{
		Items:     events,
		AfterKey:  logtools.NextAfterKey(opts, startFrom, pitID, results, b.deepPaginationCutOff),
		TotalHits: results.TotalHits(),
	}, nil
}

func (b *eventsBackend) UpdateDismissFlag(ctx context.Context, i api.ClusterInfo, events []v1.Event) (*v1.BulkResponse, error) {
	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}
	alias := b.index.Alias(i)

	// Build a bulk request using the provided events.
	bulk := b.client.Bulk()
	numToDismiss := 0
	bulkErrs := []v1.BulkError{}
	for _, event := range events {
		if err := b.checkTenancy(ctx, i, &event); err != nil {
			logrus.WithError(err).WithField("id", event.ID).Warn("Error checking tenancy for event")
			bulkErrs = append(bulkErrs, v1.BulkError{Resource: event.ID, Type: "document_missing_exception", Reason: err.Error()})
			continue
		}
		req := elastic.NewBulkUpdateRequest().Index(alias).Id(event.ID).Doc(map[string]bool{"dismissed": event.Dismissed})
		bulk.Add(req)
		numToDismiss++
	}

	if numToDismiss == 0 {
		// If there are no events to dismiss, short-circuit and return an empty response.
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
		return nil, fmt.Errorf("failed to dismiss events: %s", err)
	}

	// Convert individual success / failure responses.
	upd := []v1.BulkItem{}
	for _, i := range resp.Updated() {
		bi := v1.BulkItem{ID: i.Id, Status: i.Status}
		upd = append(upd, bi)
	}

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
		Updated:   upd,
	}, nil
}

func (b *eventsBackend) checkTenancy(ctx context.Context, i api.ClusterInfo, event *v1.Event) error {
	// If we're in single index mode, we need to check tenancy. Otherwise, we can skip this because
	// the index name already contains the cluster and tenant ID.
	if b.singleIndex {
		// We need to protect against tenancy here. In single index mode without this check, any tenant could send a request which
		// dismisses or deletes events for any other tenant if they guess the right ID.
		// Query the event to compare the tenant and cluster to the request. If they don't match, skip.
		// This is not a perfect solution, but it's better than nothing.
		// By Listing with the given cluster info and ID, we can ensure that the event is visible to that tenant.
		items, err := b.List(ctx, i, &v1.EventParams{ID: event.ID, QueryParams: v1.QueryParams{MaxPageSize: 1}})
		if err != nil {
			return err
		}
		if len(items.Items) == 0 {
			return fmt.Errorf("event not found during tenancy check")
		}
	}
	return nil
}

func (b *eventsBackend) Delete(ctx context.Context, i api.ClusterInfo, events []v1.Event) (*v1.BulkResponse, error) {
	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}
	alias := b.index.Alias(i)

	// Build a bulk request using the provided events.
	bulk := b.client.Bulk()
	numToDelete := 0
	bulkErrs := []v1.BulkError{}
	for _, event := range events {
		if err := b.checkTenancy(ctx, i, &event); err != nil {
			logrus.WithError(err).WithField("id", event.ID).Warn("Error checking tenancy for event")
			bulkErrs = append(bulkErrs, v1.BulkError{Resource: event.ID, Type: "document_missing_exception", Reason: err.Error()})
			continue
		}
		req := elastic.NewBulkDeleteRequest().Index(alias).Id(event.ID)
		bulk.Add(req)
		numToDelete++
	}

	if numToDelete == 0 {
		// If there are no events to delete, short-circuit and return an empty response, including
		// any errors that occurred during tenancy checks.
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
		return nil, fmt.Errorf("failed to delete events: %s", err)
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
