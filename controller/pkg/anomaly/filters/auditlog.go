// Copyright 2019 Tigera Inc. All rights reserved.

package filters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

const (
	Pod          = "Pod"
	ReplicaSet   = "ReplicaSet"
	AnyNamespace = "*"

	CreatedBefore = time.Hour * 1
	CreatedAfter  = 0
	DeletedBefore = time.Minute * 5
	DeletedAfter  = time.Minute * 5
)

var auditKeyTypes = map[string]string{
	"source_name":      Pod,
	"dest_name":        Pod,
	"source_name_aggr": ReplicaSet,
	"dest_name_aggr":   ReplicaSet,
}

type auditLogFilter struct {
	auditLogDB db.AuditLog
}

func NewAuditLog(al db.AuditLog) Filter {
	return &auditLogFilter{
		auditLogDB: al,
	}
}

func (al *auditLogFilter) Filter(ctx context.Context, in []elastic.RecordSpec) ([]elastic.RecordSpec, error) {
	cache := newAuditLogCache(al.auditLogDB)

	var out []elastic.RecordSpec

	for _, r := range in {
		if filtered, err := cache.areKeysFiltered(ctx, getAuditKeys(r)...); err != nil {
			return nil, err
		} else if !filtered {
			out = append(out, r)
		}
	}

	return out, nil
}

type auditKey struct {
	timestamp  time.Time
	bucketSpan int
	kind       string
	namespace  string
	name       string
}

func (k auditKey) String() string {
	return fmt.Sprintf("%d/%d/%q/%q/%q", k.timestamp.UnixNano(), k.bucketSpan, k.kind, k.namespace, k.name)
}

func getInfluencer(r elastic.RecordSpec, fieldName string) string {
	for _, i := range r.Influencers {
		if i.FieldName == fieldName {
			for _, v := range i.FieldValues {
				if res, ok := v.(string); ok {
					return res
				}
			}
		}
	}

	return ""
}

func getNamespace(r elastic.RecordSpec, fieldName string) string {
	var namespaceKey string
	switch {
	case strings.HasPrefix(fieldName, "source_"):
		namespaceKey = "source_namespace"
	case strings.HasPrefix(fieldName, "dest_"):
		namespaceKey = "dest_namespace"
	default:
		return AnyNamespace
	}

	namespace := getInfluencer(r, namespaceKey)
	if namespace != "" {
		return namespace
	}

	return AnyNamespace
}

func getAuditKeys(r elastic.RecordSpec) []auditKey {
	var keys []auditKey

	if partitionFieldType, ok := auditKeyTypes[r.PartitionFieldName]; ok {
		keys = append(keys, auditKey{
			timestamp:  r.Timestamp.Time,
			bucketSpan: r.BucketSpan,
			kind:       partitionFieldType,
			namespace:  getNamespace(r, r.PartitionFieldName),
			name:       r.PartitionFieldValue,
		})
	}

	if overFieldType, ok := auditKeyTypes[r.OverFieldName]; ok {
		keys = append(keys, auditKey{
			timestamp:  r.Timestamp.Time,
			bucketSpan: r.BucketSpan,
			kind:       overFieldType,
			namespace:  getNamespace(r, r.OverFieldName),
			name:       r.OverFieldValue,
		})
	}

	if byFieldType, ok := auditKeyTypes[r.ByFieldName]; ok {
		keys = append(keys, auditKey{
			timestamp:  r.Timestamp.Time,
			bucketSpan: r.BucketSpan,
			kind:       byFieldType,
			namespace:  getNamespace(r, r.ByFieldName),
			name:       r.ByFieldValue,
		})
	}

	return keys
}
