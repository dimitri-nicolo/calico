// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package events

import (
	"context"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type eventsBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
	templates bapi.Cache
}

func NewBackend(c lmaelastic.Client, cache bapi.Cache) bapi.EventsBackend {
	return &eventsBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
	}
}

// Create the given events in elasticsearch.
func (b *eventsBackend) Create(ctx context.Context, i bapi.ClusterInfo, events []v1.Event) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
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
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(event)
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
func (b *eventsBackend) List(ctx context.Context, i api.ClusterInfo, opts v1.EventParams) (*v1.List[v1.Event], error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Index(b.index(i)).
		Size(opts.QueryParams.GetMaxResults()).
		Sort("time", true).
		Query(b.buildQuery(i, opts))

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

	return &v1.List[v1.Event]{
		Items:    events,
		AfterKey: nil, // TODO: Support pagination.
	}, nil
}

// buildQuery builds an elastic query using the given parameters.
func (b *eventsBackend) buildQuery(i bapi.ClusterInfo, opts v1.EventParams) elastic.Query {
	// Parse times from the request. We default to a time-range query
	// if no other search parameters are given.
	var start, end time.Time
	if opts.QueryParams.TimeRange != nil {
		start = opts.QueryParams.TimeRange.From
		end = opts.QueryParams.TimeRange.To
	} else {
		// Default to the latest 5 minute window.
		start = time.Now().Add(-5 * time.Minute)
		end = time.Now()
	}
	return elastic.NewRangeQuery("time").Gt(start.Unix()).Lte(end.Unix())
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
	return fmt.Sprintf("tigera_secure_ee_events.%s.", i.Cluster)
}
