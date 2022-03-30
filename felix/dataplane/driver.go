// Copyright (c) 2020-2022 Tigera, Inc. All rights reserved.
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

//go:build !windows

package dataplane

import (
	"math/bits"
	"net"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/clock"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/felix/aws"
	"github.com/projectcalico/calico/felix/bpf"
	"github.com/projectcalico/calico/felix/bpf/conntrack"
	tcdefs "github.com/projectcalico/calico/felix/bpf/tc/defs"
	"github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/capture"
	"github.com/projectcalico/calico/felix/collector"
	"github.com/projectcalico/calico/felix/config"
	extdataplane "github.com/projectcalico/calico/felix/dataplane/external"
	"github.com/projectcalico/calico/felix/dataplane/inactive"
	intdataplane "github.com/projectcalico/calico/felix/dataplane/linux"
	"github.com/projectcalico/calico/felix/idalloc"
	"github.com/projectcalico/calico/felix/ifacemonitor"
	"github.com/projectcalico/calico/felix/ipsets"
	"github.com/projectcalico/calico/felix/logutils"
	"github.com/projectcalico/calico/felix/markbits"
	"github.com/projectcalico/calico/felix/rules"
	"github.com/projectcalico/calico/felix/wireguard"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/projectcalico/calico/libcalico-go/lib/ipam"
	"github.com/projectcalico/calico/libcalico-go/lib/security"
)

func StartDataplaneDriver(configParams *config.Config,
	healthAggregator *health.HealthAggregator,
	collector collector.Collector,
	configChangedRestartCallback func(),
	fatalErrorCallback func(error),
	childExitedRestartCallback func(),
	ipamClient ipam.Interface,
	k8sClientSet *kubernetes.Clientset,
	lc *calc.LookupsCache) (DataplaneDriver, *exec.Cmd, chan *sync.WaitGroup) {

	if !configParams.IsLeader() {
		// Return an inactive dataplane, since we're not the leader.
		log.Info("Not the leader, using an inactive dataplane")
		return &inactive.InactiveDataplane{}, nil, make(chan *sync.WaitGroup)
	}

	if configParams.UseInternalDataplaneDriver {
		log.Info("Using internal (linux) dataplane driver.")
		// If kube ipvs interface is present, enable ipvs support.  In BPF mode, we bypass kube-proxy so IPVS
		// is irrelevant.
		kubeIPVSSupportEnabled := false
		if ifacemonitor.IsInterfacePresent(intdataplane.KubeIPVSInterface) {
			if configParams.BPFEnabled {
				log.Info("kube-proxy IPVS device found but we're in BPF mode, ignoring.")
			} else {
				kubeIPVSSupportEnabled = true
				log.Info("Kube-proxy in ipvs mode, enabling felix kube-proxy ipvs support.")
			}
		}
		if configChangedRestartCallback == nil || fatalErrorCallback == nil {
			log.Panic("Starting dataplane with nil callback func.")
		}

		allowedMarkBits := configParams.IptablesMarkMask
		if configParams.BPFEnabled {
			// In BPF mode, the BPF programs use mark bits that are not configurable.  Make sure that those
			// bits are covered by our allowed mask.
			if allowedMarkBits&tcdefs.MarksMask != tcdefs.MarksMask {
				log.WithFields(log.Fields{
					"Name":            "felix-iptables",
					"MarkMask":        allowedMarkBits,
					"RequiredBPFBits": tcdefs.MarksMask,
				}).Panic("IptablesMarkMask doesn't cover bits that are used (unconditionally) by eBPF mode.")
			}
			allowedMarkBits ^= allowedMarkBits & tcdefs.MarksMask
			log.WithField("updatedBits", allowedMarkBits).Info(
				"Removed BPF program bits from available mark bits.")
		}

		markBitsManager := markbits.NewMarkBitsManager(allowedMarkBits, "felix-iptables")

		// Allocate mark bits; only the accept, scratch-0 and Wireguard bits are used in BPF mode so we
		// avoid allocating the others to minimize the number of bits in use.

		// The accept bit is a long-lived bit used to communicate between chains.
		var markAccept, markPass, markScratch0, markScratch1, markWireguard, markEndpointNonCaliEndpoint uint32
		markAccept, _ = markBitsManager.NextSingleBitMark()

		// The pass bit is used to communicate from a policy chain up to the endpoint chain.
		markPass, _ = markBitsManager.NextSingleBitMark()

		markDrop, _ := markBitsManager.NextSingleBitMark()
		var markIPsec, markEgressIP uint32
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

		// Egress mark is hard-coded in BPF mode - because then it needs to
		// interop between the BPF C code and Felix golang code - but dynamically
		// allocated in iptables mode.
		if configParams.BPFEnabled {
			markEgressIP = tcdefs.MarkEgress
		} else if configParams.EgressIPCheckEnabled() {
			log.Info("Egress IP enabled, allocating a mark bit")
			markEgressIP, _ = markBitsManager.NextSingleBitMark()
			if markEgressIP == 0 {
				log.WithFields(log.Fields{
					"Name":     "felix-iptables",
					"MarkMask": configParams.IptablesMarkMask,
				}).Panic("Failed to allocate a mark bit for Egress IP, not enough mark bits available.")
			}
		}

		// Scratch bits are short-lived bits used for calculating multi-rule results.
		markScratch0, _ = markBitsManager.NextSingleBitMark()
		markScratch1, _ = markBitsManager.NextSingleBitMark()

		if configParams.WireguardEnabled {
			log.Info("Wireguard enabled, allocating a mark bit")
			markWireguard, _ = markBitsManager.NextSingleBitMark()
			if markWireguard == 0 {
				log.WithFields(log.Fields{
					"Name":     "felix-iptables",
					"MarkMask": allowedMarkBits,
				}).Panic("Failed to allocate a mark bit for wireguard, not enough mark bits available.")
			}
		}

		var markProxy, k8sMasqMark uint32

		k8sMasqMark = 1 << configParams.KubeMasqueradeBit

		if configParams.TPROXYModeEnabled() {
			log.Infof("TPROXYMode %s, allocating a mark bit", configParams.TPROXYMode)
			markProxy, _ = markBitsManager.NextSingleBitMark()
			if markProxy == 0 {
				log.WithFields(log.Fields{
					"Name":     "felix-iptables",
					"MarkMask": allowedMarkBits,
				}).Panic("Failed to allocate a mark bit for proxy, not enough mark bits available.")
			}
		}

		if markAccept == 0 || markScratch0 == 0 || markPass == 0 || markScratch1 == 0 {
			log.WithFields(log.Fields{
				"Name":     "felix-iptables",
				"MarkMask": allowedMarkBits,
			}).Panic("Not enough mark bits available.")
		}

		var markDNSPolicy, markSkipDNSPolicyNfqueue uint32
		// If this is in BPF mode don't even try to allocate DNS policy bits. The markDNSPolicy bit is used as a flag
		// to enable the DNS policy enhancements and we don't want to somehow have the markDNSPolicy bit set without
		// the markSkipDNSPolicyNfqueue.
		if !configParams.BPFEnabled {
			markDNSPolicy, _ = markBitsManager.NextSingleBitMark()
			markSkipDNSPolicyNfqueue, _ = markBitsManager.NextSingleBitMark()
		}

		// Mark bits for endpoint mark. Currently Felix takes the rest bits from mask available for use.
		markEndpointMark, allocated := markBitsManager.NextBlockBitsMark(markBitsManager.AvailableMarkBitCount())
		if kubeIPVSSupportEnabled {
			if allocated == 0 {
				log.WithFields(log.Fields{
					"Name":     "felix-iptables",
					"MarkMask": allowedMarkBits,
				}).Panic("Not enough mark bits available for endpoint mark.")
			}
			// Take lowest bit position (position 1) from endpoint mark mask reserved for non-calico endpoint.
			markEndpointNonCaliEndpoint = uint32(1) << uint(bits.TrailingZeros32(markEndpointMark))
		}
		log.WithFields(log.Fields{
			"acceptMark":           markAccept,
			"passMark":             markPass,
			"dropMark":             markDrop,
			"scratch0Mark":         markScratch0,
			"scratch1Mark":         markScratch1,
			"endpointMark":         markEndpointMark,
			"endpointMarkNonCali":  markEndpointNonCaliEndpoint,
			"wireguardMark":        markWireguard,
			"ipsecMark":            markIPsec,
			"egressMark":           markEgressIP,
			"dnsPolicyMark":        markDNSPolicy,
			"skipDNSPolicyNfqueue": markSkipDNSPolicyNfqueue,
		}).Info("Calculated iptables mark bits")

		// Create a routing table manager. There are certain components that should take specific indices in the range
		// to simplify table tidy-up.
		reservedTables := []idalloc.IndexRange{{Min: 253, Max: 255}}
		routeTableIndexAllocator := idalloc.NewIndexAllocator(configParams.RouteTableIndices(), reservedTables)

		// Always allocate the wireguard table index (even when not enabled). This ensures we can tidy up entries
		// if wireguard is disabled after being previously enabled.
		var wireguardEnabled bool
		var wireguardTableIndex int
		if wgIdx, err := routeTableIndexAllocator.GrabIndex(); err == nil {
			log.Debugf("Assigned wireguard table index: %d", wgIdx)
			wireguardEnabled = configParams.WireguardEnabled
			wireguardTableIndex = wgIdx
		} else {
			log.WithError(err).Warning("Unable to assign table index for wireguard")
		}

		// If wireguard is enabled, update the failsafe ports to include the wireguard port.
		failsafeInboundHostPorts := configParams.FailsafeInboundHostPorts
		failsafeOutboundHostPorts := configParams.FailsafeOutboundHostPorts
		if configParams.WireguardEnabled {
			found := false
			for _, i := range failsafeInboundHostPorts {
				if i.Port == uint16(configParams.WireguardListeningPort) && i.Protocol == "udp" {
					log.WithFields(log.Fields{
						"net":      i.Net,
						"port":     i.Port,
						"protocol": i.Protocol,
					}).Debug("FailsafeInboundHostPorts is already configured for wireguard")
					found = true
					break
				}
			}
			if !found {
				failsafeInboundHostPorts = make([]config.ProtoPort, len(configParams.FailsafeInboundHostPorts)+1)
				copy(failsafeInboundHostPorts, configParams.FailsafeInboundHostPorts)
				log.Debug("Adding permissive FailsafeInboundHostPorts for wireguard")
				failsafeInboundHostPorts[len(configParams.FailsafeInboundHostPorts)] = config.ProtoPort{
					Port:     uint16(configParams.WireguardListeningPort),
					Protocol: "udp",
				}
			}

			found = false
			for _, i := range failsafeOutboundHostPorts {
				if i.Port == uint16(configParams.WireguardListeningPort) && i.Protocol == "udp" {
					log.WithFields(log.Fields{
						"net":      i.Net,
						"port":     i.Port,
						"protocol": i.Protocol,
					}).Debug("FailsafeOutboundHostPorts is already configured for wireguard")
					found = true
					break
				}
			}
			if !found {
				failsafeOutboundHostPorts = make([]config.ProtoPort, len(configParams.FailsafeOutboundHostPorts)+1)
				copy(failsafeOutboundHostPorts, configParams.FailsafeOutboundHostPorts)
				log.Debug("Adding permissive FailsafeOutboundHostPorts for wireguard")
				failsafeOutboundHostPorts[len(configParams.FailsafeOutboundHostPorts)] = config.ProtoPort{
					Port:     uint16(configParams.WireguardListeningPort),
					Protocol: "udp",
				}
			}
		}

		dpConfig := intdataplane.Config{
			Hostname:           configParams.FelixHostname,
			FloatingIPsEnabled: strings.EqualFold(configParams.FloatingIPs, string(v3.FloatingIPsEnabled)),
			IfaceMonitorConfig: ifacemonitor.Config{
				InterfaceExcludes: configParams.InterfaceExclude,
				ResyncInterval:    configParams.InterfaceRefreshInterval,
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
				KubernetesProvider:     configParams.KubernetesProvider(),

				OpenStackSpecialCasesEnabled: configParams.OpenstackActive(),
				OpenStackMetadataIP:          net.ParseIP(configParams.MetadataAddr),
				OpenStackMetadataPort:        uint16(configParams.MetadataPort),

				DNSPolicyMode:       apiv3.DNSPolicyMode(configParams.DNSPolicyMode),
				DNSPolicyNfqueueID:  int64(configParams.DNSPolicyNfqueueID),
				DNSPacketsNfqueueID: int64(configParams.DNSPacketsNfqueueID),

				IptablesMarkAccept:               markAccept,
				IptablesMarkPass:                 markPass,
				IptablesMarkDrop:                 markDrop,
				IptablesMarkIPsec:                markIPsec,
				IptablesMarkEgress:               markEgressIP,
				IptablesMarkScratch0:             markScratch0,
				IptablesMarkScratch1:             markScratch1,
				IptablesMarkEndpoint:             markEndpointMark,
				IptablesMarkNonCaliEndpoint:      markEndpointNonCaliEndpoint,
				IptablesMarkDNSPolicy:            markDNSPolicy,
				IptablesMarkSkipDNSPolicyNfqueue: markSkipDNSPolicyNfqueue,

				VXLANEnabled: configParams.VXLANEnabled,
				VXLANPort:    configParams.VXLANPort,
				VXLANVNI:     configParams.VXLANVNI,

				IPIPEnabled:        configParams.IpInIpEnabled,
				IPIPTunnelAddress:  configParams.IpInIpTunnelAddr,
				VXLANTunnelAddress: configParams.IPv4VXLANTunnelAddr,

				IPSecEnabled: configParams.IPSecEnabled(),

				EgressIPEnabled:   configParams.EgressIPCheckEnabled(),
				EgressIPVXLANPort: configParams.EgressIPVXLANPort,
				EgressIPVXLANVNI:  configParams.EgressIPVXLANVNI,
				EgressIPInterface: "egress.calico",

				AllowVXLANPacketsFromWorkloads: configParams.AllowVXLANPacketsFromWorkloads,
				AllowIPIPPacketsFromWorkloads:  configParams.AllowIPIPPacketsFromWorkloads,

				WireguardEnabled:            configParams.WireguardEnabled,
				WireguardInterfaceName:      configParams.WireguardInterfaceName,
				WireguardIptablesMark:       markWireguard,
				WireguardListeningPort:      configParams.WireguardListeningPort,
				WireguardEncryptHostTraffic: configParams.WireguardHostEncryptionEnabled,
				RouteSource:                 configParams.RouteSource,

				IptablesLogPrefix:         configParams.LogPrefix,
				IncludeDropActionInPrefix: configParams.LogDropActionOverride,
				ActionOnDrop:              configParams.DropActionOverride,
				EndpointToHostAction:      configParams.DefaultEndpointToHostAction,
				IptablesFilterAllowAction: configParams.IptablesFilterAllowAction,
				IptablesMangleAllowAction: configParams.IptablesMangleAllowAction,

				FailsafeInboundHostPorts:  failsafeInboundHostPorts,
				FailsafeOutboundHostPorts: failsafeOutboundHostPorts,

				DisableConntrackInvalid: configParams.DisableConntrackInvalidCheck,

				NATPortRange:                       configParams.NATPortRange,
				IptablesNATOutgoingInterfaceFilter: configParams.IptablesNATOutgoingInterfaceFilter,
				DNSTrustedServers:                  configParams.DNSTrustedServers,
				NATOutgoingAddress:                 configParams.NATOutgoingAddress,
				BPFEnabled:                         configParams.BPFEnabled,
				ServiceLoopPrevention:              configParams.ServiceLoopPrevention,

				TPROXYMode:             configParams.TPROXYMode,
				TPROXYPort:             configParams.TPROXYPort,
				TPROXYUpstreamConnMark: configParams.TPROXYUpstreamConnMark,
				IptablesMarkProxy:      markProxy,
				KubeMasqueradeMark:     k8sMasqMark,
			},

			Wireguard: wireguard.Config{
				Enabled:             wireguardEnabled,
				ListeningPort:       configParams.WireguardListeningPort,
				FirewallMark:        int(markWireguard),
				RoutingRulePriority: configParams.WireguardRoutingRulePriority,
				RoutingTableIndex:   wireguardTableIndex,
				InterfaceName:       configParams.WireguardInterfaceName,
				MTU:                 configParams.WireguardMTU,
				RouteSource:         configParams.RouteSource,
				EncryptHostTraffic:  configParams.WireguardHostEncryptionEnabled,
				PersistentKeepAlive: configParams.WireguardPersistentKeepAlive,
			},

			IPIPMTU:                        configParams.IpInIpMtu,
			VXLANMTU:                       configParams.VXLANMTU,
			VXLANPort:                      configParams.VXLANPort,
			IptablesBackend:                configParams.IptablesBackend,
			IptablesRefreshInterval:        configParams.IptablesRefreshInterval,
			RouteRefreshInterval:           configParams.RouteRefreshInterval,
			DeviceRouteSourceAddress:       configParams.DeviceRouteSourceAddress,
			DeviceRouteProtocol:            netlink.RouteProtocol(configParams.DeviceRouteProtocol),
			RemoveExternalRoutes:           configParams.RemoveExternalRoutes,
			IPSetsRefreshInterval:          configParams.IpsetsRefreshInterval,
			IptablesPostWriteCheckInterval: configParams.IptablesPostWriteCheckIntervalSecs,
			IptablesInsertMode:             configParams.ChainInsertMode,
			IptablesLockFilePath:           configParams.IptablesLockFilePath,
			IptablesLockTimeout:            configParams.IptablesLockTimeoutSecs,
			IptablesLockProbeInterval:      configParams.IptablesLockProbeIntervalMillis,
			MaxIPSetSize:                   configParams.MaxIpsetSize,
			IPv6Enabled:                    configParams.Ipv6Support,
			BPFIpv6Enabled:                 configParams.BpfIpv6Support,
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

			EgressIPEnabled:             configParams.EgressIPCheckEnabled(),
			EgressIPRoutingRulePriority: configParams.EgressIPRoutingRulePriority,

			ConfigChangedRestartCallback: configChangedRestartCallback,
			FatalErrorRestartCallback:    fatalErrorCallback,
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
			DebugConsoleEnabled:             configParams.DebugConsoleEnabled,
			DebugSimulateDataplaneHangAfter: configParams.DebugSimulateDataplaneHangAfter,
			DebugUseShortPollIntervals:      configParams.DebugUseShortPollIntervals,
			FelixHostname:                   configParams.FelixHostname,
			ExternalNodesCidrs:              configParams.ExternalNodesCIDRList,
			SidecarAccelerationEnabled:      configParams.SidecarAccelerationEnabled,

			BPFEnabled:                         configParams.BPFEnabled,
			BPFDisableUnprivileged:             configParams.BPFDisableUnprivileged,
			BPFConnTimeLBEnabled:               configParams.BPFConnectTimeLoadBalancingEnabled,
			BPFKubeProxyIptablesCleanupEnabled: configParams.BPFKubeProxyIptablesCleanupEnabled,
			BPFLogLevel:                        configParams.BPFLogLevel,
			BPFExtToServiceConnmark:            configParams.BPFExtToServiceConnmark,
			BPFDataIfacePattern:                configParams.BPFDataIfacePattern,
			BPFCgroupV2:                        configParams.DebugBPFCgroupV2,
			BPFMapRepin:                        configParams.DebugBPFMapRepinEnabled,
			KubeProxyMinSyncPeriod:             configParams.BPFKubeProxyMinSyncPeriod,
			BPFPSNATPorts:                      configParams.BPFPSNATPorts,
			BPFMapSizeRoute:                    configParams.BPFMapSizeRoute,
			BPFMapSizeNATFrontend:              configParams.BPFMapSizeNATFrontend,
			BPFMapSizeNATBackend:               configParams.BPFMapSizeNATBackend,
			BPFMapSizeNATAffinity:              configParams.BPFMapSizeNATAffinity,
			BPFMapSizeConntrack:                configParams.BPFMapSizeConntrack,
			BPFMapSizeIPSets:                   configParams.BPFMapSizeIPSets,
			XDPEnabled:                         configParams.XDPEnabled,
			XDPAllowGeneric:                    configParams.GenericXDPEnabled,
			BPFConntrackTimeouts:               conntrack.DefaultTimeouts(), // FIXME make timeouts configurable
			RouteTableManager:                  routeTableIndexAllocator,
			MTUIfacePattern:                    configParams.MTUIfacePattern,
			FlowLogsCollectProcessInfo:         configParams.FlowLogsCollectProcessInfo,
			FlowLogsCollectProcessPath:         configParams.FlowLogsCollectProcessPath,
			FlowLogsCollectTcpStats:            configParams.FlowLogsCollectTcpStats,
			FlowLogsFileIncludeService:         configParams.FlowLogsFileIncludeService,
			NfNetlinkBufSize:                   configParams.NfNetlinkBufSize,

			IPAMClient: ipamClient,

			FeatureDetectOverrides: configParams.FeatureDetectOverride,

			RouteSource: configParams.RouteSource,

			KubernetesProvider: configParams.KubernetesProvider(),

			Collector:            collector,
			DNSCacheFile:         configParams.GetDNSCacheFile(),
			DNSCacheSaveInterval: configParams.DNSCacheSaveInterval,
			DNSCacheEpoch:        configParams.DNSCacheEpoch,
			DNSExtraTTL:          configParams.GetDNSExtraTTL(),
			DNSLogsLatency:       configParams.DNSLogsLatency,

			AWSSecondaryIPSupport:             configParams.AWSSecondaryIPSupport,
			AWSRequestTimeout:                 configParams.AWSRequestTimeout,
			AWSSecondaryIPRoutingRulePriority: configParams.AWSSecondaryIPRoutingRulePriority,

			PacketCapture: capture.Config{
				Directory:       configParams.CaptureDir,
				MaxSizeBytes:    configParams.CaptureMaxSizeBytes,
				RotationSeconds: configParams.CaptureRotationSeconds,
				MaxFiles:        configParams.CaptureMaxFiles,
			},

			LookupsCache: lc,

			DNSPolicyMode:                    apiv3.DNSPolicyMode(configParams.DNSPolicyMode),
			DNSPolicyNfqueueID:               configParams.DNSPolicyNfqueueID,
			DNSPolicyNfqueueSize:             configParams.DNSPolicyNfqueueSize,
			DNSPacketsNfqueueID:              configParams.DNSPacketsNfqueueID,
			DNSPacketsNfqueueSize:            configParams.DNSPacketsNfqueueSize,
			DNSPacketsNfqueueMaxHoldDuration: configParams.DNSPacketsNfqueueMaxHoldDuration,
			DebugDNSResponseDelay:            configParams.DebugDNSResponseDelay,
		}
		if k8sClientSet != nil {
			dpConfig.KubeClientSet = k8sClientSet
		}

		if configParams.BPFExternalServiceMode == "dsr" {
			dpConfig.BPFNodePortDSREnabled = true
		}

		stopChan := make(chan *sync.WaitGroup, 1)
		intDP := intdataplane.NewIntDataplaneDriver(dpConfig, stopChan)
		intDP.Start()

		// Set source-destination-check on AWS EC2 instance.
		if configParams.AWSSrcDstCheck != string(apiv3.AWSSrcDstCheckOptionDoNothing) {
			c := &clock.RealClock{}
			updater := aws.NewEC2SrcDstCheckUpdater()
			go aws.WaitForEC2SrcDstCheckUpdate(configParams.AWSSrcDstCheck, healthAggregator, updater, c)
		}

		return intDP, nil, stopChan
	} else {
		log.WithField("driver", configParams.DataplaneDriver).Info(
			"Using external dataplane driver.")

		dpConn, cmd := extdataplane.StartExtDataplaneDriver(configParams.DataplaneDriver)
		return dpConn, cmd, nil
	}
}

func SupportsBPF() error {
	return bpf.SupportsBPFDataplane()
}

func SupportsBPFKprobe() error {
	return bpf.SupportsBPFKprobe()
}

func ServePrometheusMetrics(configParams *config.Config) {
	for {
		log.WithFields(log.Fields{
			"host": configParams.PrometheusMetricsHost,
			"port": configParams.PrometheusMetricsPort,
		}).Info("Starting prometheus metrics endpoint")
		if configParams.PrometheusGoMetricsEnabled && configParams.PrometheusProcessMetricsEnabled && configParams.PrometheusWireGuardMetricsEnabled {
			log.Info("Including Golang, Process and WireGuard metrics")
		} else {
			if !configParams.PrometheusGoMetricsEnabled {
				log.Info("Discarding Golang metrics")
				prometheus.Unregister(collectors.NewGoCollector())
			}
			if !configParams.PrometheusProcessMetricsEnabled {
				log.Info("Discarding process metrics")
				prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
			}
			if !configParams.PrometheusWireGuardMetricsEnabled || !configParams.WireguardEnabled {
				log.Info("Discarding WireGuard metrics")
				prometheus.Unregister(wireguard.MustNewWireguardMetrics())
			}
		}

		err := security.ServePrometheusMetrics(
			prometheus.DefaultGatherer,
			"",
			configParams.PrometheusMetricsPort,
			configParams.PrometheusMetricsCertFile,
			configParams.PrometheusMetricsKeyFile,
			configParams.PrometheusMetricsCAFile,
		)

		log.WithError(err).Error(
			"Prometheus metrics endpoint failed, trying to restart it...")
		time.Sleep(1 * time.Second)
	}
}
