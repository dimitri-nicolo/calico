// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"time"

	"github.com/projectcalico/felix/calc"
)

// L7 Update represents the data that is sent to us straight from the envoy logs?
type L7Update struct {
	Tuple Tuple
	SrcEp *calc.EndpointData
	DstEp *calc.EndpointData

	Duration         int
	DurationMax      int
	BytesReceived    int
	BytesSent        int
	ServiceName      string
	ServiceNamespace string
	ServicePort      int
	ResponseCode     string
	Method           string
	Domain           string
	Path             string
	UserAgent        string
	Type             string
	Count            int
}

// L7Log represents the log we are pushing to fluentd/elastic.
type L7Log struct {
	StartTime        int64               `json:"start_time"`
	EndTime          int64               `json:"end_time"`
	DurationMean     time.Duration       `json:"duration_mean"`
	DurationMax      time.Duration       `json:"duration_max"`
	BytesIn          int                 `json:"bytes_in"`
	BytesOut         int                 `json:"bytes_out"`
	Count            int                 `json:"count"`
	SrcNameAggr      string              `json:"src_name_aggr"`
	SrcNamespace     string              `json:"src_namespace"`
	SrcType          FlowLogEndpointType `json:"src_type"`
	ServiceName      string              `json:"dest_service_name"`
	ServiceNamespace string              `json:"dest_service_namespace"`
	ServicePort      int                 `json:"dest_service_port"`
	DstNameAggr      string              `json:"dest_name_aggr"`
	DstNamespace     string              `json:"dest_namespace"`
	DstType          FlowLogEndpointType `json:"dest_type"`
	Method           string              `json:"method"`
	UserAgent        string              `json:"user_agent"`
	URL              string              `json:"url"`
	ResponseCode     string              `json:"response_code"`
	Type             string              `json:"type"`
}

// L7Meta represents the identifiable information for an L7 log.
type L7Meta struct {
	SrcNameAggr      string
	SrcNamespace     string
	SrcType          FlowLogEndpointType
	DstNameAggr      string
	DstNamespace     string
	DstType          FlowLogEndpointType
	ServiceName      string
	ServiceNamespace string
	ServicePort      int
	ResponseCode     string
	Method           string
	Domain           string
	Path             string
	UserAgent        string
	Type             string
}

// L7Spec represents the stats and collections of L7 data
type L7Spec struct {
	Duration      int
	DurationMax   int
	BytesReceived int
	BytesSent     int
	Count         int
}

func (a *L7Spec) Merge(b L7Spec) {
	a.Duration = a.Duration + b.Duration
	if b.DurationMax > a.DurationMax {
		a.DurationMax = b.DurationMax
	}
	a.BytesReceived = a.BytesReceived + b.BytesReceived
	a.BytesSent = a.BytesSent + b.BytesSent
	a.Count = a.Count + b.Count
}

type L7Data struct {
	L7Meta
	L7Spec
}

func (ld L7Data) ToL7Log(startTime, endTime time.Time) *L7Log {
	res := &L7Log{
		StartTime:        startTime.Unix(),
		EndTime:          endTime.Unix(),
		BytesIn:          ld.BytesReceived,
		BytesOut:         ld.BytesSent,
		Count:            ld.Count,
		SrcNameAggr:      ld.SrcNameAggr,
		SrcNamespace:     ld.SrcNamespace,
		SrcType:          ld.SrcType,
		DstNameAggr:      ld.DstNameAggr,
		DstNamespace:     ld.DstNamespace,
		DstType:          ld.DstType,
		ServiceName:      ld.ServiceName,
		ServiceNamespace: ld.ServiceNamespace,
		ServicePort:      ld.ServicePort,
		Method:           ld.Method,
		UserAgent:        ld.UserAgent,
		ResponseCode:     ld.ResponseCode,
		Type:             ld.Type,
	}

	// Calculate and convert durations
	res.DurationMean = time.Duration(ld.Duration/ld.Count) * time.Millisecond
	res.DurationMax = time.Duration(ld.DurationMax) * time.Millisecond

	// Create the URL from the domain and path
	// Path is expected to have a leading "/" character.
	if ld.Domain != flowLogFieldNotIncluded && ld.Path != flowLogFieldNotIncluded {
		res.URL = fmt.Sprintf("%s%s", ld.Domain, ld.Path)
	} else if ld.Domain != flowLogFieldNotIncluded && ld.Path == flowLogFieldNotIncluded {
		res.URL = ld.Domain
	}

	return res
}
