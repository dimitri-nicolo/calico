// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package felixclient

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/tigera/ingress-collector/pkg/collector"
	"github.com/tigera/ingress-collector/proto"
)

type FelixClient interface {
	SendStats(context.Context, proto.PolicySyncClient, collector.IngressLog) error
	CollectAndSend(context.Context, collector.IngressCollector)
}

// felixClient provides the means to send data to Felix
type felixClient struct {
	target   string
	dialOpts []grpc.DialOption
}

func NewFelixClient(target string, opts []grpc.DialOption) FelixClient {
	return &felixClient{
		target:   target,
		dialOpts: opts,
	}
}

// TODO: Move this into another object to manage sending
func (fc *felixClient) CollectAndSend(ctx context.Context, collector collector.IngressCollector) {
	ctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}

	// Start the log collection go routine.
	wg.Add(1)
	go func() {
		// TODO: Don't make direct calls to the collector here
		// Replace this with non-test code
		for {
			collector.ReadLogs()
			time.Sleep(300 * time.Second)
		}
		cancel()
		wg.Done()
	}()

	// Start the DataplaneStats reporting go routine.
	wg.Add(1)
	go func() {
		fc.SendLoop(ctx, collector)
		cancel()
		wg.Done()
	}()

	// Wait for the go routine to complete before exiting
	wg.Wait()
}

// TODO: Move this into another object for managing sending
// SendLoop is the main stats reporting loop.
func (fc *felixClient) SendLoop(ctx context.Context, collector collector.IngressCollector) error {
	log.Info("Starting sending DataplaneStats to Policy Sync server")
	conn, err := grpc.Dial(fc.target, fc.dialOpts...)
	if err != nil {
		log.Warnf("fail to dial Policy Sync server: %v", err)
		return err
	}
	log.Info("Successfully connected to Policy Sync server")
	defer conn.Close()
	client := proto.NewPolicySyncClient(conn)

	for {
		select {
		case data := <-collector.Report():
			if err := fc.SendStats(ctx, client, data); err != nil {
				// Error reporting stats, exit now to start reconnction processing.
				log.WithError(err).Warning("Error reporting stats")
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (fc *felixClient) SendStats(ctx context.Context, client proto.PolicySyncClient, logData collector.IngressLog) error {
	d := &proto.DataplaneStats{
		SrcIp:    logData.SrcIp,
		DstIp:    logData.DstIp,
		SrcPort:  logData.SrcPort,
		DstPort:  logData.DstPort,
		Protocol: &proto.Protocol{&proto.Protocol_Name{logData.Protocol}},
	}

	// TODO: Add reporting of L7 data too
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
	log.Info("Sent DataplaneStats to Policy Sync server")
	return nil
}
