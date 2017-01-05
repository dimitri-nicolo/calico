// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.
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
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/collector"
	"github.com/projectcalico/felix/go/felix/collector/stats"
	"github.com/projectcalico/felix/go/felix/ifacemonitor"
	"github.com/projectcalico/felix/go/felix/ipfix"
	"github.com/projectcalico/felix/go/felix/ipsets"
	"github.com/projectcalico/felix/go/felix/iptables"
	"github.com/projectcalico/felix/go/felix/jitter"
	"github.com/projectcalico/felix/go/felix/lookup"
	"github.com/projectcalico/felix/go/felix/proto"
	"github.com/projectcalico/felix/go/felix/routetable"
	"github.com/projectcalico/felix/go/felix/rules"
	"github.com/projectcalico/felix/go/felix/set"
	"time"
)

type Config struct {
	DisableIPv6             bool
	RuleRendererOverride    rules.RuleRenderer
	IpfixPort               int
	IpfixAddr               net.IP
	IPIPMTU                 int
	IptablesRefreshInterval time.Duration
	RulesConfig             rules.Config
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
	iptablesNATTables    []*iptables.Table
	iptablesRawTables    []*iptables.Table
	iptablesFilterTables []*iptables.Table
	ipSetRegistries      []*ipsets.Registry

	ifaceMonitor     *ifacemonitor.InterfaceMonitor
	ifaceUpdates     chan *ifaceUpdate
	ifaceAddrUpdates chan *ifaceAddrsUpdate

	endpointStatusCombiner *endpointStatusCombiner

	allManagers []Manager

	ruleRenderer rules.RuleRenderer

	lookupManager *lookup.LookupManager

	interfacePrefixes []string

	routeTables []*routetable.RouteTable

	dataplaneNeedsSync bool
	refreshIptables    bool
	cleanupPending     bool

	config Config
}

func NewIntDataplaneDriver(config Config) *InternalDataplane {
	ruleRenderer := config.RuleRendererOverride
	if ruleRenderer == nil {
		ruleRenderer = rules.NewRenderer(config.RulesConfig)
	}
	dp := &InternalDataplane{
		toDataplane:       make(chan interface{}, 100),
		fromDataplane:     make(chan interface{}, 100),
		ruleRenderer:      ruleRenderer,
		interfacePrefixes: config.RulesConfig.WorkloadIfacePrefixes,
		cleanupPending:    true,
		ifaceMonitor:      ifacemonitor.New(),
		ifaceUpdates:      make(chan *ifaceUpdate, 100),
		ifaceAddrUpdates:  make(chan *ifaceAddrsUpdate, 100),
		config:            config,
	}

	dp.ifaceMonitor.Callback = dp.onIfaceStateChange
	dp.ifaceMonitor.AddrCallback = dp.onIfaceAddrsChange

	natTableV4 := iptables.NewTable(
		"nat",
		4,
		rules.AllHistoricChainNamePrefixes,
		rules.RuleHashPrefix,
		rules.HistoricInsertedNATRuleRegex,
	)
	rawTableV4 := iptables.NewTable("raw", 4, rules.AllHistoricChainNamePrefixes, rules.RuleHashPrefix, "")
	filterTableV4 := iptables.NewTable("filter", 4, rules.AllHistoricChainNamePrefixes, rules.RuleHashPrefix, "")
	ipSetsConfigV4 := config.RulesConfig.IPSetConfigV4
	ipSetRegV4 := ipsets.NewRegistry(ipSetsConfigV4)
	dp.iptablesNATTables = append(dp.iptablesNATTables, natTableV4)
	dp.iptablesRawTables = append(dp.iptablesRawTables, rawTableV4)
	dp.iptablesFilterTables = append(dp.iptablesFilterTables, filterTableV4)
	dp.ipSetRegistries = append(dp.ipSetRegistries, ipSetRegV4)

	routeTableV4 := routetable.New(config.RulesConfig.WorkloadIfacePrefixes, 4)
	dp.routeTables = append(dp.routeTables, routeTableV4)

	dp.endpointStatusCombiner = newEndpointStatusCombiner(dp.fromDataplane, !config.DisableIPv6)

	dp.RegisterManager(newIPSetsManager(ipSetRegV4))
	dp.RegisterManager(newPolicyManager(filterTableV4, ruleRenderer, 4))
	dp.RegisterManager(newEndpointManager(
		filterTableV4,
		ruleRenderer,
		routeTableV4,
		4,
		config.RulesConfig.WorkloadIfacePrefixes,
		dp.endpointStatusCombiner.OnWorkloadEndpointStatusUpdate))
	dp.lookupManager = lookup.NewLookupManager()
	dp.RegisterManager(dp.lookupManager)
	dp.RegisterManager(newMasqManager(ipSetRegV4, natTableV4, ruleRenderer, 1000000, 4))
	if config.RulesConfig.IPIPEnabled {
		// Add a manger to keep the all-hosts IP set up to date.
		dp.RegisterManager(newIPIPManager(ipSetRegV4, 1000000)) // IPv4-only
	}
	if !config.DisableIPv6 {
		natTableV6 := iptables.NewTable(
			"nat",
			6,
			rules.AllHistoricChainNamePrefixes,
			rules.RuleHashPrefix,
			rules.HistoricInsertedNATRuleRegex,
		)
		rawTableV6 := iptables.NewTable("raw", 6, rules.AllHistoricChainNamePrefixes, rules.RuleHashPrefix, "")
		filterTableV6 := iptables.NewTable("filter", 6, rules.AllHistoricChainNamePrefixes, rules.RuleHashPrefix, "")

		ipSetsConfigV6 := config.RulesConfig.IPSetConfigV6
		ipSetRegV6 := ipsets.NewRegistry(ipSetsConfigV6)
		dp.ipSetRegistries = append(dp.ipSetRegistries, ipSetRegV6)
		dp.iptablesNATTables = append(dp.iptablesNATTables, natTableV6)
		dp.iptablesRawTables = append(dp.iptablesRawTables, rawTableV6)
		dp.iptablesFilterTables = append(dp.iptablesFilterTables, filterTableV6)

		routeTableV6 := routetable.New(config.RulesConfig.WorkloadIfacePrefixes, 6)
		dp.routeTables = append(dp.routeTables, routeTableV6)

		dp.RegisterManager(newIPSetsManager(ipSetRegV6))
		dp.RegisterManager(newPolicyManager(filterTableV6, ruleRenderer, 6))
		dp.RegisterManager(newEndpointManager(
			filterTableV6,
			ruleRenderer,
			routeTableV6,
			6,
			config.RulesConfig.WorkloadIfacePrefixes,
			dp.endpointStatusCombiner.OnWorkloadEndpointStatusUpdate))
		dp.RegisterManager(newMasqManager(ipSetRegV6, natTableV6, ruleRenderer, 1000000, 6))
	}

	for _, t := range dp.iptablesNATTables {
		dp.allIptablesTables = append(dp.allIptablesTables, t)
	}
	for _, t := range dp.iptablesFilterTables {
		dp.allIptablesTables = append(dp.allIptablesTables, t)
	}
	for _, t := range dp.iptablesRawTables {
		dp.allIptablesTables = append(dp.allIptablesTables, t)
	}

	return dp
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

func (d *InternalDataplane) RegisterManager(mgr Manager) {
	d.allManagers = append(d.allManagers, mgr)
}

func (d *InternalDataplane) Start() {
	go d.loopUpdatingDataplane()
	go d.loopReportingStatus()
	go d.ifaceMonitor.MonitorInterfaces()

	// TODO (Matt): This isn't really in keeping with the surrounding code.
	ctSink := make(chan stats.StatUpdate)
	conntrackDataSource := collector.NewConntrackDataSource(d.lookupManager, ctSink)
	conntrackDataSource.Start()

	nfIngressSink := make(chan stats.StatUpdate)
	nflogIngressDataSource := collector.NewNflogDataSource(d.lookupManager, nfIngressSink, 1, stats.DirIn)
	nflogIngressDataSource.Start()

	nfEgressSink := make(chan stats.StatUpdate)
	nflogEgressDataSource := collector.NewNflogDataSource(d.lookupManager, nfEgressSink, 2, stats.DirOut)
	nflogEgressDataSource.Start()

	ipfixExportSink := make(chan *ipfix.ExportRecord)
	ipfixExporter := ipfix.NewIPFIXExporter(d.config.IpfixAddr, d.config.IpfixPort, "udp", ipfixExportSink)
	ipfixExporter.Start()

	printSink := make(chan *stats.Data)
	datasources := []<-chan stats.StatUpdate{ctSink, nfIngressSink, nfEgressSink}
	datasinks := []chan<- *stats.Data{printSink}
	statsCollector := collector.NewCollector(datasources, datasinks, ipfixExportSink)
	statsCollector.Start()

	// TODO (Matt): fix signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR2)

	go func() {
		for {
			<-sigChan
			statsCollector.PrintStats()
		}
	}()

	go func() {
		for data := range printSink {
			log.Info("MD4 test output data: ", data)
		}
	}()
}

// onIfaceStateChange is our interface monitor callback.  It gets called from the monitor's thread.
func (d *InternalDataplane) onIfaceStateChange(ifaceName string, state ifacemonitor.State) {
	log.WithFields(log.Fields{
		"ifaceName": ifaceName,
		"state":     state,
	}).Info("Linux interface state changed.")
	d.ifaceUpdates <- &ifaceUpdate{
		Name:  ifaceName,
		State: state,
	}
}

type ifaceUpdate struct {
	Name  string
	State ifacemonitor.State
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

func (d *InternalDataplane) loopUpdatingDataplane() {
	log.Info("Started internal iptables dataplane driver")

	// TODO Check global RPF value is sane (can't be "loose").

	// Endure that the default value of rp_filter is set to "strict" for newly-created
	// interfaces.  This is required to prevent a race between starting an interface and
	// Felix being able to configure it.
	writeProcSys("/proc/sys/net/ipv4/conf/default/rp_filter", "1")

	// Enable conntrack packet and byte accounting.
	writeProcSys("/proc/sys/net/netfilter/nf_conntrack_acct", "1")

	for _, t := range d.iptablesFilterTables {
		filterChains := d.ruleRenderer.StaticFilterTableChains(t.IPVersion)
		t.UpdateChains(filterChains)
		t.SetRuleInsertions("FORWARD", []iptables.Rule{{
			Action: iptables.JumpAction{rules.ChainFilterForward},
		}})
		t.SetRuleInsertions("INPUT", []iptables.Rule{{
			Action: iptables.JumpAction{rules.ChainFilterInput},
		}})
		t.SetRuleInsertions("OUTPUT", []iptables.Rule{{
			Action: iptables.JumpAction{rules.ChainFilterOutput},
		}})
	}

	if d.config.RulesConfig.IPIPEnabled {
		log.Info("IPIP enabled, starting thread to keep tunnel configuration in sync.")
		go func() {
			log.Info("IPIP thread started.")
			for {
				err := configureIPIPDevice(d.config.IPIPMTU,
					d.config.RulesConfig.IPIPTunnelAddress)
				if err != nil {
					log.WithError(err).Warn("Failed configure IPIP tunnel device, retrying...")
					time.Sleep(1 * time.Second)
					continue
				}
				time.Sleep(10 * time.Second)
			}
		}()
	}

	for _, t := range d.iptablesNATTables {
		t.UpdateChains(d.ruleRenderer.StaticNATTableChains(t.IPVersion))
		t.SetRuleInsertions("PREROUTING", []iptables.Rule{{
			Action: iptables.JumpAction{rules.ChainNATPrerouting},
		}})
		t.SetRuleInsertions("POSTROUTING", []iptables.Rule{{
			Action: iptables.JumpAction{rules.ChainNATPostrouting},
		}})
	}

	// Retry any failed operations every 10s.
	retryTicker := time.NewTicker(10 * time.Second)
	var refreshC <-chan time.Time
	if d.config.IptablesRefreshInterval > 0 {
		refreshTicker := jitter.NewTicker(
			d.config.IptablesRefreshInterval,
			d.config.IptablesRefreshInterval/10,
		)
		refreshC = refreshTicker.C
	}

	datastoreInSync := false
	for {
		select {
		case msg := <-d.toDataplane:
			log.WithField("msg", msg).Info("Received update from calculation graph")
			for _, mgr := range d.allManagers {
				mgr.OnUpdate(msg)
			}
			switch msg.(type) {
			case *proto.InSync:
				log.Info("Datastore in sync, flushing the dataplane for the first time...")
				datastoreInSync = true
			}
			d.dataplaneNeedsSync = true
		case ifaceUpdate := <-d.ifaceUpdates:
			log.WithField("msg", ifaceUpdate).Info("Received interface update")
			for _, mgr := range d.allManagers {
				mgr.OnUpdate(ifaceUpdate)
			}
			for _, routeTable := range d.routeTables {
				routeTable.OnIfaceStateChanged(ifaceUpdate.Name, ifaceUpdate.State)
			}
			d.dataplaneNeedsSync = true
		case ifaceAddrsUpdate := <-d.ifaceAddrUpdates:
			log.WithField("msg", ifaceAddrsUpdate).Info("Received interface addresses update")
			for _, mgr := range d.allManagers {
				mgr.OnUpdate(ifaceAddrsUpdate)
			}
			d.dataplaneNeedsSync = true
		case <-refreshC:
			log.Debug("Refreshing iptables dataplane state")
			d.refreshIptables = true
			d.dataplaneNeedsSync = true
		case <-retryTicker.C:
		}

		if datastoreInSync && d.dataplaneNeedsSync {
			d.apply()
		}
	}
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
			d.dataplaneNeedsSync = true
		}
	}

	// Next, create/update IP sets.  We defer deletions of IP sets until after we update
	// iptables.
	for _, w := range d.ipSetRegistries {
		w.ApplyUpdates()
	}

	if d.refreshIptables {
		for _, t := range d.allIptablesTables {
			t.InvalidateDataplaneCache()
		}
		d.refreshIptables = false
	}
	// Update iptables, this should sever any references to now-unused IP sets.
	for _, t := range d.allIptablesTables {
		t.Apply()
	}

	// Update the routing table.
	for _, r := range d.routeTables {
		err := r.Apply()
		if err != nil {
			log.Warn("Failed to synchronize routing table, will retry...")
			d.dataplaneNeedsSync = true
		}
	}

	// Now clean up any left-over IP sets.
	for _, w := range d.ipSetRegistries {
		w.ApplyDeletions()
	}

	// And publish and status updates.
	d.endpointStatusCombiner.Apply()

	if d.cleanupPending {
		for _, w := range d.ipSetRegistries {
			w.AttemptCleanup()
		}
		d.cleanupPending = false
	}
}

func (d *InternalDataplane) loopReportingStatus() {
	log.Info("Started internal status report thread")
	start := time.Now()
	for {
		time.Sleep(10 * time.Second)
		now := time.Now()
		uptimeNanos := float64(now.Sub(start))
		uptimeSecs := uptimeNanos / 1000000000
		d.fromDataplane <- &proto.ProcessStatusUpdate{
			IsoTimestamp: now.UTC().Format(time.RFC3339),
			Uptime:       uptimeSecs,
		}
	}
}
