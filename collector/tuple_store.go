// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/calc"
)

type tupleKey struct {
	tuple    Tuple
	action   FlowLogAction
	reporter FlowLogReporter
}

type tupleData struct {
	// Original Source IPs
	origSourceIPs *boundedSet

	// Source/Destination endpoint
	srcEp EndpointMetadata
	dstEp EndpointMetadata

	// Labels extracted from Source/Destination endpoint
	srcLabels map[string]string
	dstLabels map[string]string

	// Rules identification
	ruleIDs []*calc.RuleID

	// Inbound/Outbound packet/byte counts.
	inMetric  MetricValue
	outMetric MetricValue

	// Started and Completed connection during a flush interval
	startedConnections   int
	completedConnections int

	// Mark whether a connection was expired or not
	hasExpired bool
}

// MarkExpired will mark a connection completed. This is called once a UpdateType.Expire is received
func (data *tupleData) MarkExpired() {
	data.hasExpired = true
	data.completedConnections++
}

// MarkExpired will mark a connection started.
func (data *tupleData) MarkStarted() {
	data.startedConnections++
	data.hasExpired = false
}

// Reset will reset in/out metric values, started/completed connections and original source ips
func (data *tupleData) Reset() {
	// resetting statistics
	data.outMetric.Reset()
	data.inMetric.Reset()

	// resetting started/completed connections
	data.startedConnections = 0
	data.completedConnections = 0
	data.hasExpired = false

	// resetting original IPs
	if data.origSourceIPs != nil {
		data.origSourceIPs.Reset()
	}
}

type tupleStore struct {
	stats map[tupleKey]*tupleData
	mutex sync.RWMutex
}

// NewTupleStore returns a tupleStore. This is synchronized k-v map that will store the traffic identification
// against the variable data retrieved from a MetricUpdate. They key is formed from the 5-data Tuple (source ip,
// source port, destination ip, destination port, type) and traffic type (allow/denied and src/dst). The variable data
// is comprised of MetricValue statistics recorded for both in and out traffic, original source IPs, and source and
// destination labels. It also stores source and destination EndpointMetadata and keeps track when a connection was
// started and completed
func NewTupleStore() *tupleStore {
	return &tupleStore{make(map[tupleKey]*tupleData), sync.RWMutex{}}
}

// Store will either add a new entry or update the previous with the metrics,
// labels and original source ips from a metric update
func (store *tupleStore) Store(mu MetricUpdate) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	var action, reporter = getFlowLogActionAndReporterFromRuleID(mu.GetLastRuleID())
	var key = tupleKey{mu.tuple, action, reporter}

	value, found := store.stats[key]
	if !found {
		value = convert(mu)
	} else {
		// increment outMetric statistics
		value.outMetric.Increment(mu.outMetric)

		// increment inMetric statistics
		value.inMetric.Increment(mu.inMetric)

		// The pointers will be same for a conntrak connection
		value.srcLabels = getFlowLogEndpointLabels(mu.srcEp)
		value.dstLabels = getFlowLogEndpointLabels(mu.dstEp)
		value.ruleIDs = mu.ruleIDs
	}

	// an expire update will increment the completed connections
	// data will ke kept around until a Reset() is explicitly called
	if mu.updateType == UpdateTypeExpire {
		value.MarkExpired()
	} else if !found {
		// marking the start of a new connection
		value.MarkStarted()
	}

	// combine original source IPs
	if value.origSourceIPs != nil {
		value.origSourceIPs.Combine(mu.origSourceIPs)
	}

	store.stats[key] = value
}

func convert(mu MetricUpdate) *tupleData {
	// Extract EndpointMetadata info
	srcMeta, err := getFlowLogEndpointMetadata(mu.srcEp, mu.tuple.src)
	if err != nil {
		log.Errorf("Could not extract metadata for source %v", mu.srcEp)
	}
	dstMeta, err := getFlowLogEndpointMetadata(mu.dstEp, mu.tuple.dst)
	if err != nil {
		log.Errorf("Could not extract metadata for source %v", mu.srcEp)
	}

	return &tupleData{origSourceIPs: copyOrigSourceIps(mu.origSourceIPs),
		srcEp:     srcMeta,
		dstEp:     dstMeta,
		srcLabels: getFlowLogEndpointLabels(mu.srcEp),
		dstLabels: getFlowLogEndpointLabels(mu.dstEp),
		ruleIDs:   mu.ruleIDs,
		inMetric:  mu.inMetric,
		outMetric: mu.outMetric,
	}
}

// IterAndReset will iterate over the k-v pairs stored and will reset all MetricValue stats for in/out traffic. Also,
// it will reset original source IPs, started connections and completed connections. Any entry that has been marked
// as expired will be removed. It will invoke the pass in callback for each k-v pair before calling Reset
func (store *tupleStore) IterAndReset(cb func(key tupleKey, value tupleData)) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for k, v := range store.stats {
		cb(k, tupleData{origSourceIPs: copyOrigSourceIps(v.origSourceIPs),
			srcEp:                v.srcEp,
			dstEp:                v.dstEp,
			srcLabels:            copyLabels(v.srcLabels),
			dstLabels:            copyLabels(v.dstLabels),
			ruleIDs:              v.ruleIDs,
			inMetric:             v.inMetric,
			outMetric:            v.outMetric,
			startedConnections:   v.startedConnections,
			completedConnections: v.completedConnections,
			hasExpired:           v.hasExpired,
		})

		//remove expired connections
		if v.hasExpired {
			delete(store.stats, k)
			continue
		}

		v.Reset()

		store.stats[k] = v
	}
}

func copyOrigSourceIps(origSourceIPs *boundedSet) *boundedSet {
	if origSourceIPs == nil {
		return origSourceIPs
	}
	return origSourceIPs.Copy()
}

func copyLabels(labels map[string]string) map[string]string {
	var copy = map[string]string{}
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}
