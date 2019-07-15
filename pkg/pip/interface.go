package pip

import (
	"context"
	"time"

	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

type PIP interface {
	GetPolicyCalculator(ctx context.Context, r *PolicyImpactParams) (policycalc.PolicyCalculator, error)
}

type PolicyImpactParams struct {
	ResourceActions []ResourceChange `json:"resourceActions"`
	FromTime        *time.Time       `json:"-"`
	ToTime          *time.Time       `json:"-"`
}
