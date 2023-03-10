// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"
	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/projectcalico/calico/app-policy/server"
	"github.com/projectcalico/calico/app-policy/uds"
	"github.com/projectcalico/calico/app-policy/waf"
	"github.com/projectcalico/calico/libcalico-go/lib/seedrng"
)

const usage = `Dikastes - the decider.

Usage:
  dikastes server [options]
  dikastes client <namespace> <account> [--method <method>] [options]

Options:
  <namespace>               Service account namespace.
  <account>                 Service account name.
  -h --help                 Show this screen.
  -l --listen <port>        Unix domain socket path [default: /var/run/dikastes/dikastes.sock]
  -d --dial <target>        Target to dial. [default: localhost:50051]
  -r --rules <target>       Directory where WAF rules are stored.
  --log-level <level>       Log at specified level e.g. [default: info].
`

var VERSION string = "dev"

func main() {
	log.Infof("Dikastes (%s) launching", VERSION)
	// Make sure the RNG is seeded.
	seedrng.EnsureSeeded()

	arguments, err := docopt.ParseArgs(usage, nil, VERSION)
	if err != nil {
		println(usage)
		return
	}

	if lvl, ok := arguments["--log-level"].(string); ok {
		setLevel, err := log.ParseLevel(lvl)
		if err != nil {
			log.WithError(err).Warn("invalid log-level value. falling back to default value 'info'")
			setLevel = log.InfoLevel
		}
		log.SetLevel(setLevel)
	}

	if arguments["server"].(bool) {
		runServer(arguments)
	} else if arguments["client"].(bool) {
		runClient(arguments)
	}
}

func runServer(arguments map[string]interface{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Lifecycle: use a buffered channel so we don't miss any signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var (
		listenNetwork    string = "unix"
		listenAddr       string = arguments["--listen"].(string)
		policySyncAdress string = arguments["--dial"].(string)
		rulesetArgument         = arguments["--rules"]
	)

	subscriptionType := getEnv("DIKASTES_SUBSCRIPTION_TYPE", "per-pod-policies")

	// Config: overwrite listen socket path with hostport, env only
	listenTCP := getEnv("DIKASTES_FORCE_LISTEN_TCP_HOSTPORT", "")
	if listenTCP != "" {
		listenNetwork = "tcp"
		listenAddr = listenTCP
	}

	log.WithFields(log.Fields{
		"listenNetwork":    listenNetwork,
		"listenAddress":    listenAddr,
		"wafRuleset":       rulesetArgument,
		"subscriptionType": subscriptionType,
	}).Info("runtime arguments")

	// WAF: initialize if enabled, also cleanup after
	waf.Initialize(rulesetArgument)
	defer waf.CleanupModSecurity()

	// Dikastes main: Setup and serve
	dikastesServer := server.NewDikastesServer(
		server.WithListenArguments(listenNetwork, listenAddr),
		server.WithDialAddress(policySyncAdress),
		server.WithSubscriptionType(subscriptionType),
	)
	go dikastesServer.Serve(ctx)

	// Istio: termination handler (i.e., quitquitquit handler)
	th := httpTerminationHandler{make(chan bool, 1)}
	if httpServerPort := os.Getenv("DIKASTES_HTTP_BIND_PORT"); httpServerPort != "" {
		httpServerAddr := os.Getenv("DIKASTES_HTTP_BIND_ADDR")
		if httpServer, httpServerWg, err := th.RunHTTPServer(httpServerAddr, httpServerPort); err == nil {
			defer httpServerWg.Wait()
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err = httpServer.Shutdown(ctx); err != nil {
					log.Fatalf("error while shutting down HTTP server: %v", err)
				}
			}()
		} else {
			log.Fatal(err)
		}
	}

	// Lifecycle: block until a signal is received.
	select {
	case sig := <-sigChan:
		log.Infof("Got signal: %v", sig)
	case <-th.termChan:
		log.Info("Received HTTP termination request")
	}
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

type httpTerminationHandler struct {
	termChan chan bool
}

func (h *httpTerminationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.termChan <- true
	if _, err := io.WriteString(w, "terminating Dikastes\n"); err != nil {
		log.Fatalf("error writing HTTP response: %v", err)
	}
}

func (h *httpTerminationHandler) RunHTTPServer(addr string, port string) (*http.Server, *sync.WaitGroup, error) {
	if i, err := strconv.Atoi(port); err != nil {
		err = fmt.Errorf("error parsing provided HTTP listen port: %v", err)
		return nil, nil, err
	} else if i < 1 {
		err = fmt.Errorf("please provide non-zero, non-negative port number for HTTP listening port")
		return nil, nil, err
	}

	if addr != "" {
		if ip := net.ParseIP(addr); ip == nil {
			err := fmt.Errorf("invalid HTTP bind address \"%v\"", addr)
			return nil, nil, err
		}
	}

	httpServerSockAddr := fmt.Sprintf("%s:%s", addr, port)
	httpServerMux := http.NewServeMux()
	httpServerMux.Handle("/terminate", h)
	httpServer := &http.Server{Addr: httpServerSockAddr, Handler: httpServerMux}
	httpServerWg := &sync.WaitGroup{}
	httpServerWg.Add(1)

	go func() {
		defer httpServerWg.Done()
		log.Infof("starting HTTP server on %v", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server closed unexpectedly: %v", err)
		}
	}()
	return httpServer, httpServerWg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
