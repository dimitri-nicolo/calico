// Copyright (c) 2017-2020 Tigera, Inc. All rights reserved.

package dataplane

import (
	"os/exec"
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/config"
	windataplane "github.com/projectcalico/felix/dataplane/windows"
	"github.com/projectcalico/felix/dataplane/windows/hns"
	"github.com/projectcalico/libcalico-go/lib/health"
)

func StartDataplaneDriver(configParams *config.Config,
	healthAggregator *health.HealthAggregator,
	collector collector.Collector,
	configChangedRestartCallback func(),
	childExitedRestartCallback func(),
	k8sClientSet *kubernetes.Clientset) (DataplaneDriver, *exec.Cmd, chan *sync.WaitGroup) {
	log.Info("Using Windows dataplane driver.")

	dpConfig := windataplane.Config{
		IPv6Enabled:      configParams.Ipv6Support,
		HealthAggregator: healthAggregator,

		Hostname:     configParams.FelixHostname,
		VXLANEnabled: configParams.VXLANEnabled,
		VXLANID:      configParams.VXLANVNI,
		VXLANPort:    configParams.VXLANPort,
	}

	winDP := windataplane.NewWinDataplaneDriver(hns.API{}, dpConfig)
	winDP.Start()

	return winDP, nil, nil
}
