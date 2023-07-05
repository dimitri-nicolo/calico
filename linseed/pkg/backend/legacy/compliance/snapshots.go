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
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/lma/pkg/list"
)

func NewSnapshotBackend(c lmaelastic.Client, cache bapi.Cache, deepPaginationCutOff int64) bapi.SnapshotsBackend {
	return &snapshotsBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
	}
}

type snapshotsBackend struct {
	client               *elastic.Client
	templates            bapi.Cache
	lmaclient            lmaelastic.Client
	deepPaginationCutOff int64
}

func (b *snapshotsBackend) List(ctx context.Context, i bapi.ClusterInfo, opts *v1.SnapshotParams) (*v1.List[v1.Snapshot], error) {
	log := bapi.ContextLogger(i)

	query, startFrom, err := b.getSearch(i, opts)
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

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index(i), results, b.deepPaginationCutOff, log)
	if err != nil {
		return nil, err
	}

	return &v1.List[v1.Snapshot]{
		Items:     snapshots,
		TotalHits: results.TotalHits(),
		AfterKey:  logtools.NextAfterKey(opts, startFrom, pitID, results, b.deepPaginationCutOff),
	}, nil
}

func (b *snapshotsBackend) Create(ctx context.Context, i bapi.ClusterInfo, l []v1.Snapshot) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
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

func (b *snapshotsBackend) getSearch(i bapi.ClusterInfo, opts *v1.SnapshotParams) (*elastic.SearchService, int, error) {
	if err := i.Valid(); err != nil {
		return nil, 0, err
	}

	q := b.buildQuery(opts)

	// Build the query, sorting by time.
	query := b.client.Search().
		Size(opts.GetMaxPageSize()).
		Query(q)

	// Configure pagination options
	var startFrom int
	var err error
	query, startFrom, err = logtools.ConfigureCurrentPage(query, opts, b.index(i))
	if err != nil {
		return nil, 0, err
	}

	// Configure sorting.
	if len(opts.Sort) != 0 {
		for _, s := range opts.Sort {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.SortBy(elastic.NewFieldSort("requestCompletedTimestamp").Order(false), elastic.SortByDoc{})
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
