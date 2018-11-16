// Copyright (c) 2018 Tigera, Inc. All rights reserved.

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

package syncher

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/projectcalico/app-policy/policystore"
	"github.com/projectcalico/app-policy/proto"
	"github.com/projectcalico/app-policy/statscache"
)

const (
	DefaultStatsFlushInterval = 1 * time.Minute
	PolicySyncRetryTime       = 500 * time.Millisecond
	MaxStatsCaches            = 5
)

type syncClient struct {
	target     string
	dialOpts   []grpc.DialOption
	stats      statscache.Interface
	unreported []map[statscache.Tuple]statscache.Values
}

type SyncClient interface {
	// Start connects to the Policy Sync API server, processes policy updates from it and sends DataplaneStats to it.
	// It collates and maintains policy updates in a PolicyStore which it sends over the channel when they are ready
	// for enforcement.  Each time we disconnect and resync with the Policy Sync API a new PolicyStore is created
	// and the previous one should be discarded.
	Start(ctx context.Context, stores chan<- *policystore.PolicyStore, dpStats <-chan statscache.DPStats)
}

type ClientOptions struct {
	StatsFlushInterval time.Duration
}

// NewClient creates a new syncClient.
func NewClient(target string, dialOpts []grpc.DialOption, clientOpts ClientOptions) SyncClient {
	statsFlushInterval := DefaultStatsFlushInterval
	if clientOpts.StatsFlushInterval != 0 {
		statsFlushInterval = clientOpts.StatsFlushInterval
	}
	return &syncClient{target: target, dialOpts: dialOpts, stats: statscache.New(statsFlushInterval)}
}

func (s *syncClient) Start(cxt context.Context, stores chan<- *policystore.PolicyStore, dpStats <-chan statscache.DPStats) {
	aggregatedStats := make(chan map[statscache.Tuple]statscache.Values, MaxStatsCaches)
	s.stats.Start(cxt, dpStats, aggregatedStats)
	for {
		select {
		case <-cxt.Done():
			return
		case a := <-aggregatedStats:
			// Aggregated stats are available. Queue them, but don't report them since we aren't connected.
			s.queueAggregatedStats(a)
		default:
			store := policystore.NewPolicyStore()
			inSync := make(chan struct{})
			done := make(chan struct{})
			go s.connect(cxt, store, inSync, aggregatedStats, done)

			// Block until we receive InSync message, or cancelled.
			select {
			case <-inSync:
				stores <- store
			// Also catch the case where syncStore ends before it gets an InSync message.
			case <-done:
				// pass
			case <-cxt.Done():
				return
			}

			// Block until syncStore() ends (e.g. disconnected), or cancelled.
			select {
			case <-done:
				// pass
			case <-cxt.Done():
				return
			}

			time.Sleep(PolicySyncRetryTime)
		}
	}
}

func (s *syncClient) connect(cxt context.Context,
	store *policystore.PolicyStore, inSync chan<- struct{},
	aggregatedStats <-chan map[statscache.Tuple]statscache.Values, done chan<- struct{},
) {
	defer close(done)
	conn, err := grpc.Dial(s.target, s.dialOpts...)
	if err != nil {
		log.Warnf("fail to dial Policy Sync server: %v", err)
		return
	}
	log.Info("Successfully connected to Policy Sync server")
	defer conn.Close()
	client := proto.NewPolicySyncClient(conn)

	cxt, cancel := context.WithCancel(cxt)
	wg := sync.WaitGroup{}

	// Start the store sync go routine.
	wg.Add(1)
	go func() {
		s.syncStore(cxt, client, store, inSync)
		cancel()
		wg.Done()
	}()

	// Start the DataplaneStats reporting go routine.
	wg.Add(1)
	go func() {
		s.sendStats(cxt, client, aggregatedStats)
		cancel()
		wg.Done()
	}()

	// Wait for both go routines to complete before exiting, since we don't want to close the connection
	// whilst it may be being accessed.
	wg.Wait()
}

func (s *syncClient) syncStore(cxt context.Context, client proto.PolicySyncClient, store *policystore.PolicyStore, inSync chan<- struct{}) {
	// Send a sync request indicating which features we support.
	stream, err := client.Sync(cxt, &proto.SyncRequest{
		SupportsDropActionOverride: true,
		SupportsDataplaneStats:     true,
	})
	if err != nil {
		log.Warnf("failed to synchronize with Policy Sync server: %v", err)
		return
	}

	log.Info("Starting synchronization with Policy Sync server")
	for {
		update, err := stream.Recv()
		if err != nil {
			log.Warnf("connection to Policy Sync server broken: %v", err)
			return
		}
		log.WithFields(log.Fields{"proto": update}).Debug("Received sync API Update")
		store.Write(func(ps *policystore.PolicyStore) { processUpdate(ps, inSync, update) })
	}
}

// sendStats is the main stats reporting loop.
func (s *syncClient) sendStats(
	cxt context.Context, client proto.PolicySyncClient,
	aggregatedStats <-chan map[statscache.Tuple]statscache.Values,
) {
	log.Info("Starting sending DataplaneStats to Policy Sync server")

	// If there are some statistics that were queued while we were connecting, then send them now.
	if err := s.maybeReportStats(cxt, client); err != nil {
		// Error reporting stats, exit now to start reconnction processing.
		log.WithError(err).Info("Error reporting stats")
		return
	}

	for {
		select {
		case a := <-aggregatedStats:
			// Aggregated stats are available, queue them and report immediately.
			s.queueAggregatedStats(a)
			if err := s.maybeReportStats(cxt, client); err != nil {
				// Error reporting stats, exit now to start reconnction processing.
				log.WithError(err).Info("Error reporting stats")
				return
			}
		case <-cxt.Done():
			return
		}
	}
}

// queueAggregatedStats appends the supplied set of aggregated stats to the backlog of unreported stats sets.
// Only a certain number of stats sets are maintained, which means we hold on to a minumum period of data until
// it is reported. The stored data may refer to a longer time period than the minimum since we don't get any
// data sets for reporting periods that generate no data.
func (s *syncClient) queueAggregatedStats(aggregatedStats map[statscache.Tuple]statscache.Values) {
	s.unreported = append(s.unreported, aggregatedStats)
	if len(s.unreported) > MaxStatsCaches {
		log.Warning("Dropping old unreported statistics")
		s.unreported = s.unreported[len(s.unreported)-MaxStatsCaches:]
	}
}

// maybeReportStats reports any queued sets of statistics to Felix. Any error hit when reporting the statistics
// will result in reconnection processing.
func (s *syncClient) maybeReportStats(cxt context.Context, client proto.PolicySyncClient) error {
	for len(s.unreported) > 0 {
		// Iterate through the stats in the oldest set, reporting each stat and then removing the stat from the
		// set. If we hit an error, exit - we'll continue reporting this stats set when connection is reestablished
		// (unless the stats set is aged-out by newer sets).
		log.Info("Reporting aggregated statistics")
		for t, v := range s.unreported[0] {
			if err := s.report(cxt, client, t, v); err != nil {
				return err
			}
			delete(s.unreported[0], t)
		}

		s.unreported = s.unreported[1:]
	}
	return nil
}

// report converts the statscache formatted stats and reports it as a proto.DataplaneStats to Felix.
func (s *syncClient) report(cxt context.Context, client proto.PolicySyncClient, t statscache.Tuple, v statscache.Values) error {
	d := &proto.DataplaneStats{
		SrcIp:    t.SrcIp,
		DstIp:    t.DstIp,
		SrcPort:  t.SrcPort,
		DstPort:  t.DstPort,
		Protocol: &proto.Protocol{&proto.Protocol_Name{t.Protocol}},
	}
	if v.HTTPRequestsAllowed > 0 {
		d.Stats = append(d.Stats, &proto.Statistic{
			Direction:  proto.Statistic_IN,
			Relativity: proto.Statistic_DELTA,
			Kind:       proto.Statistic_HTTP_REQUESTS,
			Action:     proto.Action_ALLOWED,
			Value:      v.HTTPRequestsAllowed,
		})
	}
	if v.HTTPRequestsDenied > 0 {
		d.Stats = append(d.Stats, &proto.Statistic{
			Direction:  proto.Statistic_IN,
			Relativity: proto.Statistic_DELTA,
			Kind:       proto.Statistic_HTTP_REQUESTS,
			Action:     proto.Action_DENIED,
			Value:      v.HTTPRequestsDenied,
		})
	}

	if r, err := client.Report(cxt, d); err != nil {
		// Error sending stats, must be a connection issue, so exit now to force a reconnect.
		log.Warnf("Error sending DataplaneStats: %v", err)
		return err
	} else if !r.Successful {
		// If the remote end indicates unsuccessful then the remote end is likely transitioning from having
		// stats enabled to having stats disabled. This should be transient, so log a warning, but otherwise
		// treat as a successful report.
		log.Warning("Remote end indicates DataplaneStats not processed successfully")
		return nil
	}
	return nil
}

// Update the PolicyStore with the information passed over the Sync API.
func processUpdate(store *policystore.PolicyStore, inSync chan<- struct{}, update *proto.ToDataplane) {
	switch payload := update.Payload.(type) {
	case *proto.ToDataplane_InSync:
		close(inSync)
		processInSync(store, payload.InSync)
	case *proto.ToDataplane_IpsetUpdate:
		processIPSetUpdate(store, payload.IpsetUpdate)
	case *proto.ToDataplane_IpsetDeltaUpdate:
		processIPSetDeltaUpdate(store, payload.IpsetDeltaUpdate)
	case *proto.ToDataplane_IpsetRemove:
		processIPSetRemove(store, payload.IpsetRemove)
	case *proto.ToDataplane_ActiveProfileUpdate:
		processActiveProfileUpdate(store, payload.ActiveProfileUpdate)
	case *proto.ToDataplane_ActiveProfileRemove:
		processActiveProfileRemove(store, payload.ActiveProfileRemove)
	case *proto.ToDataplane_ActivePolicyUpdate:
		processActivePolicyUpdate(store, payload.ActivePolicyUpdate)
	case *proto.ToDataplane_ActivePolicyRemove:
		processActivePolicyRemove(store, payload.ActivePolicyRemove)
	case *proto.ToDataplane_WorkloadEndpointUpdate:
		processWorkloadEndpointUpdate(store, payload.WorkloadEndpointUpdate)
	case *proto.ToDataplane_WorkloadEndpointRemove:
		processWorkloadEndpointRemove(store, payload.WorkloadEndpointRemove)
	case *proto.ToDataplane_ServiceAccountUpdate:
		processServiceAccountUpdate(store, payload.ServiceAccountUpdate)
	case *proto.ToDataplane_ServiceAccountRemove:
		processServiceAccountRemove(store, payload.ServiceAccountRemove)
	case *proto.ToDataplane_NamespaceUpdate:
		processNamespaceUpdate(store, payload.NamespaceUpdate)
	case *proto.ToDataplane_NamespaceRemove:
		processNamespaceRemove(store, payload.NamespaceRemove)
	case *proto.ToDataplane_ConfigUpdate:
		processConfigUpdate(store, payload.ConfigUpdate)
	default:
		panic(fmt.Sprintf("unknown payload %v", update.String()))
	}
}

func processInSync(store *policystore.PolicyStore, inSync *proto.InSync) {
	log.Debug("Processing InSync")
	return
}

func processConfigUpdate(store *policystore.PolicyStore, update *proto.ConfigUpdate) {
	log.WithFields(log.Fields{
		"config": update.Config,
	}).Info("Processing ConfigUpdate")

	// Update the DropActionOverride setting if it is available.
	if val, ok := update.Config["DropActionOverride"]; ok {
		log.Debug("DropActionOverride is present in config")
		var psVal policystore.DropActionOverride
		switch strings.ToLower(val) {
		case "drop":
			psVal = policystore.DROP
		case "accept":
			psVal = policystore.ACCEPT
		case "loganddrop":
			psVal = policystore.LOG_AND_DROP
		case "logandaccept":
			psVal = policystore.LOG_AND_ACCEPT
		default:
			log.Errorf("Unknown DropActionOverride value: %s", val)
			psVal = policystore.DROP
		}
		store.DropActionOverride = psVal
	}

	// Extract the flow logs settings, defaulting to false if not present.
	store.DataplaneStatsEnabledForAllowed = getBoolFromConfig(update.Config, "DataplaneStatsEnabledForAllowed", false)
	store.DataplaneStatsEnabledForDenied = getBoolFromConfig(update.Config, "DataplaneStatsEnabledForDenied", false)
}

func getBoolFromConfig(m map[string]string, name string, def bool) bool {
	b := def
	if v, ok := m[name]; ok {
		log.Debugf("%s is present in config", name)
		if p, err := strconv.ParseBool(v); err != nil {
			log.Debugf("Parsed value from Felix config: %s=%v", name, p)
			b = p
		}
	}
	return b
}

func processIPSetUpdate(store *policystore.PolicyStore, update *proto.IPSetUpdate) {
	log.WithFields(log.Fields{
		"id": update.Id,
	}).Debug("Processing IPSetUpdate")

	// IPSetUpdate replaces the existing set.
	s := policystore.NewIPSet(update.Type)
	for _, addr := range update.Members {
		s.AddString(addr)
	}
	store.IPSetByID[update.Id] = s
}

func processIPSetDeltaUpdate(store *policystore.PolicyStore, update *proto.IPSetDeltaUpdate) {
	log.WithFields(log.Fields{
		"id": update.Id,
	}).Debug("Processing IPSetDeltaUpdate")
	s := store.IPSetByID[update.Id]
	if s == nil {
		log.Errorf("Unknown IPSet id: %v", update.Id)
		panic("unknown IPSet id")
	}
	for _, addr := range update.AddedMembers {
		s.AddString(addr)
	}
	for _, addr := range update.RemovedMembers {
		s.RemoveString(addr)
	}
}

func processIPSetRemove(store *policystore.PolicyStore, update *proto.IPSetRemove) {
	log.WithFields(log.Fields{
		"id": update.Id,
	}).Debug("Processing IPSetRemove")
	delete(store.IPSetByID, update.Id)
}

func processActiveProfileUpdate(store *policystore.PolicyStore, update *proto.ActiveProfileUpdate) {
	log.WithFields(log.Fields{
		"id": update.Id,
	}).Debug("Processing ActiveProfileUpdate")
	if update.Id == nil {
		panic("got ActiveProfileUpdate with nil ProfileID")
	}
	store.ProfileByID[*update.Id] = update.Profile
}

func processActiveProfileRemove(store *policystore.PolicyStore, update *proto.ActiveProfileRemove) {
	log.WithFields(log.Fields{
		"id": update.Id,
	}).Debug("Processing ActiveProfileRemove")
	if update.Id == nil {
		panic("got ActiveProfileRemove with nil ProfileID")
	}
	delete(store.ProfileByID, *update.Id)
}

func processActivePolicyUpdate(store *policystore.PolicyStore, update *proto.ActivePolicyUpdate) {
	log.WithFields(log.Fields{
		"id": update.Id,
	}).Debug("Processing ActivePolicyUpdate")
	if update.Id == nil {
		panic("got ActivePolicyUpdate with nil PolicyID")
	}
	store.PolicyByID[*update.Id] = update.Policy
}

func processActivePolicyRemove(store *policystore.PolicyStore, update *proto.ActivePolicyRemove) {
	log.WithFields(log.Fields{
		"id": update.Id,
	}).Debug("Processing ActivePolicyRemove")
	if update.Id == nil {
		panic("got ActivePolicyRemove with nil PolicyID")
	}
	delete(store.PolicyByID, *update.Id)
}

func processWorkloadEndpointUpdate(store *policystore.PolicyStore, update *proto.WorkloadEndpointUpdate) {
	// TODO: check the WorkloadEndpointID?
	log.WithFields(log.Fields{
		"orchestratorID": update.GetId().GetOrchestratorId(),
		"workloadID":     update.GetId().GetWorkloadId(),
		"endpointID":     update.GetId().GetEndpointId(),
	}).Info("Processing WorkloadEndpointUpdate")
	store.Endpoint = update.Endpoint
}

func processWorkloadEndpointRemove(store *policystore.PolicyStore, update *proto.WorkloadEndpointRemove) {
	// TODO: maybe this isn't required, because removing the endpoint means shutting down the pod?
	log.WithFields(log.Fields{
		"orchestratorID": update.GetId().GetOrchestratorId(),
		"workloadID":     update.GetId().GetWorkloadId(),
		"endpointID":     update.GetId().GetEndpointId(),
	}).Warning("Processing WorkloadEndpointRemove")
	store.Endpoint = nil
}

func processServiceAccountUpdate(store *policystore.PolicyStore, update *proto.ServiceAccountUpdate) {
	log.WithField("id", update.Id).Debug("Processing ServiceAccountUpdate")
	if update.Id == nil {
		panic("got ServiceAccountUpdate with nil ServiceAccountID")
	}
	store.ServiceAccountByID[*update.Id] = update
}

func processServiceAccountRemove(store *policystore.PolicyStore, update *proto.ServiceAccountRemove) {
	log.WithField("id", update.Id).Debug("Processing ServiceAccountRemove")
	if update.Id == nil {
		panic("got ServiceAccountRemove with nil ServiceAccountID")
	}
	delete(store.ServiceAccountByID, *update.Id)
}

func processNamespaceUpdate(store *policystore.PolicyStore, update *proto.NamespaceUpdate) {
	log.WithField("id", update.Id).Debug("Processing NamespaceUpdate")
	if update.Id == nil {
		panic("got NamespaceUpdate with nil NamespaceID")
	}
	store.NamespaceByID[*update.Id] = update
}

func processNamespaceRemove(store *policystore.PolicyStore, update *proto.NamespaceRemove) {
	log.WithField("id", update.Id).Debug("Processing NamespaceRemove")
	if update.Id == nil {
		panic("got NamespaceRemove with nil NamespaceID")
	}
	delete(store.NamespaceByID, *update.Id)
}
