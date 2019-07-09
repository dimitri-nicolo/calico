package pip

const (
	PreviewActionPass    = "pass"
	PreviewActionAllow   = "allow"
	PreviewActionDeny    = "deny"
	PreviewActionUnknown = "unknown"

	ChangeActionCreate = "create"
	ChangeActionDelete = "delete"
	ChangeActionUpdate = "update"
)

type contextKey int

const PolicyImpactContextKey contextKey = iota
