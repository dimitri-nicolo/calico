package pip

import (
	"context"

	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

type PIP interface {
	GetPolicyCalculator(ctx context.Context, r []ResourceChange) (policycalc.PolicyCalculator, error)
}
