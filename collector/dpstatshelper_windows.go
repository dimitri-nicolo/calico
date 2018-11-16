// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package collector

import (
	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/config"
	"github.com/projectcalico/libcalico-go/lib/health"
)

// Temporary stub for windows.
func StartDataplaneStatsCollector(configParams *config.Config, lookupsCache *calc.LookupsCache, healthAggregator *health.HealthAggregator) Collector {
	return nil
}
