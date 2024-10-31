// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package cache

import (
	"time"

	prommodel "github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/dispatcherv1v3"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/labelhandler"
)

const (
	endpointTypeFailed      = "failed"
	endpointTypeUnlabeled   = "unlabeled"
	endpointTypeUnprotected = "unprotected"
)

// NewEndpointsCacheHistory creates a new instance of an EndpointsCacheHistory
func NewEndpointsCacheHistory(c *PrometheusClient, ts time.Time) EndpointsCache {
	return &endpointsCacheHistory{promClient: c, timestamp: ts}
}

// endpointsCacheHistory implements the EndpointsCache interface. It retrieves historical
// host and workload endpoints count data from Prometheus.
type endpointsCacheHistory struct {
	promClient *PrometheusClient
	timestamp  time.Time
}

func (ch *endpointsCacheHistory) TotalHostEndpoints() api.EndpointSummary {
	var eps api.EndpointSummary

	res, err := ch.promClient.Query("queryserver_host_endpoints_total", ch.timestamp)
	if err != nil {
		log.WithError(err).Warn("failed to get historical data for total host endpoints")
		return eps
	}

	if res.Type() == prommodel.ValVector {
		vec := res.(prommodel.Vector)
		for _, v := range vec {
			ch.fillHostEndpoints(v, &eps)
		}
	}
	return eps
}

func (ch *endpointsCacheHistory) TotalWorkloadEndpointsByNamespace() map[string]api.EndpointSummary {
	epsm := make(map[string]api.EndpointSummary)

	res, err := ch.promClient.Query("queryserver_workload_endpoints_total", ch.timestamp)
	if err != nil {
		log.WithError(err).Warn("failed to get historical data for total workload endpoints")
		return epsm
	}

	if res.Type() == prommodel.ValVector {
		vec := res.(prommodel.Vector)
		for _, v := range vec {
			ch.fillWorkloadEndpoints(v, epsm)
		}
	}
	return epsm
}

func (ch *endpointsCacheHistory) GetEndpoint(model.Key) api.Endpoint {
	// do nothing for historical data cache
	return nil
}

func (ch *endpointsCacheHistory) RegisterWithDispatcher(dispatcher dispatcherv1v3.Interface) {
	// do nothing for historical data cache
}

func (ch *endpointsCacheHistory) RegisterWithLabelHandler(handler labelhandler.Interface) {
	// do nothing for historical data cache
}

func (ch *endpointsCacheHistory) RegisterWithSharedInformer(factory informers.SharedInformerFactory, stopCh <-chan struct{}) {
	// do nothing for historical data cache
}

func (ch *endpointsCacheHistory) fillHostEndpoints(sample *prommodel.Sample, eps *api.EndpointSummary) {
	if t, ok := sample.Metric["type"]; ok {
		switch t {
		case endpointTypeUnlabeled:
			eps.NumWithNoLabels = int(sample.Value)
		case endpointTypeUnprotected:
			eps.NumWithNoPolicies = int(sample.Value)
		}
	} else {
		eps.Total = int(sample.Value)
	}
}

func (ch *endpointsCacheHistory) fillWorkloadEndpoints(sample *prommodel.Sample, epsm map[string]api.EndpointSummary) {
	if ns, ok := sample.Metric["namespace"]; ok {
		eps := epsm[string(ns)]

		if t, ok := sample.Metric["type"]; ok {
			switch t {
			case endpointTypeFailed:
				eps.NumFailed = int(sample.Value)
			case endpointTypeUnlabeled:
				eps.NumWithNoLabels = int(sample.Value)
			case endpointTypeUnprotected:
				eps.NumWithNoPolicies = int(sample.Value)
			}
		} else {
			eps.Total = int(sample.Value)
		}

		epsm[string(ns)] = eps
	}
}
