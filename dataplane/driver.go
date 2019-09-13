// +build !windows

// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.
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

package dataplane

import (
	"math/bits"
	"net"
	"os/exec"
	"runtime/debug"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/config"
	extdataplane "github.com/projectcalico/felix/dataplane/external"
	intdataplane "github.com/projectcalico/felix/dataplane/linux"
	"github.com/projectcalico/felix/ifacemonitor"
	"github.com/projectcalico/felix/ipsets"
	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/markbits"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/health"
)

func StartDataplaneDriver(configParams *config.Config,
	healthAggregator *health.HealthAggregator,
	collector collector.Collector,
	configChangedRestartCallback func(),
	childExitedRestartCallback func()) (DataplaneDriver, *exec.Cmd, chan *sync.WaitGroup) {
	if configParams.UseInternalDataplaneDriver {
		log.Info("Using internal (linux) dataplane driver.")
		// If kube ipvs interface is present, enable ipvs support.
		kubeIPVSSupportEnabled := ifacemonitor.IsInterfacePresent(intdataplane.KubeIPVSInterface)
		if kubeIPVSSupportEnabled {
			log.Info("Kube-proxy in ipvs mode, enabling felix kube-proxy ipvs support.")
		}
		if configChangedRestartCallback == nil {
			log.Panic("Starting dataplane with nil callback func.")
		}

		markBitsManager := markbits.NewMarkBitsManager(configParams.IptablesMarkMask, "felix-iptables")
		// Dedicated mark bits for accept and pass actions.  These are long lived bits
		// that we use for communicating between chains.
		markAccept, _ := markBitsManager.NextSingleBitMark()
		markPass, _ := markBitsManager.NextSingleBitMark()
		markDrop, _ := markBitsManager.NextSingleBitMark()
		var markIPsec uint32
		if configParams.IPSecEnabled() {
			log.Info("IPsec enabled, allocating a mark bit")
			markIPsec, _ = markBitsManager.NextSingleBitMark()
			if markIPsec == 0 {
				log.WithFields(log.Fields{
					"Name":     "felix-iptables",
					"MarkMask": configParams.IptablesMarkMask,
				}).Panic("Failed to allocate a mark bit for IPsec, not enough mark bits available.")
			}
		}
		// Short-lived mark bits for local calculations within a chain.
		markScratch0, _ := markBitsManager.NextSingleBitMark()
		markScratch1, _ := markBitsManager.NextSingleBitMark()
		if markAccept == 0 || markPass == 0 || markScratch0 == 0 || markScratch1 == 0 {
			log.WithFields(log.Fields{
				"Name":     "felix-iptables",
				"MarkMask": configParams.IptablesMarkMask,
			}).Panic("Not enough mark bits available.")
		}

		// Mark bits for end point mark. Currently felix takes the rest bits from mask available for use.
		markEndpointMark, allocated := markBitsManager.NextBlockBitsMark(markBitsManager.AvailableMarkBitCount())
		if kubeIPVSSupportEnabled && allocated == 0 {
			log.WithFields(log.Fields{
				"Name":     "felix-iptables",
				"MarkMask": configParams.IptablesMarkMask,
			}).Panic("Not enough mark bits available for endpoint mark.")
		}
		// Take lowest bit position (position 1) from endpoint mark mask reserved for non-calico endpoint.
		markEndpointNonCaliEndpoint := uint32(1) << uint(bits.TrailingZeros32(markEndpointMark))
		log.WithFields(log.Fields{
			"acceptMark":          markAccept,
			"passMark":            markPass,
			"dropMark":            markDrop,
			"scratch0Mark":        markScratch0,
			"scratch1Mark":        markScratch1,
			"endpointMark":        markEndpointMark,
			"endpointMarkNonCali": markEndpointNonCaliEndpoint,
		}).Info("Calculated iptables mark bits")

		dpConfig := intdataplane.Config{
			Hostname: configParams.FelixHostname,
			IfaceMonitorConfig: ifacemonitor.Config{
				InterfaceExcludes: configParams.InterfaceExclude,
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

				KubeNodePortRanges:     configParams.KubeNodePortRanges,
				KubeIPVSSupportEnabled: kubeIPVSSupportEnabled,

				OpenStackSpecialCasesEnabled: configParams.OpenstackActive(),
				OpenStackMetadataIP:          net.ParseIP(configParams.MetadataAddr),
				OpenStackMetadataPort:        uint16(configParams.MetadataPort),

				IptablesMarkAccept:          markAccept,
				IptablesMarkPass:            markPass,
				IptablesMarkDrop:            markDrop,
				IptablesMarkIPsec:           markIPsec,
				IptablesMarkScratch0:        markScratch0,
				IptablesMarkScratch1:        markScratch1,
				IptablesMarkEndpoint:        markEndpointMark,
				IptablesMarkNonCaliEndpoint: markEndpointNonCaliEndpoint,

				VXLANEnabled: configParams.VXLANEnabled,
				VXLANPort:    configParams.VXLANPort,
				VXLANVNI:     configParams.VXLANVNI,

				IPIPEnabled:        configParams.IpInIpEnabled,
				IPIPTunnelAddress:  configParams.IpInIpTunnelAddr,
				VXLANTunnelAddress: configParams.IPv4VXLANTunnelAddr,

				IPSecEnabled: configParams.IPSecEnabled(),

				IptablesLogPrefix:         configParams.LogPrefix,
				IncludeDropActionInPrefix: configParams.LogDropActionOverride,
				ActionOnDrop:              configParams.DropActionOverride,
				EndpointToHostAction:      configParams.DefaultEndpointToHostAction,
				IptablesFilterAllowAction: configParams.IptablesFilterAllowAction,
				IptablesMangleAllowAction: configParams.IptablesMangleAllowAction,

				FailsafeInboundHostPorts:  configParams.FailsafeInboundHostPorts,
				FailsafeOutboundHostPorts: configParams.FailsafeOutboundHostPorts,

				DisableConntrackInvalid: configParams.DisableConntrackInvalidCheck,

				EnableNflogSize: configParams.EnableNflogSize,

				NATPortRange:                       configParams.NATPortRange,
				IptablesNATOutgoingInterfaceFilter: configParams.IptablesNATOutgoingInterfaceFilter,
				DNSTrustedServers:                  configParams.DNSTrustedServers,
				NATOutgoingAddress:                 configParams.NATOutgoingAddress,
			},

			IPIPMTU:                        configParams.IpInIpMtu,
			VXLANMTU:                       configParams.VXLANMTU,
			IptablesBackend:                configParams.IptablesBackend,
			IptablesRefreshInterval:        configParams.IptablesRefreshInterval,
			RouteRefreshInterval:           configParams.RouteRefreshInterval,
			DeviceRouteSourceAddress:       configParams.DeviceRouteSourceAddress,
			DeviceRouteProtocol:            configParams.DeviceRouteProtocol,
			RemoveExternalRoutes:           configParams.RemoveExternalRoutes,
			IPSetsRefreshInterval:          configParams.IpsetsRefreshInterval,
			IptablesPostWriteCheckInterval: configParams.IptablesPostWriteCheckIntervalSecs,
			IptablesInsertMode:             configParams.ChainInsertMode,
			IptablesLockFilePath:           configParams.IptablesLockFilePath,
			IptablesLockTimeout:            configParams.IptablesLockTimeoutSecs,
			IptablesLockProbeInterval:      configParams.IptablesLockProbeIntervalMillis,
			MaxIPSetSize:                   configParams.MaxIpsetSize,
			IgnoreLooseRPF:                 configParams.IgnoreLooseRPF,
			IPv6Enabled:                    configParams.Ipv6Support,
			StatusReportingInterval:        configParams.ReportingIntervalSecs,
			XDPRefreshInterval:             configParams.XDPRefreshInterval,

			NetlinkTimeout: configParams.NetlinkTimeoutSecs,

			NodeIP:                     configParams.NodeIP,
			IPSecPSK:                   configParams.GetPSKFromFile(),
			IPSecAllowUnsecuredTraffic: configParams.IPSecAllowUnsecuredTraffic,
			IPSecIKEProposal:           configParams.IPSecIKEAlgorithm,
			IPSecESPProposal:           configParams.IPSecESPAlgorithm,
			IPSecLogLevel:              configParams.IPSecLogLevel,
			IPSecRekeyTime:             configParams.IPSecRekeyTime,
			IPSecPolicyRefreshInterval: configParams.IPSecPolicyRefreshInterval,

			ConfigChangedRestartCallback: configChangedRestartCallback,
			ChildExitedRestartCallback:   childExitedRestartCallback,

			PostInSyncCallback: func() {
				// The initial resync uses a lot of scratch space so now is
				// a good time to force a GC and return any RAM that we can.
				debug.FreeOSMemory()

				if configParams.DebugMemoryProfilePath == "" {
					return
				}
				logutils.DumpHeapMemoryProfile(configParams.DebugMemoryProfilePath)
			},
			HealthAggregator:                healthAggregator,
			DebugSimulateDataplaneHangAfter: configParams.DebugSimulateDataplaneHangAfter,
			DebugUseShortPollIntervals:      configParams.DebugUseShortPollIntervals,
			FelixHostname:                   configParams.FelixHostname,
			ExternalNodesCidrs:              configParams.ExternalNodesCIDRList,
			SidecarAccelerationEnabled:      configParams.SidecarAccelerationEnabled,
			XDPEnabled:                      configParams.XDPEnabled,
			XDPAllowGeneric:                 configParams.GenericXDPEnabled,
		}
		stopChan := make(chan *sync.WaitGroup, 1)
		intDP := intdataplane.NewIntDataplaneDriver(dpConfig, stopChan)
		intDP.Start()

		return intDP, nil, stopChan
	} else {
		log.WithField("driver", configParams.DataplaneDriver).Info(
			"Using external dataplane driver.")

		dpConn, cmd := extdataplane.StartExtDataplaneDriver(configParams.DataplaneDriver)
		return dpConn, cmd, nil
	}
}
