// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package events

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type eventsBackend struct {
	client               *elastic.Client
	lmaclient            lmaelastic.Client
	templates            bapi.Cache
	helper               lmaindex.Helper
	deepPaginationCutOff int64
}

func NewBackend(c lmaelastic.Client, cache bapi.Cache, deepPaginationCutOff int64) bapi.EventsBackend {
	return &eventsBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		helper:               lmaindex.Alerts(),
		deepPaginationCutOff: deepPaginationCutOff,
	}
}

// Create the given events in elasticsearch.
func (b *eventsBackend) Create(ctx context.Context, i bapi.ClusterInfo, events []v1.Event) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.Events, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Debugf("Writing events in bulk to index %s", alias)

	// Build a bulk request using the provided events.
	bulk := b.client.Bulk()

	for _, event := range events {
		id := event.ID
		event.ID = ""
		eventJson, err := json.Marshal(event)
		if err != nil {
			log.WithError(err).Warningf("Failed to marshal event and add it to the request %+v", event)
			continue
		}

		req := elastic.NewBulkIndexRequest().Index(alias).Doc(string(eventJson)).Id(id)
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

	start, end := logtools.ExtractTimeRange(opts.GetTimeRange())
	q, err := logtools.BuildQuery(b.helper, i, opts, start, end)
	if err != nil {
		return nil, err
	}

	// Build the query.
	query := b.client.Search().
		Size(opts.QueryParams.GetMaxPageSize()).
		Query(q)

	// Configure pagination options
	var startFrom int
	query, startFrom, err = logtools.ConfigureCurrentPage(query, opts, b.index(i))
	if err != nil {
		return nil, err
	}

	// Configure sorting.
	if len(opts.GetSortBy()) != 0 {
		for _, s := range opts.GetSortBy() {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.SortBy(elastic.NewFieldSort(b.helper.GetTimeField()).Order(true))
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
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index(i), results, b.deepPaginationCutOff, log)
	if err != nil {
		return nil, err
	}

	return &v1.List[v1.Event]{
		Items:     events,
		AfterKey:  logtools.NextAfterKey(opts, startFrom, pitID, results, b.deepPaginationCutOff),
		TotalHits: results.TotalHits(),
	}, nil
}

func (b *eventsBackend) Dismiss(ctx context.Context, i api.ClusterInfo, events []v1.Event) (*v1.BulkResponse, error) {
	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}
	alias := b.writeAlias(i)

	// Build a bulk request using the provided events.
	bulk := b.client.Bulk()
	for _, event := range events {
		req := elastic.NewBulkUpdateRequest().Index(alias).Id(event.ID).Doc(map[string]bool{"dismissed": event.Dismissed})
		bulk.Add(req)
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

func (b *eventsBackend) Delete(ctx context.Context, i api.ClusterInfo, events []v1.Event) (*v1.BulkResponse, error) {
	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}
	alias := b.writeAlias(i)

	// Build a bulk request using the provided events.
	bulk := b.client.Bulk()
	for _, event := range events {
		req := elastic.NewBulkDeleteRequest().Index(alias).Id(event.ID)
		bulk.Add(req)
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

func (b *eventsBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_events.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_events.%s.*", i.Cluster)
}

func (b *eventsBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_events.%s.%s.", i.Tenant, i.Cluster)
	}

	return fmt.Sprintf("tigera_secure_ee_events.%s.", i.Cluster)
}
