// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package ips

import (
	"fmt"

	"github.com/projectcalico/calico/libcalico-go/lib/net"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

// NormalizedIPSet converts the IP strings into a set of normalized Keys. If there is an error with any of the IPs
// this function returns the set of all IPs that could be converted along with the last error that it encountered.
func NormalizedIPSet(ips ...string) (set.Set[string], error) {
	var lastErr error
	s := set.New[string]()
	for i := range ips {
		ip, err := NormalizeIP(ips[i])
		if err != nil {
			lastErr = err
			continue
		}
		s.Add(ip)
	}
	return s, lastErr
}

// NormalizeIP converts the IP address string to a normalized form of the IP address string (e.g it will remove leading
// zeros and always use IPv4 format for IPv4 addresses). The IP address may be supplied in CIDR format.
func NormalizeIP(ip string) (string, error) {
	i, c, err := net.ParseCIDROrIP(ip)
	if err != nil {
		return "", err
	}
	if !isFullMask(c) {
		return "", fmt.Errorf("supplied CIDR is not a single IP address: %s", ip)
	}
	if i.Version() == 4 {
		return i.To4().String(), nil
	}
	return i.To16().String(), nil
}

// isFullMask checks that the mask is either /32 or 128 (depending on IP version).
func isFullMask(c *net.IPNet) bool {
	for i := range c.Mask {
		if c.Mask[i] != 0xff {
			return false
		}
	}
	return true
}
