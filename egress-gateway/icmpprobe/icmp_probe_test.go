// Copyright (c) 2022  All rights reserved.

package icmpprobe

import (
	"context"
	"net"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/health"
)

func TestMainline(t *testing.T) {
	RegisterTestingT(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	healthAgg := health.NewHealthAggregator()
	addrs := []net.IP{net.ParseIP("127.0.0.1")}
	err := StartBackgroundICMPProbes(ctx, addrs, time.Second, 2*time.Second, healthAgg)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() bool {
		summary := healthAgg.Summary()
		return summary.Ready
	}).Should(BeTrue(), "ICMP probe failed against 127.0.0.1")
}

func TestMulti(t *testing.T) {
	RegisterTestingT(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	healthAgg := health.NewHealthAggregator()
	// One good address, one bad.
	addrs := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.0.2.1")}
	err := StartBackgroundICMPProbes(ctx, addrs, time.Second, 2*time.Second, healthAgg)
	Expect(err).NotTo(HaveOccurred())
	ready := func() bool {
		summary := healthAgg.Summary()
		return summary.Ready
	}
	Eventually(ready).Should(BeTrue(), "ICMP probe failed against 127.0.0.1+192.0.2.1")
	Consistently(ready, "3s", "200ms").Should(BeTrue(), "Should ping 127.0.0.1+192.0.2.1")
}

func TestDropped(t *testing.T) {
	RegisterTestingT(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	healthAgg := health.NewHealthAggregator()
	addrs := []net.IP{net.ParseIP("192.0.2.1")} // Reserved example block, shouldn't be routable.
	err := StartBackgroundICMPProbes(ctx, addrs, time.Second, 2*time.Second, healthAgg)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() bool {
		summary := healthAgg.Summary()
		return summary.Ready
	}, "1s", "100ms").Should(BeFalse(), "Should fail to ping 192.0.2.1")
	Consistently(func() bool {
		summary := healthAgg.Summary()
		return summary.Ready
	}, "3s", "200ms").Should(BeFalse(), "Should fail to ping 192.0.2.1")
}
