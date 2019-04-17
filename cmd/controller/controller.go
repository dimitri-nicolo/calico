// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/tigera/compliance/pkg/version"
)

func main() {
	var ver bool
	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.Parse()

	if ver {
		version.Version()
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
