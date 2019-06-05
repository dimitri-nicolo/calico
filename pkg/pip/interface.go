package pip

import (
	"context"

	"github.com/tigera/es-proxy/pkg/pip/flow"
)

type PIP interface {
	CalculateFlowImpact(ctx context.Context, npcs []NetworkPolicyChange, flows []flow.Flow) ([]flow.Flow, error)
}
