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
  calicoctl captured-packets ( copy | clean ) <NAME>
                [--config=<CONFIG>] [--namespace=<NS>] [--all-namespaces] [--dest=<DEST>]

Examples:
  # Copies capture files for packet capture from default namespace in the current directory.
  calicoctl captured-packets copy my-capture
  # Delete capture files for packet capture from default namespace still left on the system
  calicoctl captured-packets clean my-capture

Options:
  -n --namespace=<NS>      Namespace of the packet capture.
                           Uses the default namespace if not specified.
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

	var cmd = capture.NewKubectlCmd(kubeConfigPath)
	var captureCmd = capture.NewCommands(cmd, capture.NewFluentDResolver(cmd))
	var name = argutils.ArgString(parsedArgs, "<NAME>")
	var destination = argutils.ArgString(parsedArgs, "--dest")
	var isCopyCommand = argutils.ArgBoolOrFalse(parsedArgs, "copy")
	var isCleanCommand = argutils.ArgBoolOrFalse(parsedArgs, "clean")
	var results int
	var errors []error

	if isCopyCommand {
		results, errors = captureCmd.Copy(namespaces, name, captureDir, destination)
	} else if isCleanCommand {
		results, errors = captureCmd.Clean(namespaces, name, captureDir)
	}

	// in case --all-namespaces is used and we have at least 1 successful result
	// we will return 0 exit code
	if argutils.ArgBoolOrFalse(parsedArgs, "--all-namespaces") {
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
