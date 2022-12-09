// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package cache

import (
	"crypto/tls"
	"fmt"

	log "github.com/sirupsen/logrus"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/dispatcherv1v3"
)

const (
	nodeTypeNoEndpoints         = "no-endpoints"
	nodeTypeNoWorkloadEndpoints = "no-workload-endpoints"
	nodeTypeNoHostEndpoints     = "no-host-endpoints"
)

// NewNodeCacheHistory creates a new instance of a NodeCacheHistory
func NewNodeCacheHistory(address, token string, tlsConfig *tls.Config, timeRange *promv1.Range) NodeCache {
	return &nodeCacheHistory{
		promClient: NewPrometheusClient(address, token, tlsConfig),
		timeRange:  timeRange,
	}
}

// nodeCacheHistory implements the NodeHistory interface. It retrieves historical
// node count data from Prometheus.
type nodeCacheHistory struct {
	promClient *PrometheusClient
	timeRange  *promv1.Range
}

func (ch *nodeCacheHistory) TotalNodes() int {
	if ch.promClient != nil {
		res, err := ch.promClient.Query(`queryserver_node_total{type=""}`, ch.timeRange.End)
		if err != nil {
			log.WithError(err).Warn("failed to get historical data for total nodes")
			return 0
		}

		if res.Type() == model.ValVector {
			vec := res.(model.Vector)
			for _, v := range vec {
				return int(v.Value)
			}
		}
	}
	return 0
}

func (ch *nodeCacheHistory) TotalNodesWithNoEndpoints() int {
	if ch.promClient != nil {
		res, err := ch.promClient.Query(fmt.Sprintf(`queryserver_node_total{type="%s"}`, nodeTypeNoEndpoints), ch.timeRange.End)
		if err != nil {
			log.WithError(err).Warn("failed to get historical data for total nodes with no endpoints")
			return 0
		}

		if res.Type() == model.ValVector {
			vec := res.(model.Vector)
			for _, v := range vec {
				return int(v.Value)
			}
		}
	}
	return 0
}

func (ch *nodeCacheHistory) TotalNodesWithNoWorkloadEndpoints() int {
	if ch.promClient != nil {
		res, err := ch.promClient.Query(fmt.Sprintf(`queryserver_node_total{type="%s"}`, nodeTypeNoWorkloadEndpoints), ch.timeRange.End)
		if err != nil {
			log.WithError(err).Warn("failed to get historical data for total nodes with no workload endpoints")
			return 0
		}

		if res.Type() == model.ValVector {
			vec := res.(model.Vector)
			for _, v := range vec {
				return int(v.Value)
			}
		}
	}
	return 0
}

func (ch *nodeCacheHistory) TotalNodesWithNoHostEndpoints() int {
	if ch.promClient != nil {
		res, err := ch.promClient.Query(fmt.Sprintf(`queryserver_node_total{type="%s"}`, nodeTypeNoHostEndpoints), ch.timeRange.End)
		if err != nil {
			log.WithError(err).Warn("failed to get historical data for total nodes with no host endpoints")
			return 0
		}

		if res.Type() == model.ValVector {
			vec := res.(model.Vector)
			for _, v := range vec {
				return int(v.Value)
			}
		}
	}
	return 0
}

func (ch *nodeCacheHistory) GetNodes() []api.Node {
	// do nothing for historical data cache
	return nil
}

func (ch *nodeCacheHistory) GetNode(string) api.Node {
	// do nothing for historical data cache
	return nil
}

func (ch *nodeCacheHistory) RegisterWithDispatcher(dispatcher dispatcherv1v3.Interface) {
	// do nothing for historical data cache
}
