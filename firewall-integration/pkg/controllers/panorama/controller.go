// Copyright 2019 Tigera Inc. All rights reserved.

package panorama

import (
	"context"
	"time"

	panw "github.com/PaloAltoNetworks/pango"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	"github.com/projectcalico/calico/firewall-integration/pkg/cache"
	"github.com/projectcalico/calico/firewall-integration/pkg/config"
	panutils "github.com/projectcalico/calico/firewall-integration/pkg/controllers/panorama/utils"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/projectcalico/calico/libcalico-go/lib/jitter"
)

const (
	healthReporterName   = "PanoramaController"
	healthReportInterval = time.Second * 10
)

type PanoramaController struct {
	ctx              context.Context
	cfg              *config.Config
	panwClient       *panw.Panorama
	calicoClient     datastore.ClientSet
	gnpCache         *cache.GnpCache
	healthAggregator *health.HealthAggregator
}

func NewPanoramaController(ctx context.Context, cfg *config.Config, h *health.HealthAggregator) *PanoramaController {
	// TODO(doublek): Inject all clients from caller.
	pCli, err := panutils.NewPANWClient(cfg)
	if err != nil {
		log.Fatalf("Error creating PANW client: %s", err)
		return nil
	}
	if err = pCli.GetClient().Initialize(); err != nil {
		log.Fatalf("Error initializing PANW client: %s", err)
		return nil
	}

	// Write to TSEE.
	cl, err := panutils.NewTSEEClient(cfg)
	if err != nil {
		log.Fatalf("Error creating TSEE client")
		return nil
	}

	// Cache global network policies.
	gnpc := cache.NewGnpCache(cl)

	h.RegisterReporter(healthReporterName, &health.HealthReport{Live: true}, healthReportInterval)

	return &PanoramaController{
		ctx:              ctx,
		cfg:              cfg,
		panwClient:       pCli.GetClient(),
		calicoClient:     cl,
		gnpCache:         gnpc,
		healthAggregator: h,
	}
}

func (pc *PanoramaController) Run() {
	minDuration := pc.cfg.FwPollInterval
	maxJitter := minDuration / 10

	log.Infof("Starting")

	healthy := func() {
		pc.healthAggregator.Report(healthReporterName, &health.HealthReport{Live: true})
	}
	// TODO(doublek): Report at healthReportInterval.
	healthy()

	ticker := jitter.NewTicker(minDuration, maxJitter)
	for {
		select {
		case <-pc.ctx.Done():
			return
		case <-ticker.C:
			pc.fwIntegrate()
		}
	}
}
