// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package felixclient

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/tigera/envoy-collector/pkg/collector"
	"github.com/tigera/envoy-collector/proto"
)

type FelixClient interface {
	SendStats(context.Context, collector.EnvoyCollector)
	SendData(context.Context, proto.PolicySyncClient, collector.EnvoyInfo) error
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

// SendStats listens for data from the collector and sends it.
func (fc *felixClient) SendStats(ctx context.Context, collector collector.EnvoyCollector) {
	log.Info("Starting sending DataplaneStats to Policy Sync server")
	conn, err := grpc.Dial(fc.target, fc.dialOpts...)
	if err != nil {
		log.Warnf("fail to dial Policy Sync server: %v", err)
		return
	}
	log.Info("Successfully connected to Policy Sync server")
	defer conn.Close()
	client := proto.NewPolicySyncClient(conn)

	for {
		select {
		case data := <-collector.Report():
			if err := fc.SendData(ctx, client, data); err != nil {
				// Error reporting stats, exit now to start reconnction processing.
				log.WithError(err).Warning("Error reporting stats")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// SendData takes EnvoyLog data and sends the it with the
// protobuf client.
func (fc *felixClient) SendData(ctx context.Context, client proto.PolicySyncClient, logData collector.EnvoyInfo) error {
	// Batch the data by 5 tuple
	data := fc.batchAndConvertEnvoyLogs(logData)

	// Send the batched data
	for _, d := range data {
		log.Debugf("Sending DataplaneStats: %s", d)
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
	}
	log.Info("Sent DataplaneStats to Policy Sync server")
	return nil
}

func (fc *felixClient) batchAndConvertEnvoyLogs(info collector.EnvoyInfo) map[collector.TupleKey]*proto.DataplaneStats {
	data := make(map[collector.TupleKey]*proto.DataplaneStats)
	for _, log := range info.Logs {
		// Convert the EnvoyLog to DataplaneStats
		d := fc.dataPlaneStatsFromIngressLog(log)

		// Join the HttpData fields by 5 tuple
		tupleKey := collector.TupleKeyFromEnvoyLog(log)
		if existing, ok := data[tupleKey]; ok {
			// Add the HttpData to the existing log
			existing.HttpData = append(existing.HttpData, d.HttpData...)
		} else {
			data[tupleKey] = d
		}

		// Add the count statistics
		httpStat := &proto.Statistic{
			Direction:  proto.Statistic_IN,
			Relativity: proto.Statistic_DELTA,
			Kind:       proto.Statistic_HTTP_DATA,
			Action:     proto.Action_ALLOWED,
			Value:      int64(info.Connections[tupleKey]),
		}
		d.Stats = append(d.Stats, httpStat)
	}

	// Create connection logs for connections which do not
	// include requests we have recorded.
	for key, count := range info.Connections {
		if _, ok := data[key]; !ok {
			log := collector.EnvoyLogFromTupleKey(key)
			d := fc.dataPlaneStatsFromIngressLog(log)
			// Add the count statistics
			httpStat := &proto.Statistic{
				Direction:  proto.Statistic_IN,
				Relativity: proto.Statistic_DELTA,
				Kind:       proto.Statistic_HTTP_DATA,
				Action:     proto.Action_ALLOWED,
				Value:      int64(count),
			}
			d.Stats = append(d.Stats, httpStat)

			data[key] = d
		}

	}
	return data
}

func (fc *felixClient) dataPlaneStatsFromIngressLog(logData collector.EnvoyLog) *proto.DataplaneStats {
	// Unless the protocol is specified, the protocol will be
	// TCP since the feature requires the user of HTTP headers
	// in order to function properly.
	if logData.Protocol == "" {
		logData.Protocol = "tcp"
	}

	d := &proto.DataplaneStats{
		SrcIp:    logData.SrcIp,
		DstIp:    logData.DstIp,
		SrcPort:  logData.SrcPort,
		DstPort:  logData.DstPort,
		Protocol: &proto.Protocol{&proto.Protocol_Name{logData.Protocol}},
	}

	// Empty values are represented as "-" in Nginx logs
	if logData.XForwardedFor == "-" {
		logData.XForwardedFor = ""
	}
	if logData.XRealIp == "-" {
		logData.XRealIp = ""
	}

	// X-Forwarded-For can be a comma separated list of IPs.
	// Make sure that only the first IP is returned.
	// Depending on the number of proxy servers a request goes through,
	// multiple IPs can be appended to this field. We only care about
	// the original source for our flow logs which is the first IP.
	ips := strings.Split(logData.XForwardedFor, ",")
	if len(ips) > 1 {
		logData.XForwardedFor = strings.Trim(ips[0], " ")
	}

	// Only capture the header information if it exists.
	if logData.XForwardedFor != "" || logData.XRealIp != "" {
		d.HttpData = []*proto.HttpData{
			&proto.HttpData{
				XForwardedFor: logData.XForwardedFor,
				XRealIp:       logData.XRealIp,
			},
		}
	}

	return d
}
