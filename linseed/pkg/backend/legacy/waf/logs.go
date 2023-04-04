// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package waf

import (
	"context"
	"fmt"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type wafLogBackend struct {
	client *elastic.Client

	lmaclient lmaelastic.Client
	templates bapi.Cache
}

func NewBackend(c lmaelastic.Client, cache bapi.Cache) bapi.WAFBackend {
	return &wafLogBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
	}
}

// Create the given logs in elasticsearch.
func (b *wafLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.WAFLog) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.WAFLogs, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Debugf("Writing WAF logs in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f)
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
	log.WithFields(fields).Debugf("WAF log bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists logs that match the given parameters.
func (b *wafLogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.WAFLogParams) (*v1.List[v1.WAFLog], error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(opts)
	if err != nil {
		return nil, err
	}

	// Build the query.
	query := b.client.Search().
		Index(b.index(i)).
		From(startFrom).
		Size(opts.QueryParams.GetMaxPageSize()).
		Query(b.buildQuery(i, opts))

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.WAFLog{}
	for _, h := range results.Hits.Hits {
		l := v1.WAFLog{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling WAF log")
			continue
		}
		logs = append(logs, l)
	}

	return &v1.List[v1.WAFLog]{
		TotalHits: int64(len(logs)),
		Items:     logs,
		AfterKey:  logtools.NextStartFromAfterKey(opts, len(results.Hits.Hits), startFrom),
	}, nil
}

// buildQuery builds an elastic query using the given parameters.
func (b *wafLogBackend) buildQuery(i bapi.ClusterInfo, opts *v1.WAFLogParams) elastic.Query {
	// Parse times from the request. We default to a time-range query
	// if no other search parameters are given.
	var start, end time.Time
	if opts != nil && opts.QueryParams.TimeRange != nil {
		start = opts.QueryParams.TimeRange.From
		end = opts.QueryParams.TimeRange.To
	} else {
		// Default to the latest 5 minute window.
		start = time.Now().Add(-5 * time.Minute)
		end = time.Now()
	}
	return elastic.NewRangeQuery("@timestamp").Gt(start).Lte(end)
}

func (b *wafLogBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_waf.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_waf.%s.*", i.Cluster)
}

func (b *wafLogBackend) writeAlias(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		return fmt.Sprintf("tigera_secure_ee_waf.%s.%s.", i.Tenant, i.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_waf.%s.", i.Cluster)
}
