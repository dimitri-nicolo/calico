package l7

import (
	"context"
	"fmt"

	elastic "github.com/olivere/elastic/v7"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type l7LogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	templates bapi.Cache
}

func NewL7LogBackend(c lmaelastic.Client, cache bapi.Cache) bapi.L7LogBackend {
	b := &l7LogBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
	}
	return b
}

// Create the given log in elasticsearch.
func (b *l7LogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []bapi.L7Log) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	err := b.templates.InitializeIfNeeded(ctx, bapi.L7Logs, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(i)
	log.Debugf("Writing L7 logs in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		if f.Cluster != "" {
			// For the legacy backend, the Cluster ID is encoded into the index
			// and not the log itself. Fail the entire batch.
			return nil, fmt.Errorf("cluster ID should not be set on L7 log")
		}

		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f)
		bulk.Add(req)

		// TODO: Set a size-limit per-bulk-request. Is it possible that we receive a batch
		// from the frontend so large that requires being sent to ES in multiple smaller batches?
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing L7 log: %s", err)
		return nil, fmt.Errorf("failed to write L7 log: %s", err)
	}

	log.WithField("count", len(logs)).Infof("Wrote L7 logs to index: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

func (b *l7LogBackend) writeAlias(i bapi.ClusterInfo) string {
	return fmt.Sprintf("tigera_secure_ee_l7.%s.", i.Cluster)
}
