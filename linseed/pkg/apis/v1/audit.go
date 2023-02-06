// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

// AuditLogParams provide query options for listing audit logs.
type AuditLogParams struct {
	QueryParams `json:",inline" validate:"required"`
	Type        AuditLogType `json:"type"`
}

type AuditLogType string

const (
	AuditLogTypeKube AuditLogType = "kube"
	AuditLogTypeEE   AuditLogType = "ee"
)
