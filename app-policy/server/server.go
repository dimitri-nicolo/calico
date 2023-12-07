// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package server

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/projectcalico/calico/app-policy/checker"
	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/statscache"
	"github.com/projectcalico/calico/app-policy/syncher"
	"github.com/projectcalico/calico/app-policy/waf"
	"github.com/projectcalico/calico/libcalico-go/lib/uds"
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

func ensureSocketFileNone(filePath string) error {
	_, err := os.Stat(filePath)
	if !os.IsNotExist(err) {
		// file exists, try to delete it.
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("unable to remove socket file: %w", err)
		}
	}
	return nil
}

func ensureSocketFileAccessible(filePath string) error {
	// anyone on system can connect.
	if err := os.Chmod(filePath, 0777); err != nil {
		return fmt.Errorf("unable to set write permission on socket: %w", err)
	}
	return nil
}

func (s *Dikastes) Serve(ctx context.Context, readyCh ...chan struct{}) {
	if s.listenNetwork == "unix" {
		if err := ensureSocketFileNone(s.listenAddress); err != nil {
			log.Fatal("could not start listener: ", err)
		}
	}

	lis, err := net.Listen(s.listenNetwork, s.listenAddress)
	if err != nil {
		log.Fatal("could not start listener: ", err)
		return
	}
	defer lis.Close()

	if s.listenNetwork == "unix" {
		if err := ensureSocketFileAccessible(s.listenAddress); err != nil {
			log.Fatal("could not start listener: ", err)
		}
	}

	log.Infof("Dikastes listening at %s", lis.Addr())
	s.listenAddress = lis.Addr().String()

	// listen ready
	for _, r := range readyCh {
		r <- struct{}{}
	}

	gs := grpc.NewServer(s.grpcServerOptions...)

	dpStats := statscache.New()
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
		syncher.WithSubscriptionType(s.subscriptionType),
	)
	syncClient.RegisterGRPCServices(gs)
	// wire up stats cache flush callback
	dpStats.RegisterFlushCallback(syncClient.OnStatsCacheFlush)
	go syncClient.Start(ctx)
	go dpStats.Start(ctx)

	if _, ok := os.LookupEnv("DIKASTES_ENABLE_CHECKER_REFLECTION"); ok {
		reflection.Register(gs)
	}
	// Run gRPC server on separate goroutine so we catch any signals and clean up.
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Errorf("failed to serve: %v", err)
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
