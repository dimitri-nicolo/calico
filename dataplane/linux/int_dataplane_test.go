// Copyright (c) 2016-2020 Tigera, Inc. All rights reserved.

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

package intdataplane_test

import (
	"net"
	"regexp"
	"time"

	"github.com/projectcalico/felix/capture"

	"github.com/google/gopacket/layers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/config"
	intdataplane "github.com/projectcalico/felix/dataplane/linux"
	"github.com/projectcalico/felix/ifacemonitor"
	"github.com/projectcalico/felix/ipsets"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/health"
)

type mockCollector struct{}

func (_ *mockCollector) ReportingChannel() chan<- *proto.DataplaneStats { return nil }

func (_ *mockCollector) Start() {}

func (_ *mockCollector) LogDNS(src, dst net.IP, dns *layers.DNS, latencyIfKnown *time.Duration) {}

func (_ *mockCollector) SetDNSLogReporter(reporter collector.DNSLogReporterInterface) {}

func (_ *mockCollector) LogL7(hd *proto.HTTPData, data *collector.Data, tuple collector.Tuple, httpDataCount int) {
}

func (_ *mockCollector) SetL7LogReporter(reporter collector.L7LogReporterInterface) {}

func (_ *mockCollector) SetPacketInfoReader(collector.PacketInfoReader) {}

func (_ *mockCollector) SetConntrackInfoReader(collector.ConntrackInfoReader) {}

var _ = Describe("Constructor test", func() {
	var configParams *config.Config
	var dpConfig intdataplane.Config
	var healthAggregator *health.HealthAggregator
	var col collector.Collector

	JustBeforeEach(func() {
		configParams = config.New()
		_, err := configParams.UpdateFrom(map[string]string{"InterfaceExclude": "/^kube.*/,/veth/,eth2"}, config.EnvironmentVariable)
		Expect(err).NotTo(HaveOccurred())
		dpConfig = intdataplane.Config{
			IfaceMonitorConfig: ifacemonitor.Config{
				InterfaceExcludes: configParams.InterfaceExclude,
				ResyncInterval:    configParams.RouteRefreshInterval,
			},
			RulesConfig: rules.Config{
				WorkloadIfacePrefixes: configParams.InterfacePrefixes(),

				IPSetConfigV4: ipsets.NewIPVersionConfig(
					ipsets.IPFamilyV4,
					rules.IPSetNamePrefix,
					rules.AllHistoricIPSetNamePrefixes,
					rules.LegacyV4IPSetNames,
				),
				IPSetConfigV6: ipsets.NewIPVersionConfig(
					ipsets.IPFamilyV6,
					rules.IPSetNamePrefix,
					rules.AllHistoricIPSetNamePrefixes,
					nil,
				),

				OpenStackSpecialCasesEnabled: configParams.OpenstackActive(),
				OpenStackMetadataIP:          net.ParseIP(configParams.MetadataAddr),
				OpenStackMetadataPort:        uint16(configParams.MetadataPort),

				IptablesMarkAccept:   0x1000000,
				IptablesMarkPass:     0x2000000,
				IptablesMarkScratch0: 0x4000000,
				IptablesMarkScratch1: 0x8000000,
				IptablesMarkDrop:     0x0800000,
				IptablesMarkIPsec:    0x0400000,
				IptablesMarkEgress:   0x0200000,
				IptablesMarkEndpoint: 0x000ff00,

				IPIPEnabled:       configParams.IpInIpEnabled,
				IPIPTunnelAddress: configParams.IpInIpTunnelAddr,

				ActionOnDrop:              configParams.DropActionOverride,
				EndpointToHostAction:      configParams.DefaultEndpointToHostAction,
				IptablesFilterAllowAction: configParams.IptablesFilterAllowAction,
				IptablesMangleAllowAction: configParams.IptablesMangleAllowAction,
			},
			IPIPMTU:          configParams.IpInIpMtu,
			HealthAggregator: healthAggregator,
			Collector:        col,

			MTUIfacePattern: regexp.MustCompile(".*"),

			LookPathOverride: func(file string) (string, error) {
				return file, nil
			},
			PacketCapture: capture.Config{
				Directory: "/tmp",
			},
		}
	})

	It("should be constructable", func() {
		var dp = intdataplane.NewIntDataplaneDriver(dpConfig, nil)
		Expect(dp).ToNot(BeNil())
	})

	Context("with health aggregator", func() {

		BeforeEach(func() {
			healthAggregator = health.NewHealthAggregator()
		})

		It("should be constructable", func() {
			var dp = intdataplane.NewIntDataplaneDriver(dpConfig, nil)
			Expect(dp).ToNot(BeNil())
		})
	})

	Context("with collector", func() {

		BeforeEach(func() {
			col = &mockCollector{}
		})

		It("should be constructable", func() {
			var dp = intdataplane.NewIntDataplaneDriver(dpConfig, nil)
			Expect(dp).ToNot(BeNil())
		})
	})
})
