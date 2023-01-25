// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package legacy

import (
	"context"
	"fmt"

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
func (b *flowLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, f bapi.FlowLog) error {
	log := contextLogger(i)

	// Initialize if we haven't yet.
	err := b.Initialize(ctx)
	if err != nil {
		return err
	}

	if i.Cluster == "" {
		log.Fatal("BUG: No cluster ID on request")
	}
	if f.Cluster != "" {
		// For the legacy backend, the Cluster ID is encoded into the index
		// and not the log itself.
		log.Fatal("BUG: Cluster ID should not be set on flow log")
	}

	// Determine the index to write to. It will be automatically created based on the configured
	// flow template if it does not already exist.
	index := fmt.Sprintf("tigera_secure_ee_flows.%s", i.Cluster)
	log.Infof("Creating flow log in index %s", index)

	// Add the flow log to the index.
	// TODO: Probably want this to be the /bulk endpoint.
	resp, err := b.client.Index().
		Index(index).
		BodyJson(f).
		Refresh("true"). // TODO: Probably want this to be false.
		Do(ctx)
	if err != nil {
		log.Errorf("Error writing flow log: %s", err)
		return fmt.Errorf("failed to write flow log: %s", err)
	}

	log.Infof("Wrote flow log to index: %+v", resp)

	return nil
}
