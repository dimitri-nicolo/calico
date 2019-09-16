// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package snapshot

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/resources"
	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/list/mock"
)

var (
	// Use a fixed "now" to prevent crossing over into the next hour mid-test.
	now = time.Now()
)

var _ = Describe("Snapshot", func() {
	var (
		cfg        *config.Config
		src        *mock.Source
		dest       *mock.Destination
		healthy    func(bool)
		isHealthy  bool
		nResources = len(resources.GetAllResourceHelpers())
	)

	BeforeEach(func() {
		cfg = &config.Config{}
		src = mock.NewSource()
		dest = mock.NewDestination(nil)
		healthy = func(h bool) { isHealthy = h }
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
		Run(ctx, cfg, src, dest, healthy)
		Expect(dest.RetrieveCalls).To(Equal(nResources))
		Expect(src.RetrieveCalls).To(Equal(0))
		Expect(dest.StoreCalls).To(Equal(0))
		Expect(isHealthy).To(BeTrue())
	})

	It("should decide that it is time to make a snapshot but fail because src is empty", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.After(time.Second)
			cancel()
		}()

		Run(ctx, cfg, src, dest, healthy)
		Expect(dest.RetrieveCalls).To(Equal(nResources))
		Expect(src.RetrieveCalls).To(Equal(nResources))
		Expect(dest.StoreCalls).To(Equal(0))
		Expect(isHealthy).To(BeFalse())
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
		Run(ctx, cfg, src, dest, healthy)
		Expect(dest.RetrieveCalls).To(Equal(nResources))
		Expect(src.RetrieveCalls).To(Equal(nResources))
		Expect(dest.StoreCalls).To(Equal(nResources))
		Expect(isHealthy).To(BeTrue())
	})
})
