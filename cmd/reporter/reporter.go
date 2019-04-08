// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/projectcalico/libcalico-go/lib/logutils"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/version"
)

var (
	ver          = flag.Bool("version", false, "Print version information")
	startTimeStr = flag.String("start-time", time.Now().Add(-24*time.Hour).Format(time.RFC3339), "RFC3339 format for start time of report generation")
	endTimeStr   = flag.String("end-time", time.Now().Format(time.RFC3339), "RFC3339 format for end time of report generation")
)

func main() {
	flag.Parse()

	if *ver {
		version.Version()
		return
	}

	// Set up logger.
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetLevel(logutils.SafeParseLogLevel(os.Getenv("LOG_LEVEL")))

	// Parse start/end times
	startTime, err := time.Parse(time.RFC3339, *startTimeStr)
	if err != nil {
		log.WithError(err).Fatal("failed to parse start time")
	}

	endTime, err := time.Parse(time.RFC3339, *endTimeStr)
	if err != nil {
		log.WithError(err).Fatal("failed to parse end time")
	}

	log.WithFields(log.Fields{"start": startTime, "end": endTime}).Info("parsed start and end times")

	// Init elastic.
	client, err := elastic.NewFromEnv()
	if err != nil {
		log.WithError(err).Fatal("failed to initialize elastic")
	}
	// Check elastic index.
	if err = client.EnsureIndices(); err != nil {
		log.WithError(err).Fatal("failed to initialize elastic indices")
	}

	// setup signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigs
		cancel()
	}()

	// run.
	replay.New(startTime, endTime, client, client, new(mockCallback)).Start(ctx)
}

type mockCallback struct {
}

func (cb *mockCallback) OnStatusUpdate(su syncer.StatusUpdate) {
	log.WithField("status", su).Info("onStatusUpdate called")
}

func (cb *mockCallback) OnUpdate(u syncer.Update) {
	log.WithField("update", u).Info("onUpdate called")
}
