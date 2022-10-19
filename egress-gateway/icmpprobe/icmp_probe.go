// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package icmpprobe

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"time"

	"github.com/projectcalico/calico/felix/jitter"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/sirupsen/logrus"
)

const HealthName = "icmp probes"

func StartBackgroundICMPProbes(ctx context.Context, addrs []net.IP, interval time.Duration, timeout time.Duration, healthAgg *health.HealthAggregator) error {
	healthAgg.RegisterReporter(HealthName, &health.HealthReport{Ready: true}, timeout)
	// Since we want the overall readiness to be "up" if _any_ probe is successful, start one goroutine for each IP.
	for _, addr := range addrs {
		go LoopDoingProbes(ctx, addr, interval, healthAgg)
	}
	return nil
}

var (
	// Should match lines like this: `64 bytes from 8.8.8.8: icmp_seq=1 ttl=116 time=5.87 ms`
	goodResponseRE = regexp.MustCompile(`(\d+) bytes from (.+): icmp_seq=(\d+) ttl=(\d+) time=([\d.]+ \w+)`)
)

func LoopDoingProbes(ctx context.Context, addr net.IP, interval time.Duration, healthAgg *health.HealthAggregator) {
	logCtx := logrus.WithField("ip", addr)
	logCtx.Info("ICMP probe goroutine started.")

	addrStr := addr.String()
	args := []string{
		"-n",                                 // "numeric"; don't look up hostnames
		"-W", fmt.Sprint(interval.Seconds()), // Wait until the next interval...
		"-c", "1", // ...for at least one response packet
		addrStr,
	}
	ticker := jitter.NewTicker(interval*95/100, interval*10/100)
	defer ticker.Stop()
	for ctx.Err() == nil {
		logCtx.Debug("About to do ping...")
		cmd := exec.Command("ping", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			if err, ok := err.(*exec.ExitError); ok {
				// Avoid logging the ExitError, it's very verbose and not useful.
				logCtx.WithField("rc", err.ExitCode()).Warnf("ICMP probe failed:\n%s", string(out))
			} else {
				logCtx.WithError(err).Errorf("ICMP probe failed with unexpected error. Output from ping (if any): %q", string(out))
			}
		} else {
			logCtx.Debug("Ping successful.")
			healthAgg.Report(HealthName, &health.HealthReport{Ready: true})
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
		}
	}

	logCtx.Info("Context canceled.")
}
