package flows_test

import (
	"fmt"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

func NewFlowLogBuilder() *flowLogBuilder {
	return &flowLogBuilder{
		// Initialize to an empty flow log.
		log: &v1.FlowLog{
			ProcessName: "-",
		},
	}
}

type flowLogBuilder struct {
	cluster string

	log *v1.FlowLog

	// For tracking how to build the log.
	randomFlowStats   bool
	randomPacketStats bool
}

func (b *flowLogBuilder) Copy() *flowLogBuilder {
	n := *b
	return &n
}

func (b *flowLogBuilder) Build() (*v1.FlowLog, error) {
	// If no start and end times were set, default them.
	if b.log.StartTime == 0 {
		b.WithStartTime(time.Now())
	}
	if b.log.EndTime == 0 {
		b.WithEndTime(time.Now())
	}

	if b.randomPacketStats {
		b.log.PacketsIn = 1
		b.log.PacketsOut = 2
		b.log.BytesIn = 32
		b.log.BytesOut = 128
	}
	if b.randomFlowStats {
		b.log.NumFlows = 1
		b.log.NumFlowsStarted = 3
		b.log.NumFlowsCompleted = 2
	}

	// Perform any validation here to ensure the log that we're building is legit.
	return b.log, nil
}

// ExpectedFlow returns a baseline flow to expect, given the flow log's configuration.
// Note that some fields on a Flow are aggregated, and so will need to be calculated based
// on the sum total of flow logs used to build the flow.
func (b *flowLogBuilder) ExpectedFlow() *v1.L3Flow {
	return &v1.L3Flow{
		Key: v1.L3FlowKey{
			Action:   b.log.Action,
			Reporter: b.log.Reporter,
			Protocol: b.log.Protocol,
			Source: v1.Endpoint{
				Namespace:      b.log.SourceNamespace,
				Type:           v1.EndpointType(b.log.SourceType),
				AggregatedName: b.log.SourceNameAggr,
			},
			Destination: v1.Endpoint{
				Namespace:      b.log.DestNamespace,
				Type:           v1.EndpointType(b.log.DestType),
				AggregatedName: b.log.DestNameAggr,
				Port:           *b.log.DestPort,
			},
		},
		TrafficStats: &v1.TrafficStats{},
		LogStats:     &v1.LogStats{},
		Service: &v1.Service{
			Name:      b.log.DestServiceName,
			Namespace: b.log.DestServiceNamespace,
			Port:      *b.log.DestServicePortNum,
			PortName:  b.log.DestServicePortName,
		},
	}
}

func (b *flowLogBuilder) WithSourceIP(ip string) *flowLogBuilder {
	b.log.SourceIP = testutils.StringPtr(ip)
	return b
}

func (b *flowLogBuilder) WithDestIP(ip string) *flowLogBuilder {
	b.log.DestIP = testutils.StringPtr(ip)
	return b
}

func (b *flowLogBuilder) WithProcessName(n string) *flowLogBuilder {
	b.log.ProcessName = n
	return b
}

func (b *flowLogBuilder) WithSourceName(n string) *flowLogBuilder {
	b.log.SourceNameAggr = n
	return b
}

func (b *flowLogBuilder) WithDestName(n string) *flowLogBuilder {
	b.log.DestNameAggr = n
	return b
}

func (b *flowLogBuilder) WithStartTime(t time.Time) *flowLogBuilder {
	b.log.StartTime = time.Now().Unix()
	return b
}

func (b *flowLogBuilder) WithEndTime(t time.Time) *flowLogBuilder {
	b.log.EndTime = time.Now().Unix()
	return b
}

func (b *flowLogBuilder) WithProtocol(p string) *flowLogBuilder {
	b.log.Protocol = p
	return b
}

func (b *flowLogBuilder) WithDestPort(port int) *flowLogBuilder {
	b.log.DestPort = testutils.Int64Ptr(int64(port))
	return b
}

func (b *flowLogBuilder) WithSourcePort(port int) *flowLogBuilder {
	b.log.SourcePort = testutils.Int64Ptr(int64(port))
	return b
}

func (b *flowLogBuilder) WithDestService(name string, port int) *flowLogBuilder {
	b.log.DestServiceName = name
	b.log.DestServicePortName = fmt.Sprintf("%d", port)
	b.log.DestServicePortNum = testutils.Int64Ptr(int64(port))
	return b
}

func (b *flowLogBuilder) WithCluster(c string) *flowLogBuilder {
	b.cluster = c
	return b
}

func (b *flowLogBuilder) WithReporter(r string) *flowLogBuilder {
	b.log.Reporter = r
	return b
}

func (b *flowLogBuilder) WithAction(a string) *flowLogBuilder {
	b.log.Action = a
	return b
}

// WithType sets both source and dest types at once.
func (b *flowLogBuilder) WithType(t string) *flowLogBuilder {
	b.log.DestType = t
	b.log.SourceType = t
	return b
}

func (b *flowLogBuilder) WithDestType(c string) *flowLogBuilder {
	b.log.DestType = c
	return b
}

func (b *flowLogBuilder) WithSourceType(c string) *flowLogBuilder {
	b.log.SourceType = c
	return b
}

// WithNamespace sets all namespace fields at once.
func (b *flowLogBuilder) WithNamespace(n string) *flowLogBuilder {
	b.log.SourceNamespace = n
	b.log.DestNamespace = n
	b.log.DestServiceNamespace = n
	return b
}

func (b *flowLogBuilder) WithSourceNamespace(n string) *flowLogBuilder {
	b.log.SourceNamespace = n
	return b
}

func (b *flowLogBuilder) WithDestNamespace(n string) *flowLogBuilder {
	b.log.DestNamespace = n
	b.log.DestServiceNamespace = n
	return b
}

func (b *flowLogBuilder) WithSourceLabels(labels ...string) *flowLogBuilder {
	b.log.SourceLabels = &v1.FlowLogLabels{
		Labels: labels,
	}
	return b
}

func (b *flowLogBuilder) WithDestLabels(labels ...string) *flowLogBuilder {
	b.log.DestLabels = &v1.FlowLogLabels{
		Labels: labels,
	}
	return b
}

func (b *flowLogBuilder) WithRandomFlowStats() *flowLogBuilder {
	b.randomFlowStats = true
	return b
}

func (b *flowLogBuilder) WithRandomPacketStats() *flowLogBuilder {
	b.randomPacketStats = true
	return b
}
