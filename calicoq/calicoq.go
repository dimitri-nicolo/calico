// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package main

import (
	"flag"
	"github.com/docopt/docopt-go"
	"github.com/golang/glog"
	"github.com/tigera/calicoq/calicoq/commands"
	"os"
)

const usage = `Calico query tool.

Usage:
  calicoq host [-s|--hide-selectors] <hostname>
  calicoq eval <selector>
  calicoq version

Options:
  -h --help            Show usage.
  -s --hide-selectors  Hide selectors from output.
`

func main() {
	var err error

	flag.CommandLine.Usage = func() {
		println(usage)
	}
	flag.Parse()

	if os.Getenv("GLOG") != "" {
		flag.Lookup("logtostderr").Value.Set("true")
		flag.Lookup("v").Value.Set(os.Getenv("GLOG"))
	}

	arguments, err := docopt.Parse(usage, nil, true, "calicoq", false, false)
	if err != nil {
		glog.V(0).Infof("Failed to parse command line arguments: %v", err)
		os.Exit(1)
	}
	glog.V(0).Info("Command line arguments: ", arguments)

	if arguments["version"].(bool) {
		err = commands.Version()
	} else if arguments["eval"].(bool) {
		err = commands.EvalSelector(arguments["<selector>"].(string))
	} else {
		err = commands.DescribeHost(arguments["<hostname>"].(string),
			arguments["--hide-selectors"].(bool))
	}

	if err != nil {
		os.Exit(1)
	}
}
