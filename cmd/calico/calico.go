// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.

package main

import (
	"os"
	"path/filepath"

	"github.com/projectcalico/cni-plugin/pkg/install"
	"github.com/projectcalico/cni-plugin/pkg/ipamplugin"
	"github.com/projectcalico/cni-plugin/pkg/plugin"
)

// VERSION is filled out during the build process (using git describe output)
var VERSION string

func main() {
	// Accomodate Windows pathname
	name := strings.Replace(os.Args[0], "\\", "/", -1)
	// Use the name of the binary to determine which routine to run.
	_, filename := filepath.Split(os.Args[0])
	switch filename {
	case "calico", "calico.exe":
		plugin.Main(VERSION)
	case "calico-ipam", "calico-ipam.exe":
		ipamplugin.Main(VERSION)
	case "install":
		err := install.Install()
		if err != nil {
			panic(err)
		}
	default:
		panic("Unknown binary name: " + filename)
	}
}
