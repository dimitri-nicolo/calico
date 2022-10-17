// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package icmpprobe

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/sirupsen/logrus"
)

const HealthName = "icmp probes"

func StartBackgroundICMPProbes(ctx context.Context, addrs []net.IP, interval time.Duration, timeout time.Duration, healthAgg *health.HealthAggregator) error {
	healthAgg.RegisterReporter(HealthName, &health.HealthReport{Ready: true}, timeout)
	// Since we want the overall readiness to be "up"" if _any_ probe is successful, start one goroutine for each
	// address, each probing a different IP.
	for _, addr := range addrs {
		go LoopDoingProbes(ctx, addr, interval, timeout, healthAgg)
	}
	return nil
}

var (
	// Should match lines like this: `64 bytes from 8.8.8.8: icmp_seq=1 ttl=116 time=5.87 ms`
	goodResponseRE = regexp.MustCompile(`(\d+) bytes from (.+): icmp_seq=(\d+) ttl=(\d+) time=([\d.]+ \w+)`)
)

func LoopDoingProbes(ctx context.Context, addr net.IP, interval time.Duration, timeout time.Duration, healthAgg *health.HealthAggregator) {
	logCtx := logrus.WithField("ip", addr)
	logCtx.Info("ICMP probe goroutine started.")

	// It's a little awkward to send pings from a user process.  We'd need CAP_NET_RAW to do it ourselves.
	// Instead, shell out to the ping binary, which has the required permissions.
	cmd := exec.Command("ping",
		"-n",                                 // "numeric"; don't look up hostnames
		"-i", fmt.Sprint(interval.Seconds()), // Interval.
		"-W", fmt.Sprint(interval.Seconds()), // Timout per packet, wait until we're due to send the next packet.
		addr.String(),
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logCtx.WithError(err).Panic("Failed to get stdout pipe.  Have we run out of file handles?")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logCtx.WithError(err).Panic("Failed to get stderr pipe.  Have we run out of file handles?")
	}
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			logCtx.Infof("Ping stderr: %q", scanner.Text())
		}
	}()
	err = cmd.Start()
	if err != nil {
		logCtx.WithError(err).Panic("Failed to start ping subprocess.")
	}
	go func() {
		<-ctx.Done()
		logCtx.Info("Context canceled. Shutting down ping.")
		err := cmd.Process.Kill()
		if err != nil {
			logCtx.WithError(err).Warn("Failed to kill ping. (Already gone?)")
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		// Lines look like this:
		// PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
		// 64 bytes from 8.8.8.8: icmp_seq=1 ttl=116 time=5.87 ms
		// ...
		line := scanner.Text()
		if goodResponseRE.MatchString(line) {
			// Note: we only set Ready=true, never Ready=false; we only want the readiness to timeout when none of
			// our probe goroutines are successful.
			logCtx.Debugf("Successful output from ping, reporting ready=true: %q", line)
			healthAgg.Report(HealthName, &health.HealthReport{Ready: true})
			continue
		} else if strings.HasPrefix(line, "PING") {
			logCtx.Debugf("Ping header line: %q", line)
			continue
		}
		logCtx.Warnf("Non-success output from ICMP probe: %q", line)
	}
	if ctx.Err() != nil {
		logCtx.Info("Context canceled.")
		return
	}
	if err := scanner.Err(); err != nil {
		logCtx.WithError(err).Panic("Failed to read from ping")
	}
	err = cmd.Wait()
	logCtx.WithError(err).Panic("Ping exited")
}
