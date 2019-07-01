package pip

import (
	"github.com/tigera/compliance/pkg/resources"
)

const (
	PreviewActionPass    = "pass"
	PreviewActionAllow   = "allow"
	PreviewActionDeny    = "deny"
	PreviewActionUnknown = "unknown"

	ChangeActionCreate = "create"
	ChangeActionDelete = "delete"
	ChangeActionUpdate = "update"
)

type ResourceChange struct {
	Action   string             `json:"action"`
	Resource resources.Resource `json:"resource"`
}

type contextKey int

const PolicyImpactContextKey contextKey = iota
