// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package flows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
)

type flowLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
	helper    lmaindex.Helper
	templates bapi.Cache
}

func NewFlowLogBackend(c lmaelastic.Client, cache bapi.Cache) bapi.FlowLogBackend {
	return &flowLogBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
		helper:    lmaindex.FlowLogs(),
	}
}

// Create the given flow log in elasticsearch.
func (b *flowLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.FlowLog) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("No cluster ID on request")
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.FlowLogs, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Infof("Writing flow logs in bulk to alias %s", alias)

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
		log.Errorf("Error writing flow log: %s", err)
		return nil, fmt.Errorf("failed to write flow log: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("Flow log bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists logs that match the given parameters.
func (b *flowLogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.FlowLogParams) (*v1.List[v1.FlowLog], error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(opts)
	if err != nil {
		return nil, err
	}

	q, err := logtools.BuildQuery(b.helper, i, opts)
	if err != nil {
		return nil, err
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Index(b.index(i)).
		Size(opts.QueryParams.GetMaxResults()).
		From(startFrom).
		Query(q)

	// Configure sorting.
	if len(opts.Sort) != 0 {
		for _, s := range opts.Sort {
			query.Sort(s.Field, !s.Descending)
		}
	} else {
		query.Sort(b.helper.GetTimeField(), true)
	}

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.FlowLog{}
	for _, h := range results.Hits.Hits {
		l := v1.FlowLog{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling log")
			continue
		}
		logs = append(logs, l)
	}

	// Determine the AfterKey to return.
	var ak map[string]interface{}
	if numHits := len(results.Hits.Hits); numHits < opts.QueryParams.GetMaxResults() {
		// We fully satisfied the request, no afterkey.
		ak = nil
	} else {
		// There are more hits, return an afterKey the client can use for pagination.
		// We add the number of hits to the start from provided on the request, if any.
		ak = map[string]interface{}{
			"startFrom": startFrom + len(results.Hits.Hits),
		}
	}

	return &v1.List[v1.FlowLog]{
		Items:     logs,
		TotalHits: results.TotalHits(),
		AfterKey:  ak,
	}, nil
}

func (b *flowLogBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_flows.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_flows.%s.*", i.Cluster)
}

func (b *flowLogBackend) writeAlias(i bapi.ClusterInfo) string {
	return fmt.Sprintf("tigera_secure_ee_flows.%s.", i.Cluster)
}
