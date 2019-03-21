// Copyright 2019 Tigera Inc. All rights reserved.

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/tigera/intrusion-detection/controller/pkg/health"
)

func main() {
	var healthzSockPath string
	flag.StringVar(&healthzSockPath, "sock", health.DefaultHealthzSockPath, "Path to healthz socket")
	flag.Parse()

	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", healthzSockPath)
			},
		},
	}
	if len(flag.Args()) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s (liveness|readiness)\n", os.Args[0])
		os.Exit(1)
	}
	path := "/" + flag.Arg(0)
	r, err := c.Get("http://unix" + path)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error getting healthz %s: %s\n", flag.Arg(0), err)
		os.Exit(2)
	}
	if r.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintf(os.Stderr, "healthz endpoint returned status %d\n", r.StatusCode)
		os.Exit(3)
	}
	return
}
