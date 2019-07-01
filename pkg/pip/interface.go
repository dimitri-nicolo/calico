package pip

import (
	"context"

	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

type PIP interface {
	CalculateFlowImpact(ctx context.Context, f *policycalc.Flow) (processed bool, before, after policycalc.Action)
}
