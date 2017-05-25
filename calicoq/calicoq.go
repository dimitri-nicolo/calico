// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docopt/docopt-go"
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

	log.SetLevel(log.DebugLevel)

	arguments, err := docopt.Parse(usage, nil, true, "calicoq", false, false)
	if err != nil {
		log.Infof("Failed to parse command line arguments: %v", err)
		os.Exit(1)
	}
	log.Info("Command line arguments: ", arguments)

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
