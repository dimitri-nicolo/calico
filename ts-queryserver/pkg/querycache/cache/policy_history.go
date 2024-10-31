// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package cache

import (
	"time"

	prommodel "github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/api"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/dispatcherv1v3"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/labelhandler"
)

const (
	policyTypeUnmatched = "unmatched"
)

// NewPoliciesCacheHistory creates a new instance of a PolicyCacheHistory
func NewPoliciesCacheHistory(c *PrometheusClient, ts time.Time) PoliciesCache {
	return &policiesCacheHistory{promClient: c, timestamp: ts}
}

// policiesCacheHistory implements the PolicyCache interface. It retrieves historical
// node count data from Prometheus.
type policiesCacheHistory struct {
	promClient *PrometheusClient
	timestamp  time.Time
}

func (ch *policiesCacheHistory) TotalGlobalNetworkPolicies() api.PolicySummary {
	var ps api.PolicySummary

	res, err := ch.promClient.Query("queryserver_global_network_policy_total", ch.timestamp)
	if err != nil {
		log.WithError(err).Warn("failed to get historical data for total global network policies")
		return ps
	}

	if res.Type() == prommodel.ValVector {
		vec := res.(prommodel.Vector)
		for _, v := range vec {
			ch.fillGlobalNetworkPolicies(v, &ps)
		}
	}
	return ps
}

func (ch *policiesCacheHistory) TotalNetworkPoliciesByNamespace() map[string]api.PolicySummary {
	psm := make(map[string]api.PolicySummary)

	res, err := ch.promClient.Query("queryserver_network_policy_total", ch.timestamp)
	if err != nil {
		log.WithError(err).Warn("failed to get historical data for total network policies")
		return psm
	}

	if res.Type() == prommodel.ValVector {
		vec := res.(prommodel.Vector)
		for _, v := range vec {
			ch.fillNetworkPolicies(v, psm)
		}
	}
	return psm
}

func (ch *policiesCacheHistory) GetPolicy(model.Key) api.Policy {
	// do nothing for historical data cache
	return nil
}

func (ch *policiesCacheHistory) GetTier(model.Key) api.Tier {
	// do nothing for historical data cache
	return nil
}

func (ch *policiesCacheHistory) GetOrderedPolicies(set.Set[model.Key]) []api.Tier {
	// do nothing for historical data cache
	return nil
}

func (ch *policiesCacheHistory) RegisterWithDispatcher(dispatcher dispatcherv1v3.Interface) {
	// do nothing for historical data cache
}

func (ch *policiesCacheHistory) RegisterWithLabelHandler(handler labelhandler.Interface) {
	// do nothing for historical data cache
}

func (ch *policiesCacheHistory) GetPolicyKeySetByRuleSelector(string) set.Set[model.Key] {
	// do nothing for historical data cache
	return nil
}

func (ch *policiesCacheHistory) fillGlobalNetworkPolicies(sample *prommodel.Sample, ps *api.PolicySummary) {
	if t, ok := sample.Metric["type"]; ok {
		switch t {
		case policyTypeUnmatched:
			ps.NumUnmatched = int(sample.Value)
		}
	} else {
		ps.Total = int(sample.Value)
	}
}

func (ch *policiesCacheHistory) fillNetworkPolicies(sample *prommodel.Sample, psm map[string]api.PolicySummary) {
	if ns, ok := sample.Metric["namespace"]; ok {
		ps := psm[string(ns)]

		if t, ok := sample.Metric["type"]; ok {
			switch t {
			case policyTypeUnmatched:
				ps.NumUnmatched = int(sample.Value)
			}
		} else {
			ps.Total = int(sample.Value)
		}

		psm[string(ns)] = ps
	}
}
