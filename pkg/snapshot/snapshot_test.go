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

	"github.com/tigera/compliance/pkg/list/mock"
	"github.com/tigera/compliance/pkg/resources"
)

var (
	healthPort int
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
		src        *mock.Source
		dest       *mock.Destination
		healthAgg  *health.HealthAggregator
		nResources = len(resources.GetAllResourceHelpers())
	)

	BeforeEach(func() {
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
		dest.Initialize(time.Now().Add(-time.Hour))

		Run(ctx, src, dest, healthAgg)
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

		Run(ctx, src, dest, healthAgg)
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
		dest.Initialize(time.Now().Add(-25 * time.Hour))
		src.Initialize(time.Now())

		Run(ctx, src, dest, healthAgg)
		Expect(dest.RetrieveCalls).To(Equal(nResources))
		Expect(src.RetrieveCalls).To(Equal(nResources))
		Expect(dest.StoreCalls).To(Equal(nResources))
		Expect(isAlive()).To(BeTrue())
	})
})
