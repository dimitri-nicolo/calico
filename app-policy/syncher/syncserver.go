// Copyright (c) 2018-2024 Tigera, Inc. All rights reserved.

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
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	healthzv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/projectcalico/calico/app-policy/health"
	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/statscache"
	"github.com/projectcalico/calico/felix/proto"
)

const (
	// The stats reporting and flush interval. Currently set to half the hardcoded expiration time of cache entries in
	// the Felix stats collector component.
	DefaultSubscriptionType   = "per-pod-policies"
	DefaultStatsFlushInterval = 5 * time.Second
	PolicySyncRetryTime       = 1000 * time.Millisecond
)

type SyncClient struct {
	target           string
	dialOpts         []grpc.DialOption
	subscriptionType string
	inSync           bool
	storeManager     policystore.PolicyStoreManager
	stats            chan map[statscache.Tuple]statscache.Values
}

type ClientOptions func(*SyncClient)

func WithSubscriptionType(subscriptionType string) ClientOptions {
	return func(s *SyncClient) {
		switch subscriptionType {
		case "":
			s.subscriptionType = "per-pod-policies"
		case "per-pod-policies", "per-host-policies":
			s.subscriptionType = subscriptionType
		default:
			log.Panicf("invalid subscription type: '%s'", subscriptionType)
		}
	}
}

// NewClient creates a new syncClient.
func NewClient(target string, policyStoreManager policystore.PolicyStoreManager, dialOpts []grpc.DialOption, clientOpts ...ClientOptions) *SyncClient {
	syncClient := &SyncClient{
		target: target, dialOpts: dialOpts,
		storeManager:     policyStoreManager,
		stats:            make(chan map[statscache.Tuple]statscache.Values),
		subscriptionType: DefaultSubscriptionType,
	}
	for _, opt := range clientOpts {
		opt(syncClient)
	}
	return syncClient
}

func (s *SyncClient) syncRequest() *proto.SyncRequest {
	return &proto.SyncRequest{
		SupportsDropActionOverride: true,
		SupportsDataplaneStats:     true,
		SubscriptionType:           s.subscriptionType,
	}
}

func (s *SyncClient) RegisterGRPCServices(gs *grpc.Server) {
	healthzv1.RegisterHealthServer(gs, health.NewHealthCheckService(s))
}

func (s *SyncClient) Start(ctx context.Context) {
	for {
		if err := s.connectAndSync(ctx); err != nil {
			log.Error("connectAndSync error: ", err)
		}
		time.Sleep(PolicySyncRetryTime)
	}
}

func (s *SyncClient) OnStatsCacheFlush(v map[statscache.Tuple]statscache.Values) {
	// Only send stats if we are in sync.
	if !s.inSync {
		return
	}
	s.stats <- v
}

func (s *SyncClient) connectAndSync(ptx context.Context, cb ...statscache.StatsCacheFlushCallback) error {
	ctx, cancel := context.WithCancel(ptx)
	defer cancel()

	s.inSync = false
	s.storeManager.OnReconnecting()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("connection to PolicySync stopped: %w", err)
	}

	cc, err := grpc.NewClient(s.target, s.dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect to PolicySync server %s. %w", s.target, err)
	}
	defer cc.Close()

	client := proto.NewPolicySyncClient(cc)
	stream, err := client.Sync(ctx, s.syncRequest())
	if err != nil {
		return fmt.Errorf("failed to stream from PolicySync server: %w", err)
	}
	go s.sendStats(ctx, client)

	return s.sync(stream, ctx)
}

func (s *SyncClient) sync(stream proto.PolicySync_SyncClient, ctx context.Context) error {
	for {
		update, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("exceeded max stream retries. retrying grpc connection %w", err)
		}
		switch update.Payload.(type) {
		case *proto.ToDataplane_InSync:
			s.inSync = true
			s.storeManager.OnInSync()
		default:
			s.storeManager.Write(func(ps *policystore.PolicyStore) {
				ps.ProcessUpdate(s.subscriptionType, update, false)
			})
		}
	}
}

// Readiness returns whether the SyncClient is InSync.
func (s *SyncClient) Readiness() (ready bool) {

	return s.inSync
}

// sendStats is the main stats reporting loop.
func (s *SyncClient) sendStats(ctx context.Context, client proto.PolicySyncClient) {
	log.Info("Starting sending DataplaneStats to Policy Sync server")
	for {
		select {
		case a := <-s.stats:
			for t, v := range a {
				if err := s.report(ctx, client, t, v); err != nil {
					// Error reporting stats, exit now to start reconnction processing.
					log.WithError(err).Warning("Error reporting stats")
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// report converts the statscache formatted stats and reports it as a proto.DataplaneStats to Felix.
func (s *SyncClient) report(ctx context.Context, client proto.PolicySyncClient, t statscache.Tuple, v statscache.Values) error {
	log.Debugf("Reporting statistic to Felix: %s=%s", t, v)

	d := &proto.DataplaneStats{
		SrcIp:    t.SrcIp,
		DstIp:    t.DstIp,
		SrcPort:  t.SrcPort,
		DstPort:  t.DstPort,
		Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: t.Protocol}},
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
	if r, err := client.Report(ctx, d); err != nil {
		// Error sending stats, must be a connection issue, so exit now to force a reconnect.
		return err
	} else if !r.Successful {
		// If the remote end indicates unsuccessful then the remote end is likely transitioning from having
		// stats enabled to having stats disabled. This should be transient, so log a warning, but otherwise
		// treat as a successful report.
		log.Warning("Remote end indicates dataplane statistics not processed successfully")
		return nil
	}
	return nil
}
