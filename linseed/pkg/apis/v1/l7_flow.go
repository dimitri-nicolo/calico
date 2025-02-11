package v1

import (
	apim "k8s.io/apimachinery/pkg/types"
)

// L7FlowParams define querying parameters to retrieve L7 Flows
type L7FlowParams struct {
	QueryParams `json:",inline" validate:"required"`
}

// Defines a port on a Kuberentes Service.
type ServicePort struct {
	Service  apim.NamespacedName `json:"service"`
	Port     int64               `json:"port"`
	PortName string              `json:"port_name"`
}

type L7FlowKey struct {
	Cluster            string      `json:"cluster"`
	Source             Endpoint    `json:"source"`
	Destination        Endpoint    `json:"destination"`
	DestinationService ServicePort `json:"destination_service"`
	Protocol           string      `json:"protocol"`
}

type L7Stats struct {
	BytesIn      int64 `json:"bytes_in"`
	BytesOut     int64 `json:"bytes_out"`
	MeanDuration int64 `json:"mean_duration"`
	MinDuration  int64 `json:"min_duration"`
	MaxDuration  int64 `json:"max_duration"`
}

type L7Flow struct {
	Key      L7FlowKey `json:"key"`
	Code     int32     `json:"code"`
	LogCount int64     `json:"log_count"`
	Stats    *L7Stats  `json:"stats"`
}
