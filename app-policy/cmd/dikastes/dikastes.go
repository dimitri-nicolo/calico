// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/docopt/docopt-go"
	authz_v2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	authz_v2alpha "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2alpha"
	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/projectcalico/calico/app-policy/checker"
	"github.com/projectcalico/calico/app-policy/health"
	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/proto"
	"github.com/projectcalico/calico/app-policy/statscache"
	"github.com/projectcalico/calico/app-policy/syncher"
	"github.com/projectcalico/calico/app-policy/uds"
	"github.com/projectcalico/calico/app-policy/waf"
)

const usage = `Dikastes - the decider.

Usage:
  dikastes server [options]
  dikastes client <namespace> <account> [--method <method>] [options]

Options:
  <namespace>            Service account namespace.
  <account>              Service account name.
  -h --help              Show this screen.
  -l --listen <port>     Unix domain socket path [default: /var/run/dikastes/dikastes.sock]
  -d --dial <target>     Target to dial. [default: localhost:50051]
  -r --rules <target>    Directory where WAF rules are stored. [default: /etc/waf/]
  --debug                Log at Debug level.`

var VERSION string

const (
	maxPendingDataplaneStats = 100
)

func main() {
	log.Info("Dikastes launching with ALP, WAF etc. logger.")
	arguments, err := docopt.ParseArgs(usage, nil, VERSION)
	if err != nil {
		println(usage)
		return
	}
	if arguments["--debug"].(bool) {
		log.SetLevel(log.DebugLevel)
	}
	if arguments["server"].(bool) {
		runServer(arguments)
	} else if arguments["client"].(bool) {
		runClient(arguments)
	}
}

func runServer(arguments map[string]interface{}) {
	filePath := arguments["--listen"].(string)
	dial := arguments["--dial"].(string)
	rulesetDirectory := arguments["--rules"].(string)

	_, err := os.Stat(filePath)
	if !os.IsNotExist(err) {
		// file exists, try to delete it.
		err := os.Remove(filePath)
		if err != nil {
			log.WithFields(log.Fields{
				"listen": filePath,
				"err":    err,
			}).Fatal("File exists and unable to remove.")
		}
	}
	lis, err := net.Listen("unix", filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"listen": filePath,
			"err":    err,
		}).Fatal("Unable to listen.")
	}
	defer lis.Close()
	err = os.Chmod(filePath, 0777) // Anyone on system can connect.
	if err != nil {
		log.Fatal("Unable to set write permission on socket.")
	}

	// Initialize WAF and load OWASP Core Rule Sets.
	waf.InitializeModSecurity()
	waf.DefineRulesSetDirectory(rulesetDirectory)
	filenames, err := waf.ExtractRulesSetFilenames()
	if err != nil {
		log.Fatalf("WAF Core Rules Set directory: '%s' does not exist!", rulesetDirectory)
	}
	err = waf.LoadModSecurityCoreRuleSet(filenames)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Check server
	gs := grpc.NewServer()
	stores := make(chan *policystore.PolicyStore)
	dpStats := make(chan statscache.DPStats, maxPendingDataplaneStats)
	checkServer := checker.NewServer(ctx, stores, dpStats)
	authz.RegisterAuthorizationServer(gs, checkServer)
	checkServerV2 := checkServer.V2Compat()
	authz_v2alpha.RegisterAuthorizationServer(gs, checkServerV2)
	authz_v2.RegisterAuthorizationServer(gs, checkServerV2)

	// Synchronize the policy store and start reporting stats.
	opts := uds.GetDialOptions()
	syncClient := syncher.NewClient(dial, opts, syncher.ClientOptions{})

	// Register the health check service, which reports the syncClient's inSync status.
	proto.RegisterHealthzServer(gs, health.NewHealthCheckService(syncClient))

	go syncClient.Start(ctx, stores, dpStats)

	// Run gRPC server on separate goroutine so we catch any signals and clean up.
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Use a buffered channel so we don't miss any signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received.
	log.Infof("Got signal: %v", <-c)

	// Cleanup WAF resources
	waf.CleanupModSecurity()

	gs.GracefulStop()
}

func runClient(arguments map[string]interface{}) {
	dial := arguments["--dial"].(string)
	namespace := arguments["<namespace>"].(string)
	account := arguments["<account>"].(string)
	useMethod := arguments["--method"].(bool)
	method := arguments["<method>"].(string)

	opts := uds.GetDialOptions()
	conn, err := grpc.Dial(dial, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := authz.NewAuthorizationClient(conn)
	req := authz.CheckRequest{
		Attributes: &authz.AttributeContext{
			Source: &authz.AttributeContext_Peer{
				Principal: fmt.Sprintf("spiffe://cluster.local/ns/%s/sa/%s",
					namespace, account),
			},
		},
	}
	if useMethod {
		req.Attributes.Request = &authz.AttributeContext_Request{
			Http: &authz.AttributeContext_HttpRequest{
				Method: method,
			},
		}
	}
	resp, err := client.Check(context.Background(), &req)
	if err != nil {
		log.Fatalf("Failed %v", err)
	}
	log.Infof("Check response:\n %v", resp)
}
