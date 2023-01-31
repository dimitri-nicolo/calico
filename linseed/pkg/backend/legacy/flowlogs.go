// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package legacy

import (
	"context"
	"fmt"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/olivere/elastic/v7"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type flowLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	// Tracks whether the backend has been initialized.
	initialized bool
}

func NewFlowLogBackend(c lmaelastic.Client) bapi.FlowLogBackend {
	return &flowLogBackend{
		client:    c.Backend(),
		lmaclient: c,
	}
}

func (b *flowLogBackend) Initialize(ctx context.Context) error {
	var err error
	if !b.initialized {
		// Create a template with mappings for all new flow log indices.
		_, err = b.client.IndexPutTemplate("flow_log_template").
			BodyString(templates.FlowLogTemplate).
			Create(false).
			Do(ctx)
		if err != nil {
			return err
		}
		b.initialized = true
	}
	return nil
}

// Create the given flow log in elasticsearch.
func (b *flowLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.FlowLog) (*v1.BulkResponse, error) {
	log := contextLogger(i)

	// Initialize if we haven't yet.
	err := b.Initialize(ctx)
	if err != nil {
		return nil, err
	}

	if i.Cluster == "" {
		log.Fatal("BUG: No cluster ID on request")
	}

	// Determine the index to write to. It will be automatically created based on the configured
	// flow template if it does not already exist.
	index := fmt.Sprintf("tigera_secure_ee_flows.%s", i.Cluster)
	log.Infof("Writing flow logs in bulk to index %s", index)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(index).Doc(f)
		bulk.Add(req)

		// TODO: Set a size-limit per-bulk-request. Is it possible that we receive a batch
		// from the frontend so large that requires being sent to ES in multiple smaller batches?
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing flow log: %s", err)
		return nil, fmt.Errorf("failed to write flow log: %s", err)
	}

	log.WithField("count", len(logs)).Infof("Wrote flow log to index: %+v", resp)

	var bulkErrors []v1.BulkError

	if resp.Errors {
		for _, fail := range resp.Failed() {
			bulkErrors = append(bulkErrors, v1.BulkError{
				Message: fmt.Sprintf("%s failed with %s due to %s", fail.Error.ResourceId, fail.Error.Type, fail.Error.Reason),
			})
		}
	}

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    bulkErrors,
	}, nil
}
