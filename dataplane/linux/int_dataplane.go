// Copyright (c) 2017-2020 Tigera, Inc. All rights reserved.
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

package intdataplane

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/arp"
	"github.com/projectcalico/felix/bpf/conntrack"
	"github.com/projectcalico/felix/bpf/events"
	bpfipsets "github.com/projectcalico/felix/bpf/ipsets"
	"github.com/projectcalico/felix/bpf/kprobe"
	"github.com/projectcalico/felix/bpf/nat"
	bpfproxy "github.com/projectcalico/felix/bpf/proxy"
	"github.com/projectcalico/felix/bpf/routes"
	"github.com/projectcalico/felix/bpf/state"
	"github.com/projectcalico/felix/bpf/tc"
	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/capture"
	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/idalloc"
	"github.com/projectcalico/felix/ifacemonitor"
	"github.com/projectcalico/felix/ipsec"
	"github.com/projectcalico/felix/ipsets"
	"github.com/projectcalico/felix/iptables"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/labelindex"
	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/routetable"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/felix/throttle"
	"github.com/projectcalico/felix/wireguard"
	"github.com/projectcalico/libcalico-go/lib/health"
	lclogutils "github.com/projectcalico/libcalico-go/lib/logutils"
	cprometheus "github.com/projectcalico/libcalico-go/lib/prometheus"
	"github.com/projectcalico/libcalico-go/lib/set"
)

const (
	// msgPeekLimit is the maximum number of messages we'll try to grab from the to-dataplane
	// channel before we apply the changes.  Higher values allow us to batch up more work on
	// the channel for greater throughput when we're under load (at cost of higher latency).
	msgPeekLimit = 100

	// Interface name used by kube-proxy to bind service ips.
	KubeIPVSInterface = "kube-ipvs0"

	// Size of a VXLAN header.
	VXLANHeaderSize = 50
)

var (
	countDataplaneSyncErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_int_dataplane_failures",
		Help: "Number of times dataplane updates failed and will be retried.",
	})
	countMessages = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "felix_int_dataplane_messages",
		Help: "Number dataplane messages by type.",
	}, []string{"type"})
	summaryApplyTime = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_int_dataplane_apply_time_seconds",
		Help: "Time in seconds that it took to apply a dataplane update.",
	})
	summaryBatchSize = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_int_dataplane_msg_batch_size",
		Help: "Number of messages processed in each batch. Higher values indicate we're " +
			"doing more batching to try to keep up.",
	})
	summaryIfaceBatchSize = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_int_dataplane_iface_msg_batch_size",
		Help: "Number of interface state messages processed in each batch. Higher " +
			"values indicate we're doing more batching to try to keep up.",
	})
	summaryAddrBatchSize = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_int_dataplane_addr_msg_batch_size",
		Help: "Number of interface address messages processed in each batch. Higher " +
			"values indicate we're doing more batching to try to keep up.",
	})

	processStartTime time.Time
	zeroKey          = wgtypes.Key{}
)

func init() {
	prometheus.MustRegister(countDataplaneSyncErrors)
	prometheus.MustRegister(summaryApplyTime)
	prometheus.MustRegister(countMessages)
	prometheus.MustRegister(summaryBatchSize)
	prometheus.MustRegister(summaryIfaceBatchSize)
	prometheus.MustRegister(summaryAddrBatchSize)
	processStartTime = time.Now()
}

func EnableTimestamping() error {
	s, err := unix.Socket(unix.AF_INET, unix.SOCK_RAW, unix.IPPROTO_RAW)
	if err != nil || s < 0 {
		return fmt.Errorf("Failed to create raw socket: %v", err)
	}

	err = unix.SetsockoptInt(s, unix.SOL_SOCKET, unix.SO_TIMESTAMP, 1)
	if err != nil {
		return fmt.Errorf("Failed to set SO_TIMESTAMP socket option: %v", err)
	}

	return nil
}

type Config struct {
	Hostname string

	IPv6Enabled          bool
	RuleRendererOverride rules.RuleRenderer
	IPIPMTU              int
	VXLANMTU             int

	MaxIPSetSize                   int
	IptablesBackend                string
	IPSetsRefreshInterval          time.Duration
	RouteRefreshInterval           time.Duration
	DeviceRouteSourceAddress       net.IP
	DeviceRouteProtocol            int
	RemoveExternalRoutes           bool
	IptablesRefreshInterval        time.Duration
	IPSecPolicyRefreshInterval     time.Duration
	IptablesPostWriteCheckInterval time.Duration
	IptablesInsertMode             string
	IptablesLockFilePath           string
	IptablesLockTimeout            time.Duration
	IptablesLockProbeInterval      time.Duration
	XDPRefreshInterval             time.Duration

	Wireguard wireguard.Config

	NetlinkTimeout time.Duration

	RulesConfig rules.Config

	IfaceMonitorConfig ifacemonitor.Config

	StatusReportingInterval time.Duration

	ConfigChangedRestartCallback func()
	ChildExitedRestartCallback   func()

	PostInSyncCallback func()
	HealthAggregator   *health.HealthAggregator
	RouteTableManager  *idalloc.IndexAllocator

	ExternalNodesCidrs []string

	BPFEnabled                         bool
	BPFDisableUnprivileged             bool
	BPFKubeProxyIptablesCleanupEnabled bool
	BPFLogLevel                        string
	BPFDataIfacePattern                *regexp.Regexp
	XDPEnabled                         bool
	XDPAllowGeneric                    bool
	BPFConntrackTimeouts               conntrack.Timeouts
	BPFCgroupV2                        string
	BPFConnTimeLBEnabled               bool
	BPFMapRepin                        bool
	BPFNodePortDSREnabled              bool
	KubeProxyMinSyncPeriod             time.Duration
	KubeProxyEndpointSlicesEnabled     bool
	FlowLogsCollectProcessInfo         bool

	SidecarAccelerationEnabled bool

	DebugSimulateDataplaneHangAfter time.Duration
	DebugUseShortPollIntervals      bool

	FelixHostname string
	NodeIP        net.IP

	IPSecPSK string
	// IPSecAllowUnsecuredTraffic controls whether
	// - IPsec is required for every packet (on a supported path), or
	// - IPsec is used opportunistically but unsecured traffic is still allowed.
	IPSecAllowUnsecuredTraffic bool
	IPSecIKEProposal           string
	IPSecESPProposal           string
	IPSecLogLevel              string
	IPSecRekeyTime             time.Duration

	EgressIPEnabled             bool
	EgressIPRoutingRulePriority int

	// Optional stats collector
	Collector collector.Collector

	// Config for DNS policy.
	DNSCacheFile         string
	DNSCacheSaveInterval time.Duration
	DNSCacheEpoch        int
	DNSExtraTTL          time.Duration
	DNSLogsLatency       bool

	LookPathOverride func(file string) (string, error)

	KubeClientSet *kubernetes.Clientset

	FeatureDetectOverrides map[string]string

	PacketCapture capture.Config

	// Populated with the smallest host MTU based on auto-detection.
	hostMTU         int
	MTUIfacePattern *regexp.Regexp
}

// InternalDataplane implements an in-process Felix dataplane driver based on iptables
// and ipsets.  It communicates with the datastore-facing part of Felix via the
// Send/RecvMessage methods, which operate on the protobuf-defined API objects.
//
// Architecture
//
// The internal dataplane driver is organised around a main event loop, which handles
// update events from the datastore and dataplane.
//
// Each pass around the main loop has two phases.  In the first phase, updates are fanned
// out to "manager" objects, which calculate the changes that are needed and pass them to
// the dataplane programming layer.  In the second phase, the dataplane layer applies the
// updates in a consistent sequence.  The second phase is skipped until the datastore is
// in sync; this ensures that the first update to the dataplane applies a consistent
// snapshot.
//
// Having the dataplane layer batch updates has several advantages.  It is much more
// efficient to batch updates, since each call to iptables/ipsets has a high fixed cost.
// In addition, it allows for different managers to make updates without having to
// coordinate on their sequencing.
//
// Requirements on the API
//
// The internal dataplane does not do consistency checks on the incoming data (as the
// old Python-based driver used to do).  It expects to be told about dependent resources
// before they are needed and for their lifetime to exceed that of the resources that
// depend on them.  For example, it is important the the datastore layer send an
// IP set create event before it sends a rule that references that IP set.
type InternalDataplane struct {
	toDataplane   chan interface{}
	fromDataplane chan interface{}

	allIptablesTables    []*iptables.Table
	iptablesMangleTables []*iptables.Table
	iptablesNATTables    []*iptables.Table
	iptablesRawTables    []*iptables.Table
	iptablesFilterTables []*iptables.Table
	ipSets               []ipsetsDataplane

	ipipManager          *ipipManager
	allHostsIpsetManager *allHostsIpsetManager

	ipSecPolTable  *ipsec.PolicyTable
	ipSecDataplane ipSecDataplane

	wireguardManager *wireguardManager

	ifaceMonitor     *ifacemonitor.InterfaceMonitor
	ifaceUpdates     chan *ifaceUpdate
	ifaceAddrUpdates chan *ifaceAddrsUpdate

	endpointStatusCombiner *endpointStatusCombiner

	domainInfoStore   *domainInfoStore
	domainInfoChanges chan *domainInfoChanged

	allManagers             []Manager
	managersWithRouteTables []ManagerWithRouteTables
	managersWithRouteRules  []ManagerWithRouteRules
	ruleRenderer            rules.RuleRenderer

	lookupCache *calc.LookupsCache

	// dataplaneNeedsSync is set if the dataplane is dirty in some way, i.e. we need to
	// call apply().
	dataplaneNeedsSync bool
	// forceIPSetsRefresh is set by the IP sets refresh timer to indicate that we should
	// check the IP sets in the dataplane.
	forceIPSetsRefresh bool
	// forceRouteRefresh is set by the route refresh timer to indicate that we should
	// check the routes in the dataplane.
	forceRouteRefresh bool
	// forceXDPRefresh is set by the XDP refresh timer to indicate that we should
	// check the XDP state in the dataplane.
	forceXDPRefresh bool
	// doneFirstApply is set after we finish the first update to the dataplane. It indicates
	// that the dataplane should now be in sync.
	doneFirstApply bool

	reschedTimer *time.Timer
	reschedC     <-chan time.Time

	applyThrottle *throttle.Throttle

	config Config

	debugHangC <-chan time.Time

	// Channel used when the Felix top level wants the dataplane to stop.
	stopChan chan *sync.WaitGroup

	xdpState          *xdpState
	sockmapState      *sockmapState
	endpointsSourceV4 endpointsSource
	ipsetsSourceV4    ipsetsSource
	callbacks         *callbacks

	loopSummarizer *logutils.Summarizer
}

const (
	healthName     = "int_dataplane"
	healthInterval = 10 * time.Second
)

func NewIntDataplaneDriver(config Config, stopChan chan *sync.WaitGroup) *InternalDataplane {

	if config.DNSLogsLatency {
		if err := EnableTimestamping(); err != nil {
			log.WithError(err).Warning("Couldn't enable timestamping, so DNS latency will not be measured")
		} else {
			log.Info("Timestamping enabled, so DNS latency will be measured")
		}
	}

	log.WithField("config", config).Info("Creating internal dataplane driver.")
	ruleRenderer := config.RuleRendererOverride
	if ruleRenderer == nil {
		ruleRenderer = rules.NewRenderer(config.RulesConfig)
	}
	epMarkMapper := rules.NewEndpointMarkMapper(
		config.RulesConfig.IptablesMarkEndpoint,
		config.RulesConfig.IptablesMarkNonCaliEndpoint)

	// Auto-detect host MTU.
	if mtu, err := findHostMTU(config.MTUIfacePattern); err != nil {
		log.WithError(err).Fatal("Unable to detect host MTU, shutting down")
	} else {
		// We found the host's MTU. Default any MTU configurations that have not been set.
		// We default the values even if the encap is not enabled, in order to match behavior
		// from earlier versions of Calico. However, they MTU will only be considered for allocation
		// to pod interfaces if the encap is enabled.
		config.hostMTU = mtu
		if config.IPIPMTU == 0 {
			log.Debug("Defaulting IPIP MTU based on host")
			config.IPIPMTU = mtu - 20
		}
		if config.VXLANMTU == 0 {
			log.Debug("Defaulting VXLAN MTU based on host")
			config.VXLANMTU = mtu - 50
		}
		if config.Wireguard.MTU == 0 {
			log.Debug("Defaulting Wireguard MTU based on host")
			config.Wireguard.MTU = mtu - 60
		}
	}
	if err := writeMTUFile(config); err != nil {
		log.WithError(err).Error("Failed to write MTU file, pod MTU may not be properly set")
	}

	dp := &InternalDataplane{
		toDataplane:       make(chan interface{}, msgPeekLimit),
		fromDataplane:     make(chan interface{}, 100),
		ruleRenderer:      ruleRenderer,
		ifaceMonitor:      ifacemonitor.New(config.IfaceMonitorConfig),
		ifaceUpdates:      make(chan *ifaceUpdate, 100),
		ifaceAddrUpdates:  make(chan *ifaceAddrsUpdate, 100),
		domainInfoChanges: make(chan *domainInfoChanged, 100),
		config:            config,
		applyThrottle:     throttle.New(10),
		loopSummarizer:    logutils.NewSummarizer("dataplane reconciliation loops"),
		stopChan:          stopChan,
	}
	dp.applyThrottle.Refill() // Allow the first apply() immediately.

	dp.ifaceMonitor.StateCallback = dp.onIfaceStateChange
	dp.ifaceMonitor.AddrCallback = dp.onIfaceAddrsChange

	backendMode := iptables.DetectBackend(config.LookPathOverride, iptables.NewRealCmd, config.IptablesBackend)

	// Most iptables tables need the same options.
	iptablesOptions := iptables.TableOptions{
		HistoricChainPrefixes: rules.AllHistoricChainNamePrefixes,
		InsertMode:            config.IptablesInsertMode,
		RefreshInterval:       config.IptablesRefreshInterval,
		PostWriteInterval:     config.IptablesPostWriteCheckInterval,
		LockTimeout:           config.IptablesLockTimeout,
		LockProbeInterval:     config.IptablesLockProbeInterval,
		BackendMode:           backendMode,
		LookPathOverride:      config.LookPathOverride,
		OnStillAlive:          dp.reportHealth,
		OpRecorder:            dp.loopSummarizer,
	}

	if config.BPFEnabled && config.BPFKubeProxyIptablesCleanupEnabled {
		// If BPF-mode is enabled, clean up kube-proxy's rules too.
		log.Info("BPF enabled, configuring iptables layer to clean up kube-proxy's rules.")
		iptablesOptions.ExtraCleanupRegexPattern = rules.KubeProxyInsertRuleRegex
		iptablesOptions.HistoricChainPrefixes = append(iptablesOptions.HistoricChainPrefixes, rules.KubeProxyChainPrefixes...)
	}

	// However, the NAT tables need an extra cleanup regex.
	iptablesNATOptions := iptablesOptions
	if iptablesNATOptions.ExtraCleanupRegexPattern == "" {
		iptablesNATOptions.ExtraCleanupRegexPattern = rules.HistoricInsertedNATRuleRegex
	} else {
		iptablesNATOptions.ExtraCleanupRegexPattern += "|" + rules.HistoricInsertedNATRuleRegex
	}

	featureDetector := iptables.NewFeatureDetector(config.FeatureDetectOverrides)
	iptablesFeatures := featureDetector.GetFeatures()

	var iptablesLock sync.Locker
	if iptablesFeatures.RestoreSupportsLock {
		log.Debug("Calico implementation of iptables lock disabled (because detected version of " +
			"iptables-restore will use its own implementation).")
		iptablesLock = dummyLock{}
	} else if config.IptablesLockTimeout <= 0 {
		log.Debug("Calico implementation of iptables lock disabled (by configuration).")
		iptablesLock = dummyLock{}
	} else {
		// Create the shared iptables lock.  This allows us to block other processes from
		// manipulating iptables while we make our updates.  We use a shared lock because we
		// actually do multiple updates in parallel (but to different tables), which is safe.
		log.WithField("timeout", config.IptablesLockTimeout).Debug(
			"Calico implementation of iptables lock enabled")
		iptablesLock = iptables.NewSharedLock(
			config.IptablesLockFilePath,
			config.IptablesLockTimeout,
			config.IptablesLockProbeInterval,
		)
	}

	mangleTableV4 := iptables.NewTable(
		"mangle",
		4,
		rules.RuleHashPrefix,
		iptablesLock,
		featureDetector,
		iptablesOptions)
	natTableV4 := iptables.NewTable(
		"nat",
		4,
		rules.RuleHashPrefix,
		iptablesLock,
		featureDetector,
		iptablesNATOptions,
	)
	rawTableV4 := iptables.NewTable(
		"raw",
		4,
		rules.RuleHashPrefix,
		iptablesLock,
		featureDetector,
		iptablesOptions)
	filterTableV4 := iptables.NewTable(
		"filter",
		4,
		rules.RuleHashPrefix,
		iptablesLock,
		featureDetector,
		iptablesOptions)
	ipSetsConfigV4 := config.RulesConfig.IPSetConfigV4
	ipSetsV4 := ipsets.NewIPSets(ipSetsConfigV4, dp.loopSummarizer)
	dp.iptablesNATTables = append(dp.iptablesNATTables, natTableV4)
	dp.iptablesRawTables = append(dp.iptablesRawTables, rawTableV4)
	dp.iptablesMangleTables = append(dp.iptablesMangleTables, mangleTableV4)
	dp.iptablesFilterTables = append(dp.iptablesFilterTables, filterTableV4)
	dp.ipSets = append(dp.ipSets, ipSetsV4)

	if config.RulesConfig.VXLANEnabled {
		routeTableVXLAN := routetable.New([]string{"^vxlan.calico$"}, 4, true, config.NetlinkTimeout,
			config.DeviceRouteSourceAddress, config.DeviceRouteProtocol, true, unix.RT_TABLE_UNSPEC,
			dp.loopSummarizer)

		vxlanManager := newVXLANManager(
			ipSetsV4,
			routeTableVXLAN,
			"vxlan.calico",
			config,
			dp.loopSummarizer,
		)
		go vxlanManager.KeepVXLANDeviceInSync(config.VXLANMTU, 10*time.Second)
		dp.RegisterManager(vxlanManager)
	} else {
		cleanUpVXLANDevice()
	}

	if config.EgressIPEnabled {
		// If IPIP or VXLAN is enabled, MTU of egress.calico device should be 50 bytes less than
		// MTU of IPIP or VXLAN device. MTU of the VETH device of a workload should be set to
		// the same value as MTU of egress.calico device.
		mtu := config.VXLANMTU

		if config.RulesConfig.VXLANEnabled {
			mtu = config.VXLANMTU - VXLANHeaderSize
		} else if config.RulesConfig.IPIPEnabled {
			mtu = config.IPIPMTU - VXLANHeaderSize
		}
		egressIpMgr := newEgressIPManager("egress.calico", config, dp.loopSummarizer)
		go egressIpMgr.KeepVXLANDeviceInSync(mtu, 10*time.Second)
		dp.RegisterManager(egressIpMgr)
	} else {
		// If Egress ip is not enabled, check to see if there is a VXLAN device and delete it if there is.
		log.Info("Checking if we need to clean up the egress VXLAN device")
		if link, err := netlink.LinkByName("egress.calico"); err != nil && err != syscall.ENODEV {
			log.WithError(err).Warnf("Failed to query egress VXLAN device")
		} else if err = netlink.LinkDel(link); err != nil {
			log.WithError(err).Error("Failed to delete unwanted egress VXLAN device")
		}
	}

	dp.endpointStatusCombiner = newEndpointStatusCombiner(dp.fromDataplane, config.IPv6Enabled)
	dp.domainInfoStore = newDomainInfoStore(dp.domainInfoChanges, &config)
	dp.RegisterManager(dp.domainInfoStore)

	callbacks := newCallbacks()
	dp.callbacks = callbacks
	if !config.BPFEnabled && config.XDPEnabled {
		if err := bpf.SupportsXDP(); err != nil {
			log.WithError(err).Warn("Can't enable XDP acceleration.")
		} else {
			st, err := NewXDPState(config.XDPAllowGeneric)
			if err != nil {
				log.WithError(err).Warn("Can't enable XDP acceleration.")
			} else {
				dp.xdpState = st
				dp.xdpState.PopulateCallbacks(callbacks)
				log.Info("XDP acceleration enabled.")
			}
		}
	} else {
		log.Info("XDP acceleration disabled.")
	}

	// TODO Integrate XDP and BPF infra.
	if !config.BPFEnabled && dp.xdpState == nil {
		xdpState, err := NewXDPState(config.XDPAllowGeneric)
		if err == nil {
			if err := xdpState.WipeXDP(); err != nil {
				log.WithError(err).Warn("Failed to cleanup preexisting XDP state")
			}
		}
		// if we can't create an XDP state it means we couldn't get a working
		// bpffs so there's nothing to clean up
	}

	if config.SidecarAccelerationEnabled {
		if err := bpf.SupportsSockmap(); err != nil {
			log.WithError(err).Warn("Can't enable Sockmap acceleration.")
		} else {
			st, err := NewSockmapState()
			if err != nil {
				log.WithError(err).Warn("Can't enable Sockmap acceleration.")
			} else {
				dp.sockmapState = st
				dp.sockmapState.PopulateCallbacks(callbacks)

				if err := dp.sockmapState.SetupSockmapAcceleration(); err != nil {
					dp.sockmapState = nil
					log.WithError(err).Warn("Failed to set up Sockmap acceleration")
				} else {
					log.Info("Sockmap acceleration enabled.")
				}
			}
		}
	}

	if dp.sockmapState == nil {
		st, err := NewSockmapState()
		if err == nil {
			st.WipeSockmap(bpf.FindInBPFFSOnly)
		}
		// if we can't create a sockmap state it means we couldn't get a working
		// bpffs so there's nothing to clean up
	}

	if !config.BPFEnabled {
		// BPF mode disabled, create the iptables-only managers.
		ipsetsManager := newIPSetsManager(ipSetsV4, config.MaxIPSetSize, dp.domainInfoStore, callbacks)
		dp.RegisterManager(ipsetsManager)
		dp.ipsetsSourceV4 = ipsetsManager
		// TODO Connect host IP manager to BPF
		dp.RegisterManager(newHostIPManager(
			config.RulesConfig.WorkloadIfacePrefixes,
			rules.IPSetIDThisHostIPs,
			ipSetsV4,
			config.MaxIPSetSize))
		dp.RegisterManager(newPolicyManager(rawTableV4, mangleTableV4, filterTableV4, ruleRenderer, 4, callbacks))

		// Clean up any leftover BPF state.
		err := nat.RemoveConnectTimeLoadBalancer("")
		if err != nil {
			log.WithError(err).Info("Failed to remove BPF connect-time load balancer, ignoring.")
		}
		tc.CleanUpProgramsAndPins()
	}

	interfaceRegexes := make([]string, len(config.RulesConfig.WorkloadIfacePrefixes))
	for i, r := range config.RulesConfig.WorkloadIfacePrefixes {
		interfaceRegexes[i] = "^" + r + ".*"
	}
	bpfMapContext := &bpf.MapContext{
		RepinningEnabled: config.BPFMapRepin,
	}

	if config.FlowLogsCollectProcessInfo {
		bpfEvnt, err := events.New(bpfMapContext, events.SourcePerfEvents)
		if err != nil {
			log.WithError(err).Panic("Failed to create perf event")
		}
		err = startEventPoller(bpfEvnt)
		if err != nil {
			log.WithError(err).Panic("Failed to start the event poller")
		}
		err = bpf.MountDebugfs()
		if err != nil {
			log.WithError(err).Panic("Failed to mount debug fs")
		}

		protov4Map := kprobe.MapProtov4(bpfMapContext)
		err = protov4Map.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create v4 protocol stats map")
		}
		err = kprobe.AttachTCPv4(config.BPFLogLevel, bpfEvnt, protov4Map)
		if err != nil {
			log.WithError(err).Panic("Failed to install TCP v4 kprobes")
		}
		err = kprobe.AttachUDPv4(config.BPFLogLevel, bpfEvnt, protov4Map)
		if err != nil {
			log.WithError(err).Panic("Failed to install UDP v4 kprobes")
		}
	}
	if config.BPFEnabled {
		log.Info("BPF enabled, starting BPF endpoint manager and map manager.")
		// Register map managers first since they create the maps that will be used by the endpoint manager.
		// Important that we create the maps before we load a BPF program with TC since we make sure the map
		// metadata name is set whereas TC doesn't set that field.
		ipSetIDAllocator := idalloc.New()
		ipSetsMap := bpfipsets.Map(bpfMapContext)
		err := ipSetsMap.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create ipsets BPF map.")
		}
		ipSetsV4 := bpfipsets.NewBPFIPSets(
			ipSetsConfigV4,
			ipSetIDAllocator,
			ipSetsMap,
		)
		dp.ipSets = append(dp.ipSets, ipSetsV4)
		dp.RegisterManager(newIPSetsManager(ipSetsV4, config.MaxIPSetSize, dp.domainInfoStore, callbacks))
		bpfRTMgr := newBPFRouteManager(config.Hostname, bpfMapContext)
		dp.RegisterManager(bpfRTMgr)

		// Forwarding into a tunnel seems to fail silently, disable FIB lookup if tunnel is enabled for now.
		fibLookupEnabled := !config.RulesConfig.IPIPEnabled && !config.RulesConfig.VXLANEnabled
		stateMap := state.Map(bpfMapContext)
		err = stateMap.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create state BPF map.")
		}

		arpMap := arp.Map(bpfMapContext)
		err = arpMap.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create ARP BPF map.")
		}

		workloadIfaceRegex := regexp.MustCompile(strings.Join(interfaceRegexes, "|"))
		dp.RegisterManager(newBPFEndpointManager(
			config.BPFLogLevel,
			config.Hostname,
			fibLookupEnabled,
			config.RulesConfig.EndpointToHostAction == "DROP",
			config.BPFDataIfacePattern,
			workloadIfaceRegex,
			ipSetIDAllocator,
			config.VXLANMTU,
			config.BPFNodePortDSREnabled,
			ipSetsMap,
			stateMap,
			ruleRenderer,
			filterTableV4,
			dp.reportHealth,
		))

		// Pre-create the NAT maps so that later operations can assume access.
		frontendMap := nat.FrontendMap(bpfMapContext)
		err = frontendMap.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create NAT frontend BPF map.")
		}
		backendMap := nat.BackendMap(bpfMapContext)
		err = backendMap.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create NAT backend BPF map.")
		}
		backendAffinityMap := nat.AffinityMap(bpfMapContext)
		err = backendAffinityMap.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create NAT backend affinity BPF map.")
		}

		routeMap := routes.Map(bpfMapContext)
		err = routeMap.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create routes BPF map.")
		}

		ctMap := conntrack.Map(bpfMapContext)
		err = ctMap.EnsureExists()
		if err != nil {
			log.WithError(err).Panic("Failed to create conntrack BPF map.")
		}

		bpfproxyOpts := []bpfproxy.Option{
			bpfproxy.WithMinSyncPeriod(config.KubeProxyMinSyncPeriod),
			bpfproxy.WithConntrackTimeouts(config.BPFConntrackTimeouts),
		}

		if config.KubeProxyEndpointSlicesEnabled {
			bpfproxyOpts = append(bpfproxyOpts, bpfproxy.WithEndpointsSlices())
		}

		if config.BPFNodePortDSREnabled {
			bpfproxyOpts = append(bpfproxyOpts, bpfproxy.WithDSREnabled())
		}

		if config.KubeClientSet != nil {
			// We have a Kubernetes connection, start watching services and populating the NAT maps.
			kp, err := bpfproxy.StartKubeProxy(
				config.KubeClientSet,
				config.Hostname,
				frontendMap,
				backendMap,
				backendAffinityMap,
				ctMap,
				bpfproxyOpts...,
			)
			if err != nil {
				log.WithError(err).Panic("Failed to start kube-proxy.")
			}
			bpfRTMgr.setHostIPUpdatesCallBack(kp.OnHostIPsUpdate)
			bpfRTMgr.setRoutesCallBacks(kp.OnRouteUpdate, kp.OnRouteDelete)
		} else {
			log.Info("BPF enabled but no Kubernetes client available, unable to run kube-proxy module.")
		}

		if config.BPFConnTimeLBEnabled {
			// Activate the connect-time load balancer.
			err = nat.InstallConnectTimeLoadBalancer(frontendMap, backendMap, routeMap, config.BPFCgroupV2, config.BPFLogLevel)
			if err != nil {
				log.WithError(err).Panic("BPFConnTimeLBEnabled but failed to attach connect-time load balancer, bailing out.")
			}
		} else {
			// Deactivate the connect-time load balancer.
			err = nat.RemoveConnectTimeLoadBalancer(config.BPFCgroupV2)
			if err != nil {
				log.WithError(err).Warn("Failed to detach connect-time load balancer. Ignoring.")
			}
		}
	}

	routeTableV4 := routetable.New(interfaceRegexes, 4, false, config.NetlinkTimeout,
		config.DeviceRouteSourceAddress, config.DeviceRouteProtocol, config.RemoveExternalRoutes, unix.RT_TABLE_UNSPEC,
		dp.loopSummarizer)

	epManager := newEndpointManager(
		rawTableV4,
		mangleTableV4,
		filterTableV4,
		ruleRenderer,
		routeTableV4,
		4,
		epMarkMapper,
		config.RulesConfig.KubeIPVSSupportEnabled,
		config.RulesConfig.WorkloadIfacePrefixes,
		dp.endpointStatusCombiner.OnEndpointStatusUpdate,
		config.BPFEnabled,
		callbacks)
	dp.RegisterManager(epManager)
	dp.endpointsSourceV4 = epManager
	dp.RegisterManager(newFloatingIPManager(natTableV4, ruleRenderer, 4))
	dp.RegisterManager(newMasqManager(ipSetsV4, natTableV4, ruleRenderer, config.MaxIPSetSize, 4))
	if config.RulesConfig.IPIPEnabled {
		// Create and maintain the IPIP tunnel device
		dp.ipipManager = newIPIPManager(ipSetsV4, config.MaxIPSetSize, config.ExternalNodesCidrs, dp.config)
	}

	if config.RulesConfig.IPIPEnabled || config.RulesConfig.IPSecEnabled || config.EgressIPEnabled {
		// Add a manager to keep the all-hosts IP set up to date.
		dp.allHostsIpsetManager = newAllHostsIpsetManager(ipSetsV4, config.MaxIPSetSize, config.ExternalNodesCidrs)
		dp.RegisterManager(dp.allHostsIpsetManager) // IPv4-only
	}

	// Add a manager for wireguard configuration. This is added irrespective of whether wireguard is actually enabled
	// because it may need to tidy up some of the routing rules when disabled.
	cryptoRouteTableWireguard := wireguard.New(config.Hostname, &config.Wireguard, config.NetlinkTimeout,
		config.DeviceRouteProtocol, func(publicKey wgtypes.Key) error {
			if publicKey == zeroKey {
				dp.fromDataplane <- &proto.WireguardStatusUpdate{PublicKey: ""}
			} else {
				dp.fromDataplane <- &proto.WireguardStatusUpdate{PublicKey: publicKey.String()}
			}
			return nil
		},
		dp.loopSummarizer)
	dp.wireguardManager = newWireguardManager(cryptoRouteTableWireguard)
	dp.RegisterManager(dp.wireguardManager) // IPv4-only

	dp.RegisterManager(newServiceLoopManager(filterTableV4, ruleRenderer, 4))

	var activeCaptures, err = capture.NewActiveCaptures(config.PacketCapture)
	if err != nil {
		log.WithError(err).Panicf("Failed create dir %s required to start packet capture", config.PacketCapture.Directory)
	}
	captureManager := newCaptureManager(activeCaptures, config.RulesConfig.WorkloadIfacePrefixes)
	dp.RegisterManager(captureManager)

	if config.IPv6Enabled {
		mangleTableV6 := iptables.NewTable(
			"mangle",
			6,
			rules.RuleHashPrefix,
			iptablesLock,
			featureDetector,
			iptablesOptions,
		)
		natTableV6 := iptables.NewTable(
			"nat",
			6,
			rules.RuleHashPrefix,
			iptablesLock,
			featureDetector,
			iptablesNATOptions,
		)
		rawTableV6 := iptables.NewTable(
			"raw",
			6,
			rules.RuleHashPrefix,
			iptablesLock,
			featureDetector,
			iptablesOptions,
		)
		filterTableV6 := iptables.NewTable(
			"filter",
			6,
			rules.RuleHashPrefix,
			iptablesLock,
			featureDetector,
			iptablesOptions,
		)

		ipSetsConfigV6 := config.RulesConfig.IPSetConfigV6
		ipSetsV6 := ipsets.NewIPSets(ipSetsConfigV6, dp.loopSummarizer)
		dp.ipSets = append(dp.ipSets, ipSetsV6)
		dp.iptablesNATTables = append(dp.iptablesNATTables, natTableV6)
		dp.iptablesRawTables = append(dp.iptablesRawTables, rawTableV6)
		dp.iptablesMangleTables = append(dp.iptablesMangleTables, mangleTableV6)
		dp.iptablesFilterTables = append(dp.iptablesFilterTables, filterTableV6)

		routeTableV6 := routetable.New(
			interfaceRegexes, 6, false, config.NetlinkTimeout,
			config.DeviceRouteSourceAddress, config.DeviceRouteProtocol, config.RemoveExternalRoutes,
			unix.RT_TABLE_UNSPEC, dp.loopSummarizer)

		if !config.BPFEnabled {
			dp.RegisterManager(newIPSetsManager(ipSetsV6, config.MaxIPSetSize, dp.domainInfoStore, callbacks))
			dp.RegisterManager(newHostIPManager(
				config.RulesConfig.WorkloadIfacePrefixes,
				rules.IPSetIDThisHostIPs,
				ipSetsV6,
				config.MaxIPSetSize))
			dp.RegisterManager(newPolicyManager(rawTableV6, mangleTableV6, filterTableV6, ruleRenderer, 6, callbacks))
		}
		dp.RegisterManager(newEndpointManager(
			rawTableV6,
			mangleTableV6,
			filterTableV6,
			ruleRenderer,
			routeTableV6,
			6,
			epMarkMapper,
			config.RulesConfig.KubeIPVSSupportEnabled,
			config.RulesConfig.WorkloadIfacePrefixes,
			dp.endpointStatusCombiner.OnEndpointStatusUpdate,
			config.BPFEnabled,
			callbacks))
		dp.RegisterManager(newFloatingIPManager(natTableV6, ruleRenderer, 6))
		dp.RegisterManager(newMasqManager(ipSetsV6, natTableV6, ruleRenderer, config.MaxIPSetSize, 6))
		dp.RegisterManager(newServiceLoopManager(filterTableV6, ruleRenderer, 6))
	}

	dp.allIptablesTables = append(dp.allIptablesTables, dp.iptablesMangleTables...)
	dp.allIptablesTables = append(dp.allIptablesTables, dp.iptablesNATTables...)
	dp.allIptablesTables = append(dp.allIptablesTables, dp.iptablesFilterTables...)
	dp.allIptablesTables = append(dp.allIptablesTables, dp.iptablesRawTables...)

	// We always create the IPsec policy table (the component that manipulates the IPsec dataplane).  That ensures
	// that we clean up our old policies if IPsec is disabled.
	ipsecEnabled := config.IPSecPSK != "" && config.IPSecESPProposal != "" && config.IPSecIKEProposal != "" && config.NodeIP != nil
	dp.ipSecPolTable = ipsec.NewPolicyTable(ipsec.ReqID, ipsecEnabled, config.DebugUseShortPollIntervals)
	if ipsecEnabled {
		// Set up IPsec.

		// Initialise charon main config file.
		charonConfig := ipsec.NewCharonConfig(ipsec.CharonConfigRootDir, ipsec.CharonMainConfigFile)
		charonConfig.SetLogLevel(config.IPSecLogLevel)
		charonConfig.SetBooleanOption(ipsec.CharonFollowRedirects, false)
		charonConfig.SetBooleanOption(ipsec.CharonMakeBeforeBreak, true)
		log.Infof("Initialising charon config %+v", charonConfig)
		charonConfig.RenderToFile()
		ikeDaemon := ipsec.NewCharonIKEDaemon(
			config.IPSecESPProposal,
			config.IPSecIKEProposal,
			config.IPSecRekeyTime,
			config.ChildExitedRestartCallback,
		)
		var charonWG sync.WaitGroup
		err := ikeDaemon.Start(context.Background(), &charonWG)
		if err != nil {
			log.WithError(err).Panic("error starting Charon.")
		}

		dp.ipSecDataplane = ipsec.NewDataplane(
			config.NodeIP,
			config.IPSecPSK,
			config.RulesConfig.IptablesMarkIPsec,
			dp.ipSecPolTable,
			ikeDaemon,
			config.IPSecAllowUnsecuredTraffic,
		)
		ipSecManager := newIPSecManager(dp.ipSecDataplane)
		dp.allManagers = append(dp.allManagers, ipSecManager)
	}

	// Register that we will report liveness and readiness.
	if config.HealthAggregator != nil {
		log.Info("Registering to report health.")
		config.HealthAggregator.RegisterReporter(
			healthName,
			&health.HealthReport{Live: true, Ready: true},
			healthInterval*2,
		)
	}

	if config.DebugSimulateDataplaneHangAfter != 0 {
		log.WithField("delay", config.DebugSimulateDataplaneHangAfter).Warn(
			"Simulating a dataplane hang.")
		dp.debugHangC = time.After(config.DebugSimulateDataplaneHangAfter)
	}

	// If required, subscribe to NFLog collection.
	if config.Collector != nil {
		log.Debug("Stats collection is required, subscribe to nflogs")
		config.Collector.SubscribeToNflog()
	}

	return dp
}

// findHostMTU auto-detects the smallest host interface MTU.
func findHostMTU(matchRegex *regexp.Regexp) (int, error) {
	// Find all the interfaces on the host.
	links, err := netlink.LinkList()
	if err != nil {
		log.WithError(err).Error("Failed to list interfaces. Unable to auto-detect MTU")
		return 0, err
	}

	// Iterate through them, keeping track of the lowest MTU.
	smallest := 0
	for _, l := range links {
		// Skip links that we know are not external interfaces.
		fields := log.Fields{"mtu": l.Attrs().MTU, "name": l.Attrs().Name}
		if matchRegex == nil || !matchRegex.MatchString(l.Attrs().Name) {
			log.WithFields(fields).Debug("Skipping interface for MTU detection")
			continue
		}
		log.WithFields(fields).Debug("Examining link for MTU calculation")
		if l.Attrs().MTU < smallest || smallest == 0 {
			smallest = l.Attrs().MTU
		}
	}

	if smallest == 0 {
		// We failed to find a usable interface. Default the MTU of the host
		// to 1460 - the smallest among common cloud providers.
		log.Warn("Failed to auto-detect host MTU - no interfaces matched the MTU interface pattern. To use auto-MTU, set mtuIfacePattern to match your host's interfaces")
		return 1460, nil
	}
	return smallest, nil
}

// writeMTUFile writes the smallest MTU among enabled encapsulation types to disk
// for use by other components (e.g., CNI plugin).
func writeMTUFile(config Config) error {
	// Make sure directory exists.
	if err := os.MkdirAll("/var/lib/calico", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory /var/lib/calico: %s", err)
	}

	// Write the smallest MTU to disk so other components can rely on this calculation consistently.
	mtu := determinePodMTU(config)
	filename := "/var/lib/calico/mtu"
	log.Debugf("Writing %d to "+filename, mtu)
	if err := ioutil.WriteFile(filename, []byte(fmt.Sprintf("%d", mtu)), 0644); err != nil {
		log.WithError(err).Error("Unable to write to " + filename)
		return err
	}
	return nil
}

// determinePodMTU looks at the configured MTUs and enabled encapsulations to determine which
// value for MTU should be used for pod interfaces.
func determinePodMTU(config Config) int {
	// Determine the smallest MTU among enabled encap methods. If none of the encap methods are
	// enabled, we'll just use the host's MTU.
	mtu := 0
	type mtuState struct {
		mtu     int
		enabled bool
	}
	for _, s := range []mtuState{
		{config.IPIPMTU, config.RulesConfig.IPIPEnabled},
		{config.VXLANMTU, config.RulesConfig.VXLANEnabled},
		{config.Wireguard.MTU, config.Wireguard.Enabled},
	} {
		if s.enabled && s.mtu != 0 && (s.mtu < mtu || mtu == 0) {
			mtu = s.mtu
		}
	}

	if mtu == 0 {
		// No enabled encapsulation. Just use the host MTU.
		mtu = config.hostMTU
	} else if mtu > config.hostMTU {
		fields := logrus.Fields{"mtu": mtu, "hostMTU": config.hostMTU}
		log.WithFields(fields).Warn("Configured MTU is larger than detected host interface MTU")
	}
	log.WithField("mtu", mtu).Info("Determined pod MTU")
	return mtu
}

func cleanUpVXLANDevice() {
	// If VXLAN is not enabled, check to see if there is a VXLAN device and delete it if there is.
	log.Debug("Checking if we need to clean up the VXLAN device")
	link, err := netlink.LinkByName("vxlan.calico")
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			log.Debug("VXLAN disabled and no VXLAN device found")
			return
		}
		log.WithError(err).Warnf("VXLAN disabled and failed to query VXLAN device.  Ignoring.")
		return
	}
	if err = netlink.LinkDel(link); err != nil {
		log.WithError(err).Error("VXLAN disabled and failed to delete unwanted VXLAN device. Ignoring.")
	}
}

type Manager interface {
	// OnUpdate is called for each protobuf message from the datastore.  May either directly
	// send updates to the IPSets and iptables.Table objects (which will queue the updates
	// until the main loop instructs them to act) or (for efficiency) may wait until
	// a call to CompleteDeferredWork() to flush updates to the dataplane.
	OnUpdate(protoBufMsg interface{})
	// Called before the main loop flushes updates to the dataplane to allow for batched
	// work to be completed.
	CompleteDeferredWork() error
}

type ManagerWithRouteTables interface {
	Manager
	GetRouteTableSyncers() []routeTableSyncer
}

type ManagerWithRouteRules interface {
	Manager
	GetRouteRules() []routeRules
}

func (d *InternalDataplane) routeTableSyncers() []routeTableSyncer {
	var rts []routeTableSyncer
	for _, mrts := range d.managersWithRouteTables {
		rts = append(rts, mrts.GetRouteTableSyncers()...)
	}

	return rts
}

func (d *InternalDataplane) routeRules() []routeRules {
	var rrs []routeRules
	for _, mrrs := range d.managersWithRouteRules {
		rrs = append(rrs, mrrs.GetRouteRules()...)
	}

	return rrs
}

func (d *InternalDataplane) RegisterManager(mgr Manager) {
	tableMgr, ok := mgr.(ManagerWithRouteTables)
	if ok {
		log.WithField("manager", reflect.TypeOf(mgr).Name()).Debug("registering ManagerWithRouteTables")
		d.managersWithRouteTables = append(d.managersWithRouteTables, tableMgr)
	}

	rulesMgr, ok := mgr.(ManagerWithRouteRules)
	if ok {
		log.WithField("manager", reflect.TypeOf(mgr).Name()).Debug("registering ManagerWithRouteRules")
		d.managersWithRouteRules = append(d.managersWithRouteRules, rulesMgr)
	}
	d.allManagers = append(d.allManagers, mgr)
}

func (d *InternalDataplane) Start() {
	// Do our start-of-day configuration.
	d.doStaticDataplaneConfig()

	// Then, start the worker threads.
	go d.loopUpdatingDataplane()
	go d.loopReportingStatus()
	go d.ifaceMonitor.MonitorInterfaces()
	go d.monitorHostMTU()

	// Start DNS response capture.
	d.domainInfoStore.Start()
}

// onIfaceStateChange is our interface monitor callback.  It gets called from the monitor's thread.
func (d *InternalDataplane) onIfaceStateChange(ifaceName string, state ifacemonitor.State, ifIndex int) {
	log.WithFields(log.Fields{
		"ifaceName": ifaceName,
		"ifIndex":   ifIndex,
		"state":     state,
	}).Info("Linux interface state changed.")
	d.ifaceUpdates <- &ifaceUpdate{
		Name:  ifaceName,
		State: state,
		Index: ifIndex,
	}
}

type ifaceUpdate struct {
	Name  string
	State ifacemonitor.State
	Index int
}

// Check if current felix ipvs config is correct when felix gets an kube-ipvs0 interface update.
// If KubeIPVSInterface is UP and felix ipvs support is disabled (kube-proxy switched from iptables to ipvs mode),
// or if KubeIPVSInterface is DOWN and felix ipvs support is enabled (kube-proxy switched from ipvs to iptables mode),
// restart felix to pick up correct ipvs support mode.
func (d *InternalDataplane) checkIPVSConfigOnStateUpdate(state ifacemonitor.State) {
	if (!d.config.RulesConfig.KubeIPVSSupportEnabled && state == ifacemonitor.StateUp) ||
		(d.config.RulesConfig.KubeIPVSSupportEnabled && state == ifacemonitor.StateDown) {
		log.WithFields(log.Fields{
			"ipvsIfaceState": state,
			"ipvsSupport":    d.config.RulesConfig.KubeIPVSSupportEnabled,
		}).Info("kube-proxy mode changed. Restart felix.")
		d.config.ConfigChangedRestartCallback()
	}
}

// onIfaceAddrsChange is our interface address monitor callback.  It gets called
// from the monitor's thread.
func (d *InternalDataplane) onIfaceAddrsChange(ifaceName string, addrs set.Set) {
	log.WithFields(log.Fields{
		"ifaceName": ifaceName,
		"addrs":     addrs,
	}).Info("Linux interface addrs changed.")
	d.ifaceAddrUpdates <- &ifaceAddrsUpdate{
		Name:  ifaceName,
		Addrs: addrs,
	}
}

type ifaceAddrsUpdate struct {
	Name  string
	Addrs set.Set
}

func (d *InternalDataplane) SendMessage(msg interface{}) error {
	d.toDataplane <- msg
	return nil
}

func (d *InternalDataplane) RecvMessage() (interface{}, error) {
	return <-d.fromDataplane, nil
}

func (d *InternalDataplane) monitorHostMTU() {
	for {
		mtu, err := findHostMTU(d.config.MTUIfacePattern)
		if err != nil {
			log.WithError(err).Error("Error detecting host MTU")
		} else if d.config.hostMTU != mtu {
			// Since log writing is done a background thread, we set the force-flush flag on this log to ensure that
			// all the in-flight logs get written before we exit.
			log.WithFields(log.Fields{lclogutils.FieldForceFlush: true}).Info("Host MTU changed")
			d.config.ConfigChangedRestartCallback()
		}
		time.Sleep(30 * time.Second)
	}
}

// doStaticDataplaneConfig sets up the kernel and our static iptables  chains.  Should be called
// once at start of day before starting the main loop.  The actual iptables programming is deferred
// to the main loop.
func (d *InternalDataplane) doStaticDataplaneConfig() {
	// Check/configure global kernel parameters.
	d.configureKernel()

	if d.config.BPFEnabled {
		d.setUpIptablesBPF()
	} else {
		d.setUpIptablesNormal()
	}

	if d.config.RulesConfig.IPIPEnabled {
		log.Info("IPIP enabled, starting thread to keep tunnel configuration in sync.")
		go d.ipipManager.KeepIPIPDeviceInSync(
			d.config.IPIPMTU,
			d.config.RulesConfig.IPIPTunnelAddress,
		)
	} else {
		log.Info("IPIP disabled. Not starting tunnel update thread.")
	}
}

func (d *InternalDataplane) setUpIptablesBPF() {
	// TODO Make make bits configurable.

	for _, t := range d.iptablesFilterTables {
		fwdRules := []iptables.Rule{
			{
				// Bypass is a strong signal from the BPF program, it means that the flow is approved
				// by the program at both ingress and egress.
				Comment: []string{"Pre-approved by BPF programs."},
				Match:   iptables.Match().MarkMatchesWithMask(tc.MarkSeenBypass, tc.MarkSeenBypassMask),
				Action:  iptables.AcceptAction{},
			},
		}

		var inputRules []iptables.Rule
		for _, prefix := range d.config.RulesConfig.WorkloadIfacePrefixes {
			fwdRules = append(fwdRules,
				// Drop packets that have come from a workload but have not been through our BPF program.
				iptables.Rule{
					Match:   iptables.Match().InInterface(prefix+"+").NotMarkMatchesWithMask(tc.MarkSeen, tc.MarkSeenMask),
					Action:  iptables.DropAction{},
					Comment: []string{"From workload without BPF seen mark"},
				},
			)

			if d.config.RulesConfig.EndpointToHostAction == "ACCEPT" {
				// Only need to worry about ACCEPT here.  Drop gets compiled into the BPF program and
				// RETURN would be a no-op since there's nothing to RETURN from.
				inputRules = append(inputRules, iptables.Rule{
					Match:  iptables.Match().InInterface(prefix+"+").MarkMatchesWithMask(tc.MarkSeen, tc.MarkSeenMask),
					Action: iptables.AcceptAction{},
				})
			}
			// Catch any workload to host packets that haven't been through the BPF program.
			inputRules = append(inputRules, iptables.Rule{
				Match:  iptables.Match().InInterface(prefix+"+").NotMarkMatchesWithMask(tc.MarkSeen, tc.MarkSeenMask),
				Action: iptables.DropAction{},
			})
		}

		if t.IPVersion == 6 {
			for _, prefix := range d.config.RulesConfig.WorkloadIfacePrefixes {
				// In BPF mode, we don't support IPv6 yet.  Drop it.
				fwdRules = append(fwdRules, iptables.Rule{
					Match:   iptables.Match().OutInterface(prefix + "+"),
					Action:  iptables.DropAction{},
					Comment: []string{"To workload, drop IPv6."},
				})
			}
		} else {
			// The packet may be about to go to a local workload.  However, the local workload may not have a BPF
			// program attached (yet).  To catch that case, we send the packet through a dispatch chain.  We only
			// add interfaces to the dispatch chain if the BPF program is in place.
			for _, prefix := range d.config.RulesConfig.WorkloadIfacePrefixes {
				// Make sure iptables rules don't drop packets that we're about to process through BPF.
				fwdRules = append(fwdRules,
					iptables.Rule{
						Match:   iptables.Match().OutInterface(prefix + "+"),
						Action:  iptables.JumpAction{Target: rules.ChainToWorkloadDispatch},
						Comment: []string{"To workload, check workload is known."},
					},
				)
			}
			// Need a final rule to accept traffic that is from a workload and going somewhere else.
			// Otherwise, if iptables has a DROP policy on the forward chain, the packet will get dropped.
			// This rule must come after the to-workload jump rules above to ensure that we don't accept too
			// early before the destination is checked.
			for _, prefix := range d.config.RulesConfig.WorkloadIfacePrefixes {
				// Make sure iptables rules don't drop packets that we're about to process through BPF.
				fwdRules = append(fwdRules,
					iptables.Rule{
						Match:   iptables.Match().InInterface(prefix + "+"),
						Action:  iptables.AcceptAction{},
						Comment: []string{"To workload, mark has already been verified."},
					},
				)
			}
		}

		t.InsertOrAppendRules("INPUT", inputRules)
		t.InsertOrAppendRules("FORWARD", fwdRules)
	}

	for _, t := range d.iptablesNATTables {
		t.UpdateChains(d.ruleRenderer.StaticNATPostroutingChains(t.IPVersion))
		t.InsertOrAppendRules("POSTROUTING", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainNATPostrouting},
		}})
	}

	for _, t := range d.iptablesRawTables {
		// Do not RPF check what is marked as to be skipped by RPF check.
		rpfRules := []iptables.Rule{{
			Match:  iptables.Match().MarkMatchesWithMask(tc.MarkSeenBypassSkipRPF, tc.MarkSeenBypassSkipRPFMask),
			Action: iptables.ReturnAction{},
		}}

		// For anything we approved for forward, permit accept_local as it is
		// traffic encapped for NodePort, ICMP replies etc. - stuff we trust.
		rpfRules = append(rpfRules, iptables.Rule{
			Match:  iptables.Match().MarkMatchesWithMask(tc.MarkSeenBypassForward, tc.MarksMask).RPFCheckPassed(true),
			Action: iptables.ReturnAction{},
		})

		rpfRules = append(rpfRules, d.ruleRenderer.RPFilter(t.IPVersion, tc.MarkSeen, tc.MarkSeenMask,
			d.config.RulesConfig.OpenStackSpecialCasesEnabled, false)...)

		rpfChain := []*iptables.Chain{{
			Name:  rules.ChainNamePrefix + "RPF",
			Rules: rpfRules,
		}}
		t.UpdateChains(rpfChain)

		rawChains := []*iptables.Chain{{
			Name: rules.ChainRawPrerouting,
			Rules: []iptables.Rule{{
				Action: iptables.JumpAction{Target: rpfChain[0].Name},
			}},
		}}
		t.UpdateChains(rawChains)

		t.InsertOrAppendRules("PREROUTING", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainRawPrerouting},
		}})
	}
}

func (d *InternalDataplane) setUpIptablesNormal() {
	for _, t := range d.iptablesRawTables {
		rawChains := d.ruleRenderer.StaticRawTableChains(t.IPVersion)
		t.UpdateChains(rawChains)
		t.InsertOrAppendRules("PREROUTING", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainRawPrerouting},
		}})
		t.InsertOrAppendRules("OUTPUT", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainRawOutput},
		}})
	}
	for _, t := range d.iptablesFilterTables {
		filterChains := d.ruleRenderer.StaticFilterTableChains(t.IPVersion)
		t.UpdateChains(filterChains)
		t.InsertOrAppendRules("FORWARD", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainFilterForward},
		}})
		t.InsertOrAppendRules("INPUT", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainFilterInput},
		}})
		t.InsertOrAppendRules("OUTPUT", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainFilterOutput},
		}})
	}
	for _, t := range d.iptablesNATTables {
		t.UpdateChains(d.ruleRenderer.StaticNATTableChains(t.IPVersion))
		t.InsertOrAppendRules("PREROUTING", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainNATPrerouting},
		}})
		if t.IPVersion == 4 && d.config.EgressIPEnabled {
			t.AppendRules("PREROUTING", []iptables.Rule{{
				Action: iptables.JumpAction{Target: rules.ChainNATPreroutingEgress},
			}})
		}
		t.InsertOrAppendRules("POSTROUTING", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainNATPostrouting},
		}})
		t.InsertOrAppendRules("OUTPUT", []iptables.Rule{{
			Action: iptables.JumpAction{Target: rules.ChainNATOutput},
		}})
	}
	for _, t := range d.iptablesMangleTables {
		chains := d.ruleRenderer.StaticMangleTableChains(t.IPVersion)
		t.UpdateChains(chains)
		rs := []iptables.Rule{}
		if t.IPVersion == 4 && d.config.EgressIPEnabled {
			// Make sure egress rule at top.
			rs = append(rs, iptables.Rule{
				Action: iptables.JumpAction{Target: rules.ChainManglePreroutingEgress},
			})
		}
		rs = append(rs, iptables.Rule{
			Action: iptables.JumpAction{Target: rules.ChainManglePrerouting},
		})
		t.InsertOrAppendRules("PREROUTING", rs)

		rs = []iptables.Rule{}
		for _, chain := range chains {
			if chain.Name == rules.ChainManglePostroutingEgress {
				rs = append(rs, iptables.Rule{
					Action: iptables.JumpAction{Target: rules.ChainManglePostroutingEgress},
				})
				break
			}
		}
		rs = append(rs, iptables.Rule{
			Action: iptables.JumpAction{Target: rules.ChainManglePostrouting},
		})
		t.InsertOrAppendRules("POSTROUTING", rs)
	}
	if d.xdpState != nil {
		if err := d.setXDPFailsafePorts(); err != nil {
			log.Warnf("failed to set XDP failsafe ports, disabling XDP: %v", err)
			if err := d.shutdownXDPCompletely(); err != nil {
				log.Warnf("failed to disable XDP: %v, will proceed anyway.", err)
			}
		}
	}
}

func stringToProtocol(protocol string) (labelindex.IPSetPortProtocol, error) {
	switch protocol {
	case "tcp":
		return labelindex.ProtocolTCP, nil
	case "udp":
		return labelindex.ProtocolUDP, nil
	case "sctp":
		return labelindex.ProtocolSCTP, nil
	}
	return labelindex.ProtocolNone, fmt.Errorf("unknown protocol %q", protocol)
}

func (d *InternalDataplane) setXDPFailsafePorts() error {
	inboundPorts := d.config.RulesConfig.FailsafeInboundHostPorts

	if _, err := d.xdpState.common.bpfLib.NewFailsafeMap(); err != nil {
		return err
	}

	for _, p := range inboundPorts {
		proto, err := stringToProtocol(p.Protocol)
		if err != nil {
			return err
		}

		if err := d.xdpState.common.bpfLib.UpdateFailsafeMap(uint8(proto), p.Port); err != nil {
			return err
		}
	}

	log.Infof("Set XDP failsafe ports: %+v", inboundPorts)
	return nil
}

// shutdownXDPCompletely attempts to disable XDP state.  This could fail in cases where XDP isn't working properly.
func (d *InternalDataplane) shutdownXDPCompletely() error {
	if d.xdpState == nil {
		return nil
	}
	if d.callbacks != nil {
		d.xdpState.DepopulateCallbacks(d.callbacks)
	}
	// spend 1 second attempting to wipe XDP, in case of a hiccup.
	maxTries := 10
	waitInterval := 100 * time.Millisecond
	var err error
	for i := 0; i < maxTries; i++ {
		err = d.xdpState.WipeXDP()
		if err == nil {
			d.xdpState = nil
			return nil
		}
		log.WithError(err).WithField("try", i).Warn("failed to wipe the XDP state")
		time.Sleep(waitInterval)
	}
	return fmt.Errorf("Failed to wipe the XDP state after %v tries over %v seconds: Error %v", maxTries, waitInterval, err)
}

func (d *InternalDataplane) loopUpdatingDataplane() {
	log.Info("Started internal iptables dataplane driver loop")
	healthTicks := time.NewTicker(healthInterval).C
	d.reportHealth()

	// Retry any failed operations every 10s.
	retryTicker := time.NewTicker(10 * time.Second)

	// If configured, start tickers to refresh the IP sets and routing table entries.
	var ipSetsRefreshC <-chan time.Time
	if d.config.IPSetsRefreshInterval > 0 {
		log.WithField("interval", d.config.IptablesRefreshInterval).Info(
			"Will refresh IP sets on timer")
		refreshTicker := jitter.NewTicker(
			d.config.IPSetsRefreshInterval,
			d.config.IPSetsRefreshInterval/10,
		)
		ipSetsRefreshC = refreshTicker.Channel()
	}
	var routeRefreshC <-chan time.Time
	if d.config.RouteRefreshInterval > 0 {
		log.WithField("interval", d.config.RouteRefreshInterval).Info(
			"Will refresh routes on timer")
		refreshTicker := jitter.NewTicker(
			d.config.RouteRefreshInterval,
			d.config.RouteRefreshInterval/10,
		)
		routeRefreshC = refreshTicker.Channel()
	}
	var xdpRefreshC <-chan time.Time
	if d.config.XDPRefreshInterval > 0 && d.xdpState != nil {
		log.WithField("interval", d.config.XDPRefreshInterval).Info(
			"Will refresh XDP on timer")
		refreshTicker := jitter.NewTicker(
			d.config.XDPRefreshInterval,
			d.config.XDPRefreshInterval/10,
		)
		xdpRefreshC = refreshTicker.Channel()
	}
	var ipSecRefreshC <-chan time.Time
	if d.config.IPSecPolicyRefreshInterval > 0 {
		log.WithField("interval", d.config.IPSecPolicyRefreshInterval).Info(
			"Will recheck IPsec policy on timer")
		refreshTicker := jitter.NewTicker(
			d.config.IPSecPolicyRefreshInterval,
			d.config.IPSecPolicyRefreshInterval/10,
		)
		ipSecRefreshC = refreshTicker.Channel()
	}

	// Fill the apply throttle leaky bucket.
	throttleC := jitter.NewTicker(100*time.Millisecond, 10*time.Millisecond).Channel()
	beingThrottled := false

	datastoreInSync := false

	processMsgFromCalcGraph := func(msg interface{}) {
		log.WithField("msg", proto.MsgStringer{Msg: msg}).Infof(
			"Received %T update from calculation graph", msg)
		d.recordMsgStat(msg)
		for _, mgr := range d.allManagers {
			mgr.OnUpdate(msg)
		}
		switch msg.(type) {
		case *proto.InSync:
			log.WithField("timeSinceStart", time.Since(processStartTime)).Info(
				"Datastore in sync, flushing the dataplane for the first time...")
			datastoreInSync = true
		}
	}

	processIfaceUpdate := func(ifaceUpdate *ifaceUpdate) {
		log.WithField("msg", ifaceUpdate).Info("Received interface update")
		if ifaceUpdate.Name == KubeIPVSInterface {
			d.checkIPVSConfigOnStateUpdate(ifaceUpdate.State)
			return
		}

		for _, mgr := range d.allManagers {
			mgr.OnUpdate(ifaceUpdate)
		}

		for _, mgr := range d.managersWithRouteTables {
			for _, routeTable := range mgr.GetRouteTableSyncers() {
				routeTable.OnIfaceStateChanged(ifaceUpdate.Name, ifaceUpdate.State)
			}
		}
	}

	processAddrsUpdate := func(ifaceAddrsUpdate *ifaceAddrsUpdate) {
		log.WithField("msg", ifaceAddrsUpdate).Info("Received interface addresses update")
		for _, mgr := range d.allManagers {
			mgr.OnUpdate(ifaceAddrsUpdate)
		}
	}

	for {
		select {
		case msg := <-d.toDataplane:
			// Process the message we received, then opportunistically process any other
			// pending messages.
			batchSize := 1
			processMsgFromCalcGraph(msg)
		msgLoop1:
			for i := 0; i < msgPeekLimit; i++ {
				select {
				case msg := <-d.toDataplane:
					processMsgFromCalcGraph(msg)
					batchSize++
				default:
					// Channel blocked so we must be caught up.
					break msgLoop1
				}
			}
			d.dataplaneNeedsSync = true
			summaryBatchSize.Observe(float64(batchSize))
		case ifaceUpdate := <-d.ifaceUpdates:
			// Process the message we received, then opportunistically process any other
			// pending messages.
			batchSize := 1
			processIfaceUpdate(ifaceUpdate)
		msgLoop2:
			for i := 0; i < msgPeekLimit; i++ {
				select {
				case ifaceUpdate := <-d.ifaceUpdates:
					processIfaceUpdate(ifaceUpdate)
					batchSize++
				default:
					// Channel blocked so we must be caught up.
					break msgLoop2
				}
			}
			d.dataplaneNeedsSync = true
			summaryIfaceBatchSize.Observe(float64(batchSize))
		case ifaceAddrsUpdate := <-d.ifaceAddrUpdates:
			batchSize := 1
			processAddrsUpdate(ifaceAddrsUpdate)
		msgLoop3:
			for i := 0; i < msgPeekLimit; i++ {
				select {
				case ifaceAddrsUpdate := <-d.ifaceAddrUpdates:
					processAddrsUpdate(ifaceAddrsUpdate)
					batchSize++
				default:
					// Channel blocked so we must be caught up.
					break msgLoop3
				}
			}
			summaryAddrBatchSize.Observe(float64(batchSize))
			d.dataplaneNeedsSync = true
		case domainInfoChange := <-d.domainInfoChanges:
			// Opportunistically read and coalesce other domain change signals that are
			// already pending on this channel.
			domainChangeSignals := []*domainInfoChanged{domainInfoChange}
			domainsChanged := set.From(domainInfoChange.domain)
		domainChangeLoop:
			for {
				select {
				case domainInfoChange := <-d.domainInfoChanges:
					if !domainsChanged.Contains(domainInfoChange.domain) {
						domainChangeSignals = append(domainChangeSignals, domainInfoChange)
						domainsChanged.Add(domainInfoChange.domain)
					}
				default:
					// Channel blocked so we've caught up.
					break domainChangeLoop
				}
			}
			for _, domainInfoChange = range domainChangeSignals {
				for _, mgr := range d.allManagers {
					if handler, ok := mgr.(DomainInfoChangeHandler); ok {
						if handler.OnDomainInfoChange(domainInfoChange) {
							d.dataplaneNeedsSync = true
						}
					}
				}
			}
		case <-ipSetsRefreshC:
			log.Debug("Refreshing IP sets state")
			d.forceIPSetsRefresh = true
			d.dataplaneNeedsSync = true
		case <-routeRefreshC:
			log.Debug("Refreshing routes")
			d.forceRouteRefresh = true
			d.dataplaneNeedsSync = true
		case <-xdpRefreshC:
			log.Debug("Refreshing XDP")
			d.forceXDPRefresh = true
			d.dataplaneNeedsSync = true
		case <-ipSecRefreshC:
			d.ipSecPolTable.QueueResync()
		case <-d.reschedC:
			log.Debug("Reschedule kick received")
			d.dataplaneNeedsSync = true
			// nil out the channel to record that the timer is now inactive.
			d.reschedC = nil
		case <-throttleC:
			d.applyThrottle.Refill()
		case <-healthTicks:
			d.reportHealth()
		case <-retryTicker.C:
		case <-d.debugHangC:
			log.Warning("Debug hang simulation timer popped, hanging the dataplane!!")
			time.Sleep(1 * time.Hour)
			log.Panic("Woke up after 1 hour, something's probably wrong with the test.")
		case stopWG := <-d.stopChan:
			defer stopWG.Done()
			if err := d.domainInfoStore.saveMappingsV1(); err != nil {
				log.WithError(err).Warning("Failed to save mappings to file on Felix shutdown")

			}
			return
		}

		if datastoreInSync && d.dataplaneNeedsSync {
			// Dataplane is out-of-sync, check if we're throttled.
			if d.applyThrottle.Admit() {
				if beingThrottled && d.applyThrottle.WouldAdmit() {
					log.Info("Dataplane updates no longer throttled")
					beingThrottled = false
				}
				log.Debug("Applying dataplane updates")
				applyStart := time.Now()

				// Actually apply the changes to the dataplane.
				d.apply()

				// Record stats.
				applyTime := time.Since(applyStart)
				summaryApplyTime.Observe(applyTime.Seconds())

				if d.dataplaneNeedsSync {
					// Dataplane is still dirty, record an error.
					countDataplaneSyncErrors.Inc()
				}

				d.loopSummarizer.EndOfIteration(applyTime)

				if !d.doneFirstApply {
					log.WithField(
						"secsSinceStart", time.Since(processStartTime).Seconds(),
					).Info("Completed first update to dataplane.")
					d.loopSummarizer.RecordOperation("first-update")
					d.doneFirstApply = true
					if d.config.PostInSyncCallback != nil {
						d.config.PostInSyncCallback()
					}
				}
				d.reportHealth()
			} else {
				if !beingThrottled {
					log.Info("Dataplane updates throttled")
					beingThrottled = true
				}
			}
		}
	}
}

func (d *InternalDataplane) configureKernel() {
	// Attempt to modprobe nf_conntrack_proto_sctp.  In some kernels this is a
	// module that needs to be loaded, otherwise all SCTP packets are marked
	// INVALID by conntrack and dropped by Calico's rules.  However, some kernels
	// (confirmed in Ubuntu 19.10's build of 5.3.0-24-generic) include this
	// conntrack without it being a kernel module, and so modprobe will fail.
	// Log result at INFO level for troubleshooting, but otherwise ignore any
	// failed modprobe calls.
	mp := newModProbe(moduleConntrackSCTP, newRealCmd)
	out, err := mp.Exec()
	log.WithError(err).WithField("output", out).Infof("attempted to modprobe %s", moduleConntrackSCTP)

	log.Info("Making sure IPv4 forwarding is enabled.")
	err = writeProcSys("/proc/sys/net/ipv4/ip_forward", "1")
	if err != nil {
		log.WithError(err).Error("Failed to set IPv4 forwarding sysctl")
	}

	if d.config.IPv6Enabled {
		log.Info("Making sure IPv6 forwarding is enabled.")
		err = writeProcSys("/proc/sys/net/ipv6/conf/all/forwarding", "1")
		if err != nil {
			log.WithError(err).Error("Failed to set IPv6 forwarding sysctl")
		}
	}

	// Enable conntrack packet and byte accounting.
	err = writeProcSys("/proc/sys/net/netfilter/nf_conntrack_acct", "1")
	if err != nil {
		log.Warnf("failed to set enable conntrack packet and byte accounting: %v\n", err)
	}

	if d.config.BPFEnabled && d.config.BPFDisableUnprivileged {
		log.Info("BPF enabled, disabling unprivileged BPF usage.")
		err := writeProcSys("/proc/sys/kernel/unprivileged_bpf_disabled", "1")
		if err != nil {
			log.WithError(err).Error("Failed to set unprivileged_bpf_disabled sysctl")
		}
	}
	if d.config.Wireguard.Enabled {
		// wireguard module is available in linux kernel >= 5.6
		mpwg := newModProbe(moduleWireguard, newRealCmd)
		out, err = mpwg.Exec()
		log.WithError(err).WithField("output", out).Infof("attempted to modprobe %s", moduleWireguard)
	}
}

func (d *InternalDataplane) recordMsgStat(msg interface{}) {
	typeName := reflect.ValueOf(msg).Elem().Type().Name()
	countMessages.WithLabelValues(typeName).Inc()
}

func (d *InternalDataplane) apply() {
	// Update sequencing is important here because iptables rules have dependencies on ipsets.
	// Creating a rule that references an unknown IP set fails, as does deleting an IP set that
	// is in use.

	// Unset the needs-sync flag, we'll set it again if something fails.
	d.dataplaneNeedsSync = false

	// First, give the managers a chance to update IP sets and iptables.
	for _, mgr := range d.allManagers {
		err := mgr.CompleteDeferredWork()
		if err != nil {
			log.WithField("manager", reflect.TypeOf(mgr).Name()).WithError(err).Debug(
				"couldn't complete deferred work for manager, will try again later")
			d.dataplaneNeedsSync = true
		}
		d.reportHealth()
	}

	if d.xdpState != nil {
		if d.forceXDPRefresh {
			// Refresh timer popped.
			d.xdpState.QueueResync()
			d.forceXDPRefresh = false
		}

		var applyXDPError error
		d.xdpState.ProcessPendingDiffState(d.endpointsSourceV4)
		if err := d.applyXDPActions(); err != nil {
			applyXDPError = err
		} else {
			err := d.xdpState.ProcessMemberUpdates()
			d.xdpState.DropPendingDiffState()
			if err != nil {
				log.WithError(err).Warning("Failed to process XDP member updates, will resync later...")
				if err := d.applyXDPActions(); err != nil {
					applyXDPError = err
				}
			}
			d.xdpState.UpdateState()
		}
		if applyXDPError != nil {
			log.WithError(applyXDPError).Info("Applying XDP actions did not succeed, disabling XDP")
			if err := d.shutdownXDPCompletely(); err != nil {
				log.Warnf("failed to disable XDP: %v, will proceed anyway.", err)
			}
		}
	}
	d.reportHealth()

	d.ipSecPolTable.Apply()

	if d.forceRouteRefresh {
		// Refresh timer popped.
		for _, r := range d.routeTableSyncers() {
			// Queue a resync on the next Apply().
			r.QueueResync()
		}
		for _, r := range d.routeRules() {
			// Queue a resync on the next Apply().
			r.QueueResync()
		}
		d.forceRouteRefresh = false
	}

	if d.forceIPSetsRefresh {
		// Refresh timer popped.
		for _, r := range d.ipSets {
			// Queue a resync on the next Apply().
			r.QueueResync()
		}
		d.forceIPSetsRefresh = false
	}

	// Next, create/update IP sets.  We defer deletions of IP sets until after we update
	// iptables.
	var ipSetsWG sync.WaitGroup
	for _, ipSets := range d.ipSets {
		ipSetsWG.Add(1)
		go func(ipSets ipsetsDataplane) {
			ipSets.ApplyUpdates()
			d.reportHealth()
			ipSetsWG.Done()
		}(ipSets)
	}

	// Update the routing table in parallel with the other updates.  We'll wait for it to finish
	// before we return.
	var routesWG sync.WaitGroup
	for _, r := range d.routeTableSyncers() {
		routesWG.Add(1)
		go func(r routeTableSyncer) {
			err := r.Apply()
			if err != nil {
				log.Warn("Failed to synchronize routing table, will retry...")
				d.dataplaneNeedsSync = true
			}
			d.reportHealth()
			routesWG.Done()
		}(r)
	}

	// Update the routing rules in parallel with the other updates.  We'll wait for it to finish
	// before we return.
	var rulesWG sync.WaitGroup
	for _, r := range d.routeRules() {
		rulesWG.Add(1)
		go func(r routeRules) {
			err := r.Apply()
			if err != nil {
				log.Warn("Failed to synchronize routing rules, will retry...")
				d.dataplaneNeedsSync = true
			}
			d.reportHealth()
			rulesWG.Done()
		}(r)
	}

	// Wait for the IP sets update to finish.  We can't update iptables until it has.
	ipSetsWG.Wait()

	// Update iptables, this should sever any references to now-unused IP sets.
	var reschedDelayMutex sync.Mutex
	var reschedDelay time.Duration
	var iptablesWG sync.WaitGroup
	for _, t := range d.allIptablesTables {
		iptablesWG.Add(1)
		go func(t *iptables.Table) {
			tableReschedAfter := t.Apply()

			reschedDelayMutex.Lock()
			defer reschedDelayMutex.Unlock()
			if tableReschedAfter != 0 && (reschedDelay == 0 || tableReschedAfter < reschedDelay) {
				reschedDelay = tableReschedAfter
			}
			d.reportHealth()
			iptablesWG.Done()
		}(t)
	}
	iptablesWG.Wait()

	// Now clean up any left-over IP sets.
	for _, ipSets := range d.ipSets {
		ipSetsWG.Add(1)
		go func(s ipsetsDataplane) {
			s.ApplyDeletions()
			d.reportHealth()
			ipSetsWG.Done()
		}(ipSets)
	}
	ipSetsWG.Wait()

	// Wait for the route updates to finish.
	routesWG.Wait()

	// Wait for the rule updates to finish.
	rulesWG.Wait()

	// And publish and status updates.
	d.endpointStatusCombiner.Apply()

	// Set up any needed rescheduling kick.
	if d.reschedC != nil {
		// We have an active rescheduling timer, stop it so we can restart it with a
		// different timeout below if it is still needed.
		// This snippet comes from the docs for Timer.Stop().
		if !d.reschedTimer.Stop() {
			// Timer had already popped, drain its channel.
			<-d.reschedC
		}
		// Nil out our copy of the channel to record that the timer is inactive.
		d.reschedC = nil
	}
	if reschedDelay != 0 {
		// We need to reschedule.
		log.WithField("delay", reschedDelay).Debug("Asked to reschedule.")
		if d.reschedTimer == nil {
			// First time, create the timer.
			d.reschedTimer = time.NewTimer(reschedDelay)
		} else {
			// Have an existing timer, reset it.
			d.reschedTimer.Reset(reschedDelay)
		}
		d.reschedC = d.reschedTimer.C
	}
}

func (d *InternalDataplane) applyXDPActions() error {
	var err error = nil
	for i := 0; i < 10; i++ {
		err = d.xdpState.ResyncIfNeeded(d.ipsetsSourceV4)
		if err != nil {
			return err
		}
		if err = d.xdpState.ApplyBPFActions(d.ipsetsSourceV4); err == nil {
			return nil
		} else {
			log.WithError(err).Info("Applying XDP BPF actions did not succeed, will retry with resync...")
		}
	}
	return err
}

func (d *InternalDataplane) loopReportingStatus() {
	log.Info("Started internal status report thread")
	if d.config.StatusReportingInterval <= 0 {
		log.Info("Process status reports disabled")
		return
	}
	// Wait before first report so that we don't check in if we're in a tight cyclic restart.
	time.Sleep(10 * time.Second)
	for {
		uptimeSecs := time.Since(processStartTime).Seconds()
		d.fromDataplane <- &proto.ProcessStatusUpdate{
			IsoTimestamp: time.Now().UTC().Format(time.RFC3339),
			Uptime:       uptimeSecs,
		}
		time.Sleep(d.config.StatusReportingInterval)
	}
}

// iptablesTable is a shim interface for iptables.Table.
type iptablesTable interface {
	UpdateChain(chain *iptables.Chain)
	UpdateChains([]*iptables.Chain)
	RemoveChains([]*iptables.Chain)
	RemoveChainByName(name string)
}

func (d *InternalDataplane) reportHealth() {
	if d.config.HealthAggregator != nil {
		d.config.HealthAggregator.Report(
			healthName,
			&health.HealthReport{Live: true, Ready: d.doneFirstApply},
		)
	}
}

type dummyLock struct{}

func (d dummyLock) Lock() {

}

func (d dummyLock) Unlock() {

}
