// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package snapshot

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/health"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/list/mock"
	"github.com/tigera/compliance/pkg/resources"
)

var (
	healthPort int

	// Use a fixed "now" to prevent
	now = time.Now()
)

func isAlive() bool {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/liveness", healthPort))
	if err != nil {
		log.WithError(err).Error("liveness check failed")
		return false
	}
	log.WithField("statusCode", resp.StatusCode).Debug("liveness check")
	return resp.StatusCode < http.StatusBadRequest
}

var _ = Describe("Snapshot", func() {
	var (
		cfg        *config.Config
		src        *mock.Source
		dest       *mock.Destination
		healthAgg  *health.HealthAggregator
		nResources = len(resources.GetAllResourceHelpers())
	)

	BeforeEach(func() {
		cfg = &config.Config{}
		src = mock.NewSource()
		dest = mock.NewDestination(nil)
		healthPort = rand.Int()%55536 + 10000
		healthAgg = health.NewHealthAggregator()
		healthAgg.RegisterReporter(HealthName, &health.HealthReport{true, true}, 10*time.Minute)
		healthAgg.ServeHTTP(true, "localhost", healthPort)
	})

	It("should decide that it is not yet time to make a snapshot", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.After(time.Second)
			cancel()
		}()
		By("Taking a snapshot 2hrs ago")
		dest.Initialize(now.Add(-2 * time.Hour))

		By("Configuring the snapshot hour to be the next hour")
		cfg.SnapshotHour = now.Add(time.Hour).Hour()

		By("Starting the snapshotter")
		Run(ctx, cfg, src, dest, healthAgg)
		Expect(dest.RetrieveCalls).To(Equal(nResources))
		Expect(src.RetrieveCalls).To(Equal(0))
		Expect(dest.StoreCalls).To(Equal(0))
		Expect(isAlive()).To(BeTrue())
	})

	It("should decide that it is time to make a snapshot but fail because src is empty", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.After(time.Second)
			cancel()
		}()

		Run(ctx, cfg, src, dest, healthAgg)
		Expect(dest.RetrieveCalls).To(Equal(nResources))
		Expect(src.RetrieveCalls).To(Equal(nResources))
		Expect(dest.StoreCalls).To(Equal(0))
		Expect(isAlive()).To(BeFalse())
	})

	It("should decide that it is time to make a snapshot and successfully store list from src to dest", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.After(2 * time.Second)
			cancel()
		}()
		By("Taking a snapshot 2hrs ago")
		dest.Initialize(now.Add(-2 * time.Hour))
		src.Initialize(now)

		By("Configuring the snapshot hour to be the current hour")
		cfg.SnapshotHour = now.Hour()

		By("Starting the snapshotter")
		Run(ctx, cfg, src, dest, healthAgg)
		Expect(dest.RetrieveCalls).To(Equal(nResources))
		Expect(src.RetrieveCalls).To(Equal(nResources))
		Expect(dest.StoreCalls).To(Equal(nResources))
		Expect(isAlive()).To(BeTrue())
	})
})
