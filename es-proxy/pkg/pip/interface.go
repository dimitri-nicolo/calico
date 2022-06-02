package pip

import (
	"context"
	"time"

	elastic "github.com/olivere/elastic/v7"

	pelastic "github.com/projectcalico/calico/lma/pkg/elastic"

	"github.com/projectcalico/calico/es-proxy/pkg/pip/policycalc"
)

type PIP interface {
	// This is the main entrypoint into PIP.
	GetFlows(ctx context.Context, params *PolicyImpactParams, flowFilter pelastic.FlowFilter) (*FlowLogResults, error)

	// The following public interface methods are here more for convenience than anything else. The PIPHandler
	// should just use GetCompositeAggrFlows().
	GetPolicyCalculator(ctx context.Context, r *PolicyImpactParams) (policycalc.PolicyCalculator, error)
	SearchAndProcessFlowLogs(
		ctx context.Context,
		query *pelastic.CompositeAggregationQuery,
		startAfterKey pelastic.CompositeAggregationKey,
		calc policycalc.PolicyCalculator,
		limit int32,
		impactedOnly bool,
		flowFilter pelastic.FlowFilter,
	) (<-chan ProcessedFlows, <-chan error)
}

type PolicyImpactParams struct {
	ResourceActions []ResourceChange `json:"resourceActions"`
	FromTime        *time.Time       `json:"-"`
	ToTime          *time.Time       `json:"-"`
	Query           elastic.Query    `json:"-"`
	ClusterName     string           `json:"-"`
	DocumentIndex   string           `json:"-"`
	Limit           int32            `json:"-"`
	ImpactedOnly    bool             `json:"-"`
}
