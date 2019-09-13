// Copyright (c) 2019 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package daemon

import (
	lclient "github.com/tigera/licensing/client"
	v1 "k8s.io/api/core/v1"

	"github.com/projectcalico/felix/config"

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

var _ = Describe("Typha address discovery", func() {

	getKubernetesService := func(namespace, name string) (*v1.Service, error) {
		return &v1.Service{
			Spec: v1.ServiceSpec{
				ClusterIP: "fd5f:65af::2",
				Ports: []v1.ServicePort{
					v1.ServicePort{
						Name: "calico-typha",
						Port: 8156,
					},
				},
			},
		}, nil
	}

	It("should bracket an IPv6 Typha address", func() {
		configParams := config.New()
		_, err := configParams.UpdateFrom(map[string]string{
			"TyphaK8sServiceName": "calico-typha",
		}, config.EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		typhaAddr, err := discoverTyphaAddr(configParams, getKubernetesService)
		Expect(err).NotTo(HaveOccurred())
		Expect(typhaAddr).To(Equal("[fd5f:65af::2]:8156"))
	})
})
