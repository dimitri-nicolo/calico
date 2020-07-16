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
package calico

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/kelseyhightower/confd/pkg/buildinfo"
	"github.com/kelseyhightower/confd/pkg/config"
	logutils "github.com/kelseyhightower/confd/pkg/log"
	log "github.com/sirupsen/logrus"

	"github.com/kelseyhightower/confd/pkg/resource/template"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/bgpsyncer"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	lerr "github.com/projectcalico/libcalico-go/lib/errors"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/selector"
	"github.com/projectcalico/typha/pkg/syncclientutils"
	"github.com/projectcalico/typha/pkg/syncproto"
)

const globalLogging = "/calico/bgp/v1/global/loglevel"

// Handle a few keys that we need to default if not specified.
var globalDefaults = map[string]string{
	"/calico/bgp/v1/global/as_num":    "64512",
	"/calico/bgp/v1/global/node_mesh": `{"enabled": true}`,
	globalLogging:                     "info",
}

var (
	globalConfigName        = "default"
	perNodeConfigNamePrefix = "node."
	nodeToNodeMeshEnabled   = "{\"enabled\":true}"
	nodeToNodeMeshDisabled  = "{\"enabled\":false}"
	standardCommunity       = regexp.MustCompile(`^(\d+):(\d+)$`)
	largeCommunity          = regexp.MustCompile(`^(\d+):(\d+):(\d+)$`)
)

// backendClientAccessor is an interface to access the backend client from the main v2 client.
type backendClientAccessor interface {
	Backend() api.Client
}

func NewCalicoClient(confdConfig *config.Config) (*client, error) {
	// Load the client clientCfg.  This loads from the environment if a filename
	// has not been specified.
	clientCfg, err := apiconfig.LoadClientConfig(confdConfig.CalicoConfig)
	if err != nil {
		log.Errorf("Failed to load Calico client configuration: %v", err)
		return nil, err
	}

	// Query the current BGP configuration to determine if the node to node mesh is enabled or
	// not.  If it is we need to monitor all node configuration.  If it is not enabled then we
	// only need to monitor our own node.  If this setting changes, we terminate confd (so that
	// when restarted it will start watching the correct resources).
	cc, err := clientv3.New(*clientCfg)
	if err != nil {
		log.Errorf("Failed to create main Calico client: %v", err)
		return nil, err
	}
	cfg, err := cc.BGPConfigurations().Get(
		context.Background(),
		globalConfigName,
		options.GetOptions{},
	)
	if _, ok := err.(lerr.ErrorResourceDoesNotExist); err != nil && !ok {
		// Failed to get the BGP configuration (and not because it doesn't exist).
		// Exit.
		log.Errorf("Failed to query current BGP settings: %v", err)
		return nil, err
	}
	nodeMeshEnabled := true
	if cfg != nil && cfg.Spec.NodeToNodeMeshEnabled != nil {
		nodeMeshEnabled = *cfg.Spec.NodeToNodeMeshEnabled
	}

	// We know the v2 client implements the backendClientAccessor interface.  Use it to
	// get the backend client.
	bc := cc.(backendClientAccessor).Backend()

	// Create the client.  Initialize the cache revision to 1 so that the watcher
	// code can handle the first iteration by always rendering.
	c := &client{
		client:            bc,
		cache:             make(map[string]string),
		peeringCache:      make(map[string]string),
		cacheRevision:     1,
		revisionsByPrefix: make(map[string]uint64),
		nodeMeshEnabled:   nodeMeshEnabled,
		nodeLabels:        make(map[string]map[string]string),
		bgpPeers:          make(map[string]*apiv3.BGPPeer),
		sourceReady:       make(map[string]bool),
		nodeListenPorts:   make(map[string]int),

		// This channel, for the syncer calling OnUpdates and OnStatusUpdated, has 0
		// capacity so that the caller blocks in the same way as it did before when its
		// calls were processed synchronously.
		syncerC: make(chan interface{}),

		// This channel holds a trigger for existing BGP peerings to be recomputed.  We only
		// ever need 1 pending trigger, hence capacity 1.  recheckPeerConfig() does a
		// non-blocking write into this channel, so as not to block if a trigger is already
		// pending.
		recheckC: make(chan struct{}, 1),
	}
	for k, v := range globalDefaults {
		c.cache[k] = v
	}

	// Create secret watcher.  Must do this before the syncer, because updates from
	// the syncer can trigger calling c.secretWatcher.MarkStale().
	if c.secretWatcher, err = NewSecretWatcher(c); err != nil {
		log.WithError(err).Warning("Failed to create secret watcher, not running under Kubernetes?")
	}

	// Create a conditional that we use to wake up all of the watcher threads when there
	// may some actionable updates.
	c.watcherCond = sync.NewCond(&c.cacheLock)

	// Increment the waitForSync wait group.  This blocks the GetValues call until the
	// syncer has completed its initial snapshot and is in sync.
	c.waitForSync.Add(1)

	// Get cluster CIDRs. Prefer the env var, if specified.
	clusterCIDRs := []string{}
	if clusterCIDR := os.Getenv(envAdvertiseClusterIPs); len(clusterCIDR) != 0 {
		clusterCIDRs = []string{clusterCIDR}
	} else if cfg != nil && cfg.Spec.ServiceClusterIPs != nil {
		for _, c := range cfg.Spec.ServiceClusterIPs {
			clusterCIDRs = append(clusterCIDRs, c.CIDR)
		}
	}
	// Note: do this initial update before starting the syncer, so there's no chance of this
	// racing with syncer-derived updates.
	c.onClusterIPsUpdate(clusterCIDRs)

	// Get external IP CIDRs.
	externalCIDRs := []string{}
	if cfg != nil && cfg.Spec.ServiceExternalIPs != nil {
		for _, c := range cfg.Spec.ServiceExternalIPs {
			externalCIDRs = append(externalCIDRs, c.CIDR)
		}
	}
	// Note: do this initial update before starting the syncer, so there's no chance of this
	// racing with syncer-derived updates.
	c.onExternalIPsUpdate(externalCIDRs)

	// Start the main syncer loop.  If the node-to-node mesh is enabled then we need to
	// monitor all nodes.  If this setting changes (which we will monitor in the OnUpdates
	// callback) then we terminate confd - the calico/node init process will restart the
	// confd process.
	c.nodeLogKey = fmt.Sprintf("/calico/bgp/v1/host/%s/loglevel", template.NodeName)
	c.nodeIPv4Key = fmt.Sprintf("/calico/bgp/v1/host/%s/ip_addr_v4", template.NodeName)
	c.nodeIPv6Key = fmt.Sprintf("/calico/bgp/v1/host/%s/ip_addr_v6", template.NodeName)
	c.nodeV1Processor = updateprocessors.NewBGPNodeUpdateProcessor(clientCfg.Spec.K8sUsePodCIDR)
	if syncclientutils.MustStartSyncerClientIfTyphaConfigured(
		&confdConfig.Typha, syncproto.SyncerTypeBGP,
		buildinfo.GitVersion, template.NodeName, fmt.Sprintf("confd %s", buildinfo.GitVersion),
		c,
	) {
		log.Debug("Using typha syncclient")
	} else {
		// Use the syncer locally.
		log.Debug("Using local syncer")
		c.syncer = bgpsyncer.New(c.client, c, template.NodeName, clientCfg.Spec)
		c.syncer.Start()
	}

	if len(clusterCIDRs) != 0 || len(externalCIDRs) != 0 {
		// Create and start route generator, if configured to do so. This can either be through
		// environment variable, or the data store via BGPConfiguration.
		// We only turn it on if configured to do so, to avoid needing to watch services / endpoints.
		log.Info("Starting route generator for service advertisement")
		if c.rg, err = NewRouteGenerator(c); err != nil {
			log.WithError(err).Error("Failed to start route generator, routes will not be advertised")
			c.OnSyncChange(SourceRouteGenerator, true)
			c.rg = nil
		} else {
			c.rg.Start()
		}
	} else {
		c.OnSyncChange(SourceRouteGenerator, true)
	}

	// Start a goroutine to process updates in a way that's decoupled from their sources.
	go func() {
		for {
			select {
			case e := <-c.syncerC:
				switch event := e.(type) {
				case []api.Update:
					c.onUpdates(event, false)
				case api.SyncStatus:
					c.onStatusUpdated(event)
				default:
					log.Panicf("Unknown type %T in syncer channel", event)
				}
			case <-c.recheckC:
				log.Info("Recompute v1 BGP peerings")
				c.onUpdates(nil, true)
			}
		}
	}()

	return c, nil
}

var (
	SourceSyncer         string = "SourceSyncer"
	SourceRouteGenerator string = "SourceRouteGenerator"
)

// client implements the StoreClient interface for confd, and also implements the
// Calico api.SyncerCallbacks and api.SyncerParseFailCallbacks interfaces for the
// BGP Syncer.
type client struct {
	// The Calico backend client.
	client api.Client

	// The BGP syncer.
	syncer           api.Syncer
	nodeV1Processor  watchersyncer.SyncerUpdateProcessor
	nodeLabels       map[string]map[string]string
	bgpPeers         map[string]*apiv3.BGPPeer
	globalListenPort int
	nodeListenPorts  map[string]int

	// The route generator
	rg *routeGenerator

	// Readiness signals for individual data sources.
	sourceReady map[string]bool

	// Indicates whether all data sources have synced. We cannot start rendering until
	// all sources have synced, so we block calls to GetValues until this is true.
	syncedOnce  bool
	waitForSync sync.WaitGroup

	// Our internal cache of key/values, and our (internally defined) cache revision.
	cache         map[string]string
	peeringCache  map[string]string
	cacheRevision uint64

	// The current revision for each prefix.  A revision is updated when we have a sync
	// event that updates any keys with that prefix.
	revisionsByPrefix map[string]uint64

	// Lock used to synchronize access to any of the shared mutable data.
	cacheLock   sync.Mutex
	watcherCond *sync.Cond

	// Whether the node to node mesh is enabled or not.
	nodeMeshEnabled bool

	// This node's log level key.
	nodeLogKey string

	// Current values of <bgpconfig>.spec.serviceExternalIPs and
	// <bgpconfig>.spec.serviceClusterIPs.
	externalIPs    []string
	externalIPNets []*net.IPNet // same as externalIPs but parsed
	clusterCIDRs   []string

	// This node's IP address keys.
	nodeIPv4Key string
	nodeIPv6Key string

	// Subcomponent for accessing and watching secrets (that hold BGP passwords).
	secretWatcher *secretWatcher

	// Channels used to decouple update and status processing.
	syncerC  chan interface{}
	recheckC chan struct{}
}

// SetPrefixes is called from confd to notify this client of the full set of prefixes that will
// be watched.
// This client uses this information to initialize the revision map used to keep track of the
// revision number of each prefix that the template is monitoring.
func (c *client) SetPrefixes(keys []string) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	log.Debugf("Set prefixes called with: %v", keys)
	for _, k := range keys {
		// Initialise the revision that we are watching for this prefix.  This will be updated
		// if we receive any syncer events for keys with this prefix.  The Watcher function will
		// then check the revisions it is interested in to see if there is an updated revision
		// that it needs to process.
		c.revisionsByPrefix[k] = 0
	}

	return nil
}

// OnStatusUpdated is called from the BGP syncer to indicate that the sync status is updated.
// This client handles InSync and WaitForDatastore statuses. When we receive InSync, we unblock GetValues calls.
// When we receive WaitForDatastore and are already InSync, we reset the client's syncer status which blocks
// GetValues calls.
func (c *client) OnStatusUpdated(status api.SyncStatus) {
	c.syncerC <- status
}

func (c *client) onStatusUpdated(status api.SyncStatus) {
	log.Debugf("Got status update: %s", status)
	switch status {
	case api.InSync:
		c.OnSyncChange(SourceSyncer, true)
	case api.WaitForDatastore:
		c.OnSyncChange(SourceSyncer, false)
	}
}

// OnInSync handles multiplexing in-sync messages from multiple data sources
// into a single representation of readiness.
func (c *client) OnSyncChange(source string, ready bool) {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	if ready == c.sourceReady[source] {
		log.Debugf("No change for source %v, ready %v", source, ready)
		return
	}

	log.Infof("Source %v readiness changed, ready=%v", source, ready)

	// Check if we are fully in sync, before applying this change.
	oldFullSync := c.sourceReady[SourceSyncer] && c.sourceReady[SourceRouteGenerator]

	// Apply the change.
	c.sourceReady[source] = ready

	// Check if we are fully in sync now.
	newFullSync := c.sourceReady[SourceSyncer] && c.sourceReady[SourceRouteGenerator]

	if newFullSync == oldFullSync {
		log.Debugf("No change to full sync status (%v)", newFullSync)
		return
	}

	if newFullSync {
		// All data sources are ready.
		c.syncedOnce = true
		c.waitForSync.Done()
		log.Info("Data is now syncd, can start rendering templates")

		// Now that we're in-sync, check if we should update our log level
		// based on the datastore config.
		c.updateLogLevel()
	} else {
		log.Info("Full sync lost")
		c.waitForSync.Add(1)
	}
}

type bgpPeer struct {
	PeerIP            cnet.IP              `json:"ip"`
	ASNum             numorstring.ASNumber `json:"as_num,string"`
	RRClusterID       string               `json:"rr_cluster_id"`
	Extensions        map[string]string    `json:"extensions"`
	Password          *string              `json:"password"`
	SourceAddr        string               `json:"source_addr"`
	DirectlyConnected bool                 `json:"directly_connected"`
	RestartMode       string               `json:"restart_mode"`
	RestartTime       string               `json:"restart_time"`
	GatewayMode       string               `json:"gateway_mode"`
	EnableBFD         bool                 `json:"enable_bfd"`
	Port              string               `json:"port"`
}

type bgpPrefix struct {
	CIDR        string   `json:"cidr"`
	Communities []string `json:"communities"`
}

func (c *client) getPassword(v3res *apiv3.BGPPeer) *string {
	if c.secretWatcher != nil && v3res.Spec.Password != nil && v3res.Spec.Password.SecretKeyRef != nil {
		password, err := c.secretWatcher.GetSecret(
			v3res.Spec.Password.SecretKeyRef.Name,
			v3res.Spec.Password.SecretKeyRef.Key,
		)
		if err == nil {
			return &password
		}
		log.WithError(err).Warningf("Can't read password for BGPPeer %v", v3res.Name)
	}
	return nil
}

func (c *client) updatePeersV1() {
	// A map that will contain the v1 peerings that should exist, with the same key and
	// value form as c.peeringCache.
	peersV1 := make(map[string]string)

	// Common subroutine for emitting both global and node-specific peerings.
	emit := func(key model.Key, peer *bgpPeer) {
		log.WithFields(log.Fields{"key": key, "peer": peer}).Debug("Maybe emit peering")

		// Compute etcd v1 path for this peering key.
		k, err := model.KeyToDefaultPath(key)
		if err != nil {
			log.Errorf("Ignoring update: unable to create path from Key %v: %v", key, err)
			return
		}

		// If we already have an entry for that path, it wins.  When we're
		// emitting reverse peerings to ensure symmetry, this is what ensures
		// that an explicit forwards peering is not overwritten by an implicit
		// reverse peering.
		if _, ok := peersV1[k]; ok {
			log.Debug("Peering already exists")
			return
		}

		// If we would be emitting a node-specific peering to a peer IP, and we
		// already have a global peering to that IP, skip emitting the node-specific
		// one.
		if nodeKey, ok := key.(model.NodeBGPPeerKey); ok {
			globalKey := model.GlobalBGPPeerKey{PeerIP: nodeKey.PeerIP}
			globalPath, _ := model.KeyToDefaultPath(globalKey)
			if _, ok = peersV1[globalPath]; ok {
				log.Debug("Global peering already exists")
				return
			}
		}

		// Serialize and store the value for this peering.
		value, err := json.Marshal(peer)
		if err != nil {
			log.Errorf("Ignoring update: unable to serialize value %v: %v", peer, err)
			return
		}
		peersV1[k] = string(value)
	}

	// Mark currently watched secrets as stale, so that they can be cleaned up if no
	// longer needed.
	if c.secretWatcher != nil {
		c.secretWatcher.MarkStale()
	}

	// Loop through v3 BGPPeers twice, first to emit global peerings, then for
	// node-specific ones.  The point here is to emit all of the possible global peerings
	// _first_, so that we can then skip emitting any node-specific peerings that would
	// duplicate those on particular nodes.
	for _, globalPass := range []bool{true, false} {
		for _, v3res := range c.bgpPeers {
			log.WithField("peer", v3res).Debug("Process v3 BGPPeer")
			if globalPass != ((v3res.Spec.NodeSelector == "") && (v3res.Spec.Node == "")) {
				log.WithField("globalPass", globalPass).Debug("Skip BGPPeer on this pass")
				continue
			}

			var localNodeNames []string
			if v3res.Spec.NodeSelector != "" {
				localNodeNames = c.nodesMatching(v3res.Spec.NodeSelector)
			} else if v3res.Spec.Node != "" {
				localNodeNames = []string{v3res.Spec.Node}
			}
			log.Debugf("Local nodes %#v", localNodeNames)

			var peers []*bgpPeer
			if v3res.Spec.PeerSelector != "" {
				for _, peerNodeName := range c.nodesMatching(v3res.Spec.PeerSelector) {
					peers = append(peers, c.nodeAsBGPPeers(peerNodeName)...)
				}
			} else {
				// Separate port from Ip if it uses <ip>:<port> format
				host, port := parseIPPort(v3res.Spec.PeerIP)
				ip := cnet.ParseIP(host)
				if ip == nil {
					log.Warning("PeerIP is not assigned or is malformed")
					continue
				}

				// If port is not empty, we use the given value to peer.
				// If port is empty, check if BGP Peer is a calico/node, and use its listenPort to peer.
				if port == "" {
					// If port is empty, nodesWithIPPortAndAS() returns list of calico/node that matches IP and ASNumber.
					nodeNames := c.nodesWithIPPortAndAS(host, v3res.Spec.ASNumber, port)
					if len(nodeNames) != 0 {
						if nodePort, ok := c.nodeListenPorts[nodeNames[0]]; ok {
							port = strconv.Itoa(nodePort)
						} else if c.globalListenPort != 0 {
							port = strconv.Itoa(c.globalListenPort)
						}
					}
				}

				peers = append(peers, &bgpPeer{
					PeerIP:     *ip,
					ASNum:      v3res.Spec.ASNumber,
					SourceAddr: string(v3res.Spec.SourceAddress),
					Port:       port,
				})
			}
			log.Debugf("Peers %#v", peers)

			if len(peers) == 0 {
				continue
			}

			c.setPeerConfigFieldsFromV3Resource(peers, v3res)

			for _, peer := range peers {
				log.Debugf("Peer: %#v", peer)
				if globalPass {
					key := model.GlobalBGPPeerKey{PeerIP: peer.PeerIP}
					emit(key, peer)
				} else {
					for _, localNodeName := range localNodeNames {
						log.Debugf("Local node name: %#v", localNodeName)
						key := model.NodeBGPPeerKey{Nodename: localNodeName, PeerIP: peer.PeerIP}
						emit(key, peer)
					}
				}
			}
		}
	}

	// Loop through v3 BGPPeers again to add in any missing reverse peerings.
	for _, v3res := range c.bgpPeers {
		log.WithField("peer", v3res).Debug("Second pass with v3 BGPPeer")

		// This time, the "local" nodes are actually those matching the remote fields
		// in BGPPeer, i.e. PeerIP, ASNumber and PeerSelector...
		var localNodeNames []string
		if v3res.Spec.PeerSelector != "" {
			localNodeNames = c.nodesMatching(v3res.Spec.PeerSelector)
		} else {
			ip, port := parseIPPort(v3res.Spec.PeerIP)
			localNodeNames = c.nodesWithIPPortAndAS(ip, v3res.Spec.ASNumber, port)
		}
		log.Debugf("Local nodes %#v", localNodeNames)

		// Skip peer computation if there are no local nodes.
		if len(localNodeNames) == 0 {
			continue
		}

		// ...and the "peer" nodes are those matching the local fields in BGPPeer, i.e
		// Node and NodeSelector.
		var peerNodeNames []string
		if v3res.Spec.NodeSelector != "" {
			peerNodeNames = c.nodesMatching(v3res.Spec.NodeSelector)
		} else if v3res.Spec.Node != "" {
			peerNodeNames = []string{v3res.Spec.Node}
		} else {
			peerNodeNames = c.nodesMatching("all()")
		}
		log.Debugf("Peers %#v", peerNodeNames)

		if len(peerNodeNames) == 0 {
			continue
		}

		var peers []*bgpPeer
		for _, peerNodeName := range peerNodeNames {
			peers = append(peers, c.nodeAsBGPPeers(peerNodeName)...)
		}
		if len(peers) == 0 {
			continue
		}

		c.setPeerConfigFieldsFromV3Resource(peers, v3res)

		for _, peer := range peers {
			for _, localNodeName := range localNodeNames {
				key := model.NodeBGPPeerKey{Nodename: localNodeName, PeerIP: peer.PeerIP}
				emit(key, peer)
			}
		}
	}

	// Clean up any secrets that are no longer of interest.
	if c.secretWatcher != nil {
		c.secretWatcher.SweepStale()
	}

	// Now reconcile against the cache.
	for k, value := range c.peeringCache {
		newValue, ok := peersV1[k]
		if !ok {
			// This cache entry should be deleted.
			delete(c.peeringCache, k)
			c.keyUpdated(k)
		} else if newValue != value {
			// This cache entry should be updated.
			c.peeringCache[k] = newValue
			c.keyUpdated(k)
			delete(peersV1, k)
		} else {
			// Value in cache is already correct.  Delete from peersV1 so that we
			// don't generate a spurious keyUpdated for this key.
			delete(peersV1, k)
		}
	}
	// peersV1 now only contains peerings to add to the cache.
	for k, newValue := range peersV1 {
		c.peeringCache[k] = newValue
		c.keyUpdated(k)
	}
}

func parseIPPort(ipPort string) (string, string) {
	host, port, err := net.SplitHostPort(ipPort)
	if err != nil {
		log.Debug("No custom port set for peer.")
		return ipPort, ""
	}
	return host, port
}

func (c *client) nodesMatching(rawSelector string) []string {
	nodeNames := []string{}
	sel, err := selector.Parse(rawSelector)
	if err != nil {
		log.Errorf("Couldn't parse selector: %v", rawSelector)
		return nodeNames
	}
	for nodeName, labels := range c.nodeLabels {
		if sel.Evaluate(labels) {
			nodeNames = append(nodeNames, nodeName)
		}
	}
	return nodeNames
}

func (c *client) nodesWithIPPortAndAS(ip string, asNum numorstring.ASNumber, port string) []string {
	globalAS := c.globalAS()
	var asStr string
	if asNum == numorstring.ASNumber(0) {
		asStr = globalAS
	} else {
		asStr = asNum.String()
	}
	ipPort, _ := strconv.Atoi(port)
	nodeNames := []string{}
	for nodeName := range c.nodeLabels {
		nodeIPv4, nodeIPv6, nodeAS, _ := c.nodeToBGPFields(nodeName)
		if (nodeIPv4 != ip) && (nodeIPv6 != ip) {
			continue
		}
		if nodeAS == "" {
			nodeAS = globalAS
		}
		if nodeAS != asStr {
			continue
		}
		// Port in PeerIP is optional, do not compare with listenPort if it is not set.
		if ipPort != 0 {
			if nodePort, ok := c.nodeListenPorts[nodeName]; ok && ipPort != nodePort {
				continue
			} else if c.globalListenPort != 0 && c.globalListenPort != ipPort {
				continue
			}
		}
		nodeNames = append(nodeNames, nodeName)
	}
	return nodeNames
}

func (c *client) nodeToBGPFields(nodeName string) (string, string, string, string) {
	ipv4Key, _ := model.KeyToDefaultPath(model.NodeBGPConfigKey{Nodename: nodeName, Name: "ip_addr_v4"})
	ipv6Key, _ := model.KeyToDefaultPath(model.NodeBGPConfigKey{Nodename: nodeName, Name: "ip_addr_v6"})
	asKey, _ := model.KeyToDefaultPath(model.NodeBGPConfigKey{Nodename: nodeName, Name: "as_num"})
	rrKey, _ := model.KeyToDefaultPath(model.NodeBGPConfigKey{Nodename: nodeName, Name: "rr_cluster_id"})
	return c.cache[ipv4Key], c.cache[ipv6Key], c.cache[asKey], c.cache[rrKey]
}

func (c *client) globalAS() string {
	asKey, _ := model.KeyToDefaultPath(model.GlobalBGPConfigKey{Name: "as_num"})
	return c.cache[asKey]
}

func (c *client) nodeAsBGPPeers(nodeName string) (peers []*bgpPeer) {
	ipv4Str, ipv6Str, asNum, rrClusterID := c.nodeToBGPFields(nodeName)
	for version, ipStr := range map[string]string{
		"IPv4": ipv4Str,
		"IPv6": ipv6Str,
	} {
		peer := &bgpPeer{}
		if ipStr == "" {
			log.Debugf("No %v for node %v", version, nodeName)
			continue
		}
		ip := cnet.ParseIP(ipStr)
		if ip == nil {
			log.Warningf("Couldn't parse %v %v for node %v", version, ipStr, nodeName)
			continue
		}
		peer.PeerIP = *ip

		// If peer node has listenPort set in BGPConfiguration, use that.
		if port, ok := c.nodeListenPorts[nodeName]; ok {
			peer.Port = strconv.Itoa(port)
		} else if c.globalListenPort != 0 {
			peer.Port = strconv.Itoa(c.globalListenPort)
		}

		var err error
		if asNum != "" {
			log.Debugf("ASNum for %v is %#v", nodeName, asNum)
			peer.ASNum, err = numorstring.ASNumberFromString(asNum)
			if err != nil {
				log.WithError(err).Warningf("Problem parsing AS number %v for node %v", asNum, nodeName)
			}
		} else {
			asNum = c.globalAS()
			log.Debugf("Global ASNum for %v is %#v", nodeName, asNum)
			peer.ASNum, err = numorstring.ASNumberFromString(asNum)
			if err != nil {
				log.WithError(err).Warningf("Problem parsing global AS number %v for node %v", asNum, nodeName)
			}
		}
		peer.RRClusterID = rrClusterID
		peers = append(peers, peer)
	}
	return
}

// OnUpdates is called from the BGP syncer to indicate that new updates are available from the
// Calico datastore.
// This client does the following:
// -  stores the updates in its local cache
// -  increments the revision number associated with each of the affected watch prefixes
// -  wakes up the watchers so that they can check if any of the prefixes they are
//    watching have been updated.
func (c *client) OnUpdates(updates []api.Update) {
	c.syncerC <- updates
}

func (c *client) onUpdates(updates []api.Update, needUpdatePeersV1 bool) {

	// Update our cache from the updates.
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	// Indicate that our cache has been updated.
	c.incrementCacheRevision()

	// Track whether these updates require BGP peerings to be recomputed.
	needUpdatePeersReasons := []string{}

	// Track whether these updates require service advertisement to be recomputed.
	needServiceAdvertisementUpdates := false

	log.WithField("cacheRevision", c.cacheRevision).Debug("Processing OnUpdates from syncer")
	for _, u := range updates {
		log.Debugf("Update: %#v", u)

		// confd now receives Nodes, BGPPeers and BGPConfig as v3 resources.
		//
		// For each Node, we save off the node's labels, then convert to v1 so that
		// the same etcd key/value pairs appear as before (so that existing confd
		// templates will continue to work).
		//
		// BGPPeers are saved off and then the whole set is processed to generate a
		// corresponding set of v1 BGPPeers, bearing in mind (a) the possible use of
		// v3 BGPPeer selector fields, and (b) that we fill in any reverse peerings
		// that are needed for symmetry between Calico nodes.  Each v1 BGPPeer then
		// generates etcd key/value pairs as expected by existing confd templates.
		//
		// For BGP configuration recalculate peers when we receive updates with AS number.
		v3key, ok := u.Key.(model.ResourceKey)
		if !ok {
			// Not a v3 resource.
			continue
		}

		// It's a v3 resource - we care about some of these.
		if v3key.Kind == apiv3.KindNode {
			// Convert to v1 key/value pairs.
			log.Debugf("Node: %#v", u.Value)
			if u.Value != nil {
				log.Debugf("BGPSpec: %#v", u.Value.(*apiv3.Node).Spec.BGP)
			}
			kvps, err := c.nodeV1Processor.Process(&u.KVPair)
			if err != nil {
				log.Errorf("Problem converting Node resource: %v", err)
				continue
			}
			for _, kvp := range kvps {
				log.Debugf("KVP: %#v", kvp)
				if kvp.Value == nil {
					if c.updateCache(api.UpdateTypeKVDeleted, kvp) {
						needUpdatePeersV1 = true
						needUpdatePeersReasons = append(needUpdatePeersReasons, fmt.Sprintf("%s deleted", kvp.Key.String()))
					}
				} else {
					if c.updateCache(u.UpdateType, kvp) {
						needUpdatePeersV1 = true
						needUpdatePeersReasons = append(needUpdatePeersReasons, fmt.Sprintf("%s updated", kvp.Key.String()))
					}
				}
			}

			// Update our cache of node labels.
			if u.Value == nil {
				// This was a delete - remove node labels.
				if _, ok := c.nodeLabels[v3key.Name]; ok {
					delete(c.nodeLabels, v3key.Name)
					needUpdatePeersV1 = true
					needUpdatePeersReasons = append(needUpdatePeersReasons, v3key.Name+" deleted")
				}
			} else {
				// This was a create or update - update node labels.
				v3res, ok := u.Value.(*apiv3.Node)
				if !ok {
					log.Warning("Bad value for Node resource")
					continue
				}
				existingLabels, isSet := c.nodeLabels[v3key.Name]
				if !isSet || !reflect.DeepEqual(existingLabels, v3res.Labels) {
					c.nodeLabels[v3key.Name] = v3res.Labels
					needUpdatePeersV1 = true
					needUpdatePeersReasons = append(needUpdatePeersReasons, v3key.Name+" updated")
				}
			}
		}

		if v3key.Kind == apiv3.KindBGPPeer {
			// Update our cache of v3 BGPPeer resources.
			if u.Value == nil || u.UpdateType == api.UpdateTypeKVDeleted {
				delete(c.bgpPeers, v3key.Name)
			} else if v3res, ok := u.Value.(*apiv3.BGPPeer); ok {
				c.bgpPeers[v3key.Name] = v3res
			} else {
				log.Warning("Bad value for BGPPeer resource")
				continue
			}

			// Note need to recompute equivalent v1 peerings.
			needUpdatePeersV1 = true
			needUpdatePeersReasons = append(needUpdatePeersReasons, "BGP peer updated or deleted")
		}
	}

	// Update our cache from each of the individual updates, and keep track of
	// any of the prefixes that are impacted.
	for _, u := range updates {
		if v3key, ok := u.Key.(model.ResourceKey); ok && v3key.Kind == apiv3.KindBGPConfiguration {
			// Convert v3 BGPConfiguration to equivalent v1 cache values
			v3res, _ := u.KVPair.Value.(*apiv3.BGPConfiguration)
			c.updateBGPConfigCache(v3key.Name, v3res, &needServiceAdvertisementUpdates, &needUpdatePeersV1, &needUpdatePeersReasons)
		}
		c.updateCache(u.UpdateType, &u.KVPair)
	}

	// If configuration relevant to BGP peerings has changed, recalculate the set of v1
	// peerings that should exist, and update the cache accordingly.
	if needUpdatePeersV1 {
		log.Info("Recompute BGP peerings: " + strings.Join(needUpdatePeersReasons, "; "))
		c.updatePeersV1()
	}

	// If we need to update Service advertisement based on the updates, then do so.
	if needServiceAdvertisementUpdates {
		log.Info("Updates included service advertisement changes.")
		if c.rg == nil {
			// If this is the first time we've needed to start the route generator, then do so here.
			log.Info("Starting route generator due to service advertisement update")
			var err error
			if c.rg, err = NewRouteGenerator(c); err != nil {
				log.WithError(err).Error("Failed to start route generator, unable to advertise node-specific service routes")
				c.rg = nil
			} else {
				c.rg.Start()
			}
		}

		// Update external IP CIDRs. In v1 format, they are a single comma-separated
		// string. If the string isn't empty, split on the comma and pass a list of strings
		// to the route generator.  An empty string indicates a withdrawal of that set of
		// service IPs.
		var externalIPs []string
		if len(c.cache["/calico/bgp/v1/global/svc_external_ips"]) > 0 {
			externalIPs = strings.Split(c.cache["/calico/bgp/v1/global/svc_external_ips"], ",")
		}
		c.onExternalIPsUpdate(externalIPs)

		// Same for cluster CIDRs.
		var clusterIPs []string
		if len(c.cache["/calico/bgp/v1/global/svc_cluster_ips"]) > 0 {
			clusterIPs = strings.Split(c.cache["/calico/bgp/v1/global/svc_cluster_ips"], ",")
		}
		c.onClusterIPsUpdate(clusterIPs)

		if c.rg != nil {
			// Trigger the route generator to recheck and advertise or withdraw
			// node-specific routes.
			c.rg.TriggerResync()
		}
	}

	// Notify watcher thread that we've received new updates.
	log.WithField("cacheRevision", c.cacheRevision).Debug("Done processing OnUpdates from syncer, notify watchers")
	c.onNewUpdates()
}

func (c *client) updateBGPConfigCache(resName string, v3res *apiv3.BGPConfiguration, svcAdvertisement *bool, updatePeersV1 *bool, updateReasons *[]string) {

	if resName == globalConfigName {
		c.getPrefixAdvertisementsKVPair(v3res, model.GlobalBGPConfigKey{})
		c.getListenPortKVPair(v3res, model.GlobalBGPConfigKey{})
		c.getASNumberKVPair(v3res, model.GlobalBGPConfigKey{}, updatePeersV1, updateReasons)
		c.getServiceExternalIPsKVPair(v3res, model.GlobalBGPConfigKey{}, svcAdvertisement)
		c.getServiceClusterIPsKVPair(v3res, model.GlobalBGPConfigKey{}, svcAdvertisement)
		c.getNodeToNodeMeshKVPair(v3res, model.GlobalBGPConfigKey{})
		c.getLogSeverityKVPair(v3res, model.GlobalBGPConfigKey{})
	} else if strings.HasPrefix(resName, perNodeConfigNamePrefix) {
		// The name of a configuration resource has a strict format.  It is either "default"
		// for the global default values, or "node.<nodename>" for the node specific vales.
		nodeName := resName[len(perNodeConfigNamePrefix):]
		c.getPrefixAdvertisementsKVPair(v3res, model.NodeBGPConfigKey{Nodename: nodeName})
		c.getListenPortKVPair(v3res, model.NodeBGPConfigKey{Nodename: nodeName})
		c.getLogSeverityKVPair(v3res, model.NodeBGPConfigKey{Nodename: nodeName})
	} else {
		log.Warningf("Bad value for BGPConfiguration resource name: %s.", resName)
	}
}

func getBGPConfigKey(v1KeyName string, key interface{}) model.Key {
	switch k := key.(type) {
	case model.NodeBGPConfigKey:
		k.Name = v1KeyName
		return &k
	case model.GlobalBGPConfigKey:
		k.Name = v1KeyName
		return &k
	default:
		log.Warning("Bad value for BGP Configuration key.")
		return nil
	}
}

// Returns a model.KVPair for the given key and value. If no value is provided, returns model.KVPair with key.
func getKVPair(key model.Key, value ...string) *model.KVPair {
	if len(value) > 0 {
		return &model.KVPair{
			Key:   key,
			Value: value[0],
		}
	}
	return &model.KVPair{
		Key: key,
	}
}

func (c *client) getPrefixAdvertisementsKVPair(v3res *apiv3.BGPConfiguration, key interface{}) {
	ipv4Key := getBGPConfigKey("prefix_advertisements/ip_v4", key)
	ipv6Key := getBGPConfigKey("prefix_advertisements/ip_v6", key)

	if v3res != nil && v3res.Spec.PrefixAdvertisements != nil {
		definedCommunities := v3res.Spec.Communities
		var ipv4PrefixToAdvertise []bgpPrefix
		var ipv6PrefixToAdvertise []bgpPrefix
		for _, prefixAdvertisement := range v3res.Spec.PrefixAdvertisements {
			cidr := prefixAdvertisement.CIDR
			var communityValues []string
			for _, c := range prefixAdvertisement.Communities {
				isCommunity := isValidCommunity(c)
				// if c is a community value, use it directly, else get the community value from defined definedCommunities.
				if !isCommunity {
					for _, definedCommunity := range definedCommunities {
						if definedCommunity.Name == c {
							communityValues = append(communityValues, definedCommunity.Value)
							break
						}
					}
				} else {
					communityValues = append(communityValues, c)
				}
			}
			if strings.Contains(cidr, ":") {
				ipv6PrefixToAdvertise = append(ipv6PrefixToAdvertise, bgpPrefix{
					CIDR:        cidr,
					Communities: communityValues,
				})
			} else {
				ipv4PrefixToAdvertise = append(ipv4PrefixToAdvertise, bgpPrefix{
					CIDR:        cidr,
					Communities: communityValues,
				})
			}
		}

		ipv4Communities, ok := json.Marshal(ipv4PrefixToAdvertise)
		if ok != nil {
			log.Warningf("Error while marshalling BGP communities. %#v", ok)
		}
		c.updateCache(api.UpdateTypeKVUpdated, getKVPair(ipv4Key, string(ipv4Communities)))

		ipv6Communities, ok := json.Marshal(ipv6PrefixToAdvertise)
		if ok != nil {
			log.Warningf("Error while marshalling BGP communities. %#v", ok)
		}
		c.updateCache(api.UpdateTypeKVUpdated, getKVPair(ipv6Key, string(ipv6Communities)))
	} else {
		c.updateCache(api.UpdateTypeKVDeleted, getKVPair(ipv4Key))
		c.updateCache(api.UpdateTypeKVDeleted, getKVPair(ipv6Key))
	}
}

func (c *client) getListenPortKVPair(v3res *apiv3.BGPConfiguration, key interface{}) {
	listenPortKey := getBGPConfigKey("listen_port", key)

	if v3res != nil && v3res.Spec.ListenPort != 0 {
		switch key.(type) {
		case model.NodeBGPConfigKey:
			c.nodeListenPorts[getNodeName(v3res.Name)] = int(v3res.Spec.ListenPort)
		case model.GlobalBGPConfigKey:
			c.globalListenPort = int(v3res.Spec.ListenPort)
		}
		c.updateCache(api.UpdateTypeKVUpdated, getKVPair(listenPortKey, strconv.Itoa(int(v3res.Spec.ListenPort))))
	} else {
		switch k := key.(type) {
		case model.NodeBGPConfigKey:
			delete(c.nodeListenPorts, getNodeName(k.Nodename))
		case model.GlobalBGPConfigKey:
			c.globalListenPort = 0
		}
		c.updateCache(api.UpdateTypeKVDeleted, getKVPair(listenPortKey))
	}
}

func (c *client) getASNumberKVPair(v3res *apiv3.BGPConfiguration, key interface{}, updatePeersV1 *bool, updateReasons *[]string) {
	asNumberKey := getBGPConfigKey("as_num", key)
	if v3res != nil && v3res.Spec.ASNumber != nil {
		*updateReasons = append(*updateReasons, "AS number updated.")
		c.updateCache(api.UpdateTypeKVUpdated, getKVPair(asNumberKey, v3res.Spec.ASNumber.String()))
	} else {
		*updateReasons = append(*updateReasons, "AS number deleted.")
		c.updateCache(api.UpdateTypeKVDeleted, getKVPair(asNumberKey))
	}
	*updatePeersV1 = true
}

func (c *client) getServiceExternalIPsKVPair(v3res *apiv3.BGPConfiguration, key interface{}, svcAdvertisement *bool) {
	scvExternalIPKey := getBGPConfigKey("svc_external_ips", key)

	if v3res != nil && v3res.Spec.ServiceExternalIPs != nil && len(v3res.Spec.ServiceExternalIPs) != 0 {
		// We wrap each Service external IP in a ServiceExternalIPBlock struct to
		// achieve the desired API structure, unpack that.
		ipCidrs := make([]string, len(v3res.Spec.ServiceExternalIPs))
		for i, ipBlock := range v3res.Spec.ServiceExternalIPs {
			ipCidrs[i] = ipBlock.CIDR
		}
		c.updateCache(api.UpdateTypeKVUpdated, getKVPair(scvExternalIPKey, strings.Join(ipCidrs, ",")))
	} else {
		c.updateCache(api.UpdateTypeKVDeleted, getKVPair(scvExternalIPKey))
	}
	*svcAdvertisement = true
}

func (c *client) getServiceClusterIPsKVPair(v3res *apiv3.BGPConfiguration, key interface{}, svcAdvertisement *bool) {
	svcInternalIPKey := getBGPConfigKey("svc_cluster_ips", key)

	if len(os.Getenv(envAdvertiseClusterIPs)) != 0 {
		// ClusterIPs are configurable through an environment variable. If specified,
		// that variable takes precedence over datastore config, so we should ignore the update.
		// Setting Spec.ServiceClusterIPs to nil, so we keep using the cache value set during startup.
		log.Infof("Ignoring serviceClusterIPs update due to environment variable %s", envAdvertiseClusterIPs)
	} else {
		if v3res != nil && v3res.Spec.ServiceClusterIPs != nil && len(v3res.Spec.ServiceClusterIPs) != 0 {
			// We wrap each Service Cluster IP in a ServiceClusterIPBlock to
			// achieve the desired API structure. This unpacks that.
			ipCidrs := make([]string, len(v3res.Spec.ServiceClusterIPs))
			for i, ipBlock := range v3res.Spec.ServiceClusterIPs {
				ipCidrs[i] = ipBlock.CIDR
			}
			c.updateCache(api.UpdateTypeKVUpdated, getKVPair(svcInternalIPKey, strings.Join(ipCidrs, ",")))
		} else {
			c.updateCache(api.UpdateTypeKVDeleted, getKVPair(svcInternalIPKey))
		}
		*svcAdvertisement = true
	}
}

func (c *client) getNodeToNodeMeshKVPair(v3res *apiv3.BGPConfiguration, key interface{}) {
	meshKey := getBGPConfigKey("node_mesh", key)

	if v3res != nil && v3res.Spec.NodeToNodeMeshEnabled != nil {
		enabled := *v3res.Spec.NodeToNodeMeshEnabled
		var val = nodeToNodeMeshEnabled
		if !enabled {
			val = nodeToNodeMeshDisabled
		}
		c.updateCache(api.UpdateTypeKVUpdated, getKVPair(meshKey, val))
	} else {
		c.updateCache(api.UpdateTypeKVDeleted, getKVPair(meshKey))
	}
}

func (c *client) getLogSeverityKVPair(v3res *apiv3.BGPConfiguration, key interface{}) {
	logLevelKey := getBGPConfigKey("loglevel", key)

	if v3res != nil && v3res.Spec.LogSeverityScreen != "" {
		// Bird log level currently only supports granularity of none, debug and info.  Debug/Info are
		// left unchanged, all others treated as none.
		l := strings.ToLower(v3res.Spec.LogSeverityScreen)
		switch l {
		case "debug", "info":
		default:
			l = "none"
		}
		c.updateCache(api.UpdateTypeKVUpdated, getKVPair(logLevelKey, l))
	} else {
		c.updateCache(api.UpdateTypeKVDeleted, getKVPair(logLevelKey))
	}
}

func getNodeName(nodeName string) string {
	return strings.TrimPrefix(nodeName, perNodeConfigNamePrefix)
}

func (c *client) onExternalIPsUpdate(externalIPs []string) {
	if err := c.updateGlobalRoutes(c.externalIPs, externalIPs); err == nil {
		c.externalIPs = externalIPs
		c.externalIPNets = parseIPNets(c.externalIPs)
		log.Infof("Updated with new external IP CIDRs: %s", externalIPs)
	} else {
		log.WithError(err).Error("Failed to update external IP routes")
	}
}

func (c *client) onClusterIPsUpdate(clusterCIDRs []string) {
	if err := c.updateGlobalRoutes(c.clusterCIDRs, clusterCIDRs); err == nil {
		c.clusterCIDRs = clusterCIDRs
		log.Infof("Updated with new cluster IP CIDRs: %s", clusterCIDRs)
	} else {
		log.WithError(err).Error("Failed to update cluster CIDR routes")
	}
}

func (c *client) AdvertiseClusterIPs() bool {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	return len(c.clusterCIDRs) > 0
}

func (c *client) GetExternalIPs() []*net.IPNet {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	return c.externalIPNets
}

// "Global" here means the routes for cluster IP and external IP CIDRs that are advertised from
// every node in the cluster.
func (c *client) updateGlobalRoutes(current, new []string) error {
	for _, n := range new {
		_, _, err := net.ParseCIDR(n)
		if err != nil {
			// Shouldn't ever happen, given prior validation.
			return err
		}
	}

	// Find any currently advertised CIDRs that we should withdraw.
	withdraws := []string{}
	for _, existing := range current {
		if !contains(new, existing) {
			withdraws = append(withdraws, existing)
		}
	}

	// Withdraw the old CIDRs and add the new.
	c.addRoutesLockHeld(rejectKeyPrefix, rejectKeyPrefixV6, new)
	c.addRoutesLockHeld(routeKeyPrefix, routeKeyPrefixV6, new)
	c.deleteRoutesLockHeld(rejectKeyPrefix, rejectKeyPrefixV6, withdraws)
	c.deleteRoutesLockHeld(routeKeyPrefix, routeKeyPrefixV6, withdraws)

	return nil
}

func (c *client) incrementCacheRevision() {
	// If we are in-sync then this is an incremental update, so increment our internal
	// cache revision.
	if c.syncedOnce {
		c.cacheRevision++
		log.Debugf("Processing new updates, revision is now: %d", c.cacheRevision)
	}
}

func (c *client) onNewUpdates() {
	if c.syncedOnce {
		// Wake up the watchers to let them know there may be some updates of interest.  We only
		// need to do this once we're synced because until that point all of the Watcher threads
		// will be blocked getting values.
		log.Debug("Notify watchers of new event data")
		c.watcherCond.Broadcast()
	}
}

func (c *client) recheckPeerConfig() {
	log.Info("Trigger to recheck BGP peers following possible password update")
	select {
	// Non-blocking write into the recheckC channel.  The idea here is that we don't need to add
	// a second trigger if there is already one pending.
	case c.recheckC <- struct{}{}:
	default:
	}
}

// updateChache will update a cache entry. It returns true if the entry was
// updated and false if there was an error or if the cache was already
// up-to-date.
func (c *client) updateCache(updateType api.UpdateType, kvp *model.KVPair) bool {
	// Update our cache of current entries.
	k, err := model.KeyToDefaultPath(kvp.Key)
	if err != nil {
		log.Errorf("Ignoring update: unable to create path from Key %v: %v", kvp.Key, err)
		return false
	}

	switch updateType {
	case api.UpdateTypeKVDeleted:
		// The bird templates that confd is used to render assume that some global
		// defaults are always configured.
		if globalDefault, ok := globalDefaults[k]; ok {
			if currentValue, hasKey := c.cache[k]; hasKey && currentValue == globalDefault {
				return false
			}
			c.cache[k] = globalDefault
		} else {
			if _, hasValue := c.cache[k]; !hasValue {
				return false
			}
			delete(c.cache, k)
		}
	case api.UpdateTypeKVNew, api.UpdateTypeKVUpdated:
		value, err := model.SerializeValue(kvp)
		if err != nil {
			log.Errorf("Ignoring update: unable to serialize value %v: %v", kvp.Value, err)
			return false
		}
		newValue := string(value)
		if currentValue, isSet := c.cache[k]; isSet && currentValue == newValue {
			return false
		}
		c.cache[k] = newValue
	}

	log.Debugf("Cache entry updated from event type %d: %s=%s", updateType, k, c.cache[k])
	if c.syncedOnce {
		c.keyUpdated(k)
	}
	return true
}

func isValidCommunity(communityValue string) bool {
	if standardCommunity.MatchString(communityValue) || largeCommunity.MatchString(communityValue) {
		return true
	}
	return false
}

// ParseFailed is called from the BGP syncer when an event could not be parsed.
// We use this purely for logging.
func (c *client) ParseFailed(rawKey string, rawValue string) {
	log.Errorf("Unable to parse datastore entry Key=%s; Value=%s", rawKey, rawValue)
}

// GetValues is called from confd to obtain the cached data for the required set of prefixes.
// We simply populate the values from our caches, only returning values which have the
// requested set of prefixes.
func (c *client) GetValues(keys []string) (map[string]string, error) {
	// We should block GetValues until we have the sync'd notification - until that point we
	// only have a partial snapshot and we should never write out partial config.
	c.waitForSync.Wait()

	log.Debugf("Requesting values for keys: %v", keys)

	// Lock the data and then populate the results from our caches, selecting the data
	// whose path matches the set of prefix keys.
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	values := map[string]string{}
	for k, v := range c.cache {
		if c.matchesPrefix(k, keys) {
			values[k] = v
		}
	}
	for k, v := range c.peeringCache {
		if c.matchesPrefix(k, keys) {
			values[k] = v
		}
	}

	log.Debugf("Returning %d results", len(values))

	return values, nil
}

// WatchPrefix is called from confd.  It blocks waiting for updates to the data which have any
// of the requested set of prefixes.
//
// Since we keep track of revisions per prefix, all we need to do is check the revisions for an
// update, and if there is no update we wait on the conditional which is woken by the OnUpdates
// thread after updating the cache.  If any of the watched revisions is greater than the waitIndex
// then exit to render.
func (c *client) WatchPrefix(prefix string, keys []string, lastRevision uint64, stopChan chan bool) (string, error) {
	log.WithFields(log.Fields{"prefix": prefix, "keys": keys}).Debug("WatchPrefix entry")
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	if lastRevision == 0 {
		// If this is the first iteration, we always exit to ensure we render with the initial
		// synced settings.
		log.Debug("First watch call for template - exiting to render template")
		return "", nil
	}

	for {
		// Loop through each key, if the revision associated with the key is higher than the lastRevision
		// then exit with the current cacheRevision and render with the current data.
		log.Debugf("Checking for updated key revisions, watching from rev %d", lastRevision)
		for _, key := range keys {
			rev, ok := c.revisionsByPrefix[key]
			if !ok {
				log.Fatalf("Watch prefix check for unknown prefix: %s", key)
			}
			log.Debugf("Found key prefix %s at rev %d", key, rev)
			if rev > lastRevision {
				log.Debug("Exiting to render template")
				return key, nil
			}
		}

		// No changes for this watcher, so wait until there are more syncer events.
		log.Debug("No updated keys for this template - waiting for event notification")
		c.watcherCond.Wait()
		log.WithFields(log.Fields{"prefix": prefix, "keys": keys}).Debug("WatchPrefix recheck")
	}
}

// GetCurrentRevision returns the current revision of the data in our cache.
func (c *client) GetCurrentRevision() uint64 {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	log.Debugf("Current cache revision is %v", c.cacheRevision)
	return c.cacheRevision
}

// matchesPrefix returns true if the key matches any of the supplied prefixes.
func (c *client) matchesPrefix(key string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// Called when a key is updated.  This updates the revision associated with key prefixes
// affected by this key.
// The caller should be holding the cacheLock.
func (c *client) keyUpdated(key string) {
	for prefix, rev := range c.revisionsByPrefix {
		log.Debugf("Prefix %s has rev %d", prefix, rev)
		if rev != c.cacheRevision && strings.HasPrefix(key, prefix) {
			log.Debugf("Updating prefix to rev %d", c.cacheRevision)
			c.revisionsByPrefix[prefix] = c.cacheRevision

			// If this is a change to either the global log level, or the per-node
			// log level, then configure confd's log level to match.
			if strings.HasSuffix(key, "loglevel") {
				log.WithField("key", key).Info("Potential log level configuration change on key")
				c.updateLogLevel()
			}
		}
	}
}

func (c *client) updateLogLevel() {
	if envLevel := os.Getenv("BGP_LOGSEVERITYSCREEN"); envLevel != "" {
		logutils.SetLevel(envLevel)
	} else if nodeLevel := c.cache[c.nodeLogKey]; nodeLevel != "" {
		logutils.SetLevel(nodeLevel)
	} else if globalLogLevel := c.cache[globalLogging]; globalLogLevel != "" {
		logutils.SetLevel(globalLogLevel)
	} else {
		logutils.SetLevel("info")
	}
}

var routeKeyPrefix = "/calico/staticroutes/"
var rejectKeyPrefix = "/calico/rejectcidrs/"
var routeKeyPrefixV6 = "/calico/staticroutesv6/"
var rejectKeyPrefixV6 = "/calico/rejectcidrsv6/"

func (c *client) addRoutesLockHeld(prefixV4, prefixV6 string, cidrs []string) {
	for _, cidr := range cidrs {
		var k string
		if strings.Contains(cidr, ":") {
			k = prefixV6 + strings.Replace(cidr, "/", "-", 1)
		} else {
			k = prefixV4 + strings.Replace(cidr, "/", "-", 1)
		}
		c.cache[k] = cidr
		c.keyUpdated(k)
	}
}

func (c *client) deleteRoutesLockHeld(prefixV4, prefixV6 string, cidrs []string) {
	for _, cidr := range cidrs {
		var k string
		if strings.Contains(cidr, ":") {
			k = prefixV6 + strings.Replace(cidr, "/", "-", 1)
		} else {
			k = prefixV4 + strings.Replace(cidr, "/", "-", 1)
		}
		delete(c.cache, k)
		c.keyUpdated(k)
	}
}

// AddStaticRoutes adds the given CIDRs as static routes to be advertised from this node.
func (c *client) AddStaticRoutes(cidrs []string) {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	c.incrementCacheRevision()
	c.addRoutesLockHeld(routeKeyPrefix, routeKeyPrefixV6, cidrs)
	c.onNewUpdates()
}

// DeleteStaticRoutes withdraws the given CIDRs from the set of static routes advertised
// from this node.
func (c *client) DeleteStaticRoutes(cidrs []string) {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	c.incrementCacheRevision()
	c.deleteRoutesLockHeld(routeKeyPrefix, routeKeyPrefixV6, cidrs)
	c.onNewUpdates()
}

func (c *client) setPeerConfigFieldsFromV3Resource(peers []*bgpPeer, v3res *apiv3.BGPPeer) {

	// Get the password, if one is configured
	password := c.getPassword(v3res)

	// If the BGPPeer has SourceAddress UseNodeIP, a potential direct connection
	// subnet must also contain the node IP, so get those ready.
	var sourceIPv4, sourceIPv6 net.IP
	if v3res.Spec.SourceAddress == apiv3.SourceAddressUseNodeIP || v3res.Spec.SourceAddress == "" {
		if sourceIPv4Str := c.cache[c.nodeIPv4Key]; sourceIPv4Str != "" {
			sourceIPv4 = net.ParseIP(sourceIPv4Str)
			if sourceIPv4 == nil {
				log.Warnf("Failed to parse IPv4 %v", sourceIPv4Str)
			}
		}
		if sourceIPv6Str := c.cache[c.nodeIPv6Key]; sourceIPv6Str != "" {
			sourceIPv6 = net.ParseIP(sourceIPv6Str)
			if sourceIPv6 == nil {
				log.Warnf("Failed to parse IPv6 %v", sourceIPv6Str)
			}
		}
	}

	// To compute which peers are directly connected, first collect all of the subnets
	// associated with local interfaces, and which include the source IP if specified.
	var localSubnets []*net.IPNet
	if ifaces, err := net.Interfaces(); err == nil {
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				log.Warnf("Cannot get interface %v address(es): %v", iface, err)
				continue
			}
			for _, addr := range addrs {
				addrStr := addr.String()
				_, ipNet, err := net.ParseCIDR(addrStr)
				if err != nil {
					log.WithError(err).WithField("Address", addrStr).Warning("Failed to parse CIDR")
					continue
				}
				if sourceIPv4 != nil && ipNet.IP.To4() != nil && !(*ipNet).Contains(sourceIPv4) {
					// IPv4 subnet does not contain the wanted source IP.
					continue
				}
				if sourceIPv6 != nil && ipNet.IP.To16() != nil && !(*ipNet).Contains(sourceIPv6) {
					// IPv6 subnet does not contain the wanted source IP.
					continue
				}
				localSubnets = append(localSubnets, ipNet)
			}
		}
	} else {
		log.WithError(err).Warnf("Failed to enumerate interfaces")
	}
	log.Infof("Local subnets for IPv4 %v and IPv6 %v are: %v", sourceIPv4, sourceIPv6, localSubnets)

	for _, peer := range peers {
		peer.Password = password
		peer.Extensions = v3res.Spec.Extensions
		peer.SourceAddr = withDefault(string(v3res.Spec.SourceAddress), string(apiv3.SourceAddressUseNodeIP))
		peer.RestartMode = withDefault(string(v3res.Spec.RestartMode), string(apiv3.RestartModeGracefulRestart))
		if v3res.Spec.MaxRestartTime != nil {
			peer.RestartTime = fmt.Sprintf("%v", int(math.Round(v3res.Spec.MaxRestartTime.Duration.Seconds())))
		}
		for _, subnet := range localSubnets {
			if subnet.Contains(peer.PeerIP.IP) {
				log.Infof("Local subnet %v contains peer IP %v", subnet, peer.PeerIP)
				peer.DirectlyConnected = true
				break
			}
		}
		if v3res.Spec.BIRDGatewayMode == apiv3.BIRDGatewayModeDirectIfDirectlyConnected && peer.DirectlyConnected {
			peer.GatewayMode = "direct"
		} else {
			peer.GatewayMode = "recursive"
		}
		if v3res.Spec.FailureDetectionMode == apiv3.FailureDetectionModeBFDIfDirectlyConnected && peer.DirectlyConnected {
			peer.EnableBFD = true
		}
	}
}

func withDefault(val, dflt string) string {
	if val != "" {
		return val
	}
	return dflt
}
