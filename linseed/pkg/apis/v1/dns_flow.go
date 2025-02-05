package v1

// DNSFlowParams provide query options for listing DNS flows.
type DNSFlowParams struct {
	QueryParams `json:",inline" validate:"required"`
}

// DNSFlowKey are the identifiers for a DNS flow.
type DNSFlowKey struct {
	Source       Endpoint `json:"source"`
	ResponseCode string   `json:"response_code"`
	Cluster      string   `json:"cluster"`
}

// DNSFlow represents an aggregation of DNS logs from a given source with a given response code.
type DNSFlow struct {
	Key          DNSFlowKey       `json:"key"`
	LatencyStats *DNSLatencyStats `json:"latency_stats"`
	Count        int64            `json:"count"`
}

type DNSLatencyStats struct {
	MeanRequestLatency float64 `json:"mean_request_latency"`
	MaxRequestLatency  float64 `json:"max_request_latency"`
	MinRequestLatency  float64 `json:"min_request_latency"`

	// The number of logs used to generate this latency information.
	LatencyCount int `json:"latency_count"`
}
