// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns

import (
	"context"
	"fmt"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/olivere/elastic/v7"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
)

type dnsLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
	helper    lmaindex.Helper
	templates bapi.Cache
}

func NewDNSLogBackend(c lmaelastic.Client, cache bapi.Cache) bapi.DNSLogBackend {
	return &dnsLogBackend{
		client:    c.Backend(),
		lmaclient: c,
		helper:    lmaindex.DnsLogs(),
		templates: cache,
	}
}

func (b *dnsLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.DNSLog) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.DNSLogs, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Debugf("Writing DNS logs in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		// Add this log to the bulk request.
		dnsLog, err := json.Marshal(f)
		if err != nil {
			log.WithError(err).Warningf("Failed to marshal dns log and add it to the request %+v", f)
			continue
		}
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(string(dnsLog))
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing DNS log: %s", err)
		return nil, fmt.Errorf("failed to write DNS log: %s", err)
	}
	log.WithField("count", len(logs)).Debugf("Wrote DNS log to index: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists logs that match the given parameters.
func (b *dnsLogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.DNSLogParams) (*v1.List[v1.DNSLog], error) {
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

	// Build the query.
	query := b.client.Search().
		Index(b.index(i)).
		Size(opts.QueryParams.GetMaxPageSize()).
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

	logs := []v1.DNSLog{}
	for _, h := range results.Hits.Hits {
		l := v1.DNSLog{}
		err = json.Unmarshal(h.Source, &l)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling log")
			continue
		}
		logs = append(logs, l)
	}

	// Determine the AfterKey to return.
	var ak map[string]interface{}
	if numHits := len(results.Hits.Hits); numHits < opts.QueryParams.GetMaxPageSize() {
		// We fully satisfied the request, no afterkey.
		ak = nil
	} else {
		// There are more hits, return an afterKey the client can use for pagination.
		// We add the number of hits to the start from provided on the request, if any.
		ak = map[string]interface{}{
			"startFrom": startFrom + len(results.Hits.Hits),
		}
	}

	return &v1.List[v1.DNSLog]{
		Items:     logs,
		TotalHits: results.TotalHits(),
		AfterKey:  ak,
	}, nil
}

func (b *dnsLogBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_dns.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_dns.%s.*", i.Cluster)
}

func (b *dnsLogBackend) writeAlias(i bapi.ClusterInfo) string {
	// TODO: Not multi-tenant
	return fmt.Sprintf("tigera_secure_ee_dns.%s.", i.Cluster)
}
