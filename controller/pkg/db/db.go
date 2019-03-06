package db

import (
	"context"

	"github.com/tigera/intrusion-detection/controller/pkg/feed"
)

type IPSet interface {
	// Put a set of IPs in the database. IPs are sent as strings to avoid
	// overhead of decoding and encoding net.IP, since they are strings on the
	// wire to elastic.
	PutIPSet(ctx context.Context, name string, set feed.IPSet) error
	GetIPSet(name string) ([]string, error)
}

type SuspiciousIP interface {
	QueryIPSet(ctx context.Context, name string) (FlowLogIterator, error)
}

type FlowLogIterator interface {
	Next() bool
	Value() FlowLog
	Err() error
}

type Events interface {
	PutFlowLog(context.Context, FlowLog) error
}

type FlowLog struct {
	Time             int      `json:"time"`
	Type             string   `json:"type"`
	Description      string   `json:"description"`
	Severity         int      `json:"severity"`
	ID               string   `json:"id"`
	FlowLogIndex     *string  `json:"flow_log_index"`
	FlowLogID        *string  `json:"flow_log_id"`
	Protocol         *string  `json:"protocol"`
	SourceIP         *string  `json:"source_ip"`
	SourcePort       *int     `json:"source_port"`
	SourceNamespace  *string  `json:"source_namespace"`
	SourceName       *string  `json:"source_name"`
	DestIP           *string  `json:"dest_ip"`
	DestPort         *int     `json:"dest_port"`
	DestName         *string  `json:"dest_name"`
	FlowAction       *string  `json:"flow_action"`
	Feeds            []string `json:"feeds,omitempty"`
	SuspiciousPrefix *string  `json:"suspicious_prefix"`
}
