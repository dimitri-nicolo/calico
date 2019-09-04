// +build fvtests

// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package fv_test

import (
	"context"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/daemon"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

var _ = infrastructure.DatastoreDescribe("CloudWatch metrics tests", []apiconfig.DatastoreType{apiconfig.EtcdV3}, func(getInfra infrastructure.InfraFactory) {

	var (
		infra                               infrastructure.DatastoreInfra
		felix                               *infrastructure.Felix
		client                              client.Interface
		opts                                infrastructure.TopologyOptions
		collectorStartLogC, healthStartLogC chan struct{}
	)

	BeforeEach(func() {
		infra = getInfra()
		opts = infrastructure.DefaultTopologyOptions()
		felix, client = infrastructure.StartSingleNodeTopology(opts, infra)

		// Watch the felix log to detect whether the collector and health reporters start up.  Need to do this before
		// we enable CloudWatch.
		collectorStartLogC = felix.WatchStdoutFor(regexp.MustCompile(regexp.QuoteMeta(collector.StartupLog)))
		healthStartLogC = felix.WatchStdoutFor(regexp.MustCompile(regexp.QuoteMeta(daemon.HealthReporterStartupLog)))
	})

	enableCloudWatch := func() {
		fc := v3.NewFelixConfiguration()
		fc.Name = "default"
		t := true
		fc.Spec.CloudWatchMetricsReporterEnabled = &t
		fc.Spec.CloudWatchNodeHealthStatusEnabled = &t
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fc, err := client.FelixConfigurations().Create(ctx, fc, options.SetOptions{})
		Expect(err).NotTo(HaveOccurred())
	}

	It("should enable CloudWatch metrics with valid license", func() {
		enableCloudWatch()
		Eventually(collectorStartLogC, "10s", "100ms").Should(BeClosed())
		Eventually(healthStartLogC, "10s", "100ms").Should(BeClosed())
	})

	It("should not enable CloudWatch metrics with expired license", func() {
		infrastructure.ApplyExpiredLicense(client)
		enableCloudWatch()
		Consistently(collectorStartLogC, "10s", "100ms").ShouldNot(BeClosed())
		Expect(healthStartLogC).ToNot(BeClosed())
	})

	AfterEach(func() {
		felix.Stop()

		if CurrentGinkgoTestDescription().Failed {
			infra.DumpErrorData()
		}
		infra.Stop()
	})
})
