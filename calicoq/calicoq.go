// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package main

import (
	"flag"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docopt/docopt-go"
	"github.com/golang/glog"
	"github.com/tigera/calicoq/calicoq/commands"
)

const usage = `Calico query tool.

Usage:
  calicoq host [-s|--hide-selectors] [-r|--include-rule-matches] <hostname>
  calicoq eval <selector>
  calicoq version

Options:
  -h --help                  Show usage.
  -s --hide-selectors        Hide selectors from output.
  -r --include-rule-matches  Show policies whose rules match endpoints on the host.
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
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.FatalLevel)
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
			arguments["--hide-selectors"].(bool),
			arguments["--include-rule-matches"].(bool))
	}

	if err != nil {
		os.Exit(1)
	}
}
