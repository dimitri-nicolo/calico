// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/apis/audit"

	kaudit "k8s.io/apiserver/pkg/audit"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type backend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	// Tracks whether the backend has been initialized.
	initialized bool
}

func NewBackend(c lmaelastic.Client) bapi.AuditBackend {
	return &backend{
		client:    c.Backend(),
		lmaclient: c,
	}
}

func (b *backend) index(kind v1.AuditLogType, cluster string) string {
	return fmt.Sprintf("tigera_secure_ee_audit_%s.%s", kind, cluster)
}

func (b *backend) Initialize(ctx context.Context) error {
	var err error
	if !b.initialized {
		// Create a template with mappings for all new log indices.
		_, err = b.client.IndexPutTemplate("audit_template").
			BodyString(templates.AuditTemplate).
			Create(false).
			Do(ctx)
		if err != nil {
			return err
		}
		b.initialized = true
	}
	return nil
}

// Create the given logs in elasticsearch.
func (b *backend) Create(ctx context.Context, kind v1.AuditLogType, i bapi.ClusterInfo, logs []audit.Event) error {
	log := bapi.ContextLogger(i)

	// Initialize if we haven't yet.
	err := b.Initialize(ctx)
	if err != nil {
		return err
	}

	if i.Cluster == "" {
		return fmt.Errorf("no cluster ID on request")
	}

	// Determine the index to write to. It will be automatically created based on the configured
	// template if it does not already exist.
	index := b.index(kind, i.Cluster)
	log.Debugf("Writing audit logs in bulk to index %s", index)

	// Build a bulk request using the provided logs.
	bulk := b.client.Bulk()

	for _, f := range logs {
		// Kubernetes audit.Entry objects require special serialization that differs from the
		// default json implementation. So use that here. This is taken from the k8s source:
		// https://github.com/kubernetes/kubernetes/blob/v1.25.0/staging/src/k8s.io/apiserver/plugin/pkg/audit/log/backend.go#L76-L81
		groupVersion := schema.GroupVersion{
			Group:   "audit.k8s.io",
			Version: "v1",
		}
		encoder := kaudit.Codecs.LegacyCodec(groupVersion)
		bs, err := runtime.Encode(encoder, &f)
		if err != nil {
			return err
		}

		// Add this log to the bulk request.
		req := elastic.NewBulkIndexRequest().Index(index).Doc(string(bs))
		bulk.Add(req)
	}

	// Send the bulk request.
	resp, err := bulk.Do(ctx)
	if err != nil {
		log.Errorf("Error writing log: %s", err)
		return fmt.Errorf("failed to write log: %s", err)
	}

	log.WithField("count", len(logs)).Debugf("Wrote log to index: %+v", resp)

	return nil
}

// List lists logs that match the given parameters.
func (b *backend) List(ctx context.Context, i api.ClusterInfo, opts v1.AuditLogParams) ([]audit.Event, error) {
	log := bapi.ContextLogger(i)

	if i.Cluster == "" {
		return nil, fmt.Errorf("no cluster ID on request")
	}

	switch opts.Type {
	case v1.AuditLogTypeEE:
	case v1.AuditLogTypeKube:
	case "":
		return nil, fmt.Errorf("no audit log type provided on List request")
	default:
		return nil, fmt.Errorf("invalid audit log type: %s", opts.Type)
	}

	// Parse times from the request. We default to a time-range query
	// if no other search parameters are given.
	var start, end time.Time
	if opts.QueryParams != nil && opts.QueryParams.TimeRange != nil {
		start = opts.QueryParams.TimeRange.From
		end = opts.QueryParams.TimeRange.To
	} else {
		// Default to the latest 5 minute window.
		start = time.Now().Add(-5 * time.Minute)
		end = time.Now()
	}
	q := elastic.NewRangeQuery("requestReceivedTimestamp").Gt(start).Lte(end)

	// Build the query.
	query := b.client.Search().
		Index(b.index(opts.Type, i.Cluster)).
		Size(opts.QueryParams.GetMaxResults()).
		Query(q)

	results, err := query.Do(ctx)
	if err != nil {
		return nil, err
	}

	events := []audit.Event{}
	for _, h := range results.Hits.Hits {
		e := audit.Event{}
		err = json.Unmarshal(h.Source, &e)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling audit log")
			continue
		}
		events = append(events, e)
	}

	return events, nil
}
