package pip

import v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

const (
	PreviewActionPass    = "pass"
	PreviewActionAllow   = "allow"
	PreviewActionDeny    = "deny"
	PreviewActionUnknown = "unknown"

	ChangeActionCreate = "create"
	ChangeActionDelete = "delete"
	ChangeActionUpdate = "update"
)

type NetworkPolicyChange struct {
	ChangeAction  string           `json:"action"`
	NetworkPolicy v3.NetworkPolicy `json:"policy"`
}

type contextKey int

const PolicyImpactContextKey contextKey = iota
