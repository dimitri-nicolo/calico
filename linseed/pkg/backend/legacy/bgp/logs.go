// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package bgp

import (
	"context"
	"fmt"
	"time"

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

type bgpLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	templates            bapi.IndexInitializer
	deepPaginationCutOff int64

	queryHelper lmaindex.Helper
	singleIndex bool
	index       bapi.Index
}

func NewBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64) bapi.BGPBackend {
	return &bgpLogBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		queryHelper:          lmaindex.MultiIndexBGPLogs(),
		singleIndex:          false,
		index:                index.BGPLogMultiIndex,
	}
}

func NewSingleIndexBackend(c lmaelastic.Client, cache bapi.IndexInitializer, deepPaginationCutOff int64, options ...index.Option) bapi.BGPBackend {
	return &bgpLogBackend{
		client:               c.Backend(),
		lmaclient:            c,
		templates:            cache,
		deepPaginationCutOff: deepPaginationCutOff,
		queryHelper:          lmaindex.SingleIndexBGPLogs(),
		singleIndex:          true,
		index:                index.BGPLogIndex(options...),
	}
}

type logWithExtras struct {
	v1.BGPLog `json:",inline"`
	Tenant    string `json:"tenant,omitempty"`
}

// prepareForWrite sets the cluster field, and wraps the log in a document to set tenant if
// the backend is configured to write to a single index.
func (b *bgpLogBackend) prepareForWrite(i bapi.ClusterInfo, l v1.BGPLog) interface{} {
	l.Cluster = i.Cluster

	if b.singleIndex {
		return &logWithExtras{
			BGPLog: l,
			Tenant: i.Tenant,
		}
	}
	return l
}

// Create the given logs in elasticsearch.
func (b *bgpLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.BGPLog) (*v1.BulkResponse, error) {
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
	log.Debugf("Writing BGP logs in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, l := range logs {
		// Populate the log's GeneratedTime field.  This field exists to enable a way for
		// clients to efficiently query newly generated logs, and having Linseed fill it in
		// - instead of an upstream client - makes this less vulnerable to time skew between
		// clients, and between clients and Linseed.
		generatedTime := time.Now().UTC()
		l.GeneratedTime = &generatedTime

		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(b.prepareForWrite(i, l))
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing log: %s", err)
		return nil, fmt.Errorf("failed to write log: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("BGP log bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists logs that match the given parameters.
func (b *bgpLogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.BGPLogParams) (*v1.List[v1.BGPLog], error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	// Build the query.
	q, err := b.buildQuery(i, opts)
	if err != nil {
		return nil, err
	}

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
	if len(opts.Sort) != 0 {
		for _, s := range opts.Sort {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.Sort("logtime", true)
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.BGPLog{}
	for _, h := range results.Hits.Hits {
		l := v1.BGPLog{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling BGP log")
			continue
		}
		logs = append(logs, l)
	}

	// If an index has more than 10000 items or other value configured via index.max_result_window
	// setting in Elastic, we need to perform deep pagination
	pitID, err := logtools.NextPointInTime(ctx, b.client, b.index.Index(i), results, b.deepPaginationCutOff, log)
	if err != nil {
		return nil, err
	}

	return &v1.List[v1.BGPLog]{
		TotalHits: results.TotalHits(),
		Items:     logs,
		AfterKey:  logtools.NextAfterKey(opts, startFrom, pitID, results, b.deepPaginationCutOff),
	}, nil
}

// buildQuery builds an elastic query using the given parameters.
func (b *bgpLogBackend) buildQuery(i bapi.ClusterInfo, opts *v1.BGPLogParams) (elastic.Query, error) {
	// Start with the base query for this index.
	query, err := b.queryHelper.BaseQuery(i, opts)
	if err != nil {
		return nil, err
	}

	// Add the time range to the query.
	query.Filter(b.queryHelper.NewTimeRangeQuery(
		logtools.WithDefaultLast5Minutes(opts.QueryParams.TimeRange),
	))

	return query, nil
}
