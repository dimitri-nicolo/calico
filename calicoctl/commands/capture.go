// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/projectcalico/calicoctl/calicoctl/commands/argutils"
	"github.com/projectcalico/calicoctl/calicoctl/commands/capture"
	"github.com/projectcalico/calicoctl/calicoctl/commands/clientmgr"
	"github.com/projectcalico/libcalico-go/lib/options"

	"github.com/projectcalico/calicoctl/calicoctl/commands/common"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calicoctl/calicoctl/commands/constants"
)

const defaultCaptureDir = "/var/log/calico/pcap"

func Capture(args []string) error {
	doc := constants.DatastoreIntro + `Usage:
  calicoctl captured-packets ( copy | clean ) <NAME>
                [--config=<CONFIG>] [--namespace=<NS>] [--all-namespaces] [--dest=<DEST>]

Examples:
  # Copies capture files for packet capture from default namespace in the current directory.
  calicoctl captured-packets copy my-capture
  # Delete capture files for packet capture from default namespace still left on the system
  calicoctl captured-packets clean my-capture

Options:
  -n --namespace=<NS>      Namespace of the packet capture.
                           Uses the default namespace if not specified. [default: default]
  -a --all-namespaces      If present, list the requested packet capture(s) across all namespaces.
  -d --dest=<DEST>         If present, uses the directory specified as the destination. [default: .]
  -h --help                Show this screen.
  -c --config=<CONFIG>     Path to the file containing connection configuration in
                           YAML or JSON format.
                           [default: ` + constants.DefaultConfigPath + `]

Description:
  Commands for accessing Capture related information.

  See 'calicoctl captured-packets <command> --help' to read about a specific subcommand.`

	parsedArgs, err := docopt.Parse(doc, args, true, "", false, false)
	if err != nil {
		return fmt.Errorf("Invalid option: 'calicoctl %s'. Use flag '--help' to read about a specific subcommand.", strings.Join(args, " "))
	}

	if len(parsedArgs) == 0 {
		return nil
	}

	// Resolve string parameters
	cfgStr, err := argutils.ArgString(parsedArgs, "--config")
	if err != nil {
		return nil
	}
	name, err := argutils.ArgString(parsedArgs, "<NAME>")
	if err != nil {
		return err
	}
	destination, err := argutils.ArgString(parsedArgs, "--dest")
	if err != nil {
		return err
	}
	namespace, err := argutils.ArgString(parsedArgs, "--namespace")
	if err != nil {
		return err
	}
	// Resolve boolean parameters
	var isCopyCommand = argutils.ArgBoolOrFalse(parsedArgs, "copy")
	var isCleanCommand = argutils.ArgBoolOrFalse(parsedArgs, "clean")
	var allNamespaces = argutils.ArgBoolOrFalse(parsedArgs, "--all-namespaces")

	// Resolve capture dir location
	captureDir, err := resolveCaptureDir(cfgStr)
	if err != nil {
		return err
	}
	log.Debugf("Resolved capture directory to %s", captureDir)

	// Ensure kubectl command is available
	if err := common.KubectlExists(); err != nil {
		return fmt.Errorf("an error occurred checking if kubctl exists: %w", err)
	}
	// Extract kubeconfig variable
	cfg, err := clientmgr.LoadClientConfig(cfgStr)
	if err != nil {
		return err
	}

	// Create the capture commands
	captureCmd := capture.NewCommands(common.NewKubectlCmd(cfg.Spec.Kubeconfig))

	// Resolve the locations of the capture files
	locations, err := captureCmd.Resolve(captureDir, name)
	if err != nil {
		return err
	}

	var results int
	var errors []error

	// Run copy or clean
	if isCopyCommand {
		results, errors = captureCmd.Copy(filterByNamespace(locations, allNamespaces, namespace), destination)
	} else if isCleanCommand {
		results, errors = captureCmd.Clean(filterByNamespace(locations, allNamespaces, namespace))
	}

	// in case --all-namespaces is used and we have at least 1 successful result
	// we will return 0 exit code
	if allNamespaces {
		if results != 0 {
			return nil
		}
	}

	if errors != nil {
		var result []string
		for _, e := range errors {
			result = append(result, e.Error())
		}
		return fmt.Errorf(strings.Join(result, ";"))
	}

	return nil
}

func filterByNamespace(locations []capture.Location, allNamespaces bool, namespace string) []capture.Location {
	if allNamespaces {
		return locations
	}

	var filter []capture.Location
	for _, loc := range locations {
		if loc.Namespace == namespace {
			filter = append(filter, loc)
		}
	}
	return filter
}

func resolveCaptureDir(cfg string) (string, error) {
	client, err := clientmgr.NewClient(cfg)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	felixConfig, err := client.FelixConfigurations().Get(ctx, "default", options.GetOptions{ResourceVersion: ""})
	if err != nil {
		return "", err
	}

	if felixConfig.Spec.CaptureDir == nil {
		return defaultCaptureDir, nil
	}

	return *felixConfig.Spec.CaptureDir, nil
}
