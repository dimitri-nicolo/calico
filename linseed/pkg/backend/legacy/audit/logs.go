// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type auditLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	templates bapi.Cache
}

func NewBackend(c lmaelastic.Client, cache bapi.Cache) bapi.AuditBackend {
	return &auditLogBackend{
		client:    c.Backend(),
		lmaclient: c,
		templates: cache,
	}
}

// Create the given logs in elasticsearch.
func (b *auditLogBackend) Create(ctx context.Context, kind v1.AuditLogType, i bapi.ClusterInfo, logs []v1.AuditLog) (*v1.BulkResponse, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	var logType bapi.LogsType
	switch kind {
	case v1.AuditLogTypeEE:
		logType = bapi.AuditEELogs
	case v1.AuditLogTypeKube:
		logType = bapi.AuditKubeLogs
	case "":
		return nil, fmt.Errorf("no audit log type provided on List request")
	default:
		return nil, fmt.Errorf("invalid audit log type: %s", kind)
	}

	err := b.templates.InitializeIfNeeded(ctx, logType, i)
	if err != nil {
		return nil, err
	}

	// Determine the index to write to using an alias
	alias := b.writeAlias(kind, i)
	log.Debugf("Writing audit logs in bulk to alias %s", alias)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		bs, err := f.MarshalJSON()
		if err != nil {
			return nil, err
		}

		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(alias).Doc(string(bs))
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing log: %s", err)
		return nil, fmt.Errorf("failed to write log: %s", err)
	}
	fields := logrus.Fields{
		"succeeded": len(resp.Succeeded()),
		"failed":    len(resp.Failed()),
	}
	log.WithFields(fields).Debugf("Audit log bulk request complete: %+v", resp)

	return &v1.BulkResponse{
		Total:     len(resp.Items),
		Succeeded: len(resp.Succeeded()),
		Failed:    len(resp.Failed()),
		Errors:    v1.GetBulkErrors(resp),
	}, nil
}

// List lists logs that match the given parameters.
func (b *auditLogBackend) List(ctx context.Context, i api.ClusterInfo, opts *v1.AuditLogParams) (*v1.List[v1.AuditLog], error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	switch opts.Type {
	case v1.AuditLogTypeEE:
	case v1.AuditLogTypeKube:
	case v1.AuditLogTypeAny:
	case "":
		return nil, fmt.Errorf("no audit log type provided on List request")
	default:
		return nil, fmt.Errorf("invalid audit log type: %s", opts.Type)
	}

	// Get the startFrom param, if any.
	startFrom, err := logtools.StartFrom(opts)
	if err != nil {
		return nil, err
	}

	// Build the query.
	query := b.client.Search().
		Index(b.index(opts.Type, i)).
		Size(opts.GetMaxPageSize()).
		From(startFrom).
		Query(b.buildQuery(i, opts)).
		Sort("stageTimestamp", false)

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	events := []v1.AuditLog{}
	for _, h := range results.Hits.Hits {
		e := v1.AuditLog{}
		err = json.Unmarshal(h.Source, &e)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling audit log")
			continue
		}
		events = append(events, e)
	}

	return &v1.List[v1.AuditLog]{
		TotalHits: results.TotalHits(),
		Items:     events,
		AfterKey:  logtools.NextStartFromAfterKey(opts, len(results.Hits.Hits), startFrom),
	}, nil
}

// buildQuery builds an elastic query using the given parameters.
func (b *auditLogBackend) buildQuery(i bapi.ClusterInfo, opts *v1.AuditLogParams) elastic.Query {
	query := elastic.NewBoolQuery()

	// Time-range based query.
	var start, end time.Time
	if opts.TimeRange != nil {
		start = opts.TimeRange.From
		end = opts.TimeRange.To
		query.Filter(elastic.NewRangeQuery("requestReceivedTimestamp").Gt(start).Lte(end))
	}

	// Check if any resource kinds were specified.
	if len(opts.Kinds) > 0 {
		values := []interface{}{}
		for _, a := range opts.Kinds {
			values = append(values, a)
		}
		query.Filter(elastic.NewTermsQuery("objectRef.resource", values...))
	}

	// Match on author.
	if len(opts.Authors) > 0 {
		values := []interface{}{}
		for _, a := range opts.Authors {
			values = append(values, a)
		}
		query.Filter(elastic.NewTermsQuery("user.username", values...))
	}

	// Match on verb.
	if len(opts.Verbs) > 0 {
		values := []interface{}{}
		for _, a := range opts.Verbs {
			values = append(values, a)
		}
		query.Must(elastic.NewTermsQuery("verb", values...))
	}

	// Match on object.
	if opts.ObjectRef != nil {
		if opts.ObjectRef.Name != "" {
			query.Filter(elastic.NewTermQuery("objectRef.name", opts.ObjectRef.Name))
		}
		if opts.ObjectRef.Namespace != "" {
			query.Filter(elastic.NewTermQuery("objectRef.namespace", opts.ObjectRef.Namespace))
		}
	}

	// Match on response codes.
	if len(opts.ResponseCodes) > 0 {
		values := []interface{}{}
		for _, a := range opts.ResponseCodes {
			values = append(values, a)
		}
		query.Filter(elastic.NewTermsQuery("responseStatus.code", values...))
	}

	// Exclude any empty logs.
	query.MustNot(
		elastic.NewTermQuery("responseObject.metadata", "{}"),
		elastic.NewTermQuery("objectRef", "{}"),
		elastic.NewTermQuery("RequestObject", "{}"),
	)

	// Only match on RequestResponse level audit logs.
	query.Must(elastic.NewMatchQuery("level", "RequestResponse"))

	// Only match on logs who have a complete response.
	query.Must(elastic.NewMatchQuery("stage", "ResponseComplete"))

	return query
}

func (b *auditLogBackend) index(kind v1.AuditLogType, i bapi.ClusterInfo) string {
	base := fmt.Sprintf("tigera_secure_ee_audit_%s", kind)
	if kind == v1.AuditLogTypeAny {
		// Return both kube and EE logs.
		base = "tigera_secure_ee_audit_*"
	}
	if i.Tenant != "" {
		// If a tenant is provided, then we must include it in the index.
		return fmt.Sprintf("%s.%s.%s.*", base, i.Tenant, i.Cluster)
	}

	// Otherwise, this is a single-tenant cluster and we only need the cluster.
	return fmt.Sprintf("%s.%s.*", base, i.Cluster)
}

func (b *auditLogBackend) writeAlias(kind v1.AuditLogType, i bapi.ClusterInfo) string {
	return fmt.Sprintf("tigera_secure_ee_audit_%s.%s.", kind, i.Cluster)
}
