// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import (
	"time"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

type RuntimeReport struct {
	ID      string `json:"id,omitempty"`
	Tenant  string `json:"tenant,omitempty"`
	Cluster string `json:"cluster,omitempty"`
	Report  Report `json:",inline"`
}

// Report is a copy of https://github.com/tigera/runtime-security/blob/master/pkg/api/report.go#L12-L30
// We are currently storing this temporarily until https://tigera.atlassian.net/browse/EV-3460
// gets resolved
type Report struct {
	GeneratedTime *time.Time   `json:"generated_time,omitempty"`
	Count         int          `json:"count"`
	Type          string       `json:"type"`
	ConfigName    string       `json:"config_name"`
	StartTime     time.Time    `json:"start_time"`
	EndTime       time.Time    `json:"end_time"`
	Pod           PodInfo      `json:"pod"`
	File          File         `json:"file"`
	ProcessStart  ProcessStart `json:"process_start"`
	FileAccess    FileAccess   `json:"file_access"`
	Host          string       `json:"host,omitempty"`
}

// RuntimeReportParams define querying parameters to retrieve runtime reports
type RuntimeReportParams struct {
	QueryParams     `json:",inline" validate:"required"`
	QuerySortParams `json:",inline"`
	LegacyTimeRange *lmav1.TimeRange `json:"legacy_time_range"`
}

type PodInfo struct {
	Namespace     string    `json:"namespace"`
	Name          string    `json:"name"`
	NameAggr      string    `json:"name_aggr"`
	ContainerName string    `json:"container_name"`
	StartTime     time.Time `json:"start_time"`
	Ready         bool      `json:"ready"`
}

type File struct {
	Path     string `json:"path"`
	HostPath string `json:"host_path"`
}

type ProcessStart struct {
	Invocation string        `json:"invocation"`
	Hashes     ProcessHashes `json:"hashes"`
}

type ProcessHashes struct {
	MD5    string `json:"md5"`
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
}

type FileAccess struct {
	Created bool `json:"created"`

	NumReadCalls      int `json:"num_read_calls_since_last"`
	NumReadBytes      int `json:"num_read_bytes_since_last"`
	TotalNumReadCalls int `json:"total_num_read_calls"`
	TotalNumReadBytes int `json:"total_num_read_bytes"`

	NumWriteCalls      int `json:"num_write_calls_since_last"`
	NumWriteBytes      int `json:"num_write_bytes_since_last"`
	TotalNumWriteCalls int `json:"total_num_write_calls"`
	TotalNumWriteBytes int `json:"total_num_write_bytes"`

	IsMMapped  int `json:"is_mmapped"`
	MMapOffset int `json:"mmap_offset"`

	IsDupped int `json:"is_dupped"`
}
