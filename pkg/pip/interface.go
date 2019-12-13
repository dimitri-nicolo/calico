package pip

import (
	"context"
	"time"

	"github.com/olivere/elastic/v7"

	"github.com/tigera/es-proxy/pkg/pip/policycalc"
	pelastic "github.com/tigera/lma/pkg/elastic"
)

type PIP interface {
	// This is the main entrypoint into PIP.
	GetFlows(ctx context.Context, params *PolicyImpactParams) (*FlowLogResults, error)

	// The following public interface methods are here more for convenience than anything else. The PIPHandler
	// should just use GetFlows().
	GetPolicyCalculator(ctx context.Context, r *PolicyImpactParams) (policycalc.PolicyCalculator, error)
	SearchAndProcessFlowLogs(
		ctx context.Context,
		query *pelastic.CompositeAggregationQuery,
		startAfterKey pelastic.CompositeAggregationKey,
		calc policycalc.PolicyCalculator,
		limit int32,
	) (<-chan ProcessedFlows, <-chan error)
}

type PolicyImpactParams struct {
	ResourceActions []ResourceChange `json:"resourceActions"`
	FromTime        *time.Time       `json:"-"`
	ToTime          *time.Time       `json:"-"`
	Query           elastic.Query    `json:"-"`
	DocumentIndex   string           `json:"-"`
	Limit           int32            `json:"-"`
}
