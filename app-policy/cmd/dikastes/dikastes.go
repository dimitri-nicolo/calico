// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/app-policy/flags"
	"github.com/projectcalico/calico/app-policy/server"
)

var VERSION string = "dev"

func main() {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := flags.New()
	if err := config.Parse(os.Args); err != nil {
		log.Fatal(err)
		return
	}

	log.Infof("Dikastes (%s) launching", VERSION)
	runServer(ctx, config)
}

func runServer(ctx context.Context, config *flags.Config, readyCh ...chan struct{}) {
	// setup log level
	setLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.WithError(err).Warn("invalid log-level value. falling back to default value 'info'")
		setLevel = log.InfoLevel
	}
	log.SetLevel(setLevel)

	// Lifecycle: use a buffered channel so we don't miss any signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.WithFields(config.Fields()).Info("runtime arguments")

	// Dikastes main: Setup and serve
	dikastesServer := server.NewDikastesServer(
		server.WithListenArguments(config.ListenNetwork, config.ListenAddress),
		server.WithDialAddress(config.DialNetwork, config.DialAddress),
		server.WithSubscriptionType(config.SubscriptionType),
		server.WithWAFConfig(
			config.WAFEnabled,
			config.WAFLogFile,
			config.WAFRulesetFiles.Value(),
			config.WAFDirectives.Value(),
		),
		server.WithWAFFlushDuration(config.WAFEventsFlushInterval),
	)
	go dikastesServer.Serve(ctx, readyCh...)

	// Istio: termination handler (i.e., quitquitquit handler)
	thChan := make(chan struct{}, 1)
	if config.HTTPServerPort != "" {
		th := httpTerminationHandler{thChan}
		log.Info("http server port is", config.HTTPServerPort)
		if httpServer, httpServerWg, err := th.RunHTTPServer(config.HTTPServerAddr, config.HTTPServerPort); err == nil {
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
	case <-ctx.Done():
		log.Info("Context cancelled")
	case sig := <-sigChan:
		log.Infof("Got signal: %v", sig)
	case <-thChan:
		log.Info("Received HTTP termination request")
	}
}
