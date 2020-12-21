// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
package collector

import (
	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/config"
	"github.com/projectcalico/libcalico-go/lib/health"
)

// Temporary stub for windows.
func New(configParams *config.Config, lookupsCache *calc.LookupsCache, healthAggregator *health.HealthAggregator) Collector {
	return nil
}
