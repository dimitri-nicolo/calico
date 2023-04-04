// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package server

import (
	"context"
	"net"
	"net/url"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/projectcalico/calico/app-policy/checker"
	"github.com/projectcalico/calico/app-policy/policystore"
	dikastesproto "github.com/projectcalico/calico/app-policy/proto"
	"github.com/projectcalico/calico/app-policy/statscache"
	"github.com/projectcalico/calico/app-policy/syncher"
	"github.com/projectcalico/calico/app-policy/waf"
	"github.com/projectcalico/calico/libcalico-go/lib/uds"
)

const (
	maxPendingDataplaneStats = 100
)

var (
	_ dikastesproto.HealthzServer
)

type Dikastes struct {
	subscriptionType             string
	dialNetwork, dialAddress     string
	listenNetwork, listenAddress string
	grpcServerOptions            []grpc.ServerOption

	Ready chan struct{}
}

type DikastesServerOptions func(*Dikastes)

func WithDialAddress(network, addr string) DikastesServerOptions {
	return func(ds *Dikastes) {
		ds.dialNetwork = network
		ds.dialAddress = addr
	}
}

func WithListenArguments(network, address string) DikastesServerOptions {
	return func(ds *Dikastes) {
		ds.listenNetwork = network
		ds.listenAddress = address
	}
}

func WithSubscriptionType(s string) DikastesServerOptions {
	return func(ds *Dikastes) {
		ds.subscriptionType = s
	}
}

func WithGRPCServerOpts(opts ...grpc.ServerOption) DikastesServerOptions {
	return func(ds *Dikastes) {
		ds.grpcServerOptions = append(ds.grpcServerOptions, opts...)
	}
}

func NewDikastesServer(opts ...DikastesServerOptions) *Dikastes {
	s := &Dikastes{Ready: make(chan struct{})}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Dikastes) Serve(ctx context.Context) {
	lis, err := net.Listen(s.listenNetwork, s.listenAddress)
	if err != nil {
		log.Fatal("could not start listener: ", err)
		return
	}
	defer lis.Close()

	switch s.listenNetwork {
	case "unix":
		err = os.Chmod(s.listenAddress, 0777) // Anyone on system can connect.
		if err != nil {
			log.Fatal("Unable to set write permission on socket.")
		}
		defer func() {
			log.Error(os.Remove(s.listenAddress))
		}()
	}

	log.Infof("Dikastes listening at %s", lis.Addr())
	s.listenAddress = lis.Addr().String()

	gs := grpc.NewServer(s.grpcServerOptions...)

	dpStats := make(chan statscache.DPStats, maxPendingDataplaneStats)
	policyStoreManager := policystore.NewPolicyStoreManager()

	checkServerOptions := []checker.AuthServerOption{
		checker.WithSubscriptionType(s.subscriptionType),
		// register alp check provider. registrations are ordered (first-registered-processed-first)
		checker.WithRegisteredCheckProvider(checker.NewALPCheckProvider(s.subscriptionType)),
	}

	// waf checks are expensive, do it after alp
	if waf.IsEnabled() {
		checkServerOptions = append(checkServerOptions, checker.WithRegisteredCheckProvider(checker.NewWAFCheckProvider(s.subscriptionType)))
	}

	// checkServer provides envoy v3, v2, v2 alpha ext authz services
	checkServer := checker.NewServer(
		ctx, policyStoreManager, dpStats,
		checkServerOptions...,
	)
	checkServer.RegisterGRPCServices(gs)

	// syncClient provides synchronization with the policy store and start reporting stats.
	opts := uds.GetDialOptionsWithNetwork(s.dialNetwork)
	syncClient := syncher.NewClient(
		s.dialAddress,
		policyStoreManager,
		opts,
		syncher.ClientOptions{
			StatsFlushInterval: time.Second * 5,
			SubscriptionType:   s.subscriptionType,
		},
	)
	syncClient.RegisterGRPCServices(gs)
	go syncClient.Start(ctx, dpStats)

	if _, ok := os.LookupEnv("DIKASTES_ENABLE_CHECKER_REFLECTION"); ok {
		reflection.Register(gs)
	}
	// Run gRPC server on separate goroutine so we catch any signals and clean up.
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	close(s.Ready)
	<-ctx.Done()

	gs.GracefulStop()
}

func (s *Dikastes) Addr() string {
	u := url.URL{Scheme: s.listenNetwork}
	if s.listenNetwork == "unix" {
		u.Path = s.listenAddress
		return u.String()
	}
	u.Host = s.listenAddress
	return u.String()
}
