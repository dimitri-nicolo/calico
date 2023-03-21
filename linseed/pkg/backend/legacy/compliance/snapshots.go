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
	"github.com/projectcalico/calico/lma/pkg/list"
)

func NewSnapshotBackend(c lmaelastic.Client, cache bapi.Cache) bapi.SnapshotsBackend {
	return &snapshotsBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
	}
}

type snapshotsBackend struct {
	client    *elastic.Client
	templates bapi.Cache
	lmaclient lmaelastic.Client
}

func (b *snapshotsBackend) List(ctx context.Context, i bapi.ClusterInfo, p *v1.SnapshotParams) (*v1.List[v1.Snapshot], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(ctx, i, p)
	if err != nil {
		return nil, err
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	snapshots := []v1.Snapshot{}
	for _, h := range results.Hits.Hits {
		rl := list.TimestampedResourceList{}
		err = json.Unmarshal(h.Source, &rl)
		if err != nil {
			log.WithError(err).WithField("_source", string(h.Source)).Error("Error unmarshalling snapshot")
			continue
		}
		snapshot := v1.Snapshot{}
		snapshot.ID = h.Id
		snapshot.ResourceList = rl
		snapshots = append(snapshots, snapshot)
	}

	return &v1.List[v1.Snapshot]{
		Items:     snapshots,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextStartFromAfterKey(p, len(results.Hits.Hits), startFrom),
	}, nil
}

func (b *snapshotsBackend) Create(ctx context.Context, i bapi.ClusterInfo, l []v1.Snapshot) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("No cluster ID on request")
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.Snapshots, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Infof("Writing snapshot data in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range l {
		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(&f.ResourceList).Id(f.ResourceList.String())
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing snapshot data: %s", err)
		return nil, fmt.Errorf("failed to write snapshot data: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("Compliance snapshot bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

func (b *snapshotsBackend) getSearch(ctx context.Context, i api.ClusterInfo, p *v1.SnapshotParams) (*elastic.SearchService, int, error) {
	if i.Cluster == "" {
		return nil, 0, fmt.Errorf("no cluster ID on request")
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
		query.Sort("requestCompletedTimestamp", false)
	}
	return query, startFrom, nil
}

func (b *snapshotsBackend) buildQuery(p *v1.SnapshotParams) elastic.Query {
	query := elastic.NewBoolQuery()
	if p.TimeRange != nil {
		unset := time.Time{}
		tr := elastic.NewRangeQuery("requestCompletedTimestamp")
		if p.TimeRange.From != unset {
			tr.From(p.TimeRange.From)
		}
		if p.TimeRange.To != unset {
			tr.To(p.TimeRange.To)
		}
		query.Must(tr)
	}
	if p.TypeMatch != nil {
		if p.TypeMatch.Kind != "" {
			query.Must(elastic.NewTermQuery("kind", p.TypeMatch.Kind))
		}
		if p.TypeMatch.APIVersion != "" {
			query.Must(elastic.NewTermQuery("apiVersion", p.TypeMatch.APIVersion))
		}
	}
	return query
}

func (b *snapshotsBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_snapshots.%s.%s.*", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_snapshots.%s.*", i.Cluster)
}

func (b *snapshotsBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_snapshots.%s.%s.", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_snapshots.%s.", i.Cluster)
}
