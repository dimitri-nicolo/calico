// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package main

import (
	"flag"
	"github.com/docopt/docopt-go"
	"github.com/golang/glog"
	"github.com/tigera/calicoq/calicoq/commands"
	ctlcommands "github.com/tigera/libcalico-go/calicoctl/commands"
	"os"
)

const usage = `Usage:
  calicoq host <hostname>
  calicoq eval <selector>
  calicoq version
`

func main() {
	var err error

	flag.Parse()

	if os.Getenv("GLOG") != "" {
		flag.Lookup("logtostderr").Value.Set("true")
		flag.Lookup("v").Value.Set(os.Getenv("GLOG"))
	}

	doc := ctlcommands.EtcdIntro + usage
	arguments, err := docopt.Parse(doc, nil, true, "calicoq", true, false)
	if err != nil {
		os.Exit(1)
	}
	glog.V(0).Info("Command line arguments: ", arguments)

	if arguments["version"].(bool) {
		err = commands.Version()
	} else if arguments["eval"].(bool) {
		err = commands.EvalSelector(arguments["<selector>"].(string))
	} else {
		err = commands.DescribeHost(arguments["<hostname>"].(string))
	}

	if err != nil {
		os.Exit(1)
	}
}
