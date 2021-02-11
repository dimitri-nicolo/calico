// +build !windows

// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
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
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sys/unix"

	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/projectcalico/libcalico-go/lib/backend/model"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/polprog"
	"github.com/projectcalico/felix/bpf/tc"
	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/idalloc"
	"github.com/projectcalico/felix/ifacemonitor"
	"github.com/projectcalico/felix/iptables"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/ratelimited"
	"github.com/projectcalico/felix/rules"
)

const jumpMapCleanupInterval = 10 * time.Second

var (
	bpfEndpointsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_bpf_dataplane_endpoints",
		Help: "Number of BPF endpoints managed in the dataplane.",
	})
	bpfDirtyEndpointsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_bpf_dirty_dataplane_endpoints",
		Help: "Number of BPF endpoints managed in the dataplane that are left dirty after a failure.",
	})
	bpfHappyEndpointsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_bpf_happy_dataplane_endpoints",
		Help: "Number of BPF endpoints that are successfully programmed.",
	})
)

func init() {
	prometheus.MustRegister(bpfEndpointsGauge)
	prometheus.MustRegister(bpfDirtyEndpointsGauge)
	prometheus.MustRegister(bpfHappyEndpointsGauge)
}

type bpfInterface struct {
	// info contains the information about the interface sent to us from external sources. For example,
	// the ID of the controlling workload interface and our current expectation of its "oper state".
	// When the info changes, we mark the interface dirty and refresh its dataplane state.
	info bpfInterfaceInfo
	// dpState contains the dataplane state that we've derived locally.  It caches the result of updating
	// the interface (so changes to dpState don't cause the interface to be marked dirty).
	dpState bpfInterfaceState
}

type bpfInterfaceInfo struct {
	ifaceIsUp  bool
	endpointID *proto.WorkloadEndpointID
}

type bpfInterfaceState struct {
	jumpMapFDs [2]bpf.MapFD
}

type bpfEndpointManager struct {
	// Main store of information about interfaces; indexed on interface name.
	ifacesLock  sync.Mutex
	nameToIface map[string]bpfInterface

	allWEPs        map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint
	happyWEPs      map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint
	happyWEPsDirty bool
	policies       map[proto.PolicyID]*proto.Policy
	profiles       map[proto.ProfileID]*proto.Profile

	// Indexes
	policiesToWorkloads map[proto.PolicyID]set.Set  /*proto.WorkloadEndpointID*/
	profilesToWorkloads map[proto.ProfileID]set.Set /*proto.WorkloadEndpointID*/

	dirtyIfaceNames set.Set

	bpfLogLevel        string
	hostname           string
	hostIP             net.IP
	fibLookupEnabled   bool
	dataIfaceRegex     *regexp.Regexp
	workloadIfaceRegex *regexp.Regexp
	ipSetIDAlloc       *idalloc.IDAllocator
	epToHostAction     string
	vxlanMTU           int
	vxlanPort          uint16
	dsrEnabled         bool
	enableTcpStats     bool

	ipSetMap bpf.Map
	stateMap bpf.Map

	ruleRenderer        bpfAllowChainRenderer
	iptablesFilterTable iptablesTable

	startupOnce      sync.Once
	mapCleanupRunner *ratelimited.Runner

	// onStillAlive is called from loops to reset the watchdog.
	onStillAlive func()

	// HEP processing.
	hostIfaceToEpMap     map[string]proto.HostEndpoint
	wildcardHostEndpoint proto.HostEndpoint
	wildcardExists       bool

	// CaliEnt features below

	lookupsCache *calc.LookupsCache
	actionOnDrop string
}

type bpfAllowChainRenderer interface {
	WorkloadInterfaceAllowChains(endpoints map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint) []*iptables.Chain
}

func newBPFEndpointManager(
	bpfLogLevel string,
	hostname string,
	fibLookupEnabled bool,
	epToHostAction string,
	dataIfaceRegex *regexp.Regexp,
	workloadIfaceRegex *regexp.Regexp,
	ipSetIDAlloc *idalloc.IDAllocator,
	vxlanMTU int,
	vxlanPort uint16,
	dsrEnabled bool,
	ipSetMap bpf.Map,
	stateMap bpf.Map,
	iptablesRuleRenderer bpfAllowChainRenderer,
	iptablesFilterTable iptablesTable,
	livenessCallback func(),

	// CaliEnt args below

	lookupsCache *calc.LookupsCache,
	actionOnDrop string,
	enableTcpStats bool,
) *bpfEndpointManager {
	if livenessCallback == nil {
		livenessCallback = func() {}
	}

	switch actionOnDrop {
	case "ACCEPT", "LOGandACCEPT":
		actionOnDrop = "allow"
	case "DROP", "LOGandDROP":
		actionOnDrop = "deny"
	}
	return &bpfEndpointManager{
		allWEPs:             map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint{},
		happyWEPs:           map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint{},
		happyWEPsDirty:      true,
		policies:            map[proto.PolicyID]*proto.Policy{},
		profiles:            map[proto.ProfileID]*proto.Profile{},
		nameToIface:         map[string]bpfInterface{},
		policiesToWorkloads: map[proto.PolicyID]set.Set{},
		profilesToWorkloads: map[proto.ProfileID]set.Set{},
		dirtyIfaceNames:     set.New(),
		bpfLogLevel:         bpfLogLevel,
		hostname:            hostname,
		fibLookupEnabled:    fibLookupEnabled,
		dataIfaceRegex:      dataIfaceRegex,
		workloadIfaceRegex:  workloadIfaceRegex,
		ipSetIDAlloc:        ipSetIDAlloc,
		epToHostAction:      epToHostAction,
		vxlanMTU:            vxlanMTU,
		vxlanPort:           vxlanPort,
		dsrEnabled:          dsrEnabled,
		ipSetMap:            ipSetMap,
		stateMap:            stateMap,
		ruleRenderer:        iptablesRuleRenderer,
		iptablesFilterTable: iptablesFilterTable,
		mapCleanupRunner: ratelimited.NewRunner(jumpMapCleanupInterval, func(ctx context.Context) {
			log.Debug("Jump map cleanup triggered.")
			tc.CleanUpJumpMaps()
		}),
		onStillAlive:     livenessCallback,
		lookupsCache:     lookupsCache,
		hostIfaceToEpMap: map[string]proto.HostEndpoint{},
		actionOnDrop:     actionOnDrop,
		enableTcpStats:   enableTcpStats,
	}
}

// withIface handles the bookkeeping for working with a particular bpfInterface value.  It
// * creates the value if needed
// * calls the giving callback with the value so it can be edited
// * if the bpfInterface's info field changes, it marks it as dirty
// * if the bpfInterface is now empty (no info or state), it cleans it up.
func (m *bpfEndpointManager) withIface(ifaceName string, fn func(iface *bpfInterface) (forceDirty bool)) {
	iface := m.nameToIface[ifaceName]
	ifaceCopy := iface
	dirty := fn(&iface)
	logCtx := log.WithField("name", ifaceName)

	var zeroIface bpfInterface
	if iface == zeroIface {
		logCtx.Debug("Interface info is now empty.")
		delete(m.nameToIface, ifaceName)
	} else {
		// Always store the result (rather than checking the dirty flag) because dirty only covers the info..
		m.nameToIface[ifaceName] = iface
	}

	dirty = dirty || iface.info != ifaceCopy.info

	if !dirty {
		return
	}

	logCtx.Debug("Marking iface dirty.")
	m.dirtyIfaceNames.Add(ifaceName)
}

func (m *bpfEndpointManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	// Updates from the dataplane:

	// Interface updates.
	case *ifaceUpdate:
		m.onInterfaceUpdate(msg)

	// Updates from the datamodel:

	// Workloads.
	case *proto.WorkloadEndpointUpdate:
		m.onWorkloadEndpointUpdate(msg)
	case *proto.WorkloadEndpointRemove:
		m.onWorkloadEnpdointRemove(msg)
	// Policies.
	case *proto.ActivePolicyUpdate:
		m.onPolicyUpdate(msg)
	case *proto.ActivePolicyRemove:
		m.onPolicyRemove(msg)
	// Profiles.
	case *proto.ActiveProfileUpdate:
		m.onProfileUpdate(msg)
	case *proto.ActiveProfileRemove:
		m.onProfileRemove(msg)

	case *proto.HostMetadataUpdate:
		if msg.Hostname == m.hostname {
			log.WithField("HostMetadataUpdate", msg).Info("Host IP changed")
			ip := net.ParseIP(msg.Ipv4Addr)
			if ip != nil {
				m.hostIP = ip
				// Should be safe without the lock since there shouldn't be any active background threads
				// but taking it now makes us robust to refactoring.
				m.ifacesLock.Lock()
				for ifaceName := range m.nameToIface {
					m.dirtyIfaceNames.Add(ifaceName)
				}
				m.ifacesLock.Unlock()
			} else {
				log.WithField("HostMetadataUpdate", msg).Warn("Cannot parse IP, no change applied")
			}
		}
	}
}

func (m *bpfEndpointManager) onInterfaceUpdate(update *ifaceUpdate) {
	log.Debugf("Interface update for %v, state %v", update.Name, update.State)

	// Should be safe without the lock since there shouldn't be any active background threads
	// but taking it now makes us robust to refactoring.
	m.ifacesLock.Lock()
	defer m.ifacesLock.Unlock()

	if !m.isDataIface(update.Name) && !m.isWorkloadIface(update.Name) {
		log.WithField("update", update).Debug("Ignoring interface that's neither data nor workload.")
		return
	}

	m.withIface(update.Name, func(iface *bpfInterface) bool {
		iface.info.ifaceIsUp = update.State == ifacemonitor.StateUp
		// Note, only need to handle the mapping and unmapping of the host-* endpoint here.
		// For specific host endpoints OnHEPUpdate doesn't depend on iface state, and has
		// already stored and mapped as needed.
		if iface.info.ifaceIsUp {
			if _, hostEpConfigured := m.hostIfaceToEpMap[update.Name]; m.wildcardExists && !hostEpConfigured {
				log.Debugf("Map host-* endpoint for %v", update.Name)
				m.addHEPToIndexes(update.Name, &m.wildcardHostEndpoint)
				m.hostIfaceToEpMap[update.Name] = m.wildcardHostEndpoint
			}
		} else {
			if m.wildcardExists && reflect.DeepEqual(m.hostIfaceToEpMap[update.Name], m.wildcardHostEndpoint) {
				log.Debugf("Unmap host-* endpoint for %v", update.Name)
				m.removeHEPFromIndexes(update.Name, &m.wildcardHostEndpoint)
				delete(m.hostIfaceToEpMap, update.Name)
			}
		}
		return true // Force interface to be marked dirty in case we missed a transition during a resync.
	})
}

// onWorkloadEndpointUpdate adds/updates the workload in the cache along with the index from active policy to
// workloads using that policy.
func (m *bpfEndpointManager) onWorkloadEndpointUpdate(msg *proto.WorkloadEndpointUpdate) {
	log.WithField("wep", msg.Endpoint).Debug("Workload endpoint update")
	wlID := *msg.Id
	oldWEP := m.allWEPs[wlID]
	m.removeWEPFromIndexes(wlID, oldWEP)

	wl := msg.Endpoint
	m.allWEPs[wlID] = wl
	m.addWEPToIndexes(wlID, wl)
	m.withIface(wl.Name, func(iface *bpfInterface) bool {
		iface.info.endpointID = &wlID
		return true // Force interface to be marked dirty in case policies changed.
	})
}

// onWorkloadEndpointRemove removes the workload from the cache and the index, which maps from policy to workload.
func (m *bpfEndpointManager) onWorkloadEnpdointRemove(msg *proto.WorkloadEndpointRemove) {
	wlID := *msg.Id
	log.WithField("id", wlID).Debug("Workload endpoint removed")
	oldWEP := m.allWEPs[wlID]
	m.removeWEPFromIndexes(wlID, oldWEP)
	delete(m.allWEPs, wlID)

	if m.happyWEPs[wlID] != nil {
		delete(m.happyWEPs, wlID)
		m.happyWEPsDirty = true
	}

	m.withIface(oldWEP.Name, func(iface *bpfInterface) bool {
		iface.info.endpointID = nil
		return false
	})
}

// onPolicyUpdate stores the policy in the cache and marks any endpoints using it dirty.
func (m *bpfEndpointManager) onPolicyUpdate(msg *proto.ActivePolicyUpdate) {
	polID := *msg.Id
	log.WithField("id", polID).Debug("Policy update")
	m.policies[polID] = msg.Policy
	m.markEndpointsDirty(m.policiesToWorkloads[polID], "policy")
}

// onPolicyRemove removes the policy from the cache and marks any endpoints using it dirty.
// The latter should be a no-op due to the ordering guarantees of the calc graph.
func (m *bpfEndpointManager) onPolicyRemove(msg *proto.ActivePolicyRemove) {
	polID := *msg.Id
	log.WithField("id", polID).Debug("Policy removed")
	m.markEndpointsDirty(m.policiesToWorkloads[polID], "policy")
	delete(m.policies, polID)
	delete(m.policiesToWorkloads, polID)
}

// onProfileUpdate stores the profile in the cache and marks any endpoints that use it as dirty.
func (m *bpfEndpointManager) onProfileUpdate(msg *proto.ActiveProfileUpdate) {
	profID := *msg.Id
	log.WithField("id", profID).Debug("Profile update")
	m.profiles[profID] = msg.Profile
	m.markEndpointsDirty(m.profilesToWorkloads[profID], "profile")
}

// onProfileRemove removes the profile from the cache and marks any endpoints that were using it as dirty.
// The latter should be a no-op due to the ordering guarantees of the calc graph.
func (m *bpfEndpointManager) onProfileRemove(msg *proto.ActiveProfileRemove) {
	profID := *msg.Id
	log.WithField("id", profID).Debug("Profile removed")
	m.markEndpointsDirty(m.profilesToWorkloads[profID], "profile")
	delete(m.profiles, profID)
	delete(m.profilesToWorkloads, profID)
}

func (m *bpfEndpointManager) markEndpointsDirty(ids set.Set, kind string) {
	if ids == nil {
		// Hear about the policy/profile before the endpoint.
		return
	}
	ids.Iter(func(item interface{}) error {
		switch id := item.(type) {
		case proto.WorkloadEndpointID:
			m.markExistingWEPDirty(id, kind)
		case string:
			if id == allInterfaces {
				for ifaceName := range m.nameToIface {
					if m.isWorkloadIface(ifaceName) {
						log.Debugf("Mark WEP iface dirty, for host-* endpoint %v change", kind)
						m.dirtyIfaceNames.Add(ifaceName)
					}
				}
			} else {
				log.Debugf("Mark host iface dirty, for host %v change", kind)
				m.dirtyIfaceNames.Add(id)
			}
		}
		return nil
	})
}

func (m *bpfEndpointManager) markExistingWEPDirty(wlID proto.WorkloadEndpointID, mapping string) {
	wep := m.allWEPs[wlID]
	if wep == nil {
		log.WithField("wlID", wlID).Panicf(
			"BUG: %s mapping points to unknown workload.", mapping)
	} else {
		m.dirtyIfaceNames.Add(wep.Name)
	}
}

func (m *bpfEndpointManager) CompleteDeferredWork() error {
	// Do one-off initialisation.
	m.ensureStarted()

	m.applyProgramsToDirtyDataInterfaces()
	m.updateWEPsInDataplane()

	bpfEndpointsGauge.Set(float64(len(m.nameToIface)))
	bpfDirtyEndpointsGauge.Set(float64(m.dirtyIfaceNames.Len()))

	if m.happyWEPsDirty {
		chains := m.ruleRenderer.WorkloadInterfaceAllowChains(m.happyWEPs)
		m.iptablesFilterTable.UpdateChains(chains)
		m.happyWEPsDirty = false
	}
	bpfHappyEndpointsGauge.Set(float64(len(m.happyWEPs)))

	return nil
}

func (m *bpfEndpointManager) setAcceptLocal(iface string, val bool) error {
	numval := "0"
	if val {
		numval = "1"
	}

	path := fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/accept_local", iface)
	err := writeProcSys(path, numval)
	if err != nil {
		log.WithField("err", err).Errorf("Failed to  set %s to %s", path, numval)
		return err
	}

	log.Infof("%s set to %s", path, numval)
	return nil
}

func (m *bpfEndpointManager) ensureStarted() {
	m.startupOnce.Do(func() {
		log.Info("Starting map cleanup runner.")
		m.mapCleanupRunner.Start(context.Background())
	})
}

func (m *bpfEndpointManager) applyProgramsToDirtyDataInterfaces() {
	var mutex sync.Mutex
	errs := map[string]error{}
	var wg sync.WaitGroup
	m.dirtyIfaceNames.Iter(func(item interface{}) error {
		iface := item.(string)
		if !m.isDataIface(iface) {
			log.WithField("iface", iface).Debug(
				"Ignoring interface that doesn't match the host data interface regex")
			return nil
		}
		if !m.ifaceIsUp(iface) {
			log.WithField("iface", iface).Debug("Ignoring interface that is down")
			return set.RemoveItem
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			// Attach the qdisc first; it is shared between the directions.
			err := tc.EnsureQdisc(iface)
			if err != nil {
				mutex.Lock()
				errs[iface] = err
				mutex.Unlock()
				return
			}

			var hepPtr *proto.HostEndpoint
			if hep, hepExists := m.hostIfaceToEpMap[iface]; hepExists {
				hepPtr = &hep
			}

			var ingressWG sync.WaitGroup
			var ingressErr error
			ingressWG.Add(1)
			go func() {
				defer ingressWG.Done()
				ingressErr = m.attachDataIfaceProgram(iface, hepPtr, PolDirnIngress)
			}()
			err = m.attachDataIfaceProgram(iface, hepPtr, PolDirnEgress)
			ingressWG.Wait()
			if err == nil {
				err = ingressErr
			}
			if err == nil {
				// This is required to allow NodePort forwarding with
				// encapsulation with the host's IP as the source address
				err = m.setAcceptLocal(iface, true)
			}
			mutex.Lock()
			errs[iface] = err
			mutex.Unlock()
		}()
		return nil
	})
	wg.Wait()
	m.dirtyIfaceNames.Iter(func(item interface{}) error {
		iface := item.(string)
		if !m.isDataIface(iface) {
			log.WithField("iface", iface).Debug(
				"Ignoring interface that doesn't match the host data interface regex")
			return nil
		}
		err := errs[iface]
		if err == nil {
			log.WithField("id", iface).Info("Applied program to host interface")
			return set.RemoveItem
		}
		if errors.Is(err, tc.ErrDeviceNotFound) {
			log.WithField("iface", iface).Debug(
				"Tried to apply BPF program to interface but the interface wasn't present.  " +
					"Will retry if it shows up.")
			return set.RemoveItem
		}
		log.WithError(err).Warn("Failed to apply policy to interface")
		return nil
	})
}

func (m *bpfEndpointManager) updateWEPsInDataplane() {
	var mutex sync.Mutex
	errs := map[string]error{}
	var wg sync.WaitGroup

	// Limit the number of parallel workers.  Without this, all the workers vie for CPU and complete slowly.
	// On a constrained system, we can end up taking too long and going non-ready.
	maxWorkers := runtime.GOMAXPROCS(0)
	sem := semaphore.NewWeighted(int64(maxWorkers))

	m.dirtyIfaceNames.Iter(func(item interface{}) error {
		ifaceName := item.(string)

		if !m.isWorkloadIface(ifaceName) {
			return nil
		}

		if err := sem.Acquire(context.Background(), 1); err != nil {
			// Should only happen if the context finishes.
			log.WithError(err).Panic("Failed to acquire semaphore")
		}
		m.onStillAlive()

		wg.Add(1)
		go func(ifaceName string) {
			defer wg.Done()
			defer sem.Release(1)
			err := m.applyPolicy(ifaceName)
			mutex.Lock()
			errs[ifaceName] = err
			mutex.Unlock()
		}(ifaceName)
		return nil
	})
	wg.Wait()

	if m.dirtyIfaceNames.Len() > 0 {
		// Clean up any left-over jump maps in the background...
		m.mapCleanupRunner.Trigger()
	}

	m.dirtyIfaceNames.Iter(func(item interface{}) error {
		ifaceName := item.(string)

		if !m.isWorkloadIface(ifaceName) {
			return nil
		}

		err := errs[ifaceName]
		wlID := m.nameToIface[ifaceName].info.endpointID
		if err == nil {
			log.WithField("iface", ifaceName).Info("Updated workload interface.")
			if wlID != nil && m.allWEPs[*wlID] != nil {
				if m.happyWEPs[*wlID] == nil {
					log.WithField("id", wlID).Info("Adding workload interface to iptables allow list.")
					m.happyWEPsDirty = true
				}
				m.happyWEPs[*wlID] = m.allWEPs[*wlID]
			}
			return set.RemoveItem
		} else {
			if wlID != nil && m.happyWEPs[*wlID] != nil {
				if !errors.Is(err, tc.ErrDeviceNotFound) {
					log.WithField("id", *wlID).WithError(err).Warning(
						"Failed to add policy to workload, removing from iptables allow list")
				}
				delete(m.happyWEPs, *wlID)
				m.happyWEPsDirty = true
			}
		}
		if errors.Is(err, tc.ErrDeviceNotFound) {
			log.WithField("wep", wlID).Debug(
				"Tried to apply BPF program to interface but the interface wasn't present.  " +
					"Will retry if it shows up.")
			return set.RemoveItem
		}
		log.WithError(err).WithFields(log.Fields{
			"wepID": wlID,
			"name":  ifaceName,
		}).Warn("Failed to apply policy to endpoint, leaving it dirty")
		return nil
	})
}

// applyPolicy actually applies the policy to the given workload.
func (m *bpfEndpointManager) applyPolicy(ifaceName string) error {
	startTime := time.Now()

	// Other threads might be filling in jump map FDs in the map so take the lock.
	m.ifacesLock.Lock()
	var endpointID *proto.WorkloadEndpointID
	var ifaceUp bool
	m.withIface(ifaceName, func(iface *bpfInterface) (forceDirty bool) {
		ifaceUp = iface.info.ifaceIsUp
		endpointID = iface.info.endpointID
		if !ifaceUp {
			log.WithField("iface", ifaceName).Debug("Interface is down/gone, closing jump maps.")
			for i := range iface.dpState.jumpMapFDs {
				if iface.dpState.jumpMapFDs[i] > 0 {
					err := iface.dpState.jumpMapFDs[i].Close()
					if err != nil {
						log.WithError(err).Error("Failed to close jump map.")
					}
					iface.dpState.jumpMapFDs[i] = 0
				}
			}
		}
		return false
	})
	m.ifacesLock.Unlock()

	if !ifaceUp {
		// Interface is gone, nothing to do.
		log.WithField("ifaceName", ifaceName).Debug(
			"Ignoring request to program interface that is not present.")
		return nil
	}

	// Otherwise, the interface appears to be present but we may or may not have an endpoint from the
	// datastore.  If we don't have an endpoint then we'll attach a program to block traffic and we'll
	// get the jump map ready to insert the policy if the endpoint shows up.

	// Attach the qdisc first; it is shared between the directions.
	err := tc.EnsureQdisc(ifaceName)
	if err != nil {
		if errors.Is(err, tc.ErrDeviceNotFound) {
			// Interface is gone, nothing to do.
			log.WithField("ifaceName", ifaceName).Debug(
				"Ignoring request to program interface that is not present.")
			return nil
		}
		return err
	}

	var ingressErr, egressErr error
	var wg sync.WaitGroup
	var wep *proto.WorkloadEndpoint
	if endpointID != nil {
		wep = m.allWEPs[*endpointID]
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		ingressErr = m.attachWorkloadProgram(ifaceName, wep, PolDirnIngress)
	}()
	go func() {
		defer wg.Done()
		egressErr = m.attachWorkloadProgram(ifaceName, wep, PolDirnEgress)
	}()
	wg.Wait()

	if ingressErr != nil {
		return ingressErr
	}
	if egressErr != nil {
		return egressErr
	}

	applyTime := time.Since(startTime)
	log.WithField("timeTaken", applyTime).Info("Finished applying BPF programs for workload")
	return nil
}

var calicoRouterIP = net.IPv4(169, 254, 1, 1).To4()

func (m *bpfEndpointManager) attachWorkloadProgram(ifaceName string, endpoint *proto.WorkloadEndpoint, polDirection PolDirection) error {
	ap := m.calculateTCAttachPoint(polDirection, ifaceName)
	// Host side of the veth is always configured as 169.254.1.1.
	ap.HostIP = calicoRouterIP
	// * VXLAN MTU should be the host ifaces MTU -50, in order to allow space for VXLAN.
	// * We also expect that to be the MTU used on veths.
	// * We do encap on the veths, and there's a bogus kernel MTU check in the BPF helper
	//   for resizing the packet, so we have to reduce the apparent MTU by another 50 bytes
	//   when we cannot encap the packet - non-GSO & too close to veth MTU
	ap.TunnelMTU = uint16(m.vxlanMTU - 50)

	l, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return err
	}
	ap.VethNS = uint16(l.Attrs().NetNsID)
	ap.EnableTCPStats = false
	if m.enableTcpStats {
		ap.EnableTCPStats = true
	}

	jumpMapFD, err := m.ensureProgramAttached(&ap, polDirection)
	if err != nil {
		return err
	}

	var profileIDs []string
	var tiers []*proto.TierInfo
	if endpoint != nil {
		profileIDs = endpoint.ProfileIds
		tiers = endpoint.Tiers
	} else {
		log.WithField("name", ifaceName).Debug(
			"Workload interface with no endpoint in datastore, installing default-drop program.")
	}

	// If tiers or profileIDs is nil, this will return an empty set of rules but updatePolicyProgram appends a
	// drop rule, giving us default drop behaviour in that case.
	rules := m.extractRules(tiers, profileIDs, polDirection)

	// If host-* endpoint is configured, add in its policy.
	if m.wildcardExists {
		m.addHostPolicy(&rules, &m.wildcardHostEndpoint, polDirection.Inverse())
	}

	// If workload egress and DefaultEndpointToHostAction is ACCEPT or DROP, suppress the normal
	// host-* endpoint policy.
	if polDirection == PolDirnEgress && m.epToHostAction != "RETURN" {
		rules.SuppressNormalHostPolicy = true
	}

	// If host -> workload, always suppress the normal host-* endpoint policy.
	if polDirection == PolDirnIngress {
		rules.SuppressNormalHostPolicy = true
	}

	return m.updatePolicyProgram(jumpMapFD, rules)
}

// Ensure TC program is attached to the specified interface and return its jump map FD.
func (m *bpfEndpointManager) ensureProgramAttached(ap *tc.AttachPoint, polDirection PolDirection) (bpf.MapFD, error) {
	jumpMapFD := m.getJumpMapFD(ap.Iface, polDirection)
	if jumpMapFD != 0 {
		if attached, err := ap.IsAttached(); err != nil {
			return jumpMapFD, fmt.Errorf("failed to check if interface %s had BPF program; %w", ap.Iface, err)
		} else if !attached {
			// BPF program is missing; maybe we missed a notification of the interface being recreated?
			// Close the now-defunct jump map.
			log.WithField("iface", ap.Iface).Info(
				"Detected that BPF program no longer attached to interface.")
			err := jumpMapFD.Close()
			if err != nil {
				log.WithError(err).Warn("Failed to close jump map FD. Ignoring.")
			}
			m.setJumpMapFD(ap.Iface, polDirection, 0)
			jumpMapFD = 0 // Trigger program to be re-added below.
		}
	}

	if jumpMapFD == 0 {
		// We don't have a program attached to this interface yet, attach one now.
		err := ap.AttachProgram()
		if err != nil {
			return 0, err
		}

		jumpMapFD, err = FindJumpMap(ap)
		if err != nil {
			return 0, fmt.Errorf("failed to look up jump map: %w", err)
		}
		m.setJumpMapFD(ap.Iface, polDirection, jumpMapFD)
	}

	return jumpMapFD, nil
}

func (m *bpfEndpointManager) addHostPolicy(rules *polprog.Rules, hostEndpoint *proto.HostEndpoint, polDirection PolDirection) {

	// When there is applicable pre-DNAT policy that does not explicitly Allow or Deny traffic,
	// we continue on to subsequent tiers and normal or AoF policy.
	rules.HostPreDnatTiers = m.extractTiers(hostEndpoint.PreDnatTiers, polDirection, NoEndTierDrop)

	// When there is applicable apply-on-forward policy that does not explicitly Allow or Deny
	// traffic, traffic is dropped.
	rules.HostForwardTiers = m.extractTiers(hostEndpoint.ForwardTiers, polDirection, EndTierDrop)

	// When there is applicable normal policy that does not explicitly Allow or Deny traffic,
	// traffic is dropped.
	rules.HostNormalTiers = m.extractTiers(hostEndpoint.Tiers, polDirection, EndTierDrop)
	rules.HostProfiles = m.extractProfiles(hostEndpoint.ProfileIds, polDirection)
}

func (m *bpfEndpointManager) getJumpMapFD(ifaceName string, direction PolDirection) (fd bpf.MapFD) {
	m.ifacesLock.Lock()
	defer m.ifacesLock.Unlock()
	m.withIface(ifaceName, func(iface *bpfInterface) bool {
		fd = iface.dpState.jumpMapFDs[direction]
		return false
	})
	return
}

func (m *bpfEndpointManager) setJumpMapFD(name string, direction PolDirection, fd bpf.MapFD) {
	m.ifacesLock.Lock()
	defer m.ifacesLock.Unlock()

	m.withIface(name, func(iface *bpfInterface) bool {
		iface.dpState.jumpMapFDs[direction] = fd
		return false
	})
}

func (m *bpfEndpointManager) ifaceIsUp(ifaceName string) (up bool) {
	m.ifacesLock.Lock()
	defer m.ifacesLock.Unlock()
	m.withIface(ifaceName, func(iface *bpfInterface) bool {
		up = iface.info.ifaceIsUp
		return false
	})
	return
}

func (m *bpfEndpointManager) updatePolicyProgram(jumpMapFD bpf.MapFD, rules polprog.Rules) error {
	pg := polprog.NewBuilder(m.ipSetIDAlloc, m.ipSetMap.MapFD(), m.stateMap.MapFD(), jumpMapFD,
		polprog.WithActionDropOverride(m.actionOnDrop),
	)
	insns, err := pg.Instructions(rules)
	if err != nil {
		return fmt.Errorf("failed to generate policy bytecode: %w", err)
	}
	progFD, err := bpf.LoadBPFProgramFromInsns(insns, "Apache-2.0", unix.BPF_PROG_TYPE_SCHED_CLS)
	if err != nil {
		return fmt.Errorf("failed to load BPF policy program: %w", err)
	}
	defer func() {
		// Once we've put the program in the map, we don't need its FD any more.
		err := progFD.Close()
		if err != nil {
			log.WithError(err).Panic("Failed to close program FD.")
		}
	}()
	k := make([]byte, 4)
	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, uint32(progFD))
	err = bpf.UpdateMapEntry(jumpMapFD, k, v)
	if err != nil {
		return fmt.Errorf("failed to update jump map: %w", err)
	}
	return nil
}

func (m *bpfEndpointManager) removePolicyProgram(jumpMapFD bpf.MapFD) error {
	k := make([]byte, 4)
	err := bpf.DeleteMapEntryIfExists(jumpMapFD, k, 4)
	if err != nil {
		return fmt.Errorf("failed to update jump map: %w", err)
	}
	return nil
}

func FindJumpMap(ap *tc.AttachPoint) (mapFD bpf.MapFD, err error) {
	logCtx := log.WithField("iface", ap.Iface)
	logCtx.Debug("Looking up jump map.")
	out, err := tc.ExecTC("filter", "show", "dev", ap.Iface, string(ap.Hook))
	if err != nil {
		return 0, fmt.Errorf("failed to find TC filter for interface %v: %w", ap.Iface, err)
	}

	progName := ap.ProgramName()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, progName) {
			re := regexp.MustCompile(`id (\d+)`)
			m := re.FindStringSubmatch(line)
			if len(m) > 0 {
				progIDStr := m[1]
				bpftool := exec.Command("bpftool", "prog", "show", "id", progIDStr, "--json")
				output, err := bpftool.Output()
				if err != nil {
					// We can hit this case if the interface was deleted underneath us; check that it's still there.
					if _, err := os.Stat(fmt.Sprintf("/proc/sys/net/ipv4/conf/%s", ap.Iface)); os.IsNotExist(err) {
						return 0, tc.ErrDeviceNotFound
					}

					return 0, fmt.Errorf("failed to get map metadata: %w", err)
				}
				var prog struct {
					MapIDs []int `json:"map_ids"`
				}
				err = json.Unmarshal(output, &prog)
				if err != nil {
					return 0, fmt.Errorf("failed to parse bpftool output: %w", err)
				}

				for _, mapID := range prog.MapIDs {
					mapFD, err := bpf.GetMapFDByID(mapID)
					if err != nil {
						return 0, fmt.Errorf("failed to get map FD from ID: %w", err)
					}
					mapInfo, err := bpf.GetMapInfo(mapFD)
					if err != nil {
						err = mapFD.Close()
						if err != nil {
							log.WithError(err).Panic("Failed to close FD.")
						}
						return 0, fmt.Errorf("failed to get map info: %w", err)
					}
					if mapInfo.Type == unix.BPF_MAP_TYPE_PROG_ARRAY {
						logCtx.WithField("fd", mapFD).Debug("Found jump map")
						return mapFD, nil
					}
					err = mapFD.Close()
					if err != nil {
						log.WithError(err).Panic("Failed to close FD.")
					}
				}
			}

			return 0, errors.New("failed to find map")
		}
	}
	return 0, errors.New("failed to find TC program")
}

func (m *bpfEndpointManager) attachDataIfaceProgram(ifaceName string, ep *proto.HostEndpoint, polDirection PolDirection) error {
	ap := m.calculateTCAttachPoint(polDirection, ifaceName)
	ap.HostIP = m.hostIP
	ap.TunnelMTU = uint16(m.vxlanMTU)

	jumpMapFD, err := m.ensureProgramAttached(&ap, polDirection)
	if err != nil {
		return err
	}

	if ep != nil {
		rules := polprog.Rules{
			ForHostInterface: true,
			NoProfileMatchID: m.profileNoMatchID(polDirection.RuleDir()),
		}
		m.addHostPolicy(&rules, ep, polDirection)
		return m.updatePolicyProgram(jumpMapFD, rules)
	}

	return m.removePolicyProgram(jumpMapFD)
}

// PolDirection is the Calico datamodel direction of policy.  On a host endpoint, ingress is towards the host.
// On a workload endpoint, ingress is towards the workload.
type PolDirection int

const (
	PolDirnIngress PolDirection = iota
	PolDirnEgress
)

func (polDirection PolDirection) RuleDir() rules.RuleDir {
	if polDirection == PolDirnIngress {
		return rules.RuleDirIngress
	}
	return rules.RuleDirEgress
}

func (polDirection PolDirection) Inverse() PolDirection {
	if polDirection == PolDirnIngress {
		return PolDirnEgress
	}
	return PolDirnIngress
}

func (m *bpfEndpointManager) calculateTCAttachPoint(policyDirection PolDirection, ifaceName string) tc.AttachPoint {
	var ap tc.AttachPoint
	var endpointType tc.EndpointType

	// Determine endpoint type.
	if m.isWorkloadIface(ifaceName) {
		endpointType = tc.EpTypeWorkload
	} else if ifaceName == "tunl0" {
		endpointType = tc.EpTypeTunnel
	} else if ifaceName == "wireguard.cali" {
		endpointType = tc.EpTypeWireguard
	} else if m.isDataIface(ifaceName) {
		endpointType = tc.EpTypeHost
	} else {
		log.Panicf("Unsupported ifaceName %v", ifaceName)
	}

	if endpointType == tc.EpTypeWorkload {
		// Policy direction is relative to the workload so, from the host namespace it's flipped.
		if policyDirection == PolDirnIngress {
			ap.Hook = tc.HookEgress
		} else {
			ap.Hook = tc.HookIngress
		}
	} else {
		// Host endpoints have the natural relationship between policy direction and hook.
		if policyDirection == PolDirnIngress {
			ap.Hook = tc.HookIngress
		} else {
			ap.Hook = tc.HookEgress
		}
	}

	var toOrFrom tc.ToOrFromEp
	if ap.Hook == tc.HookIngress {
		toOrFrom = tc.FromEp
	} else {
		toOrFrom = tc.ToEp
	}

	ap.Iface = ifaceName
	ap.Type = endpointType
	ap.ToOrFrom = toOrFrom
	ap.ToHostDrop = (m.epToHostAction == "DROP")
	ap.FIB = m.fibLookupEnabled
	ap.DSR = m.dsrEnabled
	ap.LogLevel = m.bpfLogLevel
	ap.VXLANPort = m.vxlanPort

	return ap
}

// Given a slice of TierInfo - as present on workload and host endpoints - that actually consists
// only of tier and policy NAMEs, build and return a slice of tier data that includes all of the
// implied policy rules as well.
const EndTierDrop = true
const NoEndTierDrop = false

func (m *bpfEndpointManager) extractTiers(tiers []*proto.TierInfo, direction PolDirection, endTierDrop bool) (rTiers []polprog.Tier) {
	dir := direction.RuleDir()
	for _, tier := range tiers {
		directionalPols := tier.IngressPolicies
		if direction == PolDirnEgress {
			directionalPols = tier.EgressPolicies
		}

		if len(directionalPols) > 0 {

			stagedOnly := true

			polTier := polprog.Tier{
				Name:     tier.Name,
				Policies: make([]polprog.Policy, len(directionalPols)),
			}

			for i, polName := range directionalPols {
				staged := model.PolicyIsStaged(polName)

				if !staged {
					stagedOnly = false
				}

				pol := m.policies[proto.PolicyID{Tier: tier.Name, Name: polName}]
				var prules []*proto.Rule
				if direction == PolDirnIngress {
					prules = pol.InboundRules
				} else {
					prules = pol.OutboundRules
				}
				policy := polprog.Policy{
					Name:   polName,
					Rules:  make([]polprog.Rule, len(prules)),
					Staged: staged,
				}

				if staged {
					policy.NoMatchID = m.policyNoMatchID(dir, polName)
				}

				for ri, r := range prules {
					policy.Rules[ri] = polprog.Rule{
						Rule:    r,
						MatchID: m.ruleMatchID(dir, r.Action, rules.RuleOwnerTypePolicy, ri, polName),
					}
				}

				polTier.Policies[i] = policy
			}

			if endTierDrop && !stagedOnly {
				polTier.EndRuleID = m.endOfTierDropID(dir, tier.Name)
				polTier.EndAction = polprog.TierEndDeny
			} else {
				polTier.EndRuleID = m.endOfTierPassID(dir, tier.Name)
				polTier.EndAction = polprog.TierEndPass
			}

			rTiers = append(rTiers, polTier)
		}
	}
	return
}

func (m *bpfEndpointManager) extractProfiles(profileNames []string, direction PolDirection) (rProfiles []polprog.Profile) {
	dir := direction.RuleDir()
	if count := len(profileNames); count > 0 {
		rProfiles = make([]polprog.Profile, count)

		for i, profName := range profileNames {
			prof := m.profiles[proto.ProfileID{Name: profName}]
			var prules []*proto.Rule
			if direction == PolDirnIngress {
				prules = prof.InboundRules
			} else {
				prules = prof.OutboundRules
			}
			profile := polprog.Profile{
				Name:  profName,
				Rules: make([]polprog.Rule, len(prules)),
			}

			for ri, r := range prules {
				profile.Rules[ri] = polprog.Rule{
					Rule:    r,
					MatchID: m.ruleMatchID(dir, r.Action, rules.RuleOwnerTypeProfile, ri, profName),
				}
			}

			rProfiles[i] = profile
		}
	}
	return
}

func (m *bpfEndpointManager) extractRules(tiers []*proto.TierInfo, profileNames []string, direction PolDirection) polprog.Rules {
	var r polprog.Rules
	dir := direction.RuleDir()

	// When there is applicable normal policy that does not explicitly Allow or Deny traffic,
	// traffic is dropped.
	r.Tiers = m.extractTiers(tiers, direction, EndTierDrop)

	r.Profiles = m.extractProfiles(profileNames, direction)

	r.NoProfileMatchID = m.profileNoMatchID(dir)

	return r
}

func strToByte64(s string) [64]byte {
	var bytes [64]byte
	copy(bytes[:], []byte(s))
	return bytes
}

func (m *bpfEndpointManager) ruleMatchIDFromNFLOGPrefix(nflogPrefix string) polprog.RuleMatchID {
	return m.lookupsCache.GetID64FromNFLOGPrefix(strToByte64(nflogPrefix))
}

func (m *bpfEndpointManager) endOfTierPassID(dir rules.RuleDir, tier string) polprog.RuleMatchID {
	return m.ruleMatchIDFromNFLOGPrefix(rules.CalculateEndOfTierPassNFLOGPrefixStr(dir, tier))
}

func (m *bpfEndpointManager) endOfTierDropID(dir rules.RuleDir, tier string) polprog.RuleMatchID {
	return m.ruleMatchIDFromNFLOGPrefix(rules.CalculateEndOfTierDropNFLOGPrefixStr(dir, tier))
}

func (m *bpfEndpointManager) policyNoMatchID(dir rules.RuleDir, policy string) polprog.RuleMatchID {
	return m.ruleMatchIDFromNFLOGPrefix(rules.CalculateNoMatchPolicyNFLOGPrefixStr(dir, policy))
}

func (m *bpfEndpointManager) profileNoMatchID(dir rules.RuleDir) polprog.RuleMatchID {
	return m.ruleMatchIDFromNFLOGPrefix(rules.CalculateNoMatchProfileNFLOGPrefixStr(dir))
}

func (m *bpfEndpointManager) ruleMatchID(
	dir rules.RuleDir,
	action string,
	owner rules.RuleOwnerType,
	idx int,
	name string) polprog.RuleMatchID {

	var a rules.RuleAction
	switch action {
	case "", "allow":
		a = rules.RuleActionAllow
	case "next-tier", "pass":
		a = rules.RuleActionPass
	case "deny":
		a = rules.RuleActionDeny
	case "log":
		// If we get it here, we dont know what to do about that, 0 means
		// invalid, but does not break anything.
		return 0
	default:
		log.WithField("action", action).Panic("Unknown rule action")
	}

	return m.ruleMatchIDFromNFLOGPrefix(rules.CalculateNFLOGPrefixStr(a, owner, dir, idx, name))
}

func (m *bpfEndpointManager) isWorkloadIface(iface string) bool {
	return m.workloadIfaceRegex.MatchString(iface)
}

func (m *bpfEndpointManager) isDataIface(iface string) bool {
	return m.dataIfaceRegex.MatchString(iface)
}

func (m *bpfEndpointManager) addWEPToIndexes(wlID proto.WorkloadEndpointID, wl *proto.WorkloadEndpoint) {
	for _, t := range wl.Tiers {
		m.addPolicyToEPMappings(t.IngressPolicies, wlID)
		m.addPolicyToEPMappings(t.EgressPolicies, wlID)
	}
	m.addProfileToEPMappings(wl.ProfileIds, wlID)
}

func (m *bpfEndpointManager) addPolicyToEPMappings(polNames []string, id interface{}) {
	for _, pol := range polNames {
		polID := proto.PolicyID{
			Tier: "default",
			Name: pol,
		}
		if m.policiesToWorkloads[polID] == nil {
			m.policiesToWorkloads[polID] = set.New()
		}
		m.policiesToWorkloads[polID].Add(id)
	}
}

func (m *bpfEndpointManager) addProfileToEPMappings(profileIds []string, id interface{}) {
	for _, profName := range profileIds {
		profID := proto.ProfileID{Name: profName}
		profSet := m.profilesToWorkloads[profID]
		if profSet == nil {
			profSet = set.New()
			m.profilesToWorkloads[profID] = profSet
		}
		profSet.Add(id)
	}
}

func (m *bpfEndpointManager) removeWEPFromIndexes(wlID proto.WorkloadEndpointID, wep *proto.WorkloadEndpoint) {
	if wep == nil {
		return
	}

	for _, t := range wep.Tiers {
		m.removePolicyToEPMappings(t.IngressPolicies, wlID)
		m.removePolicyToEPMappings(t.EgressPolicies, wlID)
	}

	m.removeProfileToEPMappings(wep.ProfileIds, wlID)

	m.withIface(wep.Name, func(iface *bpfInterface) bool {
		iface.info.endpointID = nil
		return false
	})
}

func (m *bpfEndpointManager) removePolicyToEPMappings(polNames []string, id interface{}) {
	for _, pol := range polNames {
		polID := proto.PolicyID{
			Tier: "default",
			Name: pol,
		}
		polSet := m.policiesToWorkloads[polID]
		if polSet == nil {
			continue
		}
		polSet.Discard(id)
		if polSet.Len() == 0 {
			// Defensive; we also clean up when the profile is removed.
			delete(m.policiesToWorkloads, polID)
		}
	}
}

func (m *bpfEndpointManager) removeProfileToEPMappings(profileIds []string, id interface{}) {
	for _, profName := range profileIds {
		profID := proto.ProfileID{Name: profName}
		profSet := m.profilesToWorkloads[profID]
		if profSet == nil {
			continue
		}
		profSet.Discard(id)
		if profSet.Len() == 0 {
			// Defensive; we also clean up when the policy is removed.
			delete(m.profilesToWorkloads, profID)
		}
	}
}

func (m *bpfEndpointManager) OnHEPUpdate(hostIfaceToEpMap map[string]proto.HostEndpoint) {
	log.Debugf("HEP update from generic endpoint manager: %v", hostIfaceToEpMap)

	// Pre-process the map for the host-* endpoint: if there is a host-* endpoint, any host
	// interface without its own HEP should use the host-* endpoint's policy.
	wildcardHostEndpoint, wildcardExists := hostIfaceToEpMap[allInterfaces]
	if wildcardExists {
		log.Info("Host-* endpoint is configured")
		for ifaceName := range m.nameToIface {
			if _, specificExists := hostIfaceToEpMap[ifaceName]; m.isDataIface(ifaceName) && !specificExists {
				log.Infof("Use host-* endpoint policy for %v", ifaceName)
				hostIfaceToEpMap[ifaceName] = wildcardHostEndpoint
			}
		}
		delete(hostIfaceToEpMap, allInterfaces)
	}

	// If there are parts of proto.HostEndpoint that do not affect us, we could mask those out
	// here so that they can't cause spurious updates - at the cost of having different
	// proto.HostEndpoint data here than elsewhere.  For example, the ExpectedIpv4Addrs and
	// ExpectedIpv6Addrs fields.  But currently there are no fields that are sufficiently likely
	// to change as to make this worthwhile.

	// If the host-* endpoint is changing, mark all workload interfaces as dirty.
	if (wildcardExists != m.wildcardExists) || !reflect.DeepEqual(wildcardHostEndpoint, m.wildcardHostEndpoint) {
		log.Infof("Host-* endpoint is changing; was %v, now %v", m.wildcardHostEndpoint, wildcardHostEndpoint)
		m.removeHEPFromIndexes(allInterfaces, &m.wildcardHostEndpoint)
		m.wildcardHostEndpoint = wildcardHostEndpoint
		m.wildcardExists = wildcardExists
		m.addHEPToIndexes(allInterfaces, &wildcardHostEndpoint)
		for ifaceName := range m.nameToIface {
			if m.isWorkloadIface(ifaceName) {
				log.Info("Mark WEP iface dirty, for host-* endpoint change")
				m.dirtyIfaceNames.Add(ifaceName)
			}
		}
	}

	// Loop through existing host endpoints, in case they are changing or disappearing.
	for ifaceName, existingEp := range m.hostIfaceToEpMap {
		newEp, stillExists := hostIfaceToEpMap[ifaceName]
		if stillExists && reflect.DeepEqual(newEp, existingEp) {
			log.Debugf("No change to host endpoint for ifaceName=%v", ifaceName)
		} else {
			m.removeHEPFromIndexes(ifaceName, &existingEp)
			if stillExists {
				log.Infof("Host endpoint changing for ifaceName=%v", ifaceName)
				m.addHEPToIndexes(ifaceName, &newEp)
				m.hostIfaceToEpMap[ifaceName] = newEp
			} else {
				log.Infof("Host endpoint deleted for ifaceName=%v", ifaceName)
				delete(m.hostIfaceToEpMap, ifaceName)
			}
			m.dirtyIfaceNames.Add(ifaceName)
		}
		delete(hostIfaceToEpMap, ifaceName)
	}

	// Now anything remaining in hostIfaceToEpMap must be a new host endpoint.
	for ifaceName, newEp := range hostIfaceToEpMap {
		if !m.isDataIface(ifaceName) {
			log.Warningf("Host endpoint configured for ifaceName=%v, but that doesn't match BPFDataIfacePattern; ignoring", ifaceName)
			continue
		}
		log.Infof("Host endpoint added for ifaceName=%v", ifaceName)
		m.addHEPToIndexes(ifaceName, &newEp)
		m.hostIfaceToEpMap[ifaceName] = newEp
		m.dirtyIfaceNames.Add(ifaceName)
	}
}

func (m *bpfEndpointManager) addHEPToIndexes(ifaceName string, ep *proto.HostEndpoint) {
	for _, tiers := range [][]*proto.TierInfo{ep.Tiers, ep.UntrackedTiers, ep.PreDnatTiers, ep.ForwardTiers} {
		for _, t := range tiers {
			m.addPolicyToEPMappings(t.IngressPolicies, ifaceName)
			m.addPolicyToEPMappings(t.EgressPolicies, ifaceName)
		}
	}
	m.addProfileToEPMappings(ep.ProfileIds, ifaceName)
}

func (m *bpfEndpointManager) removeHEPFromIndexes(ifaceName string, ep *proto.HostEndpoint) {
	for _, tiers := range [][]*proto.TierInfo{ep.Tiers, ep.UntrackedTiers, ep.PreDnatTiers, ep.ForwardTiers} {
		for _, t := range tiers {
			m.removePolicyToEPMappings(t.IngressPolicies, ifaceName)
			m.removePolicyToEPMappings(t.EgressPolicies, ifaceName)
		}
	}

	m.removeProfileToEPMappings(ep.ProfileIds, ifaceName)
}
