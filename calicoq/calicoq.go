// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package main

import (
	"os"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/calicoq/calicoq/commands"
)

const usage = `Calico query tool.

Usage:
  calicoq [--debug|-d] [--config=<config>] eval <selector>
  calicoq [--debug|-d] [--config=<config>] policy <policy-name> [--hide-selectors|-s] [--hide-rule-matches|-r]
  calicoq [--debug|-d] [--config=<config>] endpoint <substring> [--hide-selectors|-s] [--hide-rule-matches|-r]
  calicoq [--debug|-d] [--config=<config>] host <hostname> [--hide-selectors|-s] [--hide-rule-matches|-r]
  calicoq [--debug|-d] version

Description:
  The calicoq command line tool is used to check Calico security policies.

  calicoq eval <selector> is used to display the endpoints that are matched by <selector>.

  calicoq policy <policy-name> shows the endpoints that are relevant to the named policy,
  comprising:
  - the endpoints that the policy applies to (for which ingress or egress traffic is policed
    according to the rules in that policy)
  - the endpoints that match the policy's rule selectors (that are allowed or disallowed as data
    sources or destinations).

  calicoq endpoint <substring> shows you the Calico policies and profiles that relate to endpoints
  whose full ID includes <substring>.

  calicoq host <hostname> shows you the endpoints that are hosted on <hostname> and all the Calico
  policies and profiles that relate to those endpoints.

Notes:
  When a Calico endpoint or policy is mapped from a Kubernetes resource, its name includes the
  Kubernetes namespace and name as "<namespace>.<name>".  For a policy, this is the whole of the
  Calico policy name.

  For an endpoint, the full Calico ID is "<host>/<orchestrator>/<workload-name>/<endpoint-name>".
  In the Kubernetes case "<orchestrator>" is always "k8s", "<workload-name>" is "<pod
  namespace>.<pod name>", and "<endpoint-name>" is always "eth0".

Options:
  -c <config> --config=<config>  Path to the file containing connection
                                 configuration in YAML or JSON format.
                                 [default: /etc/calico/calicoctl.cfg]

  -r --hide-rule-matches     Don't show the list of policies and profiles whose
                             rule selectors match the specified endpoint (or an
                             endpoint on the specified host) as an allowed or
                             disallowed source/destination.

  -s --hide-selectors        Don't show the detailed selector expressions involved
                             (that cause each displayed policy or profile to apply to or match
                             various endpoints).

  -d --debug                 Log debugging information to stderr.
`

func main() {
	log.SetLevel(log.FatalLevel)

	arguments, err := docopt.Parse(usage, nil, true, commands.VERSION, false)
	if err != nil {
		log.Fatalf("Failed to parse command line arguments: %v", err)
		os.Exit(1)
	}
	if arguments["--debug"].(bool) || arguments["-d"].(bool) {
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
			// Show all the endpoints that are relevant to <policy-name>.
			return commands.EvalPolicySelectors(
				arguments["--config"].(string),
				arguments["<policy-name>"].(string),
				arguments["-s"].(bool) ||
					arguments["--hide-selectors"].(bool),
				arguments["-r"].(bool) ||
					arguments["--hide-rule-matches"].(bool),
			)
		},
		"endpoint": func() error {
			// Show the policies and profiles that relate to <substring>.
			return commands.DescribeEndpointOrHost(
				arguments["--config"].(string),
				arguments["<substring>"].(string),
				"",
				arguments["-s"].(bool) ||
					arguments["--hide-selectors"].(bool),
				arguments["-r"].(bool) ||
					arguments["--hide-rule-matches"].(bool),
			)
		},
		"host": func() error {
			// Show the policies and profiles that relate to all endpoints on
			// <hostname>.
			return commands.DescribeEndpointOrHost(
				arguments["--config"].(string),
				"",
				arguments["<hostname>"].(string),
				arguments["-s"].(bool) ||
					arguments["--hide-selectors"].(bool),
				arguments["-r"].(bool) ||
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
