// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package main

import (
	"os"
	"flag"
	"github.com/docopt/docopt-go"
	"github.com/tigera/calicoq/calicoq/commands"
	ctlcommands "github.com/tigera/libcalico-go/calicoctl/commands"
	"github.com/golang/glog"
)

const usage = `Usage:
  calicoq <hostname>
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

	err = commands.DescribeHost(arguments["<hostname>"].(string))

	if err != nil {
		os.Exit(1)
	}
}
