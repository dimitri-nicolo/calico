// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package legacy

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type dnsLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	// Tracks whether the backend has been initialized.
	initialized bool
}

func NewDNSLogBackend(c lmaelastic.Client) bapi.DNSLogBackend {
	return &dnsLogBackend{
		client:    c.Backend(),
		lmaclient: c,
	}
}

func (b *dnsLogBackend) Initialize(ctx context.Context) error {
	var err error
	if !b.initialized {
		// Create a template with mappings for all new indices.
		_, err = b.client.IndexPutTemplate("dns_log_template").
			BodyString(templates.DNSLogTemplate).
			Create(false).
			Do(ctx)
		if err != nil {
			return err
		}
		b.initialized = true
	}
	return nil
}

func (b *dnsLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.DNSLog) (*v1.BulkResponse, error) {
	log := contextLogger(i)

	// Initialize if we haven't yet.
	err := b.Initialize(ctx)
	if err != nil {
		return nil, err
	}

	if i.Cluster == "" {
		return nil, fmt.Errorf("No cluster ID on request")
	}

	// Determine the index to write to. It will be automatically created based on the configured
	// template if it does not already exist.
	index := fmt.Sprintf("tigera_secure_ee_dns.%s", i.Cluster)
	log.Debugf("Writing DNS logs in bulk to index %s", index)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(index).Doc(f)
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
