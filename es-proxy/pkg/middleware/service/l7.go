// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package service

import "time"

type l7Doc struct {
	ID     string `json:"id"`
	Index  string `json:"index"`
	Source L7Log  `json:"source"`
}

// subset from felix/collector/l7log_types.go
type L7Log struct {
	StartTime    int64         `json:"start_time"`
	EndTime      int64         `json:"end_time"`
	DurationMean time.Duration `json:"duration_mean"`
	Latency      int           `json:"latency"`
	BytesIn      int           `json:"bytes_in"`
	BytesOut     int           `json:"bytes_out"`
	Count        int           `json:"count"`

	SourceNameAggr string `json:"src_name_aggr"`

	ResponseCode string `json:"response_code"`
}
