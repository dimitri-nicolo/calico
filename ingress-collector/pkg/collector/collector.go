// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"context"

	"github.com/tigera/ingress-collector/pkg/config"
)

type IngressCollector interface {
	ReadLogs(context.Context)
	Report() <-chan IngressInfo
	ParseRawLogs(string) (IngressLog, error)
}

func NewIngressCollector(cfg *config.Config) IngressCollector {
	// Currently it will only return a log file collector but
	// this should inspect the config to return other collectors
	// once they need to be implemented.
	return NewNginxCollector(cfg)
}

type IngressInfo struct {
	Logs        []IngressLog
	Connections map[TupleKey]int
}

type IngressLog struct {
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
// Ingress log's 5 tuple information.
type TupleKey struct {
	SrcIp    string
	DstIp    string
	SrcPort  int32
	DstPort  int32
	Protocol string
}

// BatchIngressLogs represents a "batch" of logs
// that we will use to collect HTTP request level info.
// This is used to limit the amount of request level
// data we send to Felix.
type BatchIngressLog struct {
	logs map[string]IngressLog
	size int
}

func NewBatchIngressLog(size int) *BatchIngressLog {
	if size < 0 {
		return &BatchIngressLog{
			logs: make(map[string]IngressLog),
			size: size,
		}
	}
	return &BatchIngressLog{
		logs: make(map[string]IngressLog, size),
		size: size,
	}
}

func (b BatchIngressLog) Insert(log IngressLog) {
	logKey := IngressLogKey(log)
	if !b.Full() {
		b.logs[logKey] = log
	}
}

func (b BatchIngressLog) Full() bool {
	if b.size < 0 {
		return false
	}
	return len(b.logs) == b.size
}

func (b BatchIngressLog) Logs() []IngressLog {
	logs := []IngressLog{}
	for _, val := range b.logs {
		logs = append(logs, val)
	}
	return logs
}

func IngressLogKey(log IngressLog) string {
	// Only use 1 IP as the key
	if log.XRealIp != "" && log.XRealIp != "-" {
		return log.XRealIp
	}
	return log.XForwardedFor
}

func TupleKeyFromIngressLog(log IngressLog) TupleKey {
	return TupleKey{
		SrcIp:    log.SrcIp,
		DstIp:    log.DstIp,
		SrcPort:  log.SrcPort,
		DstPort:  log.DstPort,
		Protocol: log.Protocol,
	}
}

func IngressLogFromTupleKey(key TupleKey) IngressLog {
	return IngressLog{
		SrcIp:    key.SrcIp,
		DstIp:    key.DstIp,
		SrcPort:  key.SrcPort,
		DstPort:  key.DstPort,
		Protocol: key.Protocol,
	}
}
