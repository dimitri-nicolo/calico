package db

import "context"

type IPSet interface {
	// Put a set of IPs in the database. IPs are sent as strings to avoid
	// overhead of decoding and encoding net.IP, since they are strings on the
	// wire to elastic.
	PutIPSet(ctx context.Context, name string, set []string) error
	GetIPSet(name string) ([]string, error)
}

type SuspiciousIP interface {
	QueryIPSet(ctx context.Context, name string) ([]FlowLog, error)
}

type Events interface {
	PutFlowLog(context.Context, FlowLog) error
}

type FlowLog struct {
	SourceIP   string `json:"source_ip"`
	SourceName string `json:"source_name"`
	DestIP     string `json:"dest_ip"`
	DestName   string `json:"dest_name"`
	StartTime  int    `json:"start_time"`
	EndTime    int    `json:"end_time"`
}


