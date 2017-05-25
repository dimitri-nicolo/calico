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
  calicoq [-d] [-c <config>] eval <selector>
  calicoq [-d] [-c <config>] policy <policy-id>
  calicoq [-d] [-c <config>] endpoint [-s|--hide-selectors] [-r|--hide-rule-matches] <endpoint-id>
  calicoq [-d] [-c <config>] host [-s|--hide-selectors] [-r|--hide-rule-matches] <hostname>
  calicoq [-d] version

Options:
  -c <config> --config=<config>  Path to the file containing connection
                                 configuration in YAML or JSON format.
                                 [default: /etc/calico/calicoctl.cfg]

  -r --hide-rule-matches     Don't show the list of profiles and policies whose
                             rule selectors match the specified endpoint (or an
                             endpoint on the specified host) as an allowed or
                             disallowed source/destination.

  -s --hide-selectors        Don't show the detailed selector expressions involved
                             (that cause each displayed profile or policy to match
                             <endpoint-id> or <hostname>).

  -d --debug                 Log debugging information to stderr.
`

func main() {
	log.SetLevel(log.ErrorLevel)

	arguments, err := docopt.Parse(usage, nil, true, commands.VERSION, false)
	if err != nil {
		log.Errorf("Failed to parse command line arguments: %v", err)
		os.Exit(1)
	}
	if arguments["-d"].(bool) {
		log.SetLevel(log.DebugLevel)
	}
	log.Info("Command line arguments: ", arguments)

	for cmd, thunk := range map[string]func() error{
		"version": commands.Version,
		"eval": func() error {
			// Show all the endpoints that match <selector>.
			return commands.EvalSelector(
				arguments["--config"].(string),
				arguments["<selector>"].(string),
			)
		},
		"policy": func() error {
			// Show all the endpoints that are relevant to <policy-id>.
			return commands.EvalPolicySelectors(
				arguments["--config"].(string),
				arguments["<policy-id>"].(string),
			)
		},
		"endpoint": func() error {
			// Show the profiles and policies that relate to <endpoint-id>.
			return commands.DescribeEndpointOrHost(
				arguments["--config"].(string),
				arguments["<endpoint-id>"].(string),
				"",
				arguments["--hide-selectors"].(bool),
				arguments["--hide-rule-matches"].(bool),
			)
		},
		"host": func() error {
			// Show the profiles and policies that relate to all endpoints on
			// <hostname>.
			return commands.DescribeEndpointOrHost(
				arguments["--config"].(string),
				"",
				arguments["<hostname>"].(string),
				arguments["--hide-selectors"].(bool),
				arguments["--hide-rule-matches"].(bool),
			)
		},
	} {
		if arguments[cmd].(bool) {
			err = thunk()
			break
		}
	}

	if err != nil {
		log.WithError(err).Error("Command failed")
		os.Exit(1)
	}
}
