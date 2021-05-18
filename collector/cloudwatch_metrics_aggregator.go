// Copyright (c) 2018, 2021 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/rules"
)

type cloudWatchMetricsAggregator struct {
	clusterGUID string
}

func NewCloudWatchMetricsAggregator(clusterGUID string) MetricAggregator {
	return &cloudWatchMetricsAggregator{
		clusterGUID: clusterGUID,
	}
}

var (
	// resultMetrics map keeps track of all the metrics such as denied packet,
	// allowed packet (in future), allowed bytes (in future). All the metrics values for each of the metrics
	// will be aggregated into one MetricData.
	resultMetrics = MetricsMap{
		mu:   sync.RWMutex{},
		mMap: map[string]*MetricData{},
	}
)

type MetricsMap struct {
	mu   sync.RWMutex
	mMap map[string]*MetricData
}

func (c *cloudWatchMetricsAggregator) FeedUpdate(mu MetricUpdate) error {
	var dpm MetricData

	// For now, we only support denied packets, but when we need to do the other
	// metrics, we need to add cases for other actions.
	lastRuleID := mu.GetLastRuleID()
	if lastRuleID == nil {
		log.WithField("metric update", mu).Error("no last rule id present")
		return fmt.Errorf("invalid metric update")
	}
	switch lastRuleID.Action {
	case rules.RuleActionDeny:
		dpm.Name = dpMetricName
		dpm.Dimensions = map[string]string{"ClusterID": c.clusterGUID}
		dpm.Unit = defaultDPUnit
		dpm.Value = float64(mu.inMetric.deltaPackets + mu.outMetric.deltaPackets)
		// We're excluding the timestamp so all CloudWatch will put the timestamp of when it receives
		// the metric update and metric chart is not influenced by individual node clock skew.
	default:
		// We only want denied packets. Skip the rest of them.
		return nil

	}

	// Lock the result map to avoid race condition because Get() and FeedUpdate() are called
	// from different goroutines.
	resultMetrics.mu.Lock()
	_, ok := resultMetrics.mMap[dpm.Name]
	if !ok {
		resultMetrics.mMap[dpm.Name] = &dpm
	} else {
		resultMetrics.mMap[dpm.Name].Value += dpm.Value
	}

	log.WithField("Metric value", resultMetrics.mMap[dpm.Name].Value).Debugf("current aggregated packet count for action: %s", lastRuleID.Action)

	resultMetrics.mu.Unlock()
	return nil
}

func (c *cloudWatchMetricsAggregator) Get() []MetricData {
	result := []MetricData{}

	// Grab the lock so we get a consistent values if they're being written by FeedUpdate.
	resultMetrics.mu.Lock()

	for _, val := range resultMetrics.mMap {
		if val != nil {
			result = append(result, *val)

			// Now that we've read and aggregated all the metrics, we need to reset the value to 0.
			val.Value = float64(0)
		}
	}

	log.WithField("MetricData", result).Debug("aggregating metric count")
	resultMetrics.mu.Unlock()

	return result
}
