package l7

import (
	"context"
	"encoding/json"
	"fmt"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
)

type l7LogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
	helper    lmaindex.Helper
	templates bapi.Cache
}

func NewL7LogBackend(c lmaelastic.Client, cache bapi.Cache) bapi.L7LogBackend {
	b := &l7LogBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
		helper:    lmaindex.L7Logs(),
	}
	return b
}

// Create the given log in elasticsearch.
func (b *l7LogBackend) Create(ctx context.Context, i bapi.ClusterInfo, logs []v1.L7Log) (*v1.BulkResponse, error) {
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
		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(f)
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing L7 log: %s", err)
		return nil, fmt.Errorf("failed to write L7 log: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("L7 log bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists logs that match the given parameters.
func (b *l7LogBackend) List(ctx context.Context, i api.ClusterInfo, opts v1.L7LogParams) (*v1.List[v1.L7Log], error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	// Get the startFrom param, if any.
	startFrom := logtools.StartFrom(&opts)

	q, err := logtools.BuildQuery(b.helper, i, &opts)
	if err != nil {
		return nil, err
	}

	// Build the query, sorting by time.
	query := b.client.Search().
		Index(b.index(i)).
		Size(opts.QueryParams.GetMaxResults()).
		From(startFrom).
		Sort("end_time", true).
		Query(q)

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	logs := []v1.L7Log{}
	for _, h := range results.Hits.Hits {
		l := v1.L7Log{}
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

	return &v1.List[v1.L7Log]{
		Items:     logs,
		TotalHits: results.TotalHits(),
		AfterKey:  ak,
	}, nil
}

func (b *l7LogBackend) index(i bapi.ClusterInfo) string {
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("tigera_secure_ee_l7.%s.%s.*", i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("tigera_secure_ee_l7.%s.*", i.Cluster)
}

func (b *l7LogBackend) writeAlias(i bapi.ClusterInfo) string {
	// TODO: Not multi-tenant
	return fmt.Sprintf("tigera_secure_ee_l7.%s.", i.Cluster)
}
