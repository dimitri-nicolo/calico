package l7

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
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
	startFrom := b.startFrom(opts)

	q, err := b.buildQuery(i, opts)
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

// startFrom parses the given parameters to determine which log to start from in the ES query.
func (b *l7LogBackend) startFrom(opts v1.L7LogParams) int {
	if opts.AfterKey != nil {
		if val, ok := opts.AfterKey["startFrom"]; ok {
			switch v := val.(type) {
			case string:
				if sf, err := strconv.Atoi(v); err != nil {
					return sf
				} else {
					logrus.WithField("val", v).Warn("Could not parse startFrom as an integer")
				}
			case float64:
				logrus.WithField("val", val).Info("Handling float64 startFrom")
				return int(v)
			case int:
				logrus.WithField("val", val).Info("Handling int startFrom")
				return v
			default:
				logrus.WithField("val", val).Infof("Unexpected type (%T) for startFrom, will not perform paging", val)
			}
		}
	}
	logrus.Debug("Starting query from 0")
	return 0
}

// buildQuery builds an elastic query using the given parameters.
func (b *l7LogBackend) buildQuery(i bapi.ClusterInfo, opts v1.L7LogParams) (elastic.Query, error) {
	// Parse times from the request. We default to a time-range query
	// if no other search parameters are given.
	var start, end time.Time
	if opts.QueryParams.TimeRange != nil {
		start = opts.QueryParams.TimeRange.From
		end = opts.QueryParams.TimeRange.To
	} else {
		// Default to the latest 5 minute window.
		start = time.Now().Add(-5 * time.Minute)
		end = time.Now()
	}
	constraints := []elastic.Query{
		lmaindex.L7Logs().NewTimeRangeQuery(start, end),
	}

	// If RBAC constraints were given, add them in.
	// TODO: We should split the query building and the authz check. Run the authz in the frontend.
	if len(opts.Permissions) > 0 {
		rbacQuery, err := lmaindex.L7Logs().NewRBACQuery(opts.Permissions)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, rbacQuery)
	}

	// If a selector was provided, parse it and add it in.
	if len(opts.Selector) > 0 {
		selQuery, err := lmaindex.L7Logs().NewSelectorQuery(opts.Selector)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, selQuery)
	}

	if len(constraints) == 1 {
		// This is just a time-range query. We don't need to join multiple
		// constraints together.
		return constraints[0], nil
	}

	// We need to perform a boolean query with multiple constraints.
	return elastic.NewBoolQuery().Filter(constraints...), nil
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
