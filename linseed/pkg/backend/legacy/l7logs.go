package legacy

import (
	"context"
	"fmt"

	elastic "github.com/olivere/elastic/v7"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type l7LogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	// Tracks whether the backend has been initialized.
	initialized bool
}

func NewL7LogBackend(c lmaelastic.Client) bapi.L7LogBackend {
	b := &l7LogBackend{
		client:    c.Backend(),
		lmaclient: c,
	}
	return b
}

func (b *l7LogBackend) Initialize(ctx context.Context) error {
	var err error
	if !b.initialized {
		// Create a template with mappings for all new log indices.
		_, err = b.client.IndexPutTemplate("l7_log_template").
			BodyString(templates.L7LogTemplate).
			Create(false).
			Do(ctx)
		if err != nil {
			return err
		}
		b.initialized = true
	}
	return nil
}

// Create the given log in elasticsearch.
func (b *l7LogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []bapi.L7Log) error {
	log := contextLogger(i)

	// Initialize if we haven't yet.
	err := b.Initialize(ctx)
	if err != nil {
		return err
	}

	if i.Cluster == "" {
		log.Fatal("BUG: No cluster ID on request")
	}

	// Determine the index to write to. It will be automatically created based on the configured
	// template if it does not already exist.
	index := fmt.Sprintf("tigera_secure_ee_l7.%s", i.Cluster)
	log.Infof("Writing L7 logs in bulk to index %s", index)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		if f.Cluster != "" {
			// For the legacy backend, the Cluster ID is encoded into the index
			// and not the log itself. Fail the entire batch.
			return fmt.Errorf("cluster ID should not be set on L7 log")
		}

		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(index).Doc(f)
		bulk.Add(req)

		// TODO: Set a size-limit per-bulk-request. Is it possible that we receive a batch
		// from the frontend so large that requires being sent to ES in multiple smaller batches?
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing L7 log: %s", err)
		return fmt.Errorf("failed to write L7 log: %s", err)
	}

	log.WithField("count", len(logs)).Infof("Wrote L7 logs to index: %+v", resp)

	return nil
}
