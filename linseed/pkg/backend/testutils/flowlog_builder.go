package testutils

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/flows"
	"github.com/projectcalico/calico/linseed/pkg/testutils"
	lmaapi "github.com/projectcalico/calico/lma/pkg/api"
)

func NewFlowLogBuilder() *FlowLogBuilder {
	return &FlowLogBuilder{
		// Initialize to an empty flow log.
		activeLog: &v1.FlowLog{},
	}
}

type FlowLogBuilder struct {
	cluster string

	activeLog *v1.FlowLog

	// For tracking how to build the log.
	randomFlowStats   bool
	randomPacketStats bool

	// Tracking logs that we've built.
	logs []v1.FlowLog
}

func (b *FlowLogBuilder) Copy() *FlowLogBuilder {
	n := *b
	return &n
}

func (b *FlowLogBuilder) Clear() {
	b.activeLog = &v1.FlowLog{}
}

func (b *FlowLogBuilder) Build() (*v1.FlowLog, error) {
	// If no start and end times were set, default them.
	if b.activeLog.StartTime == 0 {
		b.WithStartTime(time.Now())
	}
	if b.activeLog.EndTime == 0 {
		b.WithEndTime(time.Now())
	}

	if b.randomPacketStats {
		b.activeLog.PacketsIn = 1
		b.activeLog.PacketsOut = 2
		b.activeLog.BytesIn = 32
		b.activeLog.BytesOut = 128
	}
	if b.randomFlowStats {
		b.activeLog.NumFlows = 1
		b.activeLog.NumFlowsStarted = 3
		b.activeLog.NumFlowsCompleted = 2
	}

	// Keep track of the logs that have been built so that we can
	// produce an expected flow from them if requested. We take a copy
	// so that the caller can modify the next iteration of this log if desired.
	cp := *b.activeLog
	b.logs = append(b.logs, cp)

	// Perform any validation here to ensure the log that we're building is legit.
	return &cp, nil
}

// ExpectedFlow returns a baseline flow to expect, given the flow log's configuration.
// Note that some fields on a Flow are aggregated, and so will need to be calculated based
// on the sum total of flow logs used to build the flow.
// Our aggregation logic within the builder is fairly limited.
func (b *FlowLogBuilder) ExpectedFlow(t *testing.T) *v1.L3Flow {
	// Initialize the flow with identifying information. For now, we
	// don't support multiple flows from a single builder, so we assume
	// all of the logs have the same Key fields.
	f := &v1.L3Flow{
		Key: v1.L3FlowKey{
			Action:   v1.FlowAction(b.activeLog.Action),
			Reporter: v1.FlowReporter(b.activeLog.Reporter),
			Protocol: b.activeLog.Protocol,
			Source: v1.Endpoint{
				Namespace:      b.activeLog.SourceNamespace,
				Type:           v1.EndpointType(b.activeLog.SourceType),
				AggregatedName: b.activeLog.SourceNameAggr,
			},
			Destination: v1.Endpoint{
				Namespace:      b.activeLog.DestNamespace,
				Type:           v1.EndpointType(b.activeLog.DestType),
				AggregatedName: b.activeLog.DestNameAggr,
				Port:           *b.activeLog.DestPort,
			},
		},
		TrafficStats: &v1.TrafficStats{},
		LogStats:     &v1.LogStats{},
		HTTPStats:    &v1.HTTPStats{},
		Service: &v1.Service{
			Name:      b.activeLog.DestServiceName,
			Namespace: b.activeLog.DestServiceNamespace,
			Port:      *b.activeLog.DestServicePortNum,
			PortName:  b.activeLog.DestServicePortName,
		},
	}

	slt := flows.NewLabelTracker()
	dlt := flows.NewLabelTracker()

	// Now populate the expected non-identifying information based on the logs we
	// have created, simulating aggregation done by ES.
	for _, log := range b.logs {
		f.TrafficStats.BytesIn += log.BytesIn
		f.TrafficStats.BytesOut += log.BytesOut
		f.TrafficStats.PacketsIn += log.PacketsIn
		f.TrafficStats.PacketsOut += log.PacketsOut
		f.LogStats.Completed += log.NumFlowsCompleted
		f.LogStats.Started += log.NumFlowsStarted
		f.LogStats.LogCount += log.NumFlows
		f.LogStats.FlowLogCount += 1
		f.HTTPStats.AllowedIn += log.HTTPRequestsAllowedIn
		f.HTTPStats.DeniedIn += log.HTTPRequestsDeniedIn

		// Update trackers with label information.
		for _, l := range log.SourceLabels.Labels {
			labelParts := strings.Split(l, "=")
			key := labelParts[0]
			value := labelParts[1]
			slt.Add(key, value, log.NumFlows)
		}
		for _, l := range log.DestLabels.Labels {
			labelParts := strings.Split(l, "=")
			key := labelParts[0]
			value := labelParts[1]
			dlt.Add(key, value, log.NumFlows)
		}

	}

	// Set labels.
	f.SourceLabels = slt.Labels()
	f.DestinationLabels = dlt.Labels()

	// Add in expected policies. Right now, we don't support aggregation
	// of policies across multiple logs in this builder, and we assume
	// every log in the flow has the same policies.
	if b.activeLog != nil && b.activeLog.Policies != nil {
		for _, p := range b.activeLog.Policies.AllPolicies {
			if f.Policies == nil {
				f.Policies = make([]v1.Policy, 0)
			}
			h, err := lmaapi.PolicyHitFromFlowLogPolicyString(p, 1)
			require.NoError(t, err)

			name := h.Name()
			if h.IsProfile() {
				name = fmt.Sprintf("kns.%s", name)
			}

			pol := v1.Policy{
				Tier:      h.Tier(),
				Name:      name,
				Namespace: h.Namespace(),
				Action:    string(h.Action()),
				Count:     f.LogStats.FlowLogCount,
				RuleID:    h.RuleIdIndex(),
				IsProfile: h.IsProfile(),
				IsStaged:  h.IsStaged(),
			}
			f.Policies = append(f.Policies, pol)
		}
	}

	return f
}

func (b *FlowLogBuilder) WithSourceIP(ip string) *FlowLogBuilder {
	b.activeLog.SourceIP = testutils.StringPtr(ip)
	return b
}

func (b *FlowLogBuilder) WithDestIP(ip string) *FlowLogBuilder {
	b.activeLog.DestIP = testutils.StringPtr(ip)
	return b
}

func (b *FlowLogBuilder) WithProcessName(n string) *FlowLogBuilder {
	b.activeLog.ProcessName = n
	return b
}

func (b *FlowLogBuilder) WithSourceName(n string) *FlowLogBuilder {
	b.activeLog.SourceNameAggr = n
	return b
}

func (b *FlowLogBuilder) WithDestName(n string) *FlowLogBuilder {
	b.activeLog.DestNameAggr = n
	return b
}

func (b *FlowLogBuilder) WithStartTime(t time.Time) *FlowLogBuilder {
	b.activeLog.StartTime = time.Now().Unix()
	return b
}

func (b *FlowLogBuilder) WithEndTime(t time.Time) *FlowLogBuilder {
	b.activeLog.EndTime = time.Now().Unix()
	return b
}

func (b *FlowLogBuilder) WithProtocol(p string) *FlowLogBuilder {
	b.activeLog.Protocol = p
	return b
}

func (b *FlowLogBuilder) WithDestPort(port int) *FlowLogBuilder {
	b.activeLog.DestPort = testutils.Int64Ptr(int64(port))
	return b
}

func (b *FlowLogBuilder) WithSourcePort(port int) *FlowLogBuilder {
	b.activeLog.SourcePort = testutils.Int64Ptr(int64(port))
	return b
}

func (b *FlowLogBuilder) WithDestService(name string, port int) *FlowLogBuilder {
	b.activeLog.DestServiceName = name
	b.activeLog.DestServicePortName = fmt.Sprintf("%d", port)
	b.activeLog.DestServicePortNum = testutils.Int64Ptr(int64(port))
	return b
}

func (b *FlowLogBuilder) WithCluster(c string) *FlowLogBuilder {
	b.cluster = c
	return b
}

func (b *FlowLogBuilder) WithReporter(r string) *FlowLogBuilder {
	b.activeLog.Reporter = r
	return b
}

func (b *FlowLogBuilder) WithAction(a string) *FlowLogBuilder {
	b.activeLog.Action = a
	return b
}

func (b *FlowLogBuilder) WithPolicies(p ...string) *FlowLogBuilder {
	b.activeLog.Policies = &v1.FlowLogPolicy{AllPolicies: p}
	return b
}

func (b *FlowLogBuilder) WithPolicy(p string) *FlowLogBuilder {
	if b.activeLog.Policies == nil {
		b.activeLog.Policies = &v1.FlowLogPolicy{
			AllPolicies: []string{},
		}
	}
	b.activeLog.Policies.AllPolicies = append(b.activeLog.Policies.AllPolicies, p)
	return b
}

// WithType sets both source and dest types at once.
func (b *FlowLogBuilder) WithType(t string) *FlowLogBuilder {
	b.activeLog.DestType = t
	b.activeLog.SourceType = t
	return b
}

func (b *FlowLogBuilder) WithDestType(c string) *FlowLogBuilder {
	b.activeLog.DestType = c
	return b
}

func (b *FlowLogBuilder) WithSourceType(c string) *FlowLogBuilder {
	b.activeLog.SourceType = c
	return b
}

// WithNamespace sets all namespace fields at once.
func (b *FlowLogBuilder) WithNamespace(n string) *FlowLogBuilder {
	b.activeLog.SourceNamespace = n
	b.activeLog.DestNamespace = n
	b.activeLog.DestServiceNamespace = n
	return b
}

func (b *FlowLogBuilder) WithSourceNamespace(n string) *FlowLogBuilder {
	b.activeLog.SourceNamespace = n
	return b
}

func (b *FlowLogBuilder) WithDestNamespace(n string) *FlowLogBuilder {
	b.activeLog.DestNamespace = n
	b.activeLog.DestServiceNamespace = n
	return b
}

func (b *FlowLogBuilder) WithSourceLabels(labels ...string) *FlowLogBuilder {
	b.activeLog.SourceLabels = &v1.FlowLogLabels{
		Labels: labels,
	}
	return b
}

func (b *FlowLogBuilder) WithDestLabels(labels ...string) *FlowLogBuilder {
	b.activeLog.DestLabels = &v1.FlowLogLabels{
		Labels: labels,
	}
	return b
}

func (b *FlowLogBuilder) WithRandomFlowStats() *FlowLogBuilder {
	b.randomFlowStats = true
	return b
}

func (b *FlowLogBuilder) WithRandomPacketStats() *FlowLogBuilder {
	b.randomPacketStats = true
	return b
}
