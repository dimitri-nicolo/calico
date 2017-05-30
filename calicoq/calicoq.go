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
  calicoq [--debug] [--config=<config>] eval <selector>
  calicoq [--debug] [--config=<config>] policy <policy-id> [--hide-selectors] [--hide-rule-matches]
  calicoq [--debug] [--config=<config>] endpoint <endpoint-id> [--hide-selectors] [--hide-rule-matches]
  calicoq [--debug] [--config=<config>] host <hostname> [--hide-selectors] [--hide-rule-matches]
  calicoq [--debug] version

Description:
  The calicoq command line tool is used to check Calico security policies.

  calicoq eval <selector> is used to display the endpoints that are matched by <selector>.

  calicoq policy <policy-id> shows the endpoints that are relevant to policy <policy-id>,
  comprising:
  - the endpoints for which ingress or egress traffic is policed according to the rules in that
    policy
  - the endpoints that the policy's rule selectors allow or disallow as data sources or
    destinations.

  calicoq endpoint <substring> shows you the Calico profiles and policies that relate to endpoints
  whose full name includes <substring>.

  calicoq host <hostname> shows you the endpoints that are hosted on <hostname> and all the Calico
  profiles and policies that relate to those endpoints.

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
	log.SetLevel(log.FatalLevel)

	arguments, err := docopt.Parse(usage, nil, true, commands.VERSION, false)
	if err != nil {
		log.Fatalf("Failed to parse command line arguments: %v", err)
		os.Exit(1)
	}
	if arguments["--debug"].(bool) {
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
				arguments["--hide-selectors"].(bool),
				arguments["--hide-rule-matches"].(bool),
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
