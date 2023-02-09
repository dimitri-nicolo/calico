// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type dnsLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	templates bapi.Cache
}

func NewDNSLogBackend(c lmaelastic.Client, cache bapi.Cache) bapi.DNSLogBackend {
	return &dnsLogBackend{
		client:    c.Backend(),
		lmaclient: c,
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
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f)
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

func (b *dnsLogBackend) writeAlias(i bapi.ClusterInfo) string {
	return fmt.Sprintf("tigera_secure_ee_dns.%s.", i.Cluster)
}
