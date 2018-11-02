// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package daemon

import (
	"github.com/projectcalico/felix/config"
	lclient "github.com/tigera/licensing/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/licensing/client/features"
)

// Dummy license checker for the tests.
type dlc struct {
	status   lclient.LicenseStatus
	features map[string]bool
}

func (d dlc) GetFeatureStatus(feature string) bool {
	switch d.status {
	case lclient.Valid, lclient.InGracePeriod:
		return d.features[feature]
	}
	return false
}

func (d dlc) GetLicenseStatus() lclient.LicenseStatus {
	return d.status
}

var _ = Describe("FelixDaemon license checks", func() {

	var cfg *config.Config

	BeforeEach(func() {
		// Create a config resource with all of the licensed features.
		cfg = config.New()
		cfg.UpdateFrom(map[string]string{
			"IPSecMode":                         "PSK",
			"IPSecAllowUnsecuredTraffic":        "false",
			"PrometheusReporterEnabled":         "true",
			"DropActionOverride":                "ACCEPT",
			"CloudWatchLogsReporterEnabled":     "true",
			"CloudWatchMetricsReporterEnabled":  "true",
			"CloudWatchNodeHealthStatusEnabled": "true",
			"FlowLogsFileEnabled":               "true",
		}, config.DatastoreGlobal)

		Expect(cfg.IPSecMode).To(Equal("PSK"))
		Expect(cfg.IPSecAllowUnsecuredTraffic).To(BeFalse())
		Expect(cfg.PrometheusReporterEnabled).To(BeTrue())
		Expect(cfg.DropActionOverride).To(Equal("ACCEPT"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeTrue())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeTrue())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeTrue())
		Expect(cfg.FlowLogsFileEnabled).To(BeTrue())
	})

	It("Should reset all values if there is no license", func() {
		removeUnlicensedFeaturesFromConfig(cfg, dlc{
			status: lclient.NoLicenseLoaded,
		})
		Expect(cfg.IPSecMode).To(Equal(""))
		Expect(cfg.PrometheusReporterEnabled).To(BeFalse())
		Expect(cfg.DropActionOverride).To(Equal("DROP"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeFalse())
		Expect(cfg.FlowLogsFileEnabled).To(BeFalse())
	})

	It("Should allow IPSec insecure if IPSec feature is in grace period", func() {
		removeUnlicensedFeaturesFromConfig(cfg, dlc{
			status: lclient.InGracePeriod,
			features: map[string]bool{
				features.IPSec: true,
			},
		})
		Expect(cfg.IPSecMode).To(Equal("PSK"))
		Expect(cfg.IPSecAllowUnsecuredTraffic).To(BeTrue())
		Expect(cfg.PrometheusReporterEnabled).To(BeFalse())
		Expect(cfg.DropActionOverride).To(Equal("DROP"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeFalse())
		Expect(cfg.FlowLogsFileEnabled).To(BeFalse())
	})

	It("Should leave IPSec settings unchanged if IPSec license is valid", func() {
		removeUnlicensedFeaturesFromConfig(cfg, dlc{
			status: lclient.Valid,
			features: map[string]bool{
				features.IPSec: true,
			},
		})
		Expect(cfg.IPSecMode).To(Equal("PSK"))
		Expect(cfg.IPSecAllowUnsecuredTraffic).To(BeFalse())
		Expect(cfg.PrometheusReporterEnabled).To(BeFalse())
		Expect(cfg.DropActionOverride).To(Equal("DROP"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeFalse())
		Expect(cfg.FlowLogsFileEnabled).To(BeFalse())
	})

	It("Should leave Prometheus setting unchanged if PrometheusMetrics license is valid", func() {
		removeUnlicensedFeaturesFromConfig(cfg, dlc{
			status: lclient.Valid,
			features: map[string]bool{
				features.PrometheusMetrics: true,
			},
		})
		Expect(cfg.IPSecMode).To(Equal(""))
		Expect(cfg.PrometheusReporterEnabled).To(BeTrue())
		Expect(cfg.DropActionOverride).To(Equal("DROP"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeFalse())
		Expect(cfg.FlowLogsFileEnabled).To(BeFalse())
	})

	It("Should leave DropActionOverride setting unchanged if DropActionOverride license is valid", func() {
		removeUnlicensedFeaturesFromConfig(cfg, dlc{
			status: lclient.Valid,
			features: map[string]bool{
				features.DropActionOverride: true,
			},
		})
		Expect(cfg.IPSecMode).To(Equal(""))
		Expect(cfg.PrometheusReporterEnabled).To(BeFalse())
		Expect(cfg.DropActionOverride).To(Equal("ACCEPT"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeFalse())
		Expect(cfg.FlowLogsFileEnabled).To(BeFalse())
	})

	It("Should leave AWSCloudwatchFlowLogs setting unchanged if AWSCloudwatchFlowLogs license is valid", func() {
		removeUnlicensedFeaturesFromConfig(cfg, dlc{
			status: lclient.Valid,
			features: map[string]bool{
				features.AWSCloudwatchFlowLogs: true,
			},
		})
		Expect(cfg.IPSecMode).To(Equal(""))
		Expect(cfg.PrometheusReporterEnabled).To(BeFalse())
		Expect(cfg.DropActionOverride).To(Equal("DROP"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeTrue())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeFalse())
		Expect(cfg.FlowLogsFileEnabled).To(BeFalse())
	})

	It("Should leave AWSCloudwatchMetrics setting unchanged if AWSCloudwatchMetrics license is valid", func() {
		removeUnlicensedFeaturesFromConfig(cfg, dlc{
			status: lclient.Valid,
			features: map[string]bool{
				features.AWSCloudwatchMetrics: true,
			},
		})
		Expect(cfg.IPSecMode).To(Equal(""))
		Expect(cfg.PrometheusReporterEnabled).To(BeFalse())
		Expect(cfg.DropActionOverride).To(Equal("DROP"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeTrue())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeTrue())
		Expect(cfg.FlowLogsFileEnabled).To(BeFalse())
	})

	It("Should leave FileOutputFlowLogs setting unchanged if FileOutputFlowLogs license is valid", func() {
		removeUnlicensedFeaturesFromConfig(cfg, dlc{
			status: lclient.Valid,
			features: map[string]bool{
				features.FileOutputFlowLogs: true,
			},
		})
		Expect(cfg.IPSecMode).To(Equal(""))
		Expect(cfg.PrometheusReporterEnabled).To(BeFalse())
		Expect(cfg.DropActionOverride).To(Equal("DROP"))
		Expect(cfg.CloudWatchLogsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchMetricsReporterEnabled).To(BeFalse())
		Expect(cfg.CloudWatchNodeHealthStatusEnabled).To(BeFalse())
		Expect(cfg.FlowLogsFileEnabled).To(BeTrue())
	})
})
