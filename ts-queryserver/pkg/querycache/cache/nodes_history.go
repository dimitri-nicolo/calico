// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package cache

import (
	"fmt"
	"time"

	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/dispatcherv1v3"
)

const (
	nodeTypeNoEndpoints         = "no-endpoints"
	nodeTypeNoWorkloadEndpoints = "no-workload-endpoints"
	nodeTypeNoHostEndpoints     = "no-host-endpoints"
)

// NewNodeCacheHistory creates a new instance of a NodeCacheHistory
func NewNodeCacheHistory(c *PrometheusClient, ts time.Time) NodeCache {
	return &nodeCacheHistory{promClient: c, timestamp: ts}
}

// nodeCacheHistory implements the NodeHistory interface. It retrieves historical
// node count data from Prometheus.
type nodeCacheHistory struct {
	promClient *PrometheusClient
	timestamp  time.Time
}

func (ch *nodeCacheHistory) TotalNodes() int {
	res, err := ch.promClient.Query(`queryserver_node_total{type=""}`, ch.timestamp)
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
	return 0
}

func (ch *nodeCacheHistory) TotalNodesWithNoEndpoints() int {
	res, err := ch.promClient.Query(fmt.Sprintf(`queryserver_node_total{type="%s"}`, nodeTypeNoEndpoints), ch.timestamp)
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
	return 0
}

func (ch *nodeCacheHistory) TotalNodesWithNoWorkloadEndpoints() int {
	res, err := ch.promClient.Query(fmt.Sprintf(`queryserver_node_total{type="%s"}`, nodeTypeNoWorkloadEndpoints), ch.timestamp)
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
	return 0
}

func (ch *nodeCacheHistory) TotalNodesWithNoHostEndpoints() int {
	res, err := ch.promClient.Query(fmt.Sprintf(`queryserver_node_total{type="%s"}`, nodeTypeNoHostEndpoints), ch.timestamp)
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
