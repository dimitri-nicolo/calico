// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"context"

	"github.com/tigera/envoy-collector/pkg/config"
)

type EnvoyCollector interface {
	ReadLogs(context.Context)
	Report() <-chan EnvoyInfo
	ParseRawLogs(string) (EnvoyLog, error)
}

func NewEnvoyCollector(cfg *config.Config) EnvoyCollector {
	// Currently it will only return a log file collector but
	// this should inspect the config to return other collectors
	// once they need to be implemented.
	return NewNginxCollector(cfg)
}

type EnvoyInfo struct {
	Logs        []EnvoyLog
	Connections map[TupleKey]int
}

type EnvoyLog struct {
	SrcIp    string `json:"source_ip"`
	DstIp    string `json:"destination_ip"`
	SrcPort  int32  `json:"source_port"`
	DstPort  int32  `json:"destination_port"`
	Protocol string `json:"protocol"`

	// HTTP Headers
	XForwardedFor string `json:"x-forwarded-for"`
	XRealIp       string `json:"x-real-ip"`
}

// TupleKey is an object just for holding the
// Envoy log's 5 tuple information.
type TupleKey struct {
	SrcIp    string
	DstIp    string
	SrcPort  int32
	DstPort  int32
	Protocol string
}

// BatchEnvoyLog represents a "batch" of logs
// that we will use to collect HTTP request level info.
// This is used to limit the amount of request level
// data we send to Felix.
type BatchEnvoyLog struct {
	logs map[string]EnvoyLog
	size int
}

func NewBatchEnvoyLog(size int) *BatchEnvoyLog {
	if size < 0 {
		return &BatchEnvoyLog{
			logs: make(map[string]EnvoyLog),
			size: size,
		}
	}
	return &BatchEnvoyLog{
		logs: make(map[string]EnvoyLog, size),
		size: size,
	}
}

func (b BatchEnvoyLog) Insert(log EnvoyLog) {
	logKey := EnvoyLogKey(log)
	if !b.Full() {
		b.logs[logKey] = log
	}
}

func (b BatchEnvoyLog) Full() bool {
	if b.size < 0 {
		return false
	}
	return len(b.logs) == b.size
}

func (b BatchEnvoyLog) Clear() {
	if b.size < 0 {
		b.logs = make(map[string]EnvoyLog)
	}
	b.logs = make(map[string]EnvoyLog, b.size)
}

func (b BatchEnvoyLog) Logs() []EnvoyLog {
	logs := []EnvoyLog{}
	for _, val := range b.logs {
		logs = append(logs, val)
	}
	return logs
}

func EnvoyLogKey(log EnvoyLog) string {
	// Only use 1 IP as the key
	if log.XRealIp != "" && log.XRealIp != "-" {
		return log.XRealIp
	}
	return log.XForwardedFor
}

func TupleKeyFromEnvoyLog(log EnvoyLog) TupleKey {
	return TupleKey{
		SrcIp:    log.SrcIp,
		DstIp:    log.DstIp,
		SrcPort:  log.SrcPort,
		DstPort:  log.DstPort,
		Protocol: log.Protocol,
	}
}

func EnvoyLogFromTupleKey(key TupleKey) EnvoyLog {
	return EnvoyLog{
		SrcIp:    key.SrcIp,
		DstIp:    key.DstIp,
		SrcPort:  key.SrcPort,
		DstPort:  key.DstPort,
		Protocol: key.Protocol,
	}
}
