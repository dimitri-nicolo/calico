package flows

type FlowLog struct {
	Time             int64    `json:"time"`
	Type             string   `json:"type"`
	Description      string   `json:"description"`
	Severity         int      `json:"severity"`
	ID               string   `json:"id"`
	FlowLogIndex     string   `json:"flow_log_index"`
	FlowLogID        string   `json:"flow_log_id"`
	Protocol         string   `json:"protocol"`
	SourceIP         *string  `json:"source_ip"`
	SourcePort       *int64   `json:"source_port"`
	SourceNamespace  string   `json:"source_namespace"`
	SourceName       string   `json:"source_name"`
	DestIP           *string  `json:"dest_ip"`
	DestPort         *int64   `json:"dest_port"`
	DestNamespace    string   `json:"dest_namespace"`
	DestName         string   `json:"dest_name"`
	FlowAction       string   `json:"flow_action"`
	Feeds            []string `json:"feeds,omitempty"`
	SuspiciousPrefix *string  `json:"suspicious_prefix"`
}
