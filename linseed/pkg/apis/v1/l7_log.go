package v1

// L7LogParams define querying parameters to retrieve L7 logs
type L7LogParams struct {
	QueryParams        `json:",inline" validate:"required"`
	LogSelectionParams `json:",inline"`
}

// L7Log is the structure which defines a single instance of an L7 flow log.
type L7Log struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`

	DurationMean int64 `json:"duration_mean"`
	DurationMax  int64 `json:"duration_max"`
	Latency      int64 `json:"latency"`
	BytesIn      int64 `json:"bytes_in"`
	BytesOut     int64 `json:"bytes_out"`
	Count        int64 `json:"count"`

	SourceNameAggr  string `json:"src_name_aggr"`
	SourceNamespace string `json:"src_namespace"`
	SourceType      string `json:"src_type"`
	SourcePortNum   int64  `json:"source_port_num"`

	DestServiceName      string `json:"dest_service_name"`
	DestServiceNamespace string `json:"dest_service_namespace"`

	// DestServicePort is the numerical value of the port exposed by the service which the connection is trying to reach.
	// Described in the Service resource specs as specs.[]ports.port. It will have the empty L7 Log numerical field value of 0 when:
	// - in rare cases where the Service is unavailable (ie. deleted or down) while processing the Service information
	DestServicePort int64 `json:"dest_service_port"`

	// DestServicePortName Name is the name of the port exposed by the Service which the connection is trying to reach.
	// Described in the Service resource specs as specs.[]ports.name. It will have the empty L7 Log character field value of '-' when:
	// - the optional port name field in the Service resource is not provided
	// - in rare cases where the Service is unavailable (ie. deleted or down) while processing the Service information
	DestServicePortName string `json:"dest_service_port_name"`
	DestNameAggr        string `json:"dest_name_aggr"`
	DestNamespace       string `json:"dest_namespace"`
	DestType            string `json:"dest_type"`
	DestPortNum         int64  `json:"dest_port_num"`

	URL          string `json:"url"`
	ResponseCode int32  `json:"response_code"`
	Method       string `json:"method"`
	UserAgent    string `json:"user_agent"`
	Type         string `json:"type"`
}
