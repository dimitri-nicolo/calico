// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package commands

import (
	"context"
	"fmt"
	"github.com/projectcalico/calicoctl/calicoctl/commands/argutils"
	"github.com/projectcalico/calicoctl/calicoctl/commands/capture"
	"github.com/projectcalico/calicoctl/calicoctl/commands/clientmgr"
	"github.com/projectcalico/libcalico-go/lib/options"
	"strings"

	"github.com/projectcalico/calicoctl/calicoctl/commands/common"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calicoctl/calicoctl/commands/constants"
)

func Capture(args []string) error {
	doc := constants.DatastoreIntro + `Usage:
  calicoctl captured-packets (--copy |--clean ) <NAME>
                [--config=<CONFIG>] [--namespace=<NS>] [--all-namespaces] [--dest=<DEST>] [--log-level=<level>]

Examples:
  # Copies capture files for packet capture from default namespace in the current directory.
  calicoctl captured-packets --copy my-capture
  # Delete capture files for packet capture from default namespace still left on the system
  calicoctl captured-packets --clean my-capture

Options:
  -n --namespace=<NS>      Namespace of the packet capture.
                           Uses the default namespace if not specified.
  -a --all-namespaces      If present, list the requested packet capture(s) across all namespaces.
  -d --dest=<DEST>         If present, uses the directory specified as the destination.
  -h --help                Show this screen.
  -l --log-level=<level>   Set log level to debug, warn, info or panic.
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

	// Setup log level
	if logLevel := parsedArgs["--log-level"]; logLevel != nil {
		parsedLogLevel, err := log.ParseLevel(logLevel.(string))
		if err != nil {
			return fmt.Errorf("Unknown log level: %s, expected one of: \n"+
				"panic, fatal, error, warn, info, debug.\n", logLevel)
		} else {
			log.SetLevel(parsedLogLevel)
			log.Infof("Log level set to %v", parsedLogLevel)
		}
	}

	// Ensure kubectl command is available
	if err := common.KubectlExists(); err != nil {
		return fmt.Errorf("an error occurred checking if kubctl exists: %w", err)
	}
	// Extract kubeconfig variable
	cfg, err := clientmgr.LoadClientConfig(argutils.ArgString(parsedArgs, "--config"))
	if err != nil {
		return err
	}
	var kubeConfigPath = cfg.Spec.Kubeconfig

	// Resolve capture dir location
	captureDir, err := resolveCaptureDir(parsedArgs)
	if err != nil {
		return err
	}
	log.Debugf("Resolved capture directory to %s", captureDir)

	// Resolve capture namespaces
	namespaces, err := resolveNamespaces(parsedArgs, kubeConfigPath)
	if err != nil {
		return err
	}

	var failure int
	var captureCmd = capture.NewCommands(capture.NewKubectlCmd(kubeConfigPath))

	var name = argutils.ArgString(parsedArgs, "<NAME>")
	var isCopyCommand = argutils.ArgBoolOrFalse(parsedArgs, "--copy")
	var isCleanCommand = argutils.ArgBoolOrFalse(parsedArgs, "--clean")

	for _, ns := range namespaces {
		log.Debugf("Retrieve capture files for: %s/%s", ns, name)
		var pods, podNs = captureCmd.ResolveEntryPoints(captureDir, name, ns)

		if len(pods) == 0 {
			if !argutils.ArgBoolOrFalse(parsedArgs, "--all-namespaces") {
				return fmt.Errorf("Failed to find capture files for %s/%s \n", ns, name)
			}
			failure ++
		}

		if isCopyCommand {
			var destination = resolveDestination(parsedArgs)
			err := captureCmd.Copy(pods, podNs, name, ns, captureDir, destination)
			if err != nil {
				return err
			}
		} else if isCleanCommand {
			err := captureCmd.Clean(pods, podNs, name,ns, captureDir)
			if err != nil {
				return err
			}
		}
	}

	if failure == len(namespaces) {
		return fmt.Errorf("Failed to find any capture files")
	}

	return nil
}

func resolveDestination(args map[string]interface{}) string {
	var destination = argutils.ArgStringOrBlank(args, "--dest")
	if len(destination) == 0 {
		return "."
	}

	return destination
}

func resolveNamespaces(parsedArgs map[string]interface{}, kubeConfig string) ([]string, error) {
	var namespaces []string
	if len(argutils.ArgStringOrBlank(parsedArgs, "--namespace")) != 0 {
		namespaces = append(namespaces, parsedArgs["--namespace"].(string))
	} else if argutils.ArgBoolOrFalse(parsedArgs, "--all-namespaces") {
		output, err := common.ExecCmd(fmt.Sprintf("kubectl get ns --no-headers -o=custom-columns=NAME:..metadata.name --kubeconfig %s", kubeConfig))
		if err != nil {
			return nil, err
		}
		namespaces = strings.Split(output.String(), "\n")
	} else {
		namespaces = append(namespaces, "default")
	}

	return namespaces, nil
}

func resolveCaptureDir(parsedArgs map[string]interface{}) (string, error) {
	cf := parsedArgs["--config"].(string)
	client, err := clientmgr.NewClient(cf)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	felixConfig, err := client.FelixConfigurations().Get(ctx, "default", options.GetOptions{ResourceVersion: ""})
	if err != nil {
		return "", err
	}

	if felixConfig.Spec.CaptureDir == nil {
		const defaultCaptureDir = "/var/log/calico/pcap"
		return defaultCaptureDir, nil
	}

	return *felixConfig.Spec.CaptureDir, nil
}
