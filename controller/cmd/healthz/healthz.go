// Copyright 2019 Tigera Inc. All rights reserved.

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/tigera/intrusion-detection/controller/pkg/health"
)

func main() {
	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", health.HealthzSockPath)
			},
		},
	}
	if len(os.Args) != 2 {
		fmt.Println("Usage: healthz (liveness|readiness)")
		os.Exit(1)
	}
	path := "/" + os.Args[1]
	r, err := c.Get("http://unix" + path)
	if err != nil {
		fmt.Printf("Error getting healthz %s: %s\n", os.Args[1], err)
		os.Exit(2)
	}
	if r.StatusCode != http.StatusOK {
		fmt.Printf("healtz endpoint returned status %d\n", r.StatusCode)
		os.Exit(3)
	}
	return
}
